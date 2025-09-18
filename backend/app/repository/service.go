package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/loomi-labs/arco/backend/util"
	"go.uber.org/zap"
)

// ============================================================================
// SERVICE INFRASTRUCTURE
// ============================================================================

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log          *zap.SugaredLogger
	config       *types.Config
	queueManager *QueueManager
	stateMachine *statemachine.RepositoryStateMachine

	// Dependencies to be set via Init()
	db              *ent.Client
	eventEmitter    types.EventEmitter
	borgClient      borg.Borg
	cloudRepoClient *CloudRepositoryClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new repository service instance
func NewService(log *zap.SugaredLogger, config *types.Config) *ServiceInternal {
	var maxHeavyOperations = 1
	var stateMachine = statemachine.NewRepositoryStateMachine()
	var queueManager = NewQueueManager(log, stateMachine, maxHeavyOperations)
	stateMachine.SetQueueManager(queueManager)

	return &ServiceInternal{
		Service: &Service{
			log:          log,
			config:       config,
			queueManager: queueManager,
			stateMachine: stateMachine,
		},
	}
}

// Init initializes the service with remaining dependencies
func (si *ServiceInternal) Init(db *ent.Client, eventEmitter types.EventEmitter, borgClient borg.Borg, cloudRepoClient *CloudRepositoryClient) {
	si.db = db
	si.eventEmitter = eventEmitter
	si.borgClient = borgClient
	si.cloudRepoClient = cloudRepoClient

	// Initialize queue manager with database and borg clients
	si.queueManager.Init(db, si.borgClient, si.eventEmitter)

	// TODO: Start periodic cleanup goroutine
	// go si.startPeriodicCleanup(ctx)
}

// ============================================================================
// CORE REPOSITORY METHODS
// ============================================================================

// All retrieves all repositories
func (s *Service) All(ctx context.Context) ([]*Repository, error) {
	// Query database for all repositories with eager loading
	repoEntities, err := s.db.Repository.Query().
		WithArchives(func(aq *ent.ArchiveQuery) {
			// Order by creation time descending to get most recent first
			aq.Order(ent.Desc(archive.FieldCreatedAt))
		}).
		WithCloudRepository().
		Order(func(sel *sql.Selector) {
			// Order by name, case-insensitive
			sel.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", repository.FieldName)))
		}).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}

	// Transform each entity to Repository struct using helper method
	repositories := make([]*Repository, len(repoEntities))
	for i, repoEntity := range repoEntities {
		repositories[i] = s.entityToRepository(ctx, repoEntity)
	}

	return repositories, nil
}

// AllWithQueue retrieves all repositories with queue information
func (s *Service) AllWithQueue(ctx context.Context) ([]*RepositoryWithQueue, error) {
	// TODO: Implement all repositories with queue retrieval
	return nil, nil
}

// Get retrieves a repository by ID
func (s *Service) Get(ctx context.Context, repoId int) (*Repository, error) {
	repoEntity, err := s.db.Repository.Query().
		Where(repository.ID(repoId)).
		WithArchives(func(aq *ent.ArchiveQuery) {
			// Order by creation time descending to get most recent first
			aq.Order(ent.Desc(archive.FieldCreatedAt))
		}).
		WithCloudRepository().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("repository with ID %d not found", repoId)
		}
		return nil, fmt.Errorf("failed to query repository %d: %w", repoId, err)
	}

	return s.entityToRepository(ctx, repoEntity), nil
}

// GetWithQueue retrieves a repository with queue information
func (s *Service) GetWithQueue(ctx context.Context, repoId int) (*RepositoryWithQueue, error) {
	// 1. Get base repository
	baseRepo, err := s.Get(ctx, repoId)
	if err != nil {
		return nil, err // Critical error - repository doesn't exist or can't be retrieved
	}

	// 2. Get queued operations from QueueManager
	queuedOps, err := s.queueManager.GetQueuedOperations(repoId)
	if err != nil {
		return nil, err
	}

	// 3. Get active operation from repository queue
	queue := s.queueManager.GetQueue(repoId)
	activeOp := queue.GetActive() // Can be nil if no operation is active

	// 4. Create and return RepositoryWithQueue struct
	return &RepositoryWithQueue{
		Repository:       *baseRepo,
		QueuedOperations: queuedOps,
		ActiveOperation:  activeOp,
	}, nil
}

