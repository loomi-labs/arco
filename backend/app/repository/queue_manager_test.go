package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	typesmocks "github.com/loomi-labs/arco/backend/app/types/mocks"
	"github.com/loomi-labs/arco/backend/borg/mocks"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/enttest"
	"github.com/stretchr/testify/assert"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// TEST HELPERS
// ============================================================================

// newTestQueueManager creates a test queue manager with in-memory database for testing
func newTestQueueManager(t *testing.T) (*QueueManager, *ent.Client, context.Context, *typesmocks.MockEventEmitter) {
	// Initialize a minimal Wails application for testing
	// This is needed because queue manager code calls application.Get().Context()
	_ = application.New(application.Options{
		Name: "test-app",
	})

	// Create in-memory SQLite database
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()

	// Create gomock controller and mock Borg client
	ctrl := gomock.NewController(t)
	mockBorgClient := mocks.NewMockBorg(ctrl)

	// Set up default expectations: all borg operations return success
	// Use AnyTimes() to allow any number of calls without failing
	mockBorgClient.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&borgtypes.InfoResponse{}, &borgtypes.Status{}).AnyTimes()
	mockBorgClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("test-archive", &borgtypes.Status{}).AnyTimes()
	mockBorgClient.EXPECT().Prune(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&borgtypes.Status{}).AnyTimes()
	mockBorgClient.EXPECT().Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&borgtypes.Status{}).AnyTimes()
	mockBorgClient.EXPECT().DeleteArchive(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&borgtypes.Status{}).AnyTimes()

	// Create mock event emitter
	mockEmitter := typesmocks.NewMockEventEmitter(ctrl)
	mockEmitter.EXPECT().EmitEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	log := zap.NewNop().Sugar()
	stateMachine := statemachine.NewRepositoryStateMachine()

	// Create queue manager
	qm := NewQueueManager(log, stateMachine, 2) // max 2 heavy operations

	// Set the queue manager reference in state machine
	stateMachine.SetQueueManager(qm)

	// Initialize with real database and mock borg
	qm.Init(db, mockBorgClient, mockEmitter)

	return qm, db, ctx, mockEmitter
}

// createTestRepository creates a test repository in the database
func createTestRepository(t *testing.T, db *ent.Client, ctx context.Context, id int) *ent.Repository {
	repo, err := db.Repository.Create().
		SetID(id).
		SetName(fmt.Sprintf("test-repo-%d", id)).
		SetURL(fmt.Sprintf("/tmp/test-repo-%d", id)).
		SetPassword("test-password").
		Save(ctx)
	assert.NoError(t, err)
	return repo
}

// createTestBackupProfile creates a test backup profile in the database
func createTestBackupProfile(t *testing.T, db *ent.Client, ctx context.Context, id int, repoID int) *ent.BackupProfile {
	profile, err := db.BackupProfile.Create().
		SetID(id).
		SetName(fmt.Sprintf("test-backup-%d", id)).
		SetPrefix(fmt.Sprintf("test%d-", id)).
		SetIcon("home").
		AddRepositoryIDs(repoID).
		Save(ctx)
	assert.NoError(t, err)
	return profile
}

// ============================================================================
// TESTS
// ============================================================================

// TestAddOperation_StateNotOverwrittenWithActiveOperation tests that when an active
// operation exists, adding a new queued operation doesn't overwrite the repository
// state to Queued. This validates the bug fix where repository state was incorrectly
// being set to Queued even while a backup was actively running.
func TestAddOperation_StateNotOverwrittenWithActiveOperation(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	backupID1 := types.BackupId{RepositoryId: repoID, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID, BackupProfileId: 101}

	// Create test data in database
	createTestRepository(t, db, ctx, repoID)
	createTestBackupProfile(t, db, ctx, 100, repoID)
	createTestBackupProfile(t, db, ctx, 101, repoID)

	// Set initial state to BackingUp (simulating an active backup)
	// We set it directly to avoid calling application.Get() which isn't initialized in tests
	backingUpState := statemachine.NewRepositoryStateBackingUp(statemachine.BackingUp{
		Data: statemachine.Backup{
			BackupID: backupID1,
		},
	})
	qm.statesMu.Lock()
	qm.repositoryStates[repoID] = backingUpState
	qm.statesMu.Unlock()

	// Create first backup operation and manually set it as active
	// (simulating that it's currently running)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusRunning(Running{}),
		Immediate: false,
	}

	queue := qm.GetQueue(repoID)
	operationID1 := queue.AddOperation(op1)
	err := queue.MoveToActive(operationID1)
	assert.NoError(t, err)

	// Track the operation as active in queue manager
	qm.mu.Lock()
	qm.activeHeavy[repoID] = op1
	qm.mu.Unlock()

	// ACT - Add a second backup operation while first is active
	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	_, err = qm.AddOperation(repoID, op2)

	// ASSERT
	assert.NoError(t, err)

	// Verify repository state is STILL BackingUp (NOT Queued)
	// This is the critical assertion - the bug would cause state to become Queued
	currentState := qm.GetRepositoryState(repoID)
	currentStateType := statemachine.GetRepositoryStateType(currentState)
	assert.Equal(t, statemachine.RepositoryStateTypeBackingUp, currentStateType,
		"Repository state should remain BackingUp when active operation exists, but got %s", currentStateType)

	// Verify the BackingUp state still contains the original backup ID
	backingUpVariant := currentState.(statemachine.BackingUpVariant)
	backingUpData := backingUpVariant()
	assert.Equal(t, backupID1.String(), backingUpData.Data.BackupID.String(),
		"BackingUp state should still reference the first backup")

	// Verify second operation was added to queue
	queuedOps := queue.GetQueuedOperations(nil)
	assert.Len(t, queuedOps, 1, "Expected second operation to be queued")
}

