package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/negrel/assert"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// ============================================================================
// ERROR RESPONSE STRUCTURES
// ============================================================================

// OperationErrorResponse defines comprehensive error handling strategy for operations
type OperationErrorResponse struct {
	ErrorType                 statemachine.ErrorType
	ErrorAction               statemachine.ErrorAction
	ShouldEnterErrorState     bool // Whether repository should transition to error state
	ShouldNotify              bool // Whether to send frontend notification
	ShouldPersistNotification bool // Whether to save notification to database
}

// ============================================================================
// QUEUE MANAGER
// ============================================================================

// QueueManager manages operation queues for all repositories
type QueueManager struct {
	log          *zap.SugaredLogger
	stateMachine *statemachine.RepositoryStateMachine
	db           *ent.Client
	borg         borg.Borg
	eventEmitter types.EventEmitter
	queues       map[int]*RepositoryQueue // RepoID -> Queue
	mu           sync.RWMutex

	// In-memory state tracking
	repositoryStates map[int]statemachine.RepositoryState // RepoID -> Current State
	statesMu         sync.RWMutex                         // Separate mutex for states

	// Cross-repository concurrency control
	maxHeavyOps int                      // Max heavy operations across all repositories
	activeHeavy map[int]*QueuedOperation // RepoID -> active heavy operation
	activeLight map[int]*QueuedOperation // RepoID -> active light operation
}

// NewQueueManager creates a new QueueManager with specified concurrency limits
func NewQueueManager(log *zap.SugaredLogger, stateMachine *statemachine.RepositoryStateMachine, maxHeavyOps int) *QueueManager {
	return &QueueManager{
		log:              log,
		stateMachine:     stateMachine,
		queues:           make(map[int]*RepositoryQueue),
		repositoryStates: make(map[int]statemachine.RepositoryState),
		maxHeavyOps:      maxHeavyOps,
		activeHeavy:      make(map[int]*QueuedOperation),
		activeLight:      make(map[int]*QueuedOperation),
	}
}

// Init initializes the queue manager with database and borg clients
func (qm *QueueManager) Init(db *ent.Client, borgClient borg.Borg, eventEmitter types.EventEmitter) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.db = db
	qm.borg = borgClient
	qm.eventEmitter = eventEmitter
}

// GetRepositoryState returns the current state of a repository (defaults to idle if not set)
func (qm *QueueManager) GetRepositoryState(repoID int) statemachine.RepositoryState {
	qm.statesMu.RLock()
	defer qm.statesMu.RUnlock()

	if state, exists := qm.repositoryStates[repoID]; exists {
		return state
	}

	// Default to idle state for new repositories
	return statemachine.NewRepositoryStateIdle(statemachine.Idle{})
}

// setRepositoryState updates the current state of a repository in memory
func (qm *QueueManager) setRepositoryState(repoID int, state statemachine.RepositoryState) {
	qm.statesMu.Lock()
	defer qm.statesMu.Unlock()
	qm.repositoryStates[repoID] = state

	// Emit event for state change
	qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventRepoStateChangedString(repoID))

	// Emit additional events for specific states
	switch statemachine.GetRepositoryStateType(state) {
	case statemachine.RepositoryStateTypeBackingUp:
		data := state.(statemachine.BackingUpVariant)()
		qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventBackupStateChangedString(data.Data.BackupID))
	case statemachine.RepositoryStateTypePruning:
		data := state.(statemachine.PruningVariant)()
		qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventPruneStateChangedString(data.BackupID))
	case statemachine.RepositoryStateTypeRefreshing:
		qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventArchivesChangedString(repoID))
	case statemachine.RepositoryStateTypeIdle,
		statemachine.RepositoryStateTypeQueued,
		statemachine.RepositoryStateTypeDeleting,
		statemachine.RepositoryStateTypeChecking,
		statemachine.RepositoryStateTypeMounting,
		statemachine.RepositoryStateTypeMounted,
		statemachine.RepositoryStateTypeError:
	}
}

// GetQueue returns the queue for a specific repository, creating it if it doesn't exist
func (qm *QueueManager) GetQueue(repoID int) *RepositoryQueue {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if queue, exists := qm.queues[repoID]; exists {
		return queue
	}

	// Create new queue
	queue := NewRepositoryQueue(repoID)
	qm.queues[repoID] = queue
	return queue
}

// AddOperation adds an operation to the specified repository queue
func (qm *QueueManager) AddOperation(repoID int, op *QueuedOperation) (string, error) {
	// Get repository queue
	queue := qm.GetQueue(repoID)

	// Check immediate flag requirements
	if op.Immediate {
		// Light operations can start if no operation is currently active
		// Heavy operations need fully idle repository
		opWeight := statemachine.GetOperationWeight(op.Operation)

		if queue.HasActiveOperation() {
			return "", fmt.Errorf("cannot start immediate operation: repository has active operation")
		}

		if opWeight == statemachine.WeightHeavy && qm.HasQueuedOperations(repoID) {
			return "", fmt.Errorf("cannot start immediate heavy operation: repository has queued operations")
		}

		if !qm.CanStartOperation(repoID, op) {
			return "", fmt.Errorf("cannot start immediate operation: concurrency limits exceeded")
		}
	}

	// Add operation to queue (handles idempotency internally)
	operationID := queue.AddOperation(op)

	// Attempt to start operation if possible
	err := qm.processQueue(repoID)
	if err != nil {
		// Emit repo changed event even on error
		qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventRepoStateChangedString(repoID))
		return operationID, err
	}

	// If operation wasn't started, ensure repository state reflects queued status
	// Only transition to queued state if there's NO active operation
	if !queue.HasActiveOperation() && qm.HasQueuedOperations(repoID) {
		// No active operation but operations are queued (concurrency limit reached)
		currentState := qm.GetRepositoryState(repoID)
		if _, isQueued := currentState.(statemachine.QueuedVariant); !isQueued {
			// Only transition to queued state if next operation is heavy
			nextOp := queue.GetNext()
			queueLength := queue.GetQueueLength()
			if nextOp != nil && statemachine.GetOperationWeight(nextOp.Operation) == statemachine.WeightHeavy {
				targetState := statemachine.CreateQueuedState(nextOp.Operation, queueLength)
				err = qm.stateMachine.Transition(repoID, currentState, targetState)
				if err == nil {
					qm.setRepositoryState(repoID, targetState)
				}
			}
		}
	}

	// Emit repo changed event AFTER state update
	qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventRepoStateChangedString(repoID))

	return operationID, nil
}

// RemoveOperation removes an operation from tracking
func (qm *QueueManager) RemoveOperation(repoID int, operationID string) error {
	queue := qm.GetQueue(repoID)

	// Get operation details before removing it
	operation := queue.GetOperationByID(operationID)
	if operation == nil {
		// Operation doesn't exist, nothing to remove
		return nil
	}

	// Check if this is the active operation and clean up global tracking
	activeOp := queue.GetActive()
	if activeOp != nil && activeOp.ID == operationID {
		// This is an active operation, remove from global tracking maps
		weight := statemachine.GetOperationWeight(operation.Operation)

		qm.mu.Lock()
		if weight == statemachine.WeightHeavy {
			delete(qm.activeHeavy, repoID)
			qm.log.Debugw("Removed operation from activeHeavy tracking",
				"repoID", repoID, "operationID", operationID, "operationType", fmt.Sprintf("%T", operation.Operation))
		} else {
			delete(qm.activeLight, repoID)
			qm.log.Debugw("Removed operation from activeLight tracking",
				"repoID", repoID, "operationID", operationID, "operationType", fmt.Sprintf("%T", operation.Operation))
		}
		qm.mu.Unlock()
	}

	// Remove from repository queue
	err := queue.RemoveOperation(operationID)
	if err != nil {
		return err
	}

	// Emit event for archive-affecting operations before removal
	operationType := statemachine.GetOperationType(operation.Operation)
	switch operationType {
	// Archive-affecting operations - emit event
	case statemachine.OperationTypeBackup,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeExaminePrune:
		qm.eventEmitter.EmitEvent(application.Get().Context(), types.EventArchivesChangedString(repoID))

	// Non-archive-affecting operations - no action
	case statemachine.OperationTypeDelete,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypePrune,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive:
	// No action needed
	default:
		assert.Fail("Unknown operation")
	}

	// Check if queue is now empty and transition to idle if needed
	if !queue.HasActiveOperation() && queue.GetQueueLength() == 0 {
		currentState := qm.GetRepositoryState(repoID)
		if _, isQueued := currentState.(statemachine.QueuedVariant); isQueued {
			targetState := statemachine.CreateIdleState()
			err := qm.stateMachine.Transition(repoID, currentState, targetState)
			if err == nil {
				qm.setRepositoryState(repoID, targetState)
			}
		}
	}

	return nil
}

// CanStartOperation checks if an operation can start based on concurrency limits
func (qm *QueueManager) CanStartOperation(repoID int, op *QueuedOperation) bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Check if repository already has any active operation
	if _, hasHeavy := qm.activeHeavy[repoID]; hasHeavy {
		return false
	}
	if _, hasLight := qm.activeLight[repoID]; hasLight {
		return false
	}

	// Check operation weight and global limits
	weight := statemachine.GetOperationWeight(op.Operation)
	if weight == statemachine.WeightHeavy {
		// Check global heavy operation limit
		return len(qm.activeHeavy) < qm.maxHeavyOps
	}

	// Light operations can always start if no operation active on repo
	return true
}

