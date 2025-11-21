package repository

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/negrel/assert"
)

// ============================================================================
// REPOSITORY QUEUE
// ============================================================================

// RepositoryQueue manages operations for a single repository
type RepositoryQueue struct {
	repoID        int
	operations    map[string]*QueuedOperation // By operation ID
	operationList []string                    // Ordered operation IDs (FIFO)
	active        *QueuedOperation            // ONE active operation per repository
	mu            sync.Mutex

	// Deduplication tracking
	activeBackups map[types.BackupId]string // BackupID -> OperationID
	activeDeletes map[int]string            // ArchiveID -> OperationID
	hasRepoDelete bool                      // Only one repo delete allowed
}

// NewRepositoryQueue creates a new queue for a repository
func NewRepositoryQueue(repoID int) *RepositoryQueue {
	return &RepositoryQueue{
		repoID:        repoID,
		operations:    make(map[string]*QueuedOperation),
		operationList: make([]string, 0),
		active:        nil,
		activeBackups: make(map[types.BackupId]string),
		activeDeletes: make(map[int]string),
		hasRepoDelete: false,
	}
}

func (q *RepositoryQueue) CreateQueuedOperation(operation statemachine.Operation, repoID int, backupProfileID *int, validUntil *time.Time, immediate bool) *QueuedOperation {
	return &QueuedOperation{
		ID:              uuid.New().String(),
		RepoID:          repoID,
		BackupProfileID: backupProfileID,
		Operation:       operation,
		Status:          NewOperationStatusQueued(Queued{Position: 0}), // Placeholder - will be corrected by updatePositions
		CreatedAt:       time.Now(),
		ValidUntil:      validUntil,
		Immediate:       immediate,
	}
}

// AddOperation adds an operation to the queue with deduplication
func (q *RepositoryQueue) AddOperation(op *QueuedOperation) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check for existing operation (idempotency)
	canAdd, existingOpID := q.canAddOperationLocked(op.Operation)
	if !canAdd {
		return existingOpID // Return existing operation ID
	}

	// Add to operations map
	q.operations[op.ID] = op

	// Add to ordered list
	q.operationList = append(q.operationList, op.ID)

	// Update tracking maps
	q.addToTrackingMaps(op)

	// Update positions for all queued operations
	q.updatePositions()

	return op.ID
}

// GetOperations returns all operations in the queue (including active)
func (q *RepositoryQueue) GetOperations(operationType *statemachine.OperationType) []*QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	var ops []*QueuedOperation

	// Add active operation first
	if q.active != nil {
		if operationType == nil || statemachine.GetOperationType(q.active.Operation) == *operationType {
			ops = append(ops, q.active)
		}
	}

	// Add queued operations in order
	for _, operationID := range q.operationList {
		if op, exists := q.operations[operationID]; exists {
			if operationType == nil || statemachine.GetOperationType(op.Operation) == *operationType {
				ops = append(ops, op)
			}
		}
	}

	return ops
}

// GetQueuedOperations returns only queued operations (excluding active)
func (q *RepositoryQueue) GetQueuedOperations(operationType *statemachine.OperationType) []*QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	var ops []*QueuedOperation

	// Add only queued operations in order (excluding active)
	for _, operationID := range q.operationList {
		if op, exists := q.operations[operationID]; exists {
			if operationType == nil || statemachine.GetOperationType(op.Operation) == *operationType {
				ops = append(ops, op)
			}
		}
	}

	return ops
}

// GetActive returns the currently active operation
func (q *RepositoryQueue) GetActive() *QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active
}

// GetNext returns the next operation to be processed (first in queue)
func (q *RepositoryQueue) GetNext() *QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.operationList) == 0 {
		return nil
	}

	firstOperationID := q.operationList[0]
	return q.operations[firstOperationID]
}

