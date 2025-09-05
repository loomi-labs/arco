package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/ent"
)

// ============================================================================
// QUEUE MANAGER
// ============================================================================

// QueueManager manages operation queues for all repositories
type QueueManager struct {
	stateMachine *statemachine.RepositoryStateMachine
	db           *ent.Client
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
func NewQueueManager(stateMachine *statemachine.RepositoryStateMachine, maxHeavyOps int) *QueueManager {
	return &QueueManager{
		stateMachine:     stateMachine,
		queues:           make(map[int]*RepositoryQueue),
		repositoryStates: make(map[int]statemachine.RepositoryState),
		maxHeavyOps:      maxHeavyOps,
		activeHeavy:      make(map[int]*QueuedOperation),
		activeLight:      make(map[int]*QueuedOperation),
	}
}

// SetDB sets the database client for the queue manager
func (qm *QueueManager) SetDB(db *ent.Client) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.db = db
}

// GetRepositoryState returns the current state of a repository (defaults to idle if not set)
func (qm *QueueManager) GetRepositoryState(repoID int) statemachine.RepositoryState {
	qm.statesMu.RLock()
	defer qm.statesMu.RUnlock()

	if state, exists := qm.repositoryStates[repoID]; exists {
		return state
	}

	// Default to idle state for new repositories
	return statemachine.NewStateIdle(statemachine.StateIdle{})
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

	// TODO: Start actual operation execution

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

		// TODO: Cancel operation context if running

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
	err := qm.StartOperation(context.Background(), repoID, nextOp.ID)
	if err != nil {
		// Log error but don't crash
		// TODO: Add proper logging
		_ = err
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

// getTargetStateForOperation maps an operation to its corresponding active state
func (qm *QueueManager) getTargetStateForOperation(op *QueuedOperation) (statemachine.RepositoryState, error) {
	switch v := op.Operation.(type) {
	case statemachine.BackupVariant:
		backupData := v()
		return statemachine.CreateBackingUpState(context.Background(), backupData.BackupID), nil

	case statemachine.PruneVariant:
		pruneData := v()
		return statemachine.CreatePruningState(context.Background(), pruneData.BackupID), nil

	case statemachine.DeleteVariant:
		return statemachine.CreateDeletingState(context.Background(), 0), nil // Repository delete, no specific archive

	case statemachine.ArchiveRefreshVariant:
		return statemachine.CreateRefreshingState(context.Background()), nil

	case statemachine.ArchiveDeleteVariant:
		deleteData := v()
		return statemachine.CreateDeletingState(context.Background(), deleteData.ArchiveID), nil

	case statemachine.ArchiveRenameVariant:
		// Archive rename is a lightweight operation, treat as refreshing
		return statemachine.CreateRefreshingState(context.Background()), nil

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
