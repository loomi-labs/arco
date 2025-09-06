package repository

import (
	"context"
	"fmt"

	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/platform"
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
	si.queueManager.Init(db, si.borgClient)

	// TODO: Start periodic cleanup goroutine
	// go si.startPeriodicCleanup(ctx)
}

// ============================================================================
// CORE REPOSITORY METHODS
// ============================================================================

// Get retrieves a repository by ID
func (s *Service) Get(ctx context.Context, repoId int) (*Repository, error) {
	// 1. Query database for repository with eager loading
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

	// 2. Calculate current state from queue manager
	currentState := s.queueManager.GetRepositoryState(repoId)

	// 3. Create Repository struct with basic fields
	repo := &Repository{
		ID:    repoEntity.ID,
		Name:  repoEntity.Name,
		URL:   repoEntity.URL,
		State: currentState,
	}

	// 4. Populate cloud information if available
	if repoEntity.Edges.CloudRepository != nil {
		repo.IsCloud = true
		repo.CloudID = repoEntity.Edges.CloudRepository.CloudID
	} else {
		repo.IsCloud = false
		repo.CloudID = ""
	}

	// 5. Populate metadata from loaded archives
	archives := repoEntity.Edges.Archives
	repo.ArchiveCount = len(archives)

	// Get last backup time from most recent archive
	if len(archives) > 0 {
		// Archives are already ordered by creation time descending
		lastBackupTime := archives[0].CreatedAt
		repo.LastBackupTime = &lastBackupTime
	}

	// 6. Calculate storage used from repository statistics
	repo.StorageUsed = int64(repoEntity.StatsUniqueSize)

	// 7. Populate last backup error using existing helper
	err = s.populateLastBackupError(ctx, repo)
	if err != nil {
		s.log.Warnw("Failed to populate last backup error",
			"repoID", repoId,
			"error", err.Error())
		// Don't fail the whole request for this non-critical error
	}

	return repo, nil
}

// GetWithQueue retrieves a repository with queue information
func (s *Service) GetWithQueue(ctx context.Context, repoId int) (*RepositoryWithQueue, error) {
	// TODO: Implement repository with queue retrieval:
	// 1. Get base repository
	// 2. Get queued operations from QueueManager
	// 3. Get active operation
	// 4. Return RepositoryWithQueue struct
	return nil, nil
}

// All retrieves all repositories
func (s *Service) All(ctx context.Context) ([]*Repository, error) {
	// TODO: Implement all repositories retrieval
	return nil, nil
}

// AllWithQueue retrieves all repositories with queue information
func (s *Service) AllWithQueue(ctx context.Context) ([]*RepositoryWithQueue, error) {
	// TODO: Implement all repositories with queue retrieval
	return nil, nil
}

// GetByBackupId retrieves a repository by backup ID
func (s *Service) GetByBackupId(ctx context.Context, bId types.BackupId) (*Repository, error) {
	// TODO: Implement repository lookup by backup ID
	return nil, nil
}

// Create creates a new repository
func (s *Service) Create(ctx context.Context, name, location, password string, noPassword bool) (*Repository, error) {
	// TODO: Implement repository creation:
	// 1. Validate repository name and path
	// 2. Create borg repository
	// 3. Store in database
	// 4. Initialize with Idle state
	// 5. Return Repository struct
	return nil, nil
}

// CreateCloudRepository creates a new ArcoCloud repository
func (s *Service) CreateCloudRepository(ctx context.Context, name, password string, location arcov1.RepositoryLocation) (*Repository, error) {
	// TODO: Implement cloud repository creation
	return nil, nil
}

// Update updates a repository with provided changes
func (s *Service) Update(ctx context.Context, repoId int, updates map[string]interface{}) (*Repository, error) {
	// TODO: Implement repository update
	return nil, nil
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

// ============================================================================
// QUEUED OPERATIONS
// ============================================================================

// QueueDelete queues a repository deletion operation
func (s *Service) QueueDelete(ctx context.Context, id int) (string, error) {
	// TODO: Implement delete operation queueing:
	// 1. Validate repository exists
	// 2. Check for existing delete operation (idempotency)
	// 3. Create QueuedOperation with DeleteVariant
	// 4. Add to queue via QueueManager -> cancel all other operations since the repository is deleted anyway
	// 5. Return operation ID
	return "", nil
}

// QueueBackup queues a backup operation
func (s *Service) QueueBackup(ctx context.Context, backupId types.BackupId) (string, error) {
	// TODO: Implement backup operation queueing
	return "", nil
}

// QueueBackups queues multiple backup operations (convenience method)
func (s *Service) QueueBackups(ctx context.Context, backupIds []types.BackupId) ([]string, error) {
	// TODO: Implement multiple backup queueing:
	// 1. Iterate through backup IDs
	// 2. Queue each backup individually
	// 3. Return slice of operation IDs
	return nil, nil
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
func (s *Service) GetOperationsByStatus(ctx context.Context, repoId int, status OperationStatus) ([]*QueuedOperation, error) {
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
func (s *Service) ExaminePrunes(ctx context.Context, backupProfileId int, pruningRule *ent.PruningRule) ([]ExaminePruningResult, error) {
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

// populateLastBackupError populates the LastBackupError field from the latest error/warning notifications
func (s *Service) populateLastBackupError(ctx context.Context, repo *Repository) error {
	// Query latest error or warning notification for this repository
	// Include both error and warning types to show the most recent issue
	notification, err := s.db.Notification.Query().
		Where(
			notification.HasRepositoryWith(repository.ID(repo.ID)),
			notification.TypeIn(
				notification.TypeFailedBackupRun,
				notification.TypeFailedPruningRun,
				notification.TypeWarningBackupRun,
				notification.TypeWarningPruningRun,
			),
		).
		Order(ent.Desc("created_at")).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// No error/warning notifications found - clear any existing error
			repo.LastBackupError = ""
			return nil
		}
		return err
	}

	// Check if there's a newer successful archive since this notification
	// If there is, don't show the old error/warning
	hasNewerArchive, err := s.db.Archive.Query().
		Where(
			archive.HasRepositoryWith(repository.ID(repo.ID)),
			archive.CreatedAtGT(notification.CreatedAt),
		).
		Exist(ctx)

	if err != nil {
		return err
	}

	if hasNewerArchive {
		// There's a newer archive, so clear the old error
		repo.LastBackupError = ""
	} else {
		// Show the error/warning message
		repo.LastBackupError = notification.Message
	}

	return nil
}