// RemoveOperation removes an operation from the queue
func (q *RepositoryQueue) RemoveOperation(operationID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if it's the active operation
	if q.active != nil && q.active.ID == operationID {
		q.removeFromTrackingMaps(q.active)
		q.active = nil
		return nil
	}

	// Find in queued operations
	if op, exists := q.operations[operationID]; exists {
		// Remove from tracking maps
		q.removeFromTrackingMaps(op)

		// Remove from operations map
		delete(q.operations, operationID)

		// Remove from ordered list
		for i, id := range q.operationList {
			if id == operationID {
				q.operationList = append(q.operationList[:i], q.operationList[i+1:]...)
				break
			}
		}

		// Update positions
		q.updatePositions()

		return nil
	}

	return fmt.Errorf("operation %s not found in repository %d queue", operationID, q.repoID)
}

// MoveToActive moves an operation from queue to active status
func (q *RepositoryQueue) MoveToActive(operationID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Find operation in queue
	op, exists := q.operations[operationID]
	if !exists {
		return fmt.Errorf("operation %s not found in repository %d queue", operationID, q.repoID)
	}

	// Remove from queue list
	for i, id := range q.operationList {
		if id == operationID {
			q.operationList = append(q.operationList[:i], q.operationList[i+1:]...)
			break
		}
	}

	// Remove from operations map
	delete(q.operations, operationID)

	// Set as active
	q.active = op

	// Update status to running
	q.active.Status = NewOperationStatusRunning(Running{
		StartedAt: time.Now(),
		Progress:  nil,
	})

	// Update positions for remaining operations
	q.updatePositions()

	return nil
}

// CompleteActive completes the active operation and clears it
func (q *RepositoryQueue) CompleteActive(errorMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.active == nil {
		return fmt.Errorf("no active operation to complete for repository %d", q.repoID)
	}

	// Update operation status
	if errorMsg == "" {
		q.active.Status = NewOperationStatusCompleted(Completed{
			CompletedAt: time.Now(),
		})
	} else {
		q.active.Status = NewOperationStatusFailed(Failed{
			Error:    errorMsg,
			FailedAt: time.Now(),
		})
	}

	// Remove from tracking maps
	q.removeFromTrackingMaps(q.active)

	// Clear active operation (could archive it here if needed)
	q.active = nil

	return nil
}

// FindOperation searches for an operation by type and returns its ID
func (q *RepositoryQueue) FindOperation(opType statemachine.Operation) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check active operation first
	if q.active != nil && reflect.TypeOf(q.active.Operation) == reflect.TypeOf(opType) {
		return q.active.ID
	}

	// Search queued operations
	for _, operationID := range q.operationList {
		if op, exists := q.operations[operationID]; exists {
			if reflect.TypeOf(op.Operation) == reflect.TypeOf(opType) {
				return operationID
			}
		}
	}

	return ""
}

// FindBackupOperation finds an existing backup operation for a BackupID
func (q *RepositoryQueue) FindBackupOperation(backupId types.BackupId) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if operationId, exists := q.activeBackups[backupId]; exists {
		return operationId
	}
	return ""
}

// FindArchiveDeleteOperation finds an existing archive delete operation
func (q *RepositoryQueue) FindArchiveDeleteOperation(archiveId int) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if operationId, exists := q.activeDeletes[archiveId]; exists {
		return operationId
	}
	return ""
}

// HasRepoDeleteOperation checks if there's already a repository delete operation
func (q *RepositoryQueue) HasRepoDeleteOperation() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.hasRepoDelete
}

// CanAddOperation checks if an operation can be added (deduplication check)
func (q *RepositoryQueue) CanAddOperation(op statemachine.Operation) (bool, string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	switch statemachine.GetOperationType(op) {
	case statemachine.OperationTypeBackup:
		backupVariant := op.(statemachine.BackupVariant)
		backupData := backupVariant()
		if existingOpID, exists := q.activeBackups[backupData.BackupID]; exists {
			return false, existingOpID
		}
	case statemachine.OperationTypeArchiveDelete:
		deleteVariant := op.(statemachine.ArchiveDeleteVariant)
		deleteData := deleteVariant()
		if existingOpID, exists := q.activeDeletes[deleteData.ArchiveID]; exists {
			return false, existingOpID
		}
	case statemachine.OperationTypeDelete:
		if q.hasRepoDelete {
			// Find the repository delete operation ID
			for _, operationID := range q.operationList {
				if opData, exists := q.operations[operationID]; exists {
					if statemachine.GetOperationType(opData.Operation) == statemachine.OperationTypeDelete {
						return false, operationID
					}
				}
			}
			// Check active operation
			if q.active != nil {
				if statemachine.GetOperationType(q.active.Operation) == statemachine.OperationTypeDelete {
					return false, q.active.ID
				}
			}
		}
	case statemachine.OperationTypePrune,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:
		// No deduplication needed for these operations
	default:
		assert.Fail("Unhandled OperationType in CanAddOperation")
	}

	return true, ""
}