// entityToRepository converts an ent.Repository entity to a Repository struct
// Expects the entity to have Archives and CloudRepository edges loaded
func (s *Service) entityToRepository(ctx context.Context, repoEntity *ent.Repository) *Repository {
	// Calculate current state from queue manager
	currentState := s.queueManager.GetRepositoryState(repoEntity.ID)

	// Determine repository type based on repository properties
	var repoType Location
	if repoEntity.Edges.CloudRepository != nil {
		// ArcoCloud repository
		repoType = NewLocationArcoCloud(ArcoCloud{
			CloudID: repoEntity.Edges.CloudRepository.CloudID,
		})
	} else if strings.HasPrefix(repoEntity.URL, "/") {
		// Local repository (path starts with /)
		repoType = NewLocationLocal(Local{})
	} else {
		// Remote repository (SSH)
		repoType = NewLocationRemote(Remote{})
	}

	// Extract archive metadata
	archives := repoEntity.Edges.Archives
	archiveCount := len(archives)

	var lastBackupTime *time.Time
	if len(archives) > 0 {
		// Archives are already ordered by creation time descending
		lastBackupTime = &archives[0].CreatedAt
	}

	// Calculate storage used from repository statistics
	storageUsed := int64(repoEntity.StatsUniqueSize)

	// Get last backup error and warning messages
	lastBackupError := s.getLastError(ctx, repoEntity.ID)
	lastBackupWarning := s.getLastWarning(ctx, repoEntity.ID)

	return &Repository{
		ID:                repoEntity.ID,
		Name:              repoEntity.Name,
		URL:               repoEntity.URL,
		Type:              ToLocationUnion(repoType),
		State:             statemachine.ToRepositoryStateUnion(currentState),
		ArchiveCount:      archiveCount,
		LastBackupTime:    lastBackupTime,
		LastBackupError:   lastBackupError,
		LastBackupWarning: lastBackupWarning,
		StorageUsed:       storageUsed,
	}
}

// GetByBackupId retrieves a repository by backup ID
func (s *Service) GetByBackupId(ctx context.Context, bId types.BackupId) (*Repository, error) {
	// TODO: Implement repository lookup by backup ID
	return nil, nil
}

// Create creates a new repository
func (s *Service) Create(ctx context.Context, name, location, password string, noPassword bool) (*Repository, error) {
	s.log.Debugf("Creating repository %s at %s", name, location)

	// Test repository connection first
	result, err := s.TestRepoConnection(ctx, location, password)
	if err != nil {
		return nil, err
	}

	if !result.Success && !result.IsBorgRepo {
		// Create the repository if it does not exist
		status := s.borgClient.Init(ctx, util.ExpandPath(location), password, noPassword)
		if status != nil && status.HasError() {
			return nil, fmt.Errorf("failed to initialize repository: %s", status.GetError())
		}
	} else if !result.Success {
		return nil, fmt.Errorf("could not connect to repository")
	}

	// Create a new repository entity in the database
	repoEntity, err := s.db.Repository.
		Create().
		SetName(name).
		SetURL(location).
		SetPassword(password).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository in database: %w", err)
	}
	return s.entityToRepository(ctx, repoEntity), nil
}

// CreateCloudRepository creates a new ArcoCloud repository
func (s *Service) CreateCloudRepository(ctx context.Context, name, password string, location arcov1.RepositoryLocation) (*Repository, error) {
	// TODO: Implement cloud repository creation
	return nil, nil
}

// Update updates a repository with provided changes
func (s *Service) Update(ctx context.Context, repoId int, updateReq *UpdateRequest) (*Repository, error) {
	// Update the repository in the database
	updateQuery := s.db.Repository.UpdateOneID(repoId)

	// Apply updates based on provided fields
	if updateReq.Name != "" {
		updateQuery = updateQuery.SetName(updateReq.Name)
	}

	// Execute the update
	_, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("repository with ID %d not found", repoId)
		}
		return nil, fmt.Errorf("failed to update repository %d: %w", repoId, err)
	}

	return s.Get(ctx, repoId)
}

// Remove removes a repository from database only (does not delete physical repo)
func (s *Service) Remove(ctx context.Context, id int) error {
	// TODO: Implement repository removal:
	// 1. Cancel any active operations
	// 2. Remove from database
	// 3. Clean up queue
	// 4. Remove backup profiles if they only belong to this repo
	return nil
}

