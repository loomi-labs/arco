package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/borg"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"go.uber.org/zap"
)

// ============================================================================
// QUEUE MANAGER
// ============================================================================

// QueueManager manages operation queues for all repositories
type QueueManager struct {
	log          *zap.SugaredLogger
	stateMachine *statemachine.RepositoryStateMachine
	db           *ent.Client
	borg         borg.Borg
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
func (qm *QueueManager) Init(db *ent.Client, borgClient borg.Borg) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.db = db
	qm.borg = borgClient
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

// GetCurrentState returns the current repository state based on queue status
//func (qm *QueueManager) GetCurrentState(repoID int) statemachine.RepositoryState {
//	// TODO: Implement state calculation based on queue status
//	return statemachine.CreateIdleState()
//}

// AddOperation adds an operation to the specified repository queue
func (qm *QueueManager) AddOperation(repoID int, op *QueuedOperation) (string, error) {
	// Get or create repository queue
	queue := qm.GetQueue(repoID)

	// Add operation to queue (handles idempotency internally)
	operationID, err := queue.AddOperation(op)
	if err != nil {
		return "", fmt.Errorf("failed to add operation to repository %d queue: %w", repoID, err)
	}

	// Attempt to start operation if possible
	qm.processQueue(repoID)

	return operationID, nil
}

// RemoveOperation removes an operation from tracking
func (qm *QueueManager) RemoveOperation(repoID int, operationID string) error {
	queue := qm.GetQueue(repoID)
	return queue.RemoveOperation(operationID)
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
	_ = ctx // Suppress unused parameter warning
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
	targetState, err := qm.getTargetStateForOperation(op)
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
		status, err := executor.Execute(ctx, op.Operation)
		if err != nil {
			// System error (e.g., backup profile not found)
			qm.log.Errorw("System error during operation execution",
				"repoID", repoID,
				"operationID", operationID,
				"operationType", fmt.Sprintf("%T", op.Operation),
				"error", err.Error())

			// Complete operation with system error
			if completeErr := qm.CompleteOperation(repoID, operationID, false, fmt.Sprintf("System error: %v", err)); completeErr != nil {
				qm.log.Warnw("Failed to complete operation after system error",
					"repoID", repoID,
					"operationID", operationID,
					"systemError", err.Error(),
					"completionError", completeErr.Error())
			}
			return
		}

		if status.HasError() {
			// 1. ERROR BRANCH - Enhanced with conditional notification creation
			qm.log.Errorw("Borg operation failed",
				"repoID", repoID,
				"operationID", operationID,
				"operationType", fmt.Sprintf("%T", op.Operation),
				"error", status.Error.Message,
				"category", status.Error.Category,
				"exitCode", status.Error.ExitCode)

			// Only create notifications for backup/prune operations
			if qm.shouldCreateNotification(op.Operation) {
				qm.createErrorNotification(ctx, repoID, operationID, status, op.Operation)
			}

			// Complete operation with failure
			if completeErr := qm.CompleteOperation(repoID, operationID, false, status.Error.Message); completeErr != nil {
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
			if completeErr := qm.CompleteOperation(repoID, operationID, true, ""); completeErr != nil {
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

// CompleteOperation marks an operation as completed
func (qm *QueueManager) CompleteOperation(repoID int, operationID string, success bool, errorMsg string) error {
	queue := qm.GetQueue(repoID)

	// Get active operation to determine weight
	activeOp := queue.GetActive()
	if activeOp == nil || activeOp.ID != operationID {
		return fmt.Errorf("operation %s is not currently active for repository %d", operationID, repoID)
	}

	weight := statemachine.GetOperationWeight(activeOp.Operation)

	// Remove from active tracking
	qm.mu.Lock()
	if weight == statemachine.WeightHeavy {
		delete(qm.activeHeavy, repoID)
	} else {
		delete(qm.activeLight, repoID)
	}
	qm.mu.Unlock()

	// Update operation status and complete in queue
	err := queue.CompleteActive(success, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to complete operation: %w", err)
	}

	// Transition repository state via state machine
	// Get current state from in-memory tracking
	currentState := qm.GetRepositoryState(repoID)

	// Determine target state based on completion and queue status
	var targetState statemachine.RepositoryState

	if !success {
		// On failure, transition to error state
		targetState = statemachine.CreateErrorState(statemachine.ErrorTypeSSHKey, errorMsg, statemachine.ErrorActionNone)
	} else {
		// On success, determine next state based on queue status
		targetState, err = qm.getCompletionStateForRepository(repoID)
		if err != nil {
			return fmt.Errorf("failed to determine completion state: %w", err)
		}
	}

	// Validate state transition
	err = qm.stateMachine.Transition(repoID, currentState, targetState)
	if err != nil {
		return fmt.Errorf("failed to transition repository %d from %T to %T: %w", repoID, currentState, targetState, err)
	}

	// Update repository state in memory
	qm.setRepositoryState(repoID, targetState)

	// Attempt to start next queued operation for this repo
	qm.processQueue(repoID)

	// If we completed a heavy operation, try to start waiting heavy operations on other repos
	if weight == statemachine.WeightHeavy {
		for otherRepoID := range qm.queues {
			if otherRepoID != repoID {
				qm.processQueue(otherRepoID)
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
		weight := statemachine.GetOperationWeight(activeOp.Operation)

		// Remove from active tracking
		qm.mu.Lock()
		if weight == statemachine.WeightHeavy {
			delete(qm.activeHeavy, repoID)
		} else {
			delete(qm.activeLight, repoID)
		}
		qm.mu.Unlock()

		// Cancel operation context if running
		currentState := qm.GetRepositoryState(repoID)
		if cancel, hasCancel := statemachine.GetCancel(currentState); hasCancel {
			qm.log.Infow("Cancelling operation",
				"repoID", repoID,
				"operationID", operationID,
				"stateType", fmt.Sprintf("%T", currentState))
			cancel()
		}

		// Complete with cancelled status
		err := queue.CompleteActive(false, "Operation cancelled by user")
		if err != nil {
			return fmt.Errorf("failed to cancel active operation: %w", err)
		}

		// Attempt to start next operation
		qm.processQueue(repoID)

		return nil
	}

	// Remove from queue if queued
	err := queue.RemoveOperation(operationID)
	if err != nil {
		return fmt.Errorf("failed to cancel queued operation: %w", err)
	}

	return nil
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

// GetQueuedOperations returns all operations for a repository
func (qm *QueueManager) GetQueuedOperations(repoID int) ([]*QueuedOperation, error) {
	queue := qm.GetQueue(repoID)
	return queue.GetOperations(), nil
}

// GetOperationsByStatus returns operations filtered by status for a repository
func (qm *QueueManager) GetOperationsByStatus(repoID int, status OperationStatus) ([]*QueuedOperation, error) {
	queue := qm.GetQueue(repoID)
	return queue.GetOperationsByStatus(status), nil
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

// SetMaxHeavyOps updates the maximum number of concurrent heavy operations
func (qm *QueueManager) SetMaxHeavyOps(maxOps int) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	oldLimit := qm.maxHeavyOps
	qm.maxHeavyOps = maxOps

	// If new limit is higher, try to start waiting heavy operations
	if maxOps > oldLimit {
		for repoID := range qm.queues {
			// Release lock temporarily to avoid deadlock
			qm.mu.Unlock()
			qm.processQueue(repoID)
			qm.mu.Lock()
		}
	}
}

// processQueue attempts to start the next operation in a repository queue
func (qm *QueueManager) processQueue(repoID int) {
	queue := qm.GetQueue(repoID)

	// Check if repository already has an active operation
	if queue.HasActiveOperation() {
		return
	}

	// Get next operation from queue
	nextOp := queue.GetNext()
	if nextOp == nil {
		return // No operations in queue
	}

	// Check concurrency limits
	if !qm.CanStartOperation(repoID, nextOp) {
		return // Cannot start due to concurrency limits
	}

	// Start the operation
	// TODO: pass correct context
	err := qm.StartOperation(context.TODO(), repoID, nextOp.ID)
	if err != nil {
		// Log error but don't crash
		qm.log.Warnw("Failed to start queued operation",
			"repoID", repoID,
			"operationID", nextOp.ID,
			"operationType", fmt.Sprintf("%T", nextOp.Operation),
			"error", err)
	}
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

// TODO: pass correct context
// getTargetStateForOperation maps an operation to its corresponding active state
func (qm *QueueManager) getTargetStateForOperation(op *QueuedOperation) (statemachine.RepositoryState, error) {
	switch v := op.Operation.(type) {
	case statemachine.OpBackupVariant:
		backupData := v()
		return statemachine.CreateBackingUpState(context.TODO(), backupData.BackupID), nil

	case statemachine.OpPruneVariant:
		pruneData := v()
		return statemachine.CreatePruningState(context.TODO(), pruneData.BackupID), nil

	case statemachine.OpDeleteVariant:
		return statemachine.CreateDeletingState(context.TODO(), 0), nil // Repository delete, no specific archive

	case statemachine.OpArchiveRefreshVariant:
		return statemachine.CreateRefreshingState(context.TODO()), nil

	case statemachine.OpArchiveDeleteVariant:
		deleteData := v()
		return statemachine.CreateDeletingState(context.TODO(), deleteData.ArchiveID), nil

	case statemachine.OpArchiveRenameVariant:
		// Archive rename is a lightweight operation, treat as refreshing
		return statemachine.CreateRefreshingState(context.TODO()), nil

	default:
		return nil, fmt.Errorf("unknown operation type: %T", op.Operation)
	}
}

// getCompletionStateForRepository determines the target state when an operation completes successfully
func (qm *QueueManager) getCompletionStateForRepository(repoID int) (statemachine.RepositoryState, error) {
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

	// No more operations, return to idle
	return statemachine.CreateIdleState(), nil
}

// createErrorNotification creates an error notification in the database for a failed borg operation
func (qm *QueueManager) createErrorNotification(ctx context.Context, repoID int, operationID string, status *borgtypes.Status, operation statemachine.Operation) {
	// Determine notification type based on operation
	notificationType := qm.getErrorNotificationType(operation)

	// Get backup profile ID from operation
	backupProfileID := qm.getBackupProfileIDFromOperation(operation)

	// Create rich error message with borg details
	message := fmt.Sprintf("Operation %s failed: %s (Exit Code: %d, Category: %s)",
		operationID, status.Error.Message, status.Error.ExitCode, status.Error.Category)

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

	// Create rich warning message with borg details
	message := fmt.Sprintf("Operation %s completed with warning: %s (Exit Code: %d, Category: %s)",
		operationID, status.Warning.Message, status.Warning.ExitCode, status.Warning.Category)

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

// getErrorNotificationType determines the notification type for error cases based on operation type
// This method should only be called for backup and prune operations
func (qm *QueueManager) getErrorNotificationType(operation statemachine.Operation) notification.Type {
	switch operation.(type) {
	case statemachine.OpBackupVariant:
		return notification.TypeFailedBackupRun
	case statemachine.OpPruneVariant:
		return notification.TypeFailedPruningRun
	default:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for error notification",
			"operationType", fmt.Sprintf("%T", operation))
		return notification.TypeFailedBackupRun // Fallback for safety
	}
}

// getWarningNotificationType determines the notification type for warning cases based on operation type
// This method should only be called for backup and prune operations
func (qm *QueueManager) getWarningNotificationType(operation statemachine.Operation) notification.Type {
	switch operation.(type) {
	case statemachine.OpBackupVariant:
		return notification.TypeWarningBackupRun
	case statemachine.OpPruneVariant:
		return notification.TypeWarningPruningRun
	default:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for warning notification",
			"operationType", fmt.Sprintf("%T", operation))
		return notification.TypeWarningBackupRun // Fallback for safety
	}
}

// getBackupProfileIDFromOperation extracts the backup profile ID from operation data
func (qm *QueueManager) getBackupProfileIDFromOperation(operation statemachine.Operation) int {
	switch op := operation.(type) {
	case statemachine.OpBackupVariant:
		backupData := op()
		return backupData.BackupID.BackupProfileId
	case statemachine.OpPruneVariant:
		pruneData := op()
		return pruneData.BackupID.BackupProfileId
	default:
		// This should never happen if shouldCreateNotification is used correctly
		qm.log.Errorw("Unexpected operation type for backup profile",
			"operationType", fmt.Sprintf("%T", op))
		return 0
	}
}

// shouldCreateNotification determines if error/warning notifications should be created for this operation type
func (qm *QueueManager) shouldCreateNotification(operation statemachine.Operation) bool {
	switch operation.(type) {
	case statemachine.OpBackupVariant, statemachine.OpPruneVariant:
		return true
	default:
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

// borgOperationExecutor implements OperationExecutor using borg commands
type borgOperationExecutor struct {
	borgClient  borg.Borg
	db          *ent.Client
	qm          *QueueManager
	repoID      int
	operationID string
}

// newBorgOperationExecutor creates a new borg operation executor
func (qm *QueueManager) newBorgOperationExecutor(repoID int, operationID string) OperationExecutor {
	return &borgOperationExecutor{
		borgClient:  qm.borg,
		db:          qm.db,
		qm:          qm,
		repoID:      repoID,
		operationID: operationID,
	}
}

// Execute implements OperationExecutor.Execute
func (e *borgOperationExecutor) Execute(ctx context.Context, operation statemachine.Operation) (*borgtypes.Status, error) {
	switch v := operation.(type) {
	case statemachine.OpBackupVariant:
		return e.executeBackup(ctx, v)
	case statemachine.OpPruneVariant:
		return e.executePrune(ctx, v)
	case statemachine.OpDeleteVariant:
		return e.executeRepositoryDelete(ctx, v)
	case statemachine.OpArchiveDeleteVariant:
		return e.executeArchiveDelete(ctx, v)
	case statemachine.OpArchiveRefreshVariant:
		return e.executeArchiveRefresh(ctx, v)
	case statemachine.OpArchiveRenameVariant:
		return e.executeArchiveRename(ctx, v)
	default:
		// System error: unsupported operation type (programming error)
		return nil, fmt.Errorf("unsupported operation type: %T", operation)
	}
}

// executeBackup performs a borg backup operation
func (e *borgOperationExecutor) executeBackup(ctx context.Context, backupOp statemachine.OpBackupVariant) (*borgtypes.Status, error) {
	backupData := backupOp()

	// Get backup profile with repository data in a single query
	profile, err := e.db.BackupProfile.Query().
		Where(backupprofile.ID(backupData.BackupID.BackupProfileId)).
		WithRepositories().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup profile %d not found: %w", backupData.BackupID.BackupProfileId, err)
	}

	// Get repository from backup profile relationship
	if len(profile.Edges.Repositories) == 0 {
		return nil, fmt.Errorf("backup profile %d has no associated repository", backupData.BackupID.BackupProfileId)
	}
	repo := profile.Edges.Repositories[0] // Backup profiles should have exactly one repository

	backupPaths := profile.BackupPaths
	excludePaths := profile.ExcludePaths
	prefix := profile.Prefix

	// Create progress channel
	progressCh := make(chan borgtypes.BackupProgress, 100)

	// Start progress monitoring in background
	go e.monitorBackupProgress(ctx, progressCh)

	// Execute borg create command
	archivePath, status := e.borgClient.Create(ctx, repo.URL, repo.Password, prefix, backupPaths, excludePaths, progressCh)

	// Close progress channel
	close(progressCh)

	// Refresh the newly created archive in database
	err = e.refreshNewArchive(ctx, repo, archivePath)
	if err != nil {
		e.qm.log.Warnw("Failed to refresh archive after backup",
			"archivePath", archivePath,
			"repoID", e.repoID,
			"error", err.Error())
		// Don't fail backup operation for refresh errors
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
			// TODO: Update operation status with progress information
			// For now, just ignore progress updates
			_ = progress
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

// executePrune performs a borg prune operation
func (e *borgOperationExecutor) executePrune(ctx context.Context, pruneOp statemachine.OpPruneVariant) (*borgtypes.Status, error) {
	pruneData := pruneOp()

	// Get backup profile with repository and pruning rule data in a single query
	profile, err := e.db.BackupProfile.Query().
		Where(backupprofile.ID(pruneData.BackupID.BackupProfileId)).
		WithRepositories().
		WithPruningRule().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup profile %d not found: %w", pruneData.BackupID.BackupProfileId, err)
	}

	// Get repository from backup profile relationship
	if len(profile.Edges.Repositories) == 0 {
		return nil, fmt.Errorf("backup profile %d has no associated repository", pruneData.BackupID.BackupProfileId)
	}
	repo := profile.Edges.Repositories[0] // Backup profiles should have exactly one repository

	// Get pruning configuration from database
	prefix := profile.Prefix

	// Handle pruning rule configuration
	var pruneOptions []string
	// Ensure pruning rule exists and is enabled
	if profile.Edges.PruningRule == nil {
		return nil, fmt.Errorf("backup profile %d has no pruning rule defined", pruneData.BackupID.BackupProfileId)
	}
	if !profile.Edges.PruningRule.IsEnabled {
		return nil, fmt.Errorf("backup profile %d has pruning rule disabled", pruneData.BackupID.BackupProfileId)
	}

	// Get pruning options from enabled pruning rule
	pruneOptions = buildPruneOptions(profile.Edges.PruningRule)
	e.qm.log.Infow("Using database pruning rule configuration",
		"repoID", e.repoID,
		"operationID", e.operationID,
		"backupProfileID", pruneData.BackupID.BackupProfileId,
		"pruneOptions", pruneOptions)

	// Create progress channel for prune results
	progressCh := make(chan borgtypes.PruneResult, 100)
	defer close(progressCh)

	// Start progress monitoring in background
	go e.monitorPruneProgress(ctx, progressCh)

	// Execute borg prune command
	status := e.borgClient.Prune(ctx, repo.URL, repo.Password, prefix, pruneOptions, false, progressCh)

	// Return status directly (preserves rich error information)
	_ = pruneData // Use pruneData to avoid unused variable warning
	return status, nil
}

// executeRepositoryDelete performs a borg repository delete operation
func (e *borgOperationExecutor) executeRepositoryDelete(ctx context.Context, deleteOp statemachine.OpDeleteVariant) (*borgtypes.Status, error) {
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
func (e *borgOperationExecutor) executeArchiveDelete(ctx context.Context, deleteOp statemachine.OpArchiveDeleteVariant) (*borgtypes.Status, error) {
	deleteData := deleteOp()

	// Get archive from database to get repository information
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(deleteData.ArchiveID)).
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithRepositories()
		}).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", deleteData.ArchiveID, err)
	}

	// Get repository from archive's backup profile
	repo := archiveEntity.Edges.BackupProfile.Edges.Repositories[0] // Assumes one repository per backup profile

	// Use archive name from database
	archiveName := archiveEntity.Name

	// Execute borg delete archive command
	status := e.borgClient.DeleteArchive(ctx, repo.URL, archiveName, repo.Password)

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeArchiveRefresh performs a borg list operation to refresh archive information
func (e *borgOperationExecutor) executeArchiveRefresh(ctx context.Context, refreshOp statemachine.OpArchiveRefreshVariant) (*borgtypes.Status, error) {
	refreshData := refreshOp()

	// Get repository from database using RepositoryID
	repo, err := e.db.Repository.Get(ctx, refreshData.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("repository %d not found: %w", refreshData.RepositoryID, err)
	}

	// Execute borg list command to refresh archive information
	listResponse, status := e.borgClient.List(ctx, repo.URL, repo.Password)

	// Update database with refreshed archive information
	err = e.syncArchivesToDatabase(ctx, refreshData.RepositoryID, listResponse.Archives)
	if err != nil {
		return nil, fmt.Errorf("failed to sync archives to database: %w", err)
	}

	// Return status directly (preserves rich error information)
	return status, nil
}

// executeArchiveRename performs a borg rename operation
func (e *borgOperationExecutor) executeArchiveRename(ctx context.Context, renameOp statemachine.OpArchiveRenameVariant) (*borgtypes.Status, error) {
	renameData := renameOp()

	// Get archive from database to get repository and current name
	archiveEntity, err := e.db.Archive.Query().
		Where(archive.ID(renameData.ArchiveID)).
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithRepositories()
		}).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", renameData.ArchiveID, err)
	}

	// Get repository from archive's backup profile
	repo := archiveEntity.Edges.BackupProfile.Edges.Repositories[0] // Assumes one repository per backup profile

	// Get current archive name from database
	currentArchiveName := archiveEntity.Name
	newArchiveName := fmt.Sprintf("%s%s", renameData.Prefix, renameData.Name)

	// Execute borg rename command
	status := e.borgClient.Rename(ctx, repo.URL, currentArchiveName, repo.Password, newArchiveName)

	// Return status directly (preserves rich error information)
	return status, nil
}

// monitorPruneProgress monitors prune progress and updates operation status
func (e *borgOperationExecutor) monitorPruneProgress(ctx context.Context, progressCh <-chan borgtypes.PruneResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case progress, ok := <-progressCh:
			if !ok {
				return // Channel closed
			}
			// TODO: Update operation status with prune progress information
			// For now, just ignore progress updates
			_ = progress
		}
	}
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
		e.qm.log.Infow("Deleted orphaned archives", "count", deletedCount, "repositoryID", repositoryID)
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
		e.qm.log.Infow("Created new archives", "count", newArchiveCount, "repositoryID", repositoryID)
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

		e.qm.log.Infow("Updated existing archive",
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

		e.qm.log.Infow("Created new archive",
			"archiveName", archiveData.Name,
			"borgId", archiveData.ID,
			"repositoryID", repositoryID)
	}

	return nil
}

// refreshNewArchive refreshes a single newly created archive in the database
func (e *borgOperationExecutor) refreshNewArchive(ctx context.Context, repo *ent.Repository, archivePath string) error {
	// Use repository::archive syntax to list only the specific archive
	listResponse, status := e.borgClient.List(ctx, archivePath, repo.Password)

	if status.HasError() {
		return fmt.Errorf("failed to list archive %s: %w", archivePath, status.Error)
	}

	if len(listResponse.Archives) == 0 {
		return fmt.Errorf("archive %s not found in borg repository", archivePath)
	}

	// Sync only the single archive (no deletions)
	return e.syncSingleArchiveToDatabase(ctx, e.repoID, listResponse.Archives[0])
}