// GetOperationByID retrieves a specific operation by ID
func (q *RepositoryQueue) GetOperationByID(operationID string) *QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check active operation first
	if q.active != nil && q.active.ID == operationID {
		return q.active
	}

	// Check queued operations
	return q.operations[operationID]
}

// GetOperationsByStatus returns operations filtered by status
func (q *RepositoryQueue) GetOperationsByStatus(status OperationStatus) []*QueuedOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	var result []*QueuedOperation

	// Check active operation
	if q.active != nil && isSameStatusType(q.active.Status, status) {
		result = append(result, q.active)
	}

	// Check queued operations
	for _, operationID := range q.operationList {
		if op, exists := q.operations[operationID]; exists {
			if isSameStatusType(op.Status, status) {
				result = append(result, op)
			}
		}
	}

	return result
}

// UpdateOperationStatus updates the status of an operation
func (q *RepositoryQueue) UpdateOperationStatus(operationID string, status OperationStatus) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check active operation first
	if q.active != nil && q.active.ID == operationID {
		q.active.Status = status
		return nil
	}

	// Check queued operations
	if op, exists := q.operations[operationID]; exists {
		op.Status = status
		return nil
	}

	return fmt.Errorf("operation %s not found in repository %d queue", operationID, q.repoID)
}

// GetQueueLength returns the number of queued operations (excluding active)
func (q *RepositoryQueue) GetQueueLength() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.operationList)
}

// IsEmpty returns true if the queue has no operations
func (q *RepositoryQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active == nil && len(q.operationList) == 0
}

// HasActiveOperation returns true if there's an active operation
func (q *RepositoryQueue) HasActiveOperation() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active != nil
}

// GetNextOperationType returns the type of the next operation in queue
func (q *RepositoryQueue) GetNextOperationType() statemachine.Operation {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.operationList) == 0 {
		return nil
	}

	firstOperationID := q.operationList[0]
	if op, exists := q.operations[firstOperationID]; exists {
		return op.Operation
	}

	return nil
}

// ExpireOldOperations removes operations that have passed their ValidUntil time
func (q *RepositoryQueue) ExpireOldOperations(now time.Time) []string {
	q.mu.Lock()
	defer q.mu.Unlock()

	var expiredIDs []string

	// Check queued operations (don't expire active operations)
	for i := len(q.operationList) - 1; i >= 0; i-- {
		operationID := q.operationList[i]
		if op, exists := q.operations[operationID]; exists {
			if op.ValidUntil != nil && op.ValidUntil.Before(now) {
				// Mark as expired
				op.Status = NewOperationStatusExpired(Expired{
					ExpiredAt: now,
				})

				// Remove from tracking maps
				q.removeFromTrackingMaps(op)

				// Remove from operations and list
				delete(q.operations, operationID)
				q.operationList = append(q.operationList[:i], q.operationList[i+1:]...)

				expiredIDs = append(expiredIDs, operationID)
			}
		}
	}

	// Update positions if any operations were removed
	if len(expiredIDs) > 0 {
		q.updatePositions()
	}

	return expiredIDs
}

// updatePositions recalculates position numbers for all queued operations
func (q *RepositoryQueue) updatePositions() {
	// Iterate through operationList and update positions
	for i, operationID := range q.operationList {
		if op, exists := q.operations[operationID]; exists {
			// Only update if status is currently queued
			if _, isQueued := op.Status.(QueuedVariant); isQueued {
				op.Status = NewOperationStatusQueued(Queued{
					Position: i + 1, // 1-based position
				})
			}
		}
	}
}