// StartOperation marks an operation as active and updates state
func (qm *QueueManager) StartOperation(ctx context.Context, repoID int, operationID string) error {
	queue := qm.GetQueue(repoID)

	// Get operation before moving
	op := queue.GetOperationByID(operationID)
	if op == nil {
		return fmt.Errorf("operation %s not found in repository %d queue", operationID, repoID)
	}

	// Check concurrency limits
	if !qm.CanStartOperation(repoID, op) {
		return fmt.Errorf("cannot start operation %s: concurrency limits exceeded", operationID)
	}

	// Move operation from queue to active
	err := queue.MoveToActive(operationID)
	if err != nil {
		return fmt.Errorf("failed to move operation to active: %w", err)
	}

	// Update concurrency tracking
	qm.mu.Lock()
	weight := statemachine.GetOperationWeight(op.Operation)
	if weight == statemachine.WeightHeavy {
		qm.activeHeavy[repoID] = op
	} else {
		qm.activeLight[repoID] = op
	}
	qm.mu.Unlock()

	// Transition repository state via state machine
	// Get current state from in-memory tracking
	currentState := qm.GetRepositoryState(repoID)

	// Get target state for this operation
	targetState, err := qm.getTargetStateForOperation(ctx, op)
	if err != nil {
		return fmt.Errorf("failed to determine target state for operation: %w", err)
	}

	// Validate state transition
	err = qm.stateMachine.Transition(repoID, currentState, targetState)
	if err != nil {
		return fmt.Errorf("failed to transition repository %d from %T to %T: %w", repoID, currentState, targetState, err)
	}

	// Update repository state in memory
	qm.setRepositoryState(repoID, targetState)

	// Start actual operation execution in background goroutine
	go func() {
		// Create operation executor
		executor := qm.newBorgOperationExecutor(repoID, operationID)

		// Execute the operation
		operationCtx := statemachine.GetCancelCtxOrDefault(application.Get().Context(), targetState)
		status, err := executor.Execute(operationCtx, op.Operation)
		if err != nil {
			// System error (e.g., backup profile not found)
			qm.log.Errorw("System error during operation execution",
				"repoID", repoID,
				"operationID", operationID,
				"operationType", fmt.Sprintf("%T", op.Operation),
				"error", err.Error())

			// Complete operation with system error
			systemErrorResponse := &OperationErrorResponse{
				ErrorType:                 statemachine.ErrorTypeGeneral,
				ErrorAction:               statemachine.ErrorActionNone,
				ShouldEnterErrorState:     true,
				ShouldNotify:              true,
				ShouldPersistNotification: true,
			}
			if completeErr := qm.CompleteOperation(application.Get().Context(), repoID, operationID, op.Operation, systemErrorResponse, fmt.Sprintf("System error: %v", err)); completeErr != nil {
				qm.log.Warnw("Failed to complete operation after system error",
					"repoID", repoID,
					"operationID", operationID,
					"systemError", err.Error(),
					"completionError", completeErr.Error())
			}
			return
		}

		if status.HasBeenCanceled {
			// CANCELLATION BRANCH - Operation was canceled, clean up properly
			qm.log.Infow("Borg operation was canceled",
				"repoID", repoID,
				"operationID", operationID,
				"operationType", fmt.Sprintf("%T", op.Operation))

			// Complete operation with cancellation (no error)
			if completeErr := qm.CompleteOperation(application.Get().Context(), repoID, operationID, op.Operation, nil, ""); completeErr != nil {
				qm.log.Warnw("Failed to complete canceled operation",
					"repoID", repoID,
					"operationID", operationID,
					"completionError", completeErr.Error())
			}

		} else if status.HasError() {
			// ERROR BRANCH - Enhanced with conditional notification creation
			qm.log.Errorw("Borg operation failed",
				"repoID", repoID,
				"operationID", operationID,
				"operationType", fmt.Sprintf("%T", op.Operation),
				"error", status.Error.Message,
				"category", status.Error.Category,
				"exitCode", status.Error.ExitCode)

			// Complete operation with failure using operation-aware error mapping
			errorResponse := qm.mapOperationErrorResponse(ctx, status.Error, repoID, op.Operation)
			if completeErr := qm.CompleteOperation(application.Get().Context(), repoID, operationID, op.Operation, &errorResponse, status.Error.Message); completeErr != nil {
				// Log completion error (system issue, not user-facing)
				qm.log.Warnw("Failed to complete failed operation",
					"repoID", repoID,
					"operationID", operationID,
					"completionError", completeErr.Error())
			}

		} else if status.IsCompletedWithSuccess() {
			// 2. SUCCESS BRANCH - Combined warning + pure success handling
			if status.HasWarning() {
				// Log as warning and create notifications for warnings
				qm.log.Warnw("Borg operation completed with warning",
					"repoID", repoID,
					"operationID", operationID,
					"operationType", fmt.Sprintf("%T", op.Operation),
					"warning", status.Warning.Message,
					"category", status.Warning.Category,
					"exitCode", status.Warning.ExitCode)

				// Only create notifications for certain operations
				if qm.shouldCreateNotification(op.Operation) {
					qm.createWarningNotification(ctx, repoID, operationID, status, op.Operation)
				}
			} else {
				// Log as info for pure success
				qm.log.Infow("Borg operation completed successfully",
					"repoID", repoID,
					"operationID", operationID,
					"operationType", fmt.Sprintf("%T", op.Operation))
			}

			// Complete operation with success
			if completeErr := qm.CompleteOperation(application.Get().Context(), repoID, operationID, op.Operation, nil, ""); completeErr != nil {
				// Log completion error (system issue, not user-facing)
				qm.log.Warnw("Failed to complete successful operation",
					"repoID", repoID,
					"operationID", operationID,
					"completionError", completeErr.Error())
			}
		}
	}()

	return nil
}