// Delete deletes a repository completely. This cancels all other operations
func (s *Service) Delete(ctx context.Context, id int) error {
	// TODO: Implement delete operation queueing:
	// 1. Validate repository exists
	// 2. Cancel all other operations
	// 3. Delete repository
	return nil
}

// GetBackupProfilesThatHaveOnlyRepo gets backup profiles that only have this repo
func (s *Service) GetBackupProfilesThatHaveOnlyRepo(ctx context.Context, repoId int) ([]*ent.BackupProfile, error) {
	// TODO: Implement proper backup profiles retrieval that only have this repository
	return []*ent.BackupProfile{}, nil
}

// ============================================================================
// QUEUED OPERATIONS
// ============================================================================

// QueueBackup queues a backup operation
func (s *Service) QueueBackup(ctx context.Context, backupId types.BackupId) (string, error) {
	// Generate unique operation ID
	operationID := uuid.New().String()

	// Create backup operation
	backupOp := statemachine.NewOperationBackup(statemachine.Backup{
		BackupID: backupId,
	})

	// Create initial status (queued with position 0)
	initialStatus := NewOperationStatusQueued(Queued{
		Position: 0,
	})

	// Create the queued operation
	queuedOp := &QueuedOperation{
		ID:              operationID,
		RepoID:          backupId.RepositoryId,
		BackupProfileID: &backupId.BackupProfileId,
		Operation:       backupOp,
		Status:          initialStatus,
		CreatedAt:       time.Now(),
		ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour TTL
	}

	// Queue the operation
	resultID, err := s.queueManager.AddOperation(backupId.RepositoryId, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue backup operation: %w", err)
	}

	return resultID, nil
}

// QueueBackups queues multiple backup operations (convenience method)
func (s *Service) QueueBackups(ctx context.Context, backupIds []types.BackupId) ([]string, error) {
	// Initialize result slice
	operationIDs := make([]string, 0, len(backupIds))

	// Iterate through backup IDs and queue each one
	for _, backupId := range backupIds {
		operationID, err := s.QueueBackup(ctx, backupId)
		if err != nil {
			return nil, fmt.Errorf("failed to queue backup for backupId %s: %w", backupId.String(), err)
		}
		operationIDs = append(operationIDs, operationID)
	}

	return operationIDs, nil
}

// QueuePrune queues a prune operation
func (s *Service) QueuePrune(ctx context.Context, backupId types.BackupId) (string, error) {
	// TODO: Implement prune operation queueing
	return "", nil
}

// QueueArchiveRefresh queues an archive refresh operation
func (s *Service) QueueArchiveRefresh(ctx context.Context, repoId int) (string, error) {
	// TODO: Implement archive refresh queueing
	return "", nil
}

// QueueArchiveDelete queues an archive deletion operation
func (s *Service) QueueArchiveDelete(ctx context.Context, archiveId int) (string, error) {
	// TODO: Implement archive delete queueing
	return "", nil
}

// QueueArchiveRename queues an archive rename operation
func (s *Service) QueueArchiveRename(ctx context.Context, archiveId int, prefix, name string) (string, error) {
	// TODO: Implement archive rename queueing
	return "", nil
}

// ============================================================================
// OPERATION MANAGEMENT
// ============================================================================

// GetOperation retrieves an operation by ID
func (s *Service) GetOperation(ctx context.Context, operationId string) (*QueuedOperation, error) {
	// TODO: Implement operation retrieval from QueueManager
	return nil, nil
}

// CancelOperation cancels a queued or running operation
func (s *Service) CancelOperation(ctx context.Context, operationId string) error {
	// TODO: Implement operation cancellation via QueueManager
	return nil
}

// GetQueuedOperations returns all operations for a repository
func (s *Service) GetQueuedOperations(ctx context.Context, repoId int) ([]*QueuedOperation, error) {
	// TODO: Implement queued operations retrieval
	return nil, nil
}

// GetOperationsByStatus returns operations filtered by status for a repository
func (s *Service) GetOperationsByStatus(ctx context.Context, repoId int, status OperationStatusType) ([]*QueuedOperation, error) {
	// TODO: Implement status-filtered operations retrieval
	return nil, nil
}