// addToTrackingMaps adds operation to appropriate deduplication tracking
func (q *RepositoryQueue) addToTrackingMaps(op *QueuedOperation) {
	switch statemachine.GetOperationType(op.Operation) {
	case statemachine.OperationTypeBackup:
		backupVariant := op.Operation.(statemachine.BackupVariant)
		backupData := backupVariant()
		q.activeBackups[backupData.BackupID] = op.ID
	case statemachine.OperationTypeArchiveDelete:
		deleteVariant := op.Operation.(statemachine.ArchiveDeleteVariant)
		deleteData := deleteVariant()
		q.activeDeletes[deleteData.ArchiveID] = op.ID
	case statemachine.OperationTypeDelete:
		q.hasRepoDelete = true
	case statemachine.OperationTypePrune,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:
		// No tracking needed for these operations
	default:
		assert.Fail("Unhandled OperationType in addToTrackingMaps")
	}
}

// removeFromTrackingMaps removes operation from deduplication tracking
func (q *RepositoryQueue) removeFromTrackingMaps(op *QueuedOperation) {
	switch statemachine.GetOperationType(op.Operation) {
	case statemachine.OperationTypeBackup:
		backupVariant := op.Operation.(statemachine.BackupVariant)
		backupData := backupVariant()
		delete(q.activeBackups, backupData.BackupID)
	case statemachine.OperationTypeArchiveDelete:
		deleteVariant := op.Operation.(statemachine.ArchiveDeleteVariant)
		deleteData := deleteVariant()
		delete(q.activeDeletes, deleteData.ArchiveID)
	case statemachine.OperationTypeDelete:
		q.hasRepoDelete = false
	case statemachine.OperationTypePrune,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:
		// No tracking to remove for these operations
	default:
		assert.Fail("Unhandled OperationType in removeFromTrackingMaps")
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// isSameStatusType checks if two OperationStatus values have the same underlying type
func isSameStatusType(status1, status2 OperationStatus) bool {
	return reflect.TypeOf(status1) == reflect.TypeOf(status2)
}

// canAddOperationLocked checks if an operation can be added (assumes caller holds mutex)
func (q *RepositoryQueue) canAddOperationLocked(op statemachine.Operation) (bool, string) {
	switch statemachine.GetOperationType(op) {
	case statemachine.OperationTypeBackup:
		backupVariant := op.(statemachine.BackupVariant)
		backupData := backupVariant()
		if existingOpID, exists := q.activeBackups[backupData.BackupID]; exists {
			return false, existingOpID
		}
	case statemachine.OperationTypeArchiveDelete:
		deleteVariant := op.(statemachine.ArchiveDeleteVariant)
		deleteData := deleteVariant()
		if existingOpID, exists := q.activeDeletes[deleteData.ArchiveID]; exists {
			return false, existingOpID
		}
	case statemachine.OperationTypeDelete:
		if q.hasRepoDelete {
			// Find the repository delete operation ID
			for _, operationID := range q.operationList {
				if opData, exists := q.operations[operationID]; exists {
					if statemachine.GetOperationType(opData.Operation) == statemachine.OperationTypeDelete {
						return false, operationID
					}
				}
			}
			// Check active operation
			if q.active != nil {
				if statemachine.GetOperationType(q.active.Operation) == statemachine.OperationTypeDelete {
					return false, q.active.ID
				}
			}
		}
	case statemachine.OperationTypePrune,
		statemachine.OperationTypeArchiveRefresh,
		statemachine.OperationTypeCheck,
		statemachine.OperationTypeArchiveRename,
		statemachine.OperationTypeMount,
		statemachine.OperationTypeMountArchive,
		statemachine.OperationTypeUnmount,
		statemachine.OperationTypeUnmountArchive,
		statemachine.OperationTypeExaminePrune:
		// No deduplication needed for these operations
	default:
		assert.Fail("Unhandled OperationType in canAddOperationLocked")
	}

	return true, ""
}