// CompleteOperation marks an operation as completed with comprehensive error handling
func (qm *QueueManager) CompleteOperation(ctx context.Context, repoID int, operationID string, operation statemachine.Operation, errorResponse *OperationErrorResponse, errorMsg string) error {
	queue := qm.GetQueue(repoID)

	// Get active operation to determine weight
	activeOp := queue.GetActive()
	if activeOp == nil || activeOp.ID != operationID {
		return fmt.Errorf("operation %s is not currently active for repository %d", operationID, repoID)
	}

	weight := statemachine.GetOperationWeight(activeOp.Operation)

	// Handle error notifications based on error response
	if errorResponse != nil && errorResponse.ShouldNotify && errorMsg != "" {
		if errorResponse.ShouldPersistNotification {
			// Create persistent database notification (for backup/prune operations)
			qm.createErrorNotification(ctx, repoID, operationID, &borgtypes.Status{
				Error: &borgtypes.BorgError{Message: errorMsg},
			}, operation)
		} else {
			// Send frontend-only notification (for rename/delete operations)
			qm.sendFrontendNotification(ctx, repoID, operationID, errorMsg, operation)
		}
	}

	// Remove from active tracking
	qm.mu.Lock()
	if weight == statemachine.WeightHeavy {
		delete(qm.activeHeavy, repoID)
	} else {
		delete(qm.activeLight, repoID)
	}
	qm.mu.Unlock()

	// Update operation status and complete in queue
	err := queue.CompleteActive(errorMsg)
	if err != nil {
		return fmt.Errorf("failed to complete operation: %w", err)
	}

	// Transition repository state via state machine
	// Get current state from in-memory tracking
	currentState := qm.GetRepositoryState(repoID)

	// Determine target state based on completion and queue status
	var targetState statemachine.RepositoryState

	if errorResponse != nil && errorResponse.ShouldEnterErrorState {
		// On failure, transition to error state
		targetState = statemachine.CreateErrorState(errorResponse.ErrorType, errorMsg, errorResponse.ErrorAction)
	} else {
		// On success, determine next state based on queue status and completed operation
		targetState, err = qm.getCompletionStateForRepository(repoID, activeOp)
		if err != nil {
			return fmt.Errorf("failed to determine completion state: %w", err)
		}
	}

	// Only perform state transition if state types differ
	// This avoids invalid transitions like Queued -> Queued when queue data changes
	currentStateType := statemachine.GetRepositoryStateType(currentState)
	targetStateType := statemachine.GetRepositoryStateType(targetState)

	if currentStateType != targetStateType {
		// Validate and perform state transition
		err = qm.stateMachine.Transition(repoID, currentState, targetState)
		if err != nil {
			return fmt.Errorf("failed to transition repository %d from %T to %T: %w", repoID, currentState, targetState, err)
		}
	}

	// Always update repository state in memory (even if types match, data like queue length may differ)
	qm.setRepositoryState(repoID, targetState)

	// Attempt to start next queued operation for this repo
	err = qm.processQueue(repoID)
	if err != nil {
		return err
	}

	// If we completed a heavy operation, try to start waiting heavy operations on other repos
	if weight == statemachine.WeightHeavy {
		for otherRepoID := range qm.queues {
			if otherRepoID != repoID {
				err = qm.processQueue(otherRepoID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// CancelOperation cancels a queued or running operation
func (qm *QueueManager) CancelOperation(repoID int, operationID string) error {
	queue := qm.GetQueue(repoID)

	// Check if it's the active operation
	activeOp := queue.GetActive()
	if activeOp != nil && activeOp.ID == operationID {
		// Cancel active operation
		currentState := qm.GetRepositoryState(repoID)

		// Check if operation can be canceled via context
		if cancel, hasCancel := statemachine.GetCancel(currentState); hasCancel {
			qm.log.Infow("Triggering cancellation for active operation",
				"repoID", repoID,
				"operationID", operationID,
				"stateType", fmt.Sprintf("%T", currentState))

			// Trigger cancellation and return
			// The goroutine will handle all cleanup after borg process terminates
			cancel()
			return nil
		}

		// Operation doesn't support cancellation - proceed with immediate cleanup
		qm.log.Infow("Operation doesn't support cancellation, performing immediate cleanup",
			"repoID", repoID,
			"operationID", operationID,
			"stateType", fmt.Sprintf("%T", currentState))

		// Complete operation with no error
		if completeErr := qm.CompleteOperation(application.Get().Context(), repoID, operationID, activeOp.Operation, nil, ""); completeErr != nil {
			return fmt.Errorf("failed to complete non-cancelable operation: %w", completeErr)
		}

		return nil
	}

	// Remove from queue (handles archive events and state transitions)
	return qm.RemoveOperation(repoID, operationID)
}

// GetOperation retrieves an operation by ID
func (qm *QueueManager) GetOperation(operationID string) (*QueuedOperation, error) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Search across all repository queues
	for _, queue := range qm.queues {
		if op := queue.GetOperationByID(operationID); op != nil {
			return op, nil
		}
	}

	return nil, fmt.Errorf("operation %s not found in any queue", operationID)
}

// UpdateBackupProgress updates the progress of a backup operation
func (qm *QueueManager) UpdateBackupProgress(ctx context.Context, operationID string, progress borgtypes.BackupProgress) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Search across all repository queues
	for _, queue := range qm.queues {
		op := queue.GetOperationByID(operationID)
		if op != nil {
			// Check if this is a backup operation
			if backupVariant, isBackup := op.Operation.(statemachine.BackupVariant); isBackup {

				// Update the operation's backup data with new progress
				backupData := backupVariant()
				backupData.Progress = &progress

				// Create a new BackupVariant with updated data
				updatedOperation := statemachine.NewOperationBackup(backupData)
				op.Operation = updatedOperation

				qm.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(backupData.BackupID))

				return nil
			}
			return fmt.Errorf("operation %s is not a backup operation", operationID)
		}
	}

	return fmt.Errorf("operation %s not found in any queue", operationID)
}

// GetQueuedOperations returns only queued operations for a repository (excluding active), optionally filtered by operation type
func (qm *QueueManager) GetQueuedOperations(repoID int, operationType *statemachine.OperationType) ([]*QueuedOperation, error) {
	queue := qm.GetQueue(repoID)
	return queue.GetQueuedOperations(operationType), nil
}

// GetOperationsByStatus returns operations filtered by status for a repository
func (qm *QueueManager) GetOperationsByStatus(repoID int, status OperationStatus) ([]*QueuedOperation, error) {
	queue := qm.GetQueue(repoID)
	return queue.GetOperationsByStatus(status), nil
}

// GetActiveOperation returns the currently active operation for a repository, optionally filtered by operation type
func (qm *QueueManager) GetActiveOperation(repoID int, operationType *statemachine.OperationType) *QueuedOperation {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Check heavy operations first
	if op, exists := qm.activeHeavy[repoID]; exists {
		if operationType == nil || statemachine.GetOperationType(op.Operation) == *operationType {
			return op
		}
	}

	// Check light operations
	if op, exists := qm.activeLight[repoID]; exists {
		if operationType == nil || statemachine.GetOperationType(op.Operation) == *operationType {
			return op
		}
	}

	return nil
}

// GetActiveOperations returns currently active operations across all repositories
func (qm *QueueManager) GetActiveOperations() map[int]*QueuedOperation {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make(map[int]*QueuedOperation)

	// Add heavy operations
	for repoID, op := range qm.activeHeavy {
		result[repoID] = op
	}

	// Add light operations (only if no heavy for same repo)
	for repoID, op := range qm.activeLight {
		if _, hasHeavy := qm.activeHeavy[repoID]; !hasHeavy {
			result[repoID] = op
		}
	}

	return result
}

// GetHeavyOperationCount returns the current number of active heavy operations
func (qm *QueueManager) GetHeavyOperationCount() int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return len(qm.activeHeavy)
}

// HasQueuedOperations checks if a repository has pending operations
func (qm *QueueManager) HasQueuedOperations(repoID int) bool {
	qm.mu.RLock()
	queue, exists := qm.queues[repoID]
	qm.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if there are any queued operations (excluding active)
	return queue.GetQueueLength() > 0
}

// processQueue attempts to start the next operation in a repository queue
func (qm *QueueManager) processQueue(repoID int) error {
	queue := qm.GetQueue(repoID)

	// Check if repository already has an active operation
	if queue.HasActiveOperation() {
		return nil
	}

	// Get next operation from queue
	nextOp := queue.GetNext()
	if nextOp == nil {
		return nil
	}

	// Check concurrency limits
	if !qm.CanStartOperation(repoID, nextOp) {
		return nil
	}

	// Start the operation
	err := qm.StartOperation(application.Get().Context(), repoID, nextOp.ID)
	if err != nil {
		// Log error and mark operation as failed
		qm.log.Warnw("Failed to start queued operation",
			"repoID", repoID,
			"operationID", nextOp.ID,
			"operationType", fmt.Sprintf("%T", nextOp.Operation),
			"error", err)

		// Mark the operation as failed and remove it from queue
		removeErr := qm.RemoveOperation(repoID, nextOp.ID)
		if removeErr != nil {
			qm.log.Errorw("Failed to remove operation",
				"repoID", repoID, "operationID",
				nextOp.ID, "operationType",
				fmt.Sprintf("%T", nextOp.Operation),
				"error", err)
		}
	}
	return err
}

// expireOldOperations removes expired operations from all queues
func (qm *QueueManager) expireOldOperations() {
	now := time.Now()

	qm.mu.RLock()
	queues := make(map[int]*RepositoryQueue)
	for repoID, queue := range qm.queues {
		queues[repoID] = queue
	}
	qm.mu.RUnlock()

	// Process each queue
	for repoID, queue := range queues {
		expiredIDs := queue.ExpireOldOperations(now)

		// Try to start next operation if any were expired
		if len(expiredIDs) > 0 {
			qm.processQueue(repoID)
		}
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getTargetStateForOperation maps an operation to its corresponding active state
func (qm *QueueManager) getTargetStateForOperation(ctx context.Context, op *QueuedOperation) (statemachine.RepositoryState, error) {
	switch statemachine.GetOperationType(op.Operation) {
	case statemachine.OperationTypeBackup:
		backupVariant := op.Operation.(statemachine.BackupVariant)
		backupData := backupVariant()
		return statemachine.CreateBackingUpState(ctx, backupData), nil

	case statemachine.OperationTypePrune:
		pruneVariant := op.Operation.(statemachine.PruneVariant)
		pruneData := pruneVariant()
		return statemachine.CreatePruningState(ctx, pruneData.BackupID), nil

	case statemachine.OperationTypeDelete:
		return statemachine.CreateDeletingState(ctx, 0), nil // Repository delete, no specific archive

	case statemachine.OperationTypeArchiveRefresh:
		return statemachine.CreateRefreshingState(ctx), nil

	case statemachine.OperationTypeCheck:
		return statemachine.CreateCheckingState(ctx), nil

	case statemachine.OperationTypeArchiveDelete:
		deleteVariant := op.Operation.(statemachine.ArchiveDeleteVariant)
		deleteData := deleteVariant()
		return statemachine.CreateDeletingState(ctx, deleteData.ArchiveID), nil

	case statemachine.OperationTypeArchiveRename:
		// Archive rename is a lightweight operation, treat as refreshing
		return statemachine.CreateRefreshingState(ctx), nil

	case statemachine.OperationTypeMount:
		return statemachine.CreateMountingState(nil), nil

	case statemachine.OperationTypeMountArchive:
		mountVariant := op.Operation.(statemachine.MountArchiveVariant)
		mountData := mountVariant()
		return statemachine.CreateMountingState(&mountData.ArchiveID), nil

	case statemachine.OperationTypeUnmount:
		// Unmount operations transition through refreshing state temporarily
		return statemachine.CreateRefreshingState(ctx), nil

	case statemachine.OperationTypeUnmountArchive:
		// Unmount archive operations transition through refreshing state temporarily
		return statemachine.CreateRefreshingState(ctx), nil

	case statemachine.OperationTypeExaminePrune:
		// ExaminePrune is a lightweight operation, treat as refreshing
		return statemachine.CreateRefreshingState(ctx), nil

	case statemachine.OperationTypeArchiveComment:
		// Archive comment is a lightweight operation, treat as refreshing
		return statemachine.CreateRefreshingState(ctx), nil

	default:
		assert.Fail("Unhandled OperationType in getTargetStateForOperation")
		return nil, fmt.Errorf("unknown operation type: %T", op.Operation)
	}
}

// getCompletionStateForRepository determines the target state when an operation completes successfully
func (qm *QueueManager) getCompletionStateForRepository(repoID int, completedOp *QueuedOperation) (statemachine.RepositoryState, error) {
	// Check if there are more operations queued
	if qm.HasQueuedOperations(repoID) {
		// Get queue info for state data
		queue := qm.GetQueue(repoID)
		nextOp := queue.GetNext()
		queueLength := queue.GetQueueLength()

		if nextOp != nil {
			return statemachine.CreateQueuedState(nextOp.Operation, queueLength), nil
		}
	}

	// No more operations - determine final state based on completed operation
	switch statemachine.GetOperationType(completedOp.Operation) {
	case statemachine.OperationTypeMount:
		// Mount operations transition to Mounted state
		mountVariant := completedOp.Operation.(statemachine.MountVariant)
		mountData := mountVariant()
		return statemachine.CreateMountedState([]statemachine.MountInfo{
			{
				MountType: statemachine.MountTypeRepository,
				MountPath: mountData.MountPath,
			},
		}), nil

	case statemachine.OperationTypeMountArchive:
		// Mount archive operations transition to Mounted state
		mountVariant := completedOp.Operation.(statemachine.MountArchiveVariant)
		mountData := mountVariant()
		archiveID := mountData.ArchiveID
		return statemachine.CreateMountedState([]statemachine.MountInfo{
			{
				MountType: statemachine.MountTypeArchive,
				ArchiveID: &archiveID,
				MountPath: mountData.MountPath,
			},
		}), nil

	case statemachine.OperationTypeBackup,
		statemachine.OperationTypePrune,
		statemachine.OperationTypeDelete,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:
		// All other operations return to idle
		return statemachine.CreateIdleState(), nil

	default:
		assert.Fail("Unhandled OperationType in getCompletionStateForRepository")
		return statemachine.CreateIdleState(), nil
	}
}

// createErrorNotification creates an error notification in the database for a failed borg operation
func (qm *QueueManager) createErrorNotification(ctx context.Context, repoID int, operationID string, status *borgtypes.Status, operation statemachine.Operation) {
	// Determine notification type based on operation
	notificationType := qm.getErrorNotificationType(operation)

	// Get backup profile ID from operation
	backupProfileID := qm.getBackupProfileIDFromOperation(operation)

	// Create user-friendly error message with exit code
	message := fmt.Sprintf("%s (Exit Code: %d)", status.Error.Message, status.Error.ExitCode)

	// Create notification in database
	_, err := qm.db.Notification.Create().
		SetMessage(message).
		SetType(notificationType).
		SetRepositoryID(repoID).
		SetBackupProfileID(backupProfileID).
		Save(ctx)

	if err != nil {
		qm.log.Errorw("Failed to create error notification",
			"error", err.Error(),
			"repoID", repoID,
			"operationID", operationID,
			"borgError", status.Error.Message)
	} else {
		qm.log.Infow("Created error notification in database",
			"repoID", repoID,
			"operationID", operationID,
			"notificationType", notificationType,
			"errorCategory", status.Error.Category,
			"exitCode", status.Error.ExitCode)
	}
}

// createWarningNotification creates a warning notification in the database for a borg operation with warnings
func (qm *QueueManager) createWarningNotification(ctx context.Context, repoID int, operationID string, status *borgtypes.Status, operation statemachine.Operation) {
	// Determine notification type based on operation
	notificationType := qm.getWarningNotificationType(operation)

	// Get backup profile ID from operation
	backupProfileID := qm.getBackupProfileIDFromOperation(operation)

	// Create user-friendly warning message with exit code
	message := fmt.Sprintf("%s (Exit Code: %d)", status.Warning.Message, status.Warning.ExitCode)

	// Create notification in database
	_, err := qm.db.Notification.Create().
		SetMessage(message).
		SetType(notificationType).
		SetRepositoryID(repoID).
		SetBackupProfileID(backupProfileID).
		Save(ctx)

	if err != nil {
		qm.log.Warnw("Failed to create warning notification",
			"error", err.Error(),
			"repoID", repoID,
			"operationID", operationID,
			"borgWarning", status.Warning.Message)
	} else {
		qm.log.Infow("Created warning notification in database",
			"repoID", repoID,
			"operationID", operationID,
			"notificationType", notificationType,
			"warningCategory", status.Warning.Category,
			"exitCode", status.Warning.ExitCode)
	}
}

// sendFrontendNotification sends a frontend-only notification without database persistence
func (qm *QueueManager) sendFrontendNotification(ctx context.Context, repoID int, operationID string, errorMsg string, operation statemachine.Operation) {
	// Create user-friendly error message based on operation type
	var message string
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeArchiveRename:
		archiveRename := operation.(statemachine.ArchiveRenameVariant)()
		archiveData := archiveRename
		message = fmt.Sprintf("Failed to rename archive '%s': %s", archiveData.Name, errorMsg)
	case statemachine.OperationTypeArchiveDelete:
		archiveDelete := operation.(statemachine.ArchiveDeleteVariant)()
		archiveData := archiveDelete
		message = fmt.Sprintf("Failed to delete archive (ID %d): %s", archiveData.ArchiveID, errorMsg)
	case statemachine.OperationTypeArchiveRefresh:
		message = fmt.Sprintf("Failed to refresh archives: %s", errorMsg)
	case statemachine.OperationTypeBackup:
		message = fmt.Sprintf("Failed to backup: %s", errorMsg)
	case statemachine.OperationTypePrune:
		message = fmt.Sprintf("Failed to prune repository: %s", errorMsg)
	case statemachine.OperationTypeDelete:
		message = fmt.Sprintf("Failed to delete repository: %s", errorMsg)
	case statemachine.OperationTypeMount:
		message = fmt.Sprintf("Failed to mount repository: %s", errorMsg)
	case statemachine.OperationTypeMountArchive:
		message = fmt.Sprintf("Failed to mount archive: %s", errorMsg)
	case statemachine.OperationTypeUnmount:
		message = fmt.Sprintf("Failed to unmount repository: %s", errorMsg)
	case statemachine.OperationTypeUnmountArchive:
		message = fmt.Sprintf("Failed to unmount archive: %s", errorMsg)
	case statemachine.OperationTypeExaminePrune:
		message = fmt.Sprintf("Failed to examine prune: %s", errorMsg)
	case statemachine.OperationTypeArchiveComment:
		message = fmt.Sprintf("Failed to update archive comment: %s", errorMsg)
	case statemachine.OperationTypeCheck:
		checkOp := operation.(statemachine.CheckVariant)()
		if checkOp.QuickVerification {
			message = fmt.Sprintf("Repository quick check failed: %s", errorMsg)
		} else {
			message = fmt.Sprintf("Repository full check failed: %s", errorMsg)
		}
	default:
		assert.Fail("Unhandled OperationType in sendFrontendNotification")
	}

	// Log the notification for debugging
	qm.log.Infow("Sending frontend notification",
		"repoID", repoID,
		"operationID", operationID,
		"operationType", statemachine.GetOperationType(operation),
		"message", message)

	qm.eventEmitter.EmitEvent(ctx, types.EventOperationErrorOccurred.String(), message)
}

// getErrorNotificationType determines the notification type for error cases based on operation type
// This method should only be called for backup and prune operations
func (qm *QueueManager) getErrorNotificationType(operation statemachine.Operation) notification.Type {
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup:
		return notification.TypeFailedBackupRun
	case statemachine.OperationTypePrune:
		return notification.TypeFailedPruningRun
	case statemachine.OperationTypeCheck:
		checkOp := operation.(statemachine.CheckVariant)()
		if checkOp.QuickVerification {
			return notification.TypeFailedQuickCheck
		} else {
			return notification.TypeFailedFullCheck
		}
	case statemachine.OperationTypeDelete,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeExaminePrune,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for error notification",
			"operationType", fmt.Sprintf("%T", operation))
		return notification.TypeFailedBackupRun // Fallback for safety
	default:
		assert.Fail("Unhandled OperationType in getErrorNotificationType")
		return notification.TypeFailedBackupRun // Fallback for safety
	}
}