// ============================================================================
// PHASE 1: BASIC OPERATION MANAGEMENT
// ============================================================================

// TestAddOperation_BackupIdempotency verifies that adding the same backup operation
// twice returns the same operation ID and only creates one operation in the queue.
func TestAddOperation_BackupIdempotency(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	backupID := types.BackupId{RepositoryId: repoID, BackupProfileId: 100}

	// Create test data
	createTestRepository(t, db, ctx, repoID)
	createTestBackupProfile(t, db, ctx, 100, repoID)

	op := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	// ACT - Add the same backup operation twice
	operationID1, err1 := qm.AddOperation(repoID, op)
	operationID2, err2 := qm.AddOperation(repoID, op)

	// ASSERT
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Should return the same operation ID (idempotency)
	assert.Equal(t, operationID1, operationID2, "Adding same backup twice should return same operation ID")

	// Operation may have started immediately (in active) or be queued
	// Either way, there should only be one operation total
	queue := qm.GetQueue(repoID)
	totalOps := 0
	if queue.HasActiveOperation() {
		totalOps = 1
	}
	totalOps += len(queue.GetQueuedOperations(nil))
	assert.Equal(t, 1, totalOps, "Should only have one backup operation total (active or queued)")
}

// TestAddOperation_ArchiveDeleteIdempotency verifies that adding the same archive delete
// operation twice returns the same operation ID.
func TestAddOperation_ArchiveDeleteIdempotency(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	const archiveID = 500

	// Create test data
	createTestRepository(t, db, ctx, repoID)

	// Use archive delete to test idempotency
	op := &QueuedOperation{
		Operation: statemachine.NewOperationArchiveDelete(statemachine.ArchiveDelete{
			ArchiveID: archiveID,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	// ACT - Add the same archive delete operation twice
	operationID1, err1 := qm.AddOperation(repoID, op)
	operationID2, err2 := qm.AddOperation(repoID, op)

	// ASSERT
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Should return the same operation ID (idempotency)
	assert.Equal(t, operationID1, operationID2, "Adding same archive delete operation twice should return same operation ID")

	// Operation may have started immediately (in active) or be queued
	queue := qm.GetQueue(repoID)
	totalOps := 0
	if queue.HasActiveOperation() {
		totalOps = 1
	}
	totalOps += len(queue.GetQueuedOperations(nil))
	assert.Equal(t, 1, totalOps, "Should only have one archive operation total (active or queued)")
}

// ============================================================================
// PHASE 2: IMMEDIATE FLAG VALIDATION
// ============================================================================

// TestAddOperation_ImmediateLightSucceedsWithoutActive verifies that a light
// immediate operation succeeds when there's no active operation.
func TestAddOperation_ImmediateLightSucceedsWithoutActive(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	const archiveID = 500

	// Create test data
	createTestRepository(t, db, ctx, repoID)

	// Archive rename is a light operation
	op := &QueuedOperation{
		Operation: statemachine.NewOperationArchiveRename(statemachine.ArchiveRename{
			ArchiveID: archiveID,
			Name:      "new-name",
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: true, // Immediate flag set
	}

	// ACT - Add immediate light operation with no active operations
	_, err := qm.AddOperation(repoID, op)

	// ASSERT
	assert.NoError(t, err, "Immediate light operation should succeed when no active operation exists")

	// Operation should have started immediately
	queue := qm.GetQueue(repoID)
	assert.True(t, queue.HasActiveOperation(), "Immediate operation should be active")
}

// TestAddOperation_ImmediateFailsWithActiveOperation verifies that an immediate
// operation fails when there's already an active operation.
func TestAddOperation_ImmediateFailsWithActiveOperation(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	backupID1 := types.BackupId{RepositoryId: repoID, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID, BackupProfileId: 101}

	// Create test data
	createTestRepository(t, db, ctx, repoID)
	createTestBackupProfile(t, db, ctx, 100, repoID)
	createTestBackupProfile(t, db, ctx, 101, repoID)

	// Set up an active operation
	backingUpState := statemachine.NewRepositoryStateBackingUp(statemachine.BackingUp{
		Data: statemachine.Backup{
			BackupID: backupID1,
		},
	})
	qm.statesMu.Lock()
	qm.repositoryStates[repoID] = backingUpState
	qm.statesMu.Unlock()

	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusRunning(Running{}),
		Immediate: false,
	}

	queue := qm.GetQueue(repoID)
	operationID1 := queue.AddOperation(op1)
	err := queue.MoveToActive(operationID1)
	assert.NoError(t, err)
	qm.mu.Lock()
	qm.activeHeavy[repoID] = op1
	qm.mu.Unlock()

	// ACT - Try to add immediate operation while another is active
	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: true, // Immediate flag set
	}

	_, err = qm.AddOperation(repoID, op2)

	// ASSERT
	assert.Error(t, err, "Immediate operation should fail when active operation exists")
	assert.Contains(t, err.Error(), "cannot start immediate operation", "Error message should indicate immediate operation blocked")
}

// TestAddOperation_ImmediateHeavyFailsWithQueuedOps verifies that an immediate heavy
// operation fails when there are queued operations.
func TestAddOperation_ImmediateHeavyFailsWithQueuedOps(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	backupID1 := types.BackupId{RepositoryId: repoID, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID, BackupProfileId: 101}

	// Create test data
	createTestRepository(t, db, ctx, repoID)
	createTestBackupProfile(t, db, ctx, 100, repoID)
	createTestBackupProfile(t, db, ctx, 101, repoID)

	// Add a queued operation (not immediate)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	queue := qm.GetQueue(repoID)
	_ = queue.AddOperation(op1)

	// ACT - Try to add immediate heavy operation when there are queued ops
	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: true, // Immediate heavy operation
	}

	_, err := qm.AddOperation(repoID, op2)

	// ASSERT
	assert.Error(t, err, "Immediate heavy operation should fail when queued operations exist")
	assert.Contains(t, err.Error(), "cannot start immediate heavy operation", "Error message should indicate immediate heavy operation blocked by queue")
}

// ============================================================================
// PHASE 3: STATE TRANSITIONS
// ============================================================================

// TestAddOperation_StateTransitionsWhenOperationStarts verifies that repository state
// transitions to the appropriate active state when an operation starts.
func TestAddOperation_StateTransitionsWhenOperationStarts(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID = 1
	backupID := types.BackupId{RepositoryId: repoID, BackupProfileId: 100}

	// Create test data
	createTestRepository(t, db, ctx, repoID)
	createTestBackupProfile(t, db, ctx, 100, repoID)

	// Repository starts in Idle state (default)
	initialState := qm.GetRepositoryState(repoID)
	assert.Equal(t, statemachine.RepositoryStateTypeIdle, statemachine.GetRepositoryStateType(initialState))

	// ACT - Add a backup operation (should start immediately)
	op := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	_, err := qm.AddOperation(repoID, op)

	// ASSERT
	assert.NoError(t, err)

	// Verify repository state transitioned to BackingUp
	currentState := qm.GetRepositoryState(repoID)
	currentStateType := statemachine.GetRepositoryStateType(currentState)
	assert.Equal(t, statemachine.RepositoryStateTypeBackingUp, currentStateType,
		"Repository state should transition to BackingUp when backup starts")

	// Verify the BackingUp state contains the correct backup ID
	backingUpVariant := currentState.(statemachine.BackingUpVariant)
	backingUpData := backingUpVariant()
	assert.Equal(t, backupID.String(), backingUpData.Data.BackupID.String(),
		"BackingUp state should reference the started backup")
}

// TestAddOperation_StateSetToQueuedWithoutActiveOperation verifies that repository state
// transitions to Queued when an operation can't start due to concurrency limits.
func TestAddOperation_StateSetToQueuedWithoutActiveOperation(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID1 = 1
	const repoID2 = 2
	const repoID3 = 3
	backupID1 := types.BackupId{RepositoryId: repoID1, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID2, BackupProfileId: 101}
	backupID3 := types.BackupId{RepositoryId: repoID3, BackupProfileId: 102}

	// Create test data
	createTestRepository(t, db, ctx, repoID1)
	createTestRepository(t, db, ctx, repoID2)
	createTestRepository(t, db, ctx, repoID3)
	createTestBackupProfile(t, db, ctx, 100, repoID1)
	createTestBackupProfile(t, db, ctx, 101, repoID2)
	createTestBackupProfile(t, db, ctx, 102, repoID3)

	// Start 2 heavy operations on different repos (reaching max limit of 2)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err := qm.AddOperation(repoID1, op1)
	assert.NoError(t, err)

	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID2, op2)
	assert.NoError(t, err)

	// ACT - Add a third heavy operation (should be queued, not started)
	op3 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID3,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	_, err = qm.AddOperation(repoID3, op3)

	// ASSERT
	assert.NoError(t, err)

	// Verify repository state is Queued (not BackingUp) because operation couldn't start
	currentState := qm.GetRepositoryState(repoID3)
	currentStateType := statemachine.GetRepositoryStateType(currentState)
	assert.Equal(t, statemachine.RepositoryStateTypeQueued, currentStateType,
		"Repository state should be Queued when operation can't start due to concurrency limit")

	// Verify no active operation on repo3
	queue := qm.GetQueue(repoID3)
	assert.False(t, queue.HasActiveOperation(), "Repository should have no active operation")

	// Verify operation is in queued list
	queuedOps := queue.GetQueuedOperations(nil)
	assert.Len(t, queuedOps, 1, "Operation should be in queued list")
}

// ============================================================================
// PHASE 4: CONCURRENCY CONTROL
// ============================================================================

// TestAddOperation_HeavyOperationWaitsForConcurrencyLimit verifies that heavy operations
// wait when the global concurrency limit is reached.
func TestAddOperation_HeavyOperationWaitsForConcurrencyLimit(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID1 = 1
	const repoID2 = 2
	const repoID3 = 3
	backupID1 := types.BackupId{RepositoryId: repoID1, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID2, BackupProfileId: 101}
	backupID3 := types.BackupId{RepositoryId: repoID3, BackupProfileId: 102}

	// Create test data
	createTestRepository(t, db, ctx, repoID1)
	createTestRepository(t, db, ctx, repoID2)
	createTestRepository(t, db, ctx, repoID3)
	createTestBackupProfile(t, db, ctx, 100, repoID1)
	createTestBackupProfile(t, db, ctx, 101, repoID2)
	createTestBackupProfile(t, db, ctx, 102, repoID3)

	// Start 2 heavy operations (max limit)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err := qm.AddOperation(repoID1, op1)
	assert.NoError(t, err)

	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID2, op2)
	assert.NoError(t, err)

	// Verify both are active
	assert.Len(t, qm.activeHeavy, 2, "Should have 2 active heavy operations")

	// ACT - Try to add a third heavy operation
	op3 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID3,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	_, err = qm.AddOperation(repoID3, op3)

	// ASSERT
	assert.NoError(t, err)

	// Third operation should be queued, not active
	queue3 := qm.GetQueue(repoID3)
	assert.False(t, queue3.HasActiveOperation(), "Third operation should not be active")
	assert.Len(t, queue3.GetQueuedOperations(nil), 1, "Third operation should be queued")

	// Should still only have 2 active heavy operations
	assert.Len(t, qm.activeHeavy, 2, "Should still have only 2 active heavy operations")
}

