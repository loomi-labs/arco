package repository

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// QUEUE MANAGER
// ============================================================================

// QueueManager manages operation queues for all repositories
type QueueManager struct {
	queues map[int]*RepositoryQueue // RepoID -> Queue
	mu     sync.RWMutex

	// Cross-repository concurrency control
	maxHeavyOps int                      // Max heavy operations across all repositories
	activeHeavy map[int]*QueuedOperation // RepoID -> active heavy operation
	activeLight map[int]*QueuedOperation // RepoID -> active light operation
}

// NewQueueManager creates a new QueueManager with specified concurrency limits
func NewQueueManager(maxHeavyOps int) *QueueManager {
	return &QueueManager{
		queues:      make(map[int]*RepositoryQueue),
		maxHeavyOps: maxHeavyOps,
		activeHeavy: make(map[int]*QueuedOperation),
		activeLight: make(map[int]*QueuedOperation),
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
	weight := GetOperationWeight(op.Operation)
	if weight == WeightHeavy {
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
	weight := GetOperationWeight(op.Operation)
	if weight == WeightHeavy {
		qm.activeHeavy[repoID] = op
	} else {
		qm.activeLight[repoID] = op
	}
	qm.mu.Unlock()

	// TODO: Transition repository state via state machine
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

	weight := GetOperationWeight(activeOp.Operation)

	// Remove from active tracking
	qm.mu.Lock()
	if weight == WeightHeavy {
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

	// TODO: Transition repository state via state machine

	// Attempt to start next queued operation for this repo
	qm.processQueue(repoID)

	// If we completed a heavy operation, try to start waiting heavy operations on other repos
	if weight == WeightHeavy {
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
		weight := GetOperationWeight(activeOp.Operation)

		// Remove from active tracking
		qm.mu.Lock()
		if weight == WeightHeavy {
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