// getWarningNotificationType determines the notification type for warning cases based on operation type
// This method should only be called for backup and prune operations
func (qm *QueueManager) getWarningNotificationType(operation statemachine.Operation) notification.Type {
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup:
		return notification.TypeWarningBackupRun
	case statemachine.OperationTypePrune:
		return notification.TypeWarningPruningRun
	case statemachine.OperationTypeCheck:
		checkOp := operation.(statemachine.CheckVariant)()
		if checkOp.QuickVerification {
			return notification.TypeWarningQuickCheck
		} else {
			return notification.TypeWarningFullCheck
		}
	case statemachine.OperationTypeDelete,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeExaminePrune,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for warning notification",
			"operationType", fmt.Sprintf("%T", operation))
		return notification.TypeWarningBackupRun // Fallback for safety
	default:
		assert.Fail("Unhandled OperationType in getWarningNotificationType")
		return notification.TypeWarningBackupRun // Fallback for safety
	}
}

// getBackupProfileIDFromOperation extracts the backup profile ID from operation data
func (qm *QueueManager) getBackupProfileIDFromOperation(operation statemachine.Operation) int {
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup:
		backupVariant := operation.(statemachine.BackupVariant)
		backupData := backupVariant()
		return backupData.BackupID.BackupProfileId
	case statemachine.OperationTypePrune:
		pruneVariant := operation.(statemachine.PruneVariant)
		pruneData := pruneVariant()
		return pruneData.BackupID.BackupProfileId
	case statemachine.OperationTypeExaminePrune:
		examinePruneVariant := operation.(statemachine.ExaminePruneVariant)
		examinePruneData := examinePruneVariant()
		return examinePruneData.BackupID.BackupProfileId
	case statemachine.OperationTypeCheck:
		// Check is repository-wide, get first backup profile for the repository
		checkVariant := operation.(statemachine.CheckVariant)
		checkData := checkVariant()
		// Query first backup profile for this repository
		backupProfile, err := qm.db.BackupProfile.Query().
			Where(backupprofile.HasRepositoriesWith(repository.ID(checkData.RepositoryID))).
			First(context.Background())
		if err != nil {
			qm.log.Errorw("Failed to get backup profile for check notification",
				"repositoryID", checkData.RepositoryID,
				"error", err.Error())
			return 0
		}
		return backupProfile.ID
	case statemachine.OperationTypeDelete,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for backup profile",
			"operationType", fmt.Sprintf("%T", operation))
		return 0
	default:
		assert.Fail("Unhandled OperationType in getBackupProfileIDFromOperation")
		return 0
	}
}