// ============================================================================
// IMMEDIATE OPERATIONS
// ============================================================================

// AbortBackup immediately aborts a running backup operation
func (s *Service) AbortBackup(ctx context.Context, backupId types.BackupId) error {
	// TODO: Implement backup abortion:
	// 1. Find active backup operation
	// 2. Cancel operation context
	// 3. Update operation status to cancelled
	// 4. Transition repository state
	return nil
}

// AbortBackups aborts multiple running backup operations
func (s *Service) AbortBackups(ctx context.Context, backupIds []types.BackupId) error {
	// TODO: Implement multiple backup abortion
	return nil
}

// Mount mounts a repository
func (s *Service) Mount(ctx context.Context, repoId int) (*platform.MountState, error) {
	// TODO: Implement repository mounting:
	// 1. Validate repository state (must be Idle)
	// 2. Mount repository using borg/platform
	// 3. Transition state to Mounted
	// 4. Return mount state
	return nil, nil
}

// MountArchive mounts a specific archive
func (s *Service) MountArchive(ctx context.Context, archiveId int) (*platform.MountState, error) {
	// TODO: Implement archive mounting
	return nil, nil
}

// Unmount unmounts a repository
func (s *Service) Unmount(ctx context.Context, repoId int) (*platform.MountState, error) {
	// TODO: Implement repository unmounting:
	// 1. Validate repository is mounted
	// 2. Unmount repository
	// 3. Transition state to Idle
	// 4. Return mount state
	return nil, nil
}

// UnmountArchive unmounts a specific archive
func (s *Service) UnmountArchive(ctx context.Context, archiveId int) (*platform.MountState, error) {
	// TODO: Implement archive unmounting
	return nil, nil
}

// UnmountAllForRepos unmounts all mounts for specified repositories
func (s *Service) UnmountAllForRepos(ctx context.Context, repoIds []int) error {
	// TODO: Implement bulk unmounting
	return nil
}

// ExaminePrunes analyzes what would be pruned with given rules
func (s *Service) ExaminePrunes(ctx context.Context, backupProfileId int, pruningRule *ent.PruningRule, saveResults bool) ([]ExaminePruningResult, error) {
	// TODO: Implement prune examination
	return nil, nil
}

// ChangePassword changes the password for a repository
func (s *Service) ChangePassword(ctx context.Context, repoId int, password string) error {
	// TODO: Implement password change
	return nil
}

// RegenerateSSHKey regenerates SSH key for ArcoCloud repositories
func (s *Service) RegenerateSSHKey(ctx context.Context) error {
	// TODO: Implement SSH key regeneration
	return nil
}

// BreakLock breaks a repository lock
func (s *Service) BreakLock(ctx context.Context, repoId int) error {
	// TODO: Implement lock breaking:
	// 1. Validate repository is in error state with locked error
	// 2. Break borg repository lock
	// 3. Transition state from Error to Idle
	return nil
}

// ============================================================================
// ARCHIVE METHODS
// ============================================================================

// GetArchive retrieves an archive by ID
func (s *Service) GetArchive(ctx context.Context, id int) (*ent.Archive, error) {
	// TODO: Implement archive retrieval
	return nil, nil
}

// GetLastArchiveByRepoId gets last archive for repository
func (s *Service) GetLastArchiveByRepoId(ctx context.Context, repoId int) (*ent.Archive, error) {
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query last archive for repository %d: %w", repoId, err)
	}
	return archiveEntity, nil
}

// GetPaginatedArchives retrieves paginated archives for a repository
func (s *Service) GetPaginatedArchives(ctx context.Context, req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error) {
	// TODO: Implement paginated archives retrieval
	return nil, nil
}

// GetPruningDates retrieves pruning dates for specified archives
func (s *Service) GetPruningDates(ctx context.Context, archiveIds []int) (PruningDates, error) {
	// TODO: Implement pruning dates calculation
	return PruningDates{}, nil
}

// ============================================================================
// VALIDATION METHODS
// ============================================================================

// ValidateRepoName validates a repository name
func (s *Service) ValidateRepoName(ctx context.Context, name string) (string, error) {
	// TODO: Implement repository name validation:
	// 1. Check name format and length
	// 2. Check for duplicates
	// 3. Return validation error message or empty string if valid
	return "", nil
}