// TestAddOperation_LightOperationStartsRegardlessOfHeavyLimit verifies that light
// operations can start even when heavy operation limit is reached.
func TestAddOperation_LightOperationStartsRegardlessOfHeavyLimit(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID1 = 1
	const repoID2 = 2
	const repoID3 = 3
	backupID1 := types.BackupId{RepositoryId: repoID1, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID2, BackupProfileId: 101}

	// Create test data
	createTestRepository(t, db, ctx, repoID1)
	createTestRepository(t, db, ctx, repoID2)
	createTestRepository(t, db, ctx, repoID3)
	createTestBackupProfile(t, db, ctx, 100, repoID1)
	createTestBackupProfile(t, db, ctx, 101, repoID2)

	// Start 2 heavy operations (max limit)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err := qm.AddOperation(repoID1, op1)
	assert.NoError(t, err)

	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID2, op2)
	assert.NoError(t, err)

	// ACT - Add a light operation (archive rename)
	op3 := &QueuedOperation{
		Operation: statemachine.NewOperationArchiveRename(statemachine.ArchiveRename{
			ArchiveID: 500,
			Name:      "new-name",
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}

	_, err = qm.AddOperation(repoID3, op3)

	// ASSERT
	assert.NoError(t, err)

	// Light operation should be active despite heavy limit
	queue3 := qm.GetQueue(repoID3)
	assert.True(t, queue3.HasActiveOperation(), "Light operation should be active even when heavy limit reached")

	// Should have the light operation in activeLight map
	qm.mu.RLock()
	_, hasLightOp := qm.activeLight[repoID3]
	qm.mu.RUnlock()
	assert.True(t, hasLightOp, "Light operation should be tracked in activeLight")
}

// ============================================================================
// PHASE 5: INTEGRATION
// ============================================================================

// TestAddOperation_Integration_MultipleRepositories tests the queue manager with
// multiple repositories in various states to verify cross-repository coordination.
func TestAddOperation_Integration_MultipleRepositories(t *testing.T) {
	// ARRANGE
	qm, db, ctx, _ := newTestQueueManager(t)

	const repoID1 = 1
	const repoID2 = 2
	const repoID3 = 3
	backupID1 := types.BackupId{RepositoryId: repoID1, BackupProfileId: 100}
	backupID2 := types.BackupId{RepositoryId: repoID2, BackupProfileId: 101}
	backupID3a := types.BackupId{RepositoryId: repoID3, BackupProfileId: 102}
	backupID3b := types.BackupId{RepositoryId: repoID3, BackupProfileId: 103}

	// Create test data
	createTestRepository(t, db, ctx, repoID1)
	createTestRepository(t, db, ctx, repoID2)
	createTestRepository(t, db, ctx, repoID3)
	createTestBackupProfile(t, db, ctx, 100, repoID1)
	createTestBackupProfile(t, db, ctx, 101, repoID2)
	createTestBackupProfile(t, db, ctx, 102, repoID3)
	createTestBackupProfile(t, db, ctx, 103, repoID3)

	// ACT & ASSERT

	// Repo 1: Start a backup (should be active)
	op1 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID1,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err := qm.AddOperation(repoID1, op1)
	assert.NoError(t, err)
	assert.True(t, qm.GetQueue(repoID1).HasActiveOperation(), "Repo 1 should have active backup")
	assert.Equal(t, statemachine.RepositoryStateTypeBackingUp, statemachine.GetRepositoryStateType(qm.GetRepositoryState(repoID1)))

	// Repo 2: Start a backup (should be active, reaching max heavy ops)
	op2 := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID2,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID2, op2)
	assert.NoError(t, err)
	assert.True(t, qm.GetQueue(repoID2).HasActiveOperation(), "Repo 2 should have active backup")
	assert.Len(t, qm.activeHeavy, 2, "Should have 2 active heavy operations")

	// Repo 3: Try to start a backup (should be queued due to concurrency limit)
	op3a := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID3a,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID3, op3a)
	assert.NoError(t, err)
	assert.False(t, qm.GetQueue(repoID3).HasActiveOperation(), "Repo 3 should NOT have active backup yet")
	assert.Equal(t, statemachine.RepositoryStateTypeQueued, statemachine.GetRepositoryStateType(qm.GetRepositoryState(repoID3)))

	// Repo 3: Add another backup (should be queued, idempotency should NOT apply because different backup ID)
	op3b := &QueuedOperation{
		Operation: statemachine.NewOperationBackup(statemachine.Backup{
			BackupID: backupID3b,
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID3, op3b)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(repoID3).GetQueuedOperations(nil), 2, "Repo 3 should have 2 queued backups")

	// Repo 3: Add a light operation (should start immediately despite heavy limit)
	op3Light := &QueuedOperation{
		Operation: statemachine.NewOperationArchiveRename(statemachine.ArchiveRename{
			ArchiveID: 500,
			Name:      "test-name",
		}),
		Status:    NewOperationStatusQueued(Queued{}),
		Immediate: false,
	}
	_, err = qm.AddOperation(repoID3, op3Light)
	assert.NoError(t, err)
	// Note: Light operation might not become active if there are heavy ops queued in same repo
	// This is expected behavior

	// Verify global state
	assert.Len(t, qm.activeHeavy, 2, "Should maintain 2 active heavy operations")
}