// shouldCreateNotification determines if error/warning notifications should be created for this operation type
func (qm *QueueManager) shouldCreateNotification(operation statemachine.Operation) bool {
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup, statemachine.OperationTypePrune, statemachine.OperationTypeCheck:
		return true
	case statemachine.OperationTypeDelete,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeExaminePrune,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive:
		return false
	default:
		assert.Fail("Unhandled OperationType in shouldCreateNotification")
		return false
	}
}

// ============================================================================
// OPERATION EXECUTOR
// ============================================================================

// OperationExecutor handles the actual execution of repository operations
type OperationExecutor interface {
	Execute(ctx context.Context, operation statemachine.Operation) (*borgtypes.Status, error)
}

type progressUpdater interface {
	UpdateBackupProgress(ctx context.Context, operationID string, progress borgtypes.BackupProgress) error
}

// borgOperationExecutor implements OperationExecutor using borg commands
type borgOperationExecutor struct {
	log             *zap.SugaredLogger
	db              *ent.Client
	borgClient      borg.Borg
	eventEmitter    types.EventEmitter
	repoID          int
	operationID     string
	progressUpdater progressUpdater
}

// newBorgOperationExecutor creates a new borg operation executor
func (qm *QueueManager) newBorgOperationExecutor(repoID int, operationID string) OperationExecutor {
	return &borgOperationExecutor{
		borgClient:      qm.borg,
		db:              qm.db,
		log:             qm.log,
		eventEmitter:    qm.eventEmitter,
		repoID:          repoID,
		operationID:     operationID,
		progressUpdater: qm,
	}
}

// Execute implements OperationExecutor.Execute
func (e *borgOperationExecutor) Execute(ctx context.Context, operation statemachine.Operation) (*borgtypes.Status, error) {
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup:
		return e.executeBackup(ctx, operation.(statemachine.BackupVariant))
	case statemachine.OperationTypePrune:
		pruneVariant := operation.(statemachine.PruneVariant)
		pruneData := pruneVariant()
		return e.executePrune(ctx, pruneData.BackupID, false, nil, nil, false)
	case statemachine.OperationTypeDelete:
		return e.executeRepositoryDelete(ctx, operation.(statemachine.DeleteVariant))
	case statemachine.OperationTypeArchiveDelete:
		return e.executeArchiveDelete(ctx, operation.(statemachine.ArchiveDeleteVariant))
	case statemachine.OperationTypeArchiveRefresh:
		return e.executeArchiveRefresh(ctx, operation.(statemachine.ArchiveRefreshVariant))
	case statemachine.OperationTypeCheck:
		return e.executeCheck(ctx, operation.(statemachine.CheckVariant))
	case statemachine.OperationTypeArchiveRename:
		return e.executeArchiveRename(ctx, operation.(statemachine.ArchiveRenameVariant))
	case statemachine.OperationTypeArchiveComment:
		return e.executeArchiveComment(ctx, operation.(statemachine.ArchiveCommentVariant))
	case statemachine.OperationTypeMount:
		return e.executeMount(ctx, operation.(statemachine.MountVariant))
	case statemachine.OperationTypeMountArchive:
		return e.executeMountArchive(ctx, operation.(statemachine.MountArchiveVariant))
	case statemachine.OperationTypeUnmount:
		return e.executeUnmount(ctx, operation.(statemachine.UnmountVariant))
	case statemachine.OperationTypeUnmountArchive:
		return e.executeUnmountArchive(ctx, operation.(statemachine.UnmountArchiveVariant))
	case statemachine.OperationTypeExaminePrune:
		examineVariant := operation.(statemachine.ExaminePruneVariant)
		examineData := examineVariant()
		return e.executePrune(ctx, examineData.BackupID, true, examineData.PruningRule, examineData.ResultCh, examineData.SaveResults)
	default:
		assert.Fail("Unhandled OperationType in borgOperationExecutor.Execute")
		return nil, fmt.Errorf("unsupported operation type: %T", operation)
	}
}

// executeBackup performs a borg backup operation
func (e *borgOperationExecutor) executeBackup(ctx context.Context, backupOp statemachine.BackupVariant) (*borgtypes.Status, error) {
	backupData := backupOp()

	// Get backup profile with repository data in a single query
	profile, err := e.db.BackupProfile.Query().
		Where(
			backupprofile.ID(backupData.BackupID.BackupProfileId),
			backupprofile.HasRepositoriesWith(repository.ID(backupData.BackupID.RepositoryId)),
		).
		WithRepositories(func(q *ent.RepositoryQuery) {
			q.Where(repository.ID(backupData.BackupID.RepositoryId))
		}).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup profile %d not found: %w", backupData.BackupID.BackupProfileId, err)
	}

	// Get repository from backup profile relationship
	if len(profile.Edges.Repositories) != 1 {
		return nil, fmt.Errorf("expected exactly one repository for backup profile %d and repository %d, got %d",
			backupData.BackupID.BackupProfileId, backupData.BackupID.RepositoryId, len(profile.Edges.Repositories))
	}
	repo := profile.Edges.Repositories[0] // Should be exactly the repository we requested

	backupPaths := profile.BackupPaths
	excludePaths := profile.ExcludePaths
	prefix := profile.Prefix
	compressionMode := profile.CompressionMode
	compressionLevel := profile.CompressionLevel

	// Create progress channel
	progressCh := make(chan borgtypes.BackupProgress, 100)

	// Start progress monitoring in background
	go e.monitorBackupProgress(ctx, progressCh)

	// Execute borg create command
	archivePath, status := e.borgClient.Create(ctx, repo.URL, repo.Password, prefix, backupPaths, excludePaths, compressionMode, compressionLevel, progressCh)
	if !status.IsCompletedWithSuccess() {
		return status, nil
	}

	// Refresh the newly created archive in database
	err = e.refreshNewArchive(ctx, repo, archivePath)
	if err != nil {
		e.log.Errorw("Failed to refresh archive after backup",
			"archivePath", archivePath,
			"repoID", e.repoID,
			"error", err.Error())
		// Don't fail backup operation for refresh errors
	}

	// Refresh repository stats from borg info
	err = e.refreshRepositoryStats(ctx, e.repoID)
	if err != nil {
		e.log.Errorw("Failed to refresh repository stats after backup",
			"repoID", e.repoID,
			"error", err.Error())
		// Don't fail backup operation for stats refresh errors
	}

	// Return status directly (preserves rich error information)
	_ = backupData // Use backupData to avoid unused variable warning
	return status, nil
}

// monitorBackupProgress monitors backup progress and updates operation status
func (e *borgOperationExecutor) monitorBackupProgress(ctx context.Context, progressCh <-chan borgtypes.BackupProgress) {
	for {
		select {
		case <-ctx.Done():
			return
		case progress, ok := <-progressCh:
			if !ok {
				return // Channel closed
			}
			err := e.progressUpdater.UpdateBackupProgress(ctx, e.operationID, progress)
			if err != nil {
				e.log.Errorw("Failed to update operation progress", "operationID", e.operationID, "error", err.Error())
			}
		}
	}
}

// buildPruneOptions converts PruningRule database fields into borg prune command options
func buildPruneOptions(rule *ent.PruningRule) []string {
	var options []string

	if rule.KeepHourly > 0 {
		options = append(options, fmt.Sprintf("--keep-hourly=%d", rule.KeepHourly))
	}
	if rule.KeepDaily > 0 {
		options = append(options, fmt.Sprintf("--keep-daily=%d", rule.KeepDaily))
	}
	if rule.KeepWeekly > 0 {
		options = append(options, fmt.Sprintf("--keep-weekly=%d", rule.KeepWeekly))
	}
	if rule.KeepMonthly > 0 {
		options = append(options, fmt.Sprintf("--keep-monthly=%d", rule.KeepMonthly))
	}
	if rule.KeepYearly > 0 {
		options = append(options, fmt.Sprintf("--keep-yearly=%d", rule.KeepYearly))
	}
	if rule.KeepWithinDays > 0 {
		options = append(options, fmt.Sprintf("--keep-within=%dd", rule.KeepWithinDays))
	}

	// If no options were configured, provide a sensible default
	if len(options) == 0 {
		options = []string{"--keep-daily=7", "--keep-weekly=4", "--keep-monthly=6"}
	}

	return options
}