// ValidateRepoPath validates a repository path
func (s *Service) ValidateRepoPath(ctx context.Context, path string, isLocal bool) (string, error) {
	// TODO: Implement repository path validation
	return "", nil
}

// ValidateArchiveName validates an archive name
func (s *Service) ValidateArchiveName(ctx context.Context, archiveId int, prefix, name string) (string, error) {
	// TODO: Implement archive name validation
	return "", nil
}

// TestRepoConnection tests connection to a repository
func (s *Service) TestRepoConnection(ctx context.Context, path, password string) (TestRepoConnectionResult, error) {
	// TODO: Implement repository connection testing
	return TestRepoConnectionResult{}, nil
}

// IsBorgRepository checks if a path contains a borg repository
func (s *Service) IsBorgRepository(path string) bool {
	// TODO: Implement borg repository detection
	return false
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

// transitionState transitions a repository to a new state
func (s *Service) transitionState(ctx context.Context, repoId int, newState statemachine.RepositoryState) error {
	// TODO: Implement state transition:
	// 1. Get current repository
	// 2. Validate transition via state machine
	// 3. Update repository state in database
	// 4. Emit state change event
	return nil
}

// emitStateChangeEvent emits an event for repository state changes
func (s *Service) emitStateChangeEvent(repoId int, newState statemachine.RepositoryState) {
	// TODO: Implement event emission
}

// createOperationID generates a unique operation ID
func (s *Service) createOperationID() string {
	// TODO: Implement UUID generation for operation IDs
	return ""
}

// startPeriodicCleanup starts background cleanup of expired operations
func (s *Service) startPeriodicCleanup(ctx context.Context) {
	// TODO: Implement periodic cleanup:
	// 1. Run goroutine that periodically calls QueueManager.expireOldOperations()
	// 2. Handle context cancellation for graceful shutdown
}

// getLastError returns the latest backup error message for a repository
func (s *Service) getLastError(ctx context.Context, repoID int) string {
	// Query latest error notification for this repository
	notificationEnt, err := s.db.Notification.Query().
		Where(
			notification.HasRepositoryWith(repository.ID(repoID)),
			notification.TypeIn(
				notification.TypeFailedBackupRun,
				notification.TypeFailedPruningRun,
			),
		).
		Order(ent.Desc("created_at")).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// No error notifications found
			return ""
		}
		s.log.Errorw("Failed to query error notifications",
			"repoID", repoID,
			"error", err.Error())
		return ""
	}

	// Check if there's a newer successful archive since this notification
	// If there is, don't show the old error
	hasNewerArchive, err := s.db.Archive.Query().
		Where(
			archive.HasRepositoryWith(repository.ID(repoID)),
			archive.CreatedAtGT(notificationEnt.CreatedAt),
		).
		Exist(ctx)
	if err != nil {
		s.log.Errorw("Failed to check for newer archives",
			"repoID", repoID,
			"error", err.Error())
		return ""
	}

	if hasNewerArchive {
		// There's a newer archive, so clear the old error
		return ""
	} else {
		// Show the error message
		return notificationEnt.Message
	}
}

// getLastWarning returns the latest backup warning message for a repository
func (s *Service) getLastWarning(ctx context.Context, repoID int) string {
	// Query latest warning notification for this repository
	notificationEnt, err := s.db.Notification.Query().
		Where(
			notification.HasRepositoryWith(repository.ID(repoID)),
			notification.TypeIn(
				notification.TypeWarningBackupRun,
				notification.TypeWarningPruningRun,
			),
		).
		Order(ent.Desc("created_at")).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// No warning notifications found
			return ""
		}
		s.log.Errorw("Failed to query warning notifications",
			"repoID", repoID,
			"error", err.Error())
		return ""
	}

	// Check if there's a newer successful archive since this notification
	// If there is, don't show the old warning
	hasNewerArchive, err := s.db.Archive.Query().
		Where(
			archive.HasRepositoryWith(repository.ID(repoID)),
			archive.CreatedAtGT(notificationEnt.CreatedAt),
		).
		Exist(ctx)
	if err != nil {
		s.log.Errorw("Failed to check for newer archives",
			"repoID", repoID,
			"error", err.Error())
		return ""
	}

	if hasNewerArchive {
		// There's a newer archive, so clear the old warning
		return ""
	} else {
		// Show the warning message
		return notificationEnt.Message
	}
}