// executePrune performs a borg prune operation (real or dry-run)
func (e *borgOperationExecutor) executePrune(ctx context.Context, backupID types.BackupId, isDryRun bool, customPruningRule *ent.PruningRule, resultCh chan<- borgtypes.PruneResult, saveResults bool) (*borgtypes.Status, error) {
	// Get backup profile with repository and pruning rule data
	profile, err := e.db.BackupProfile.Query().
		Where(
			backupprofile.ID(backupID.BackupProfileId),
			backupprofile.HasRepositoriesWith(repository.ID(backupID.RepositoryId)),
		).
		WithRepositories(func(q *ent.RepositoryQuery) {
			q.Where(repository.ID(backupID.RepositoryId))
		}).
		WithPruningRule().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup profile %d not found: %w", backupID.BackupProfileId, err)
	}

	// Get repository from backup profile relationship
	if len(profile.Edges.Repositories) != 1 {
		return nil, fmt.Errorf("expected exactly one repository for backup profile %d and repository %d, got %d",
			backupID.BackupProfileId, backupID.RepositoryId, len(profile.Edges.Repositories))
	}
	repo := profile.Edges.Repositories[0]

	// Get pruning configuration from database
	prefix := profile.Prefix

	// Handle pruning rule configuration - select rule source
	var pruningRule *ent.PruningRule
	if customPruningRule != nil {
		pruningRule = customPruningRule
	} else if profile.Edges.PruningRule != nil {
		if !isDryRun && !profile.Edges.PruningRule.IsEnabled {
			return nil, fmt.Errorf("backup profile %d has pruning rule disabled", backupID.BackupProfileId)
		}
		pruningRule = profile.Edges.PruningRule
	} else {
		return nil, fmt.Errorf("backup profile %d has no pruning rule available", backupID.BackupProfileId)
	}

	// Get pruning options from selected pruning rule
	pruneOptions := buildPruneOptions(pruningRule)
	e.log.Infow("Using pruning rule configuration",
		"repoID", e.repoID,
		"operationID", e.operationID,
		"backupProfileID", backupID.BackupProfileId,
		"pruneOptions", pruneOptions,
		"isDryRun", isDryRun)

	// Create result channel for prune results
	borgResultCh := make(chan borgtypes.PruneResult, 1)

	// Execute borg prune command
	status := e.borgClient.Prune(ctx, repo.URL, repo.Password, prefix, pruneOptions, isDryRun, borgResultCh)

	// Wait for prune result with timeout
	select {
	case pruneResult := <-borgResultCh:
		// Log the result based on operation type
		if isDryRun {
			e.log.Infow("Prune examination completed",
				"repoID", e.repoID,
				"operationID", e.operationID,
				"isDryRun", pruneResult.IsDryRun,
				"archivesToPrune", len(pruneResult.PruneArchives),
				"archivesToKeep", len(pruneResult.KeepArchives))
		} else {
			e.log.Infow("Prune operation completed",
				"repoID", e.repoID,
				"operationID", e.operationID,
				"isDryRun", pruneResult.IsDryRun,
				"prunedCount", len(pruneResult.PruneArchives),
				"keptCount", len(pruneResult.KeepArchives))
		}

		// Send result to examination channel if provided (for dry-run examinations)
		if isDryRun && resultCh != nil {
			select {
			case resultCh <- pruneResult:
				// Successfully sent result
			case <-time.After(5 * time.Second):
				e.log.Warnw("Timeout sending examination result to channel", "repoID", e.repoID, "operationID", e.operationID)
			}
		}

		// Save examination results to database if requested (for dry-run examinations)
		if isDryRun && saveResults {
			if err := e.saveExaminationResults(ctx, backupID.RepositoryId, pruneResult); err != nil {
				e.log.Errorw("Failed to save examination results",
					"repoID", e.repoID,
					"operationID", e.operationID,
					"backupID", backupID,
					"error", err)
			} else {
				e.log.Infow("Examination results saved to database",
					"repoID", e.repoID,
					"operationID", e.operationID,
					"backupID", backupID,
					"archivesToPrune", len(pruneResult.PruneArchives))
			}
		}

	case <-time.After(30 * time.Second):
		if isDryRun {
			e.log.Warnw("Timeout waiting for examine prune result", "repoID", e.repoID, "operationID", e.operationID)
		} else {
			e.log.Warnw("Timeout waiting for prune result", "repoID", e.repoID, "operationID", e.operationID)
		}
	case <-ctx.Done():
		if isDryRun {
			e.log.Warnw("Context canceled waiting for examine prune result", "repoID", e.repoID, "operationID", e.operationID)
		} else {
			e.log.Warnw("Context canceled waiting for prune result", "repoID", e.repoID, "operationID", e.operationID)
		}
	}

	// Refresh archives after successful real prune operation (archives may have been deleted)
	// Skip for dry-run examinations
	if !isDryRun && status.IsCompletedWithSuccess() {
		// Get updated archive list from repository
		listResponse, listStatus := e.borgClient.List(ctx, repo.URL, repo.Password, "")
		if listStatus.IsCompletedWithSuccess() {
			// Update database with refreshed archive information
			err := e.syncArchivesToDatabase(ctx, repo.ID, listResponse.Archives)
			if err != nil {
				e.log.Warnw("Failed to refresh archives after prune", "repoID", e.repoID, "error", err)
			}
		} else {
			e.log.Warnw("Failed to list archives after prune", "repoID", e.repoID, "error", listStatus.Error)
		}

		// Refresh repository stats from borg info
		err := e.refreshRepositoryStats(ctx, repo.ID)
		if err != nil {
			e.log.Errorw("Failed to refresh repository stats after prune",
				"repoID", repo.ID,
				"error", err.Error())
			// Don't fail prune operation for stats refresh errors
		}
	}
	return status, nil
}

// saveExaminationResults saves prune examination results to database by updating WillBePruned flags
func (e *borgOperationExecutor) saveExaminationResults(ctx context.Context, repositoryID int, pruneResult borgtypes.PruneResult) error {
	// Map archive names to slices for database queries
	archiveNamesToPrune := make([]string, len(pruneResult.PruneArchives))
	for i, arch := range pruneResult.PruneArchives {
		archiveNamesToPrune[i] = arch.Name
	}

	archiveNamesToKeep := make([]string, len(pruneResult.KeepArchives))
	for i, arch := range pruneResult.KeepArchives {
		archiveNamesToKeep[i] = arch.Name
	}

	// Update archives that will be pruned
	cntToTrue, err := e.db.Archive.
		Update().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(repositoryID)),
			archive.NameIn(archiveNamesToPrune...),
			archive.WillBePruned(false)),
		).
		SetWillBePruned(true).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update archives to be pruned: %w", err)
	}

	// Update archives that will be kept
	cntToFalse, err := e.db.Archive.
		Update().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(repositoryID)),
			archive.NameIn(archiveNamesToKeep...),
			archive.WillBePruned(true)),
		).
		SetWillBePruned(false).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update archives to be kept: %w", err)
	}

	e.log.Debugw("Updated examination results in database",
		"repoID", repositoryID,
		"prunedCount", cntToTrue,
		"keptCount", cntToFalse)

	return nil
}

// executeRepositoryDelete performs a borg repository delete operation
func (e *borgOperationExecutor) executeRepositoryDelete(ctx context.Context, deleteOp statemachine.DeleteVariant) (*borgtypes.Status, error) {
	deleteData := deleteOp()

	// Get repository from database using RepositoryID
	repo, err := e.db.Repository.Get(ctx, deleteData.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("repository %d not found: %w", deleteData.RepositoryID, err)
	}

	// Execute borg delete repository command
	status := e.borgClient.DeleteRepository(ctx, repo.URL, repo.Password)

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeArchiveDelete performs a borg archive delete operation
func (e *borgOperationExecutor) executeArchiveDelete(ctx context.Context, deleteOp statemachine.ArchiveDeleteVariant) (*borgtypes.Status, error) {
	deleteData := deleteOp()

	// Get archive from database to get repository information
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(deleteData.ArchiveID)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", deleteData.ArchiveID, err)
	}

	// Get repository from archive's backup profile
	repo := archiveEntity.Edges.Repository

	// Use archive name from database
	archiveName := archiveEntity.Name

	// Execute borg delete archive command
	status := e.borgClient.DeleteArchive(ctx, repo.URL, archiveName, repo.Password)

	// If deletion was successful, also delete from database and emit event
	if status.IsCompletedWithSuccess() {
		// Delete archive from database
		_, dbErr := e.db.Archive.Delete().Where(archive.ID(deleteData.ArchiveID)).Exec(ctx)
		if dbErr != nil {
			e.log.Errorw("Failed to delete archive from database",
				"archiveID", deleteData.ArchiveID,
				"archiveName", archiveName,
				"error", dbErr.Error())
		} else {
			// Emit event for archive change
			e.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repo.ID))
		}
	}

	return status, nil
}

// executeArchiveRefresh performs a borg list operation to refresh archive information
func (e *borgOperationExecutor) executeArchiveRefresh(ctx context.Context, refreshOp statemachine.ArchiveRefreshVariant) (*borgtypes.Status, error) {
	refreshData := refreshOp()

	// Get repository from database using RepositoryID
	repo, err := e.db.Repository.Get(ctx, refreshData.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("repository %d not found: %w", refreshData.RepositoryID, err)
	}

	// Execute borg list command to refresh archive information
	listResponse, status := e.borgClient.List(ctx, repo.URL, repo.Password, "")
	if !status.IsCompletedWithSuccess() {
		return status, nil
	}

	// Update database with refreshed archive information
	err = e.syncArchivesToDatabase(ctx, refreshData.RepositoryID, listResponse.Archives)
	if err != nil {
		return nil, fmt.Errorf("failed to sync archives to database: %w", err)
	}

	// Refresh repository stats from borg info
	err = e.refreshRepositoryStats(ctx, refreshData.RepositoryID)
	if err != nil {
		e.log.Errorw("Failed to refresh repository stats after archive refresh",
			"repoID", refreshData.RepositoryID,
			"error", err.Error())
		// Don't fail refresh operation for stats refresh errors
	}

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeCheck performs a borg check operation to verify repository integrity
func (e *borgOperationExecutor) executeCheck(ctx context.Context, checkOp statemachine.CheckVariant) (*borgtypes.Status, error) {
	checkData := checkOp()

	// Get repository from database using RepositoryID
	repo, err := e.db.Repository.Get(ctx, checkData.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("repository %d not found: %w", checkData.RepositoryID, err)
	}

	e.log.Infof("Starting %s check for repo %d", map[bool]string{true: "quick", false: "full"}[checkData.QuickVerification], repo.ID)

	// Execute borg check command
	result := e.borgClient.Check(ctx, repo.URL, repo.Password, checkData.QuickVerification)

	// Extract error messages for logging and storage
	errorMessages := make([]string, 0, len(result.ErrorLogs))
	for _, msg := range result.ErrorLogs {
		errorMessages = append(errorMessages, msg.Message)
	}

	// Log captured error messages
	if len(result.ErrorLogs) > 0 {
		e.log.Infow("Check operation found errors",
			"repoID", repo.ID,
			"checkType", map[bool]string{true: "quick", false: "full"}[checkData.QuickVerification],
			"errorCount", len(result.ErrorLogs),
			"errors", errorMessages)
	}

	// Update database fields based on check type and result
	now := time.Now()
	updateQuery := e.db.Repository.UpdateOne(repo)

	if checkData.QuickVerification {
		// Update quick check fields
		updateQuery = updateQuery.
			SetLastQuickCheckAt(now).
			SetQuickCheckError(errorMessages) // Empty array if no errors
	} else {
		// Update full check fields
		updateQuery = updateQuery.
			SetLastFullCheckAt(now).
			SetFullCheckError(errorMessages) // Empty array if no errors
	}

	// Save updates
	return result.Status, updateQuery.Exec(ctx)
}

// executeArchiveRename performs a borg rename operation
func (e *borgOperationExecutor) executeArchiveRename(ctx context.Context, renameOp statemachine.ArchiveRenameVariant) (*borgtypes.Status, error) {
	renameData := renameOp()

	// Get archive from database to get repository and current name
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(renameData.ArchiveID)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", renameData.ArchiveID, err)
	}

	// Get repository from archive relationship
	repo := archiveEntity.Edges.Repository

	// Get current archive name from database
	currentArchiveName := archiveEntity.Name
	newArchiveName := fmt.Sprintf("%s%s", renameData.Prefix, renameData.Name)

	// Execute borg rename command
	status := e.borgClient.Rename(ctx, repo.URL, currentArchiveName, repo.Password, newArchiveName)

	// If the operation was successful, update the archive name in the database
	if status.IsCompletedWithSuccess() {
		err := e.db.Archive.UpdateOneID(archiveEntity.ID).SetName(newArchiveName).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("borg rename succeeded but failed to update archive name in database: %w", err)
		}

		// Emit event for archive change
		e.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repo.ID))
	}

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeArchiveComment performs a borg recreate --comment operation
func (e *borgOperationExecutor) executeArchiveComment(ctx context.Context, commentOp statemachine.ArchiveCommentVariant) (*borgtypes.Status, error) {
	commentData := commentOp()

	// Get archive from database to get repository
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(commentData.ArchiveID)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", commentData.ArchiveID, err)
	}

	// Get repository from archive relationship
	repo := archiveEntity.Edges.Repository

	// Execute borg recreate --comment command
	status := e.borgClient.Recreate(ctx, repo.URL, archiveEntity.Name, repo.Password, commentData.Comment)

	// If the operation was successful, update the archive comment in the database
	if status.IsCompletedWithSuccess() {
		err := e.db.Archive.UpdateOneID(archiveEntity.ID).SetComment(commentData.Comment).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("borg recreate succeeded but failed to update archive comment in database: %w", err)
		}

		// Emit event for archive change
		e.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repo.ID))
	}

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeMount performs a borg mount operation for a repository
func (e *borgOperationExecutor) executeMount(ctx context.Context, mountOp statemachine.MountVariant) (*borgtypes.Status, error) {
	mountData := mountOp()

	// Get repository from database
	repo, err := e.db.Repository.Get(ctx, mountData.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Make sure the mount path exists
	err = ensurePathExists(mountData.MountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create mount path: %w", err)
	}

	// Execute borg mount
	status := e.borgClient.MountRepository(ctx, repo.URL, repo.Password, mountData.MountPath)

	// On success, open file manager
	if status == nil || !status.HasError() {
		go openFileManager(mountData.MountPath, e.log)
	}

	return status, nil
}

// executeMountArchive performs a borg mount operation for a specific archive
func (e *borgOperationExecutor) executeMountArchive(ctx context.Context, mountOp statemachine.MountArchiveVariant) (*borgtypes.Status, error) {
	mountData := mountOp()

	// Get archive from database
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(mountData.ArchiveID)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", mountData.ArchiveID, err)
	}

	// Get repository from archive's backup profile
	repo := archiveEntity.Edges.Repository

	// Make sure the mount path exists
	err = ensurePathExists(mountData.MountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive mount path: %w", err)
	}

	// Execute borg mount
	status := e.borgClient.MountArchive(ctx, repo.URL, archiveEntity.Name, repo.Password, mountData.MountPath)

	// On success, open file manager
	if status == nil || !status.HasError() {
		go openFileManager(mountData.MountPath, e.log)
	}

	return status, nil
}

// executeUnmount performs a borg unmount operation for a repository
func (e *borgOperationExecutor) executeUnmount(ctx context.Context, unmountOp statemachine.UnmountVariant) (*borgtypes.Status, error) {
	unmountData := unmountOp()

	// Use stored mount path
	mountPath := unmountData.MountPath

	// Execute borg umount
	status := e.borgClient.Umount(ctx, mountPath)

	return status, nil
}

// executeUnmountArchive performs a borg unmount operation for a specific archive
func (e *borgOperationExecutor) executeUnmountArchive(ctx context.Context, unmountOp statemachine.UnmountArchiveVariant) (*borgtypes.Status, error) {
	unmountData := unmountOp()

	// Use stored mount path
	mountPath := unmountData.MountPath

	// Execute borg umount
	status := e.borgClient.Umount(ctx, mountPath)

	return status, nil
}

// syncArchivesToDatabase synchronizes borg archives with the database
// It deletes archives that no longer exist in borg and creates new ones
func (e *borgOperationExecutor) syncArchivesToDatabase(ctx context.Context, repositoryID int, archives []borgtypes.ArchiveList) error {
	// Get repository with backup profiles for prefix matching
	repo, err := e.db.Repository.Query().
		Where(repository.ID(repositoryID)).
		WithBackupProfiles().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query repository %d: %w", repositoryID, err)
	}

	// Extract all borg IDs from the response
	borgIds := make([]string, len(archives))
	for i, arch := range archives {
		borgIds[i] = arch.ID
	}

	// Delete archives that no longer exist in borg
	deletedCount, err := e.db.Archive.Delete().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repositoryID)),
				archive.BorgIDNotIn(borgIds...),
			),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete orphaned archives: %w", err)
	}

	if deletedCount > 0 {
		e.log.Infow("Deleted orphaned archives", "count", deletedCount, "repositoryID", repositoryID)
	}

	// Query existing archives to identify which ones are already saved
	existingArchives, err := e.db.Archive.Query().
		Where(archive.HasRepositoryWith(repository.ID(repositoryID))).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query existing archives: %w", err)
	}

	// Create map of existing borg IDs for faster lookup
	existingBorgIds := make(map[string]bool)
	for _, arch := range existingArchives {
		existingBorgIds[arch.BorgID] = true
	}

	// Create new archives for those not already in database
	newArchiveCount := 0
	for _, arch := range archives {
		if existingBorgIds[arch.ID] {
			continue // Archive already exists, skip
		}

		// Calculate duration from start to end time
		startTime := time.Time(arch.Start)
		endTime := time.Time(arch.End)
		duration := endTime.Sub(startTime)

		// Create base archive creation query
		createQuery := e.db.Archive.Create().
			SetBorgID(arch.ID).
			SetName(arch.Name).
			SetCreatedAt(startTime).
			SetDuration(duration.Seconds()).
			SetRepositoryID(repositoryID)

		// Find matching backup profile by prefix
		for _, backupProfile := range repo.Edges.BackupProfiles {
			if strings.HasPrefix(arch.Name, backupProfile.Prefix) {
				createQuery = createQuery.SetBackupProfileID(backupProfile.ID)
				break
			}
		}

		// Save the new archive
		_, err := createQuery.Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create archive %s: %w", arch.Name, err)
		}

		newArchiveCount++
	}

	if newArchiveCount > 0 {
		e.log.Infow("Created new archives", "count", newArchiveCount, "repositoryID", repositoryID)
	}

	// Emit event if any archives were changed (deleted or created)
	if deletedCount > 0 || newArchiveCount > 0 {
		e.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repositoryID))
	}

	return nil
}

// syncSingleArchiveToDatabase adds or updates a single archive in the database
// This method does NOT delete any existing archives - it only adds/updates
func (e *borgOperationExecutor) syncSingleArchiveToDatabase(ctx context.Context, repositoryID int, archiveData borgtypes.ArchiveList) error {
	// Get repository with backup profiles for prefix matching
	repo, err := e.db.Repository.Query().
		Where(repository.ID(repositoryID)).
		WithBackupProfiles().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query repository %d: %w", repositoryID, err)
	}

	// Calculate duration from start to end time
	startTime := time.Time(archiveData.Start)
	endTime := time.Time(archiveData.End)
	duration := endTime.Sub(startTime)

	// Check if archive already exists by borg ID
	existingArchive, err := e.db.Archive.Query().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repositoryID)),
				archive.BorgID(archiveData.ID),
			),
		).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing archive %s: %w", archiveData.Name, err)
	}

	if existingArchive != nil {
		// Update existing archive
		_, err := existingArchive.Update().
			SetName(archiveData.Name).
			SetDuration(duration.Seconds()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update archive %s: %w", archiveData.Name, err)
		}

		e.log.Infow("Updated existing archive",
			"archiveName", archiveData.Name,
			"borgId", archiveData.ID,
			"repositoryID", repositoryID)
	} else {
		// Create new archive
		createQuery := e.db.Archive.Create().
			SetBorgID(archiveData.ID).
			SetName(archiveData.Name).
			SetCreatedAt(startTime).
			SetDuration(duration.Seconds()).
			SetRepositoryID(repositoryID)

		// Find matching backup profile by prefix
		for _, backupProfile := range repo.Edges.BackupProfiles {
			if strings.HasPrefix(archiveData.Name, backupProfile.Prefix) {
				createQuery = createQuery.SetBackupProfileID(backupProfile.ID)
				break
			}
		}

		// Save the new archive
		_, err := createQuery.Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create archive %s: %w", archiveData.Name, err)
		}

		e.log.Infow("Created new archive",
			"archiveName", archiveData.Name,
			"borgId", archiveData.ID,
			"repositoryID", repositoryID)
	}

	// Emit event for archive change (always emit since this method always changes something)
	e.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repositoryID))

	return nil
}

// refreshNewArchive refreshes a single newly created archive in the database
func (e *borgOperationExecutor) refreshNewArchive(ctx context.Context, repo *ent.Repository, archivePath string) error {
	// Parse archivePath to extract repository and archive name
	// archivePath format: "repository::archiveName"
	parts := strings.SplitN(archivePath, "::", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid archive path format: %s", archivePath)
	}
	repoPath := parts[0]
	archiveName := parts[1]

	// Use repository path with archive glob pattern to list only the specific archive
	listResponse, status := e.borgClient.List(ctx, repoPath, repo.Password, archiveName)
	if status.HasError() {
		return fmt.Errorf("failed to list archive %s: %w", archiveName, status.Error)
	}

	if len(listResponse.Archives) == 0 {
		return fmt.Errorf("archive %s not found in borg repository", archiveName)
	}

	// Sync only the single archive (no deletions)
	return e.syncSingleArchiveToDatabase(ctx, e.repoID, listResponse.Archives[0])
}

// refreshRepositoryStats calls borg info to update repository statistics for local/remote repositories
func (e *borgOperationExecutor) refreshRepositoryStats(ctx context.Context, repoID int) error {
	// Get repository from database
	repo, err := e.db.Repository.Get(ctx, repoID)
	if err != nil {
		return fmt.Errorf("repository %d not found: %w", repoID, err)
	}

	// Call borg info to get repository statistics
	infoResponse, status := e.borgClient.Info(ctx, repo.URL, repo.Password)
	if status.HasError() {
		return fmt.Errorf("failed to get repository info: %w", status.Error)
	}

	// Update repository stats in database
	err = e.db.Repository.UpdateOneID(repoID).
		SetStatsTotalChunks(infoResponse.Cache.Stats.TotalChunks).
		SetStatsTotalSize(infoResponse.Cache.Stats.TotalSize).
		SetStatsTotalCsize(infoResponse.Cache.Stats.TotalCSize).
		SetStatsTotalUniqueChunks(infoResponse.Cache.Stats.TotalUniqueChunks).
		SetStatsUniqueSize(infoResponse.Cache.Stats.UniqueSize).
		SetStatsUniqueCsize(infoResponse.Cache.Stats.UniqueCSize).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update repository stats: %w", err)
	}

	e.log.Debugw("Refreshed repository stats",
		"repoID", repoID,
		"uniqueSize", infoResponse.Cache.Stats.UniqueSize,
		"totalSize", infoResponse.Cache.Stats.TotalSize)

	return nil
}

// isCloudRepository checks if a repository is an ArcoCloud repository
func (qm *QueueManager) isCloudRepository(ctx context.Context, repoID int) bool {
	exists, err := qm.db.Repository.Query().
		Where(repository.And(
			repository.IDEQ(repoID),
			repository.HasCloudRepository(),
		)).
		Exist(ctx)
	if err != nil {
		qm.log.Errorw("IsCloudRepository query error", "error", err)
	}
	return exists
}

// mapOperationErrorResponse maps borg errors to comprehensive error handling strategy
// based on both the error type and the operation type
func (qm *QueueManager) mapOperationErrorResponse(ctx context.Context, borgError *borgtypes.BorgError, repoID int, operation statemachine.Operation) OperationErrorResponse {
	// First, determine base error type and action (existing logic)
	var errorType statemachine.ErrorType
	var errorAction statemachine.ErrorAction

	// Check for SSH connection errors
	if errors.Is(borgError, borgtypes.ErrorConnectionClosedWithHint) {
		// SSH key authentication failed
		// Only suggest regenerating SSH key for cloud repositories
		if qm.isCloudRepository(ctx, repoID) {
			errorType, errorAction = statemachine.ErrorTypeSSHKey, statemachine.ErrorActionRegenerateSSH
		} else {
			errorType, errorAction = statemachine.ErrorTypeSSHKey, statemachine.ErrorActionNone
		}
	} else if errors.Is(borgError, borgtypes.ErrorPassphraseWrong) {
		// Incorrect passphrase - no automatic action possible
		errorType, errorAction = statemachine.ErrorTypePassphrase, statemachine.ErrorActionChangePassphrase
	} else if errors.Is(borgError, borgtypes.ErrorLockTimeout) {
		// Repository is locked - can break lock for any repo type
		errorType, errorAction = statemachine.ErrorTypeLocked, statemachine.ErrorActionBreakLock
	} else {
		// Default fallback for all other errors
		errorType, errorAction = statemachine.ErrorTypeGeneral, statemachine.ErrorActionNone
	}

	// Now determine operation-specific behavior
	switch statemachine.GetOperationType(operation) {
	case statemachine.OperationTypeBackup, statemachine.OperationTypePrune, statemachine.OperationTypeCheck:
		// Critical operations - full error handling with persistent notifications
		return OperationErrorResponse{
			ErrorType:                 errorType,
			ErrorAction:               errorAction,
			ShouldEnterErrorState:     errorAction != statemachine.ErrorActionNone,
			ShouldNotify:              true,
			ShouldPersistNotification: true,
		}

	case statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeArchiveDelete,
		statemachine.OperationTypeArchiveComment,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeDelete,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:

		// Default is to notify but not enter error state and not persist the errors
		return OperationErrorResponse{
			ErrorType:                 errorType,
			ErrorAction:               errorAction,
			ShouldEnterErrorState:     errorAction != statemachine.ErrorActionNone, // Enter error state if error requires an action
			ShouldNotify:              true,
			ShouldPersistNotification: false,
		}

	default:
		// Catch-all for any new operation types - fail with assertion to force explicit handling
		assert.Fail("Unhandled OperationType in mapOperationErrorResponse")
		return OperationErrorResponse{
			ErrorType:                 statemachine.ErrorTypeGeneral,
			ErrorAction:               statemachine.ErrorActionNone,
			ShouldEnterErrorState:     true,
			ShouldNotify:              true,
			ShouldPersistNotification: false,
		}
	}
}

// ============================================================================
// MOUNT UTILITY FUNCTIONS
// ============================================================================

// openFileManager opens the file manager for the given path
func openFileManager(path string, log *zap.SugaredLogger) {
	openCmd, err := platform.GetOpenFileManagerCmd()
	if err != nil {
		log.Error("Error getting open file manager command: ", err)
		return
	}
	cmd := exec.Command(openCmd, path)
	err = cmd.Run()
	if err != nil {
		log.Error("Error opening file manager: ", err)
	}
}

// getMountPath returns the base mount path for the current user
func getMountPath(name string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	mountPath, err := platform.GetMountPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(mountPath, currentUser.Uid, "arco", name), nil
}

func getRepoMountPath(repo *ent.Repository) (string, error) {
	return getMountPath("repo-" + strconv.Itoa(repo.ID))
}

func getArchiveMountPath(archive *ent.Archive) (string, error) {
	return getMountPath("archive-" + strconv.Itoa(archive.ID))
}

func ensurePathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
