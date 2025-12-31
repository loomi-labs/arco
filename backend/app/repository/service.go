package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	backupprofileservice "github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/database"
	"github.com/loomi-labs/arco/backend/app/keyring"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/cloudrepository"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/predicate"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/ent/schema"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/negrel/assert"
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
	keyring         *keyring.Service
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
func (si *ServiceInternal) Init(ctx context.Context, db *ent.Client, eventEmitter types.EventEmitter, borgClient borg.Borg, cloudRepoClient *CloudRepositoryClient, keyringService *keyring.Service) {
	si.db = db
	si.eventEmitter = eventEmitter
	si.borgClient = borgClient
	si.cloudRepoClient = cloudRepoClient
	si.keyring = keyringService

	// Initialize queue manager with database and borg clients
	si.queueManager.Init(db, si.borgClient, si.eventEmitter, keyringService)

	// Initialize mount states
	si.initMountStates(ctx)
}

// initMountStates initializes the mount states of all repositories and archives at startup
// This method is called during app startup and restores mount states if things are mounted
func (si *ServiceInternal) initMountStates(ctx context.Context) {
	// Get all mounted repositories and archives in a single system call
	mountedRepos, mountedArchives, err := platform.GetArcoMounts()
	if err != nil {
		si.log.Errorw("Error getting Arco mount states", "error", err)
		return
	}

	// Process mounted repositories
	for repoID, mountState := range mountedRepos {
		mountInfo := statemachine.MountInfo{
			MountType: statemachine.MountTypeRepository,
			MountPath: mountState.MountPath,
		}
		mountedState := statemachine.CreateMountedState([]statemachine.MountInfo{mountInfo})
		si.queueManager.setRepositoryState(repoID, mountedState)
		si.log.Infow("Restored repository mount state", "repoID", repoID, "mountPath", mountState.MountPath)
	}

	// Process mounted archives - group by repository for efficiency
	archivesByRepo := make(map[int][]statemachine.MountInfo)
	for archiveID, mountState := range mountedArchives {
		// Get repository ID for this archive
		archiveEntity, err := si.db.Archive.Query().
			Where(archive.ID(archiveID)).
			WithRepository().
			Only(ctx)
		if err != nil {
			si.log.Errorw("Error getting repository for mounted archive", "archiveID", archiveID, "error", err)
			continue
		}

		repoID := archiveEntity.Edges.Repository.ID
		id := archiveID
		mountInfo := statemachine.MountInfo{
			MountType: statemachine.MountTypeArchive,
			MountPath: mountState.MountPath,
			ArchiveID: &id,
		}

		archivesByRepo[repoID] = append(archivesByRepo[repoID], mountInfo)
	}

	// Update repository states for repositories with mounted archives
	for repoID, archiveMounts := range archivesByRepo {
		// Check if repository itself is not already mounted (to avoid overriding repo mounts)
		if _, repoIsMounted := mountedRepos[repoID]; !repoIsMounted {
			mountedState := statemachine.CreateMountedState(archiveMounts)
			si.queueManager.setRepositoryState(repoID, mountedState)
			si.log.Infow("Restored archive mount states", "repoID", repoID, "mountedArchiveCount", len(archiveMounts))
		}
	}

	// Log summary
	si.log.Infow("Mount state initialization completed",
		"mountedRepos", len(mountedRepos),
		"mountedArchives", len(mountedArchives),
		"affectedRepositories", len(archivesByRepo)+len(mountedRepos))
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

	// 2. Get queued operations from QueueManager and convert to serializable format
	queuedOps, err := s.queueManager.GetQueuedOperations(repoId, nil)
	if err != nil {
		return nil, err
	}

	var serializableQueuedOps []*SerializableQueuedOperation
	for _, op := range queuedOps {
		if op != nil {
			serialized := toSerializableQueuedOperation(op)
			serializableQueuedOps = append(serializableQueuedOps, &serialized)
		}
	}

	// 3. Get active operation from repository queue and convert to serializable format
	queue := s.queueManager.GetQueue(repoId)
	activeOp := queue.GetActive() // Can be nil if no operation is active

	var serializableActiveOp *SerializableQueuedOperation
	if activeOp != nil {
		serialized := toSerializableQueuedOperation(activeOp)
		serializableActiveOp = &serialized
	}

	// 4. Create and return RepositoryWithQueue struct
	return &RepositoryWithQueue{
		Repository:       *baseRepo,
		QueuedOperations: serializableQueuedOps,
		ActiveOperation:  serializableActiveOp,
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

	// Calculate storage used from repository statistics
	sizeOnDisk := int64(repoEntity.StatsUniqueCsize)  // Compressed & deduplicated storage (actual disk usage)
	totalSize := int64(repoEntity.StatsTotalSize)     // Total uncompressed size across all archives
	uniqueSize := int64(repoEntity.StatsUniqueSize)   // Deduplicated uncompressed size
	uniqueCsize := int64(repoEntity.StatsUniqueCsize) // Deduplicated compressed size

	// Calculate deduplication ratio (totalSize / uniqueSize)
	dedupRatio := 0.0
	if uniqueSize > 0 {
		dedupRatio = float64(totalSize) / float64(uniqueSize)
	}

	// Calculate compression ratio (uniqueSize / uniqueCsize)
	compressionRatio := 0.0
	if uniqueCsize > 0 {
		compressionRatio = float64(uniqueSize) / float64(uniqueCsize)
	}

	// Calculate overall space savings percentage ((totalSize - uniqueCsize) / totalSize * 100)
	spaceSavingsPercent := 0.0
	if totalSize > 0 {
		spaceSavingsPercent = ((float64(totalSize) - float64(uniqueCsize)) / float64(totalSize)) * 100
	}

	return &Repository{
		ID:                  repoEntity.ID,
		Name:                repoEntity.Name,
		URL:                 repoEntity.URL,
		Type:                ToLocationUnion(repoType),
		State:               statemachine.ToRepositoryStateUnion(currentState),
		ArchiveCount:        archiveCount,
		LastBackup:          s.getLastBackup(ctx, repoEntity.ID),
		LastAttempt:         s.getLastAttempt(ctx, repoEntity.ID),
		LastQuickCheckAt:    repoEntity.LastQuickCheckAt,
		QuickCheckError:     repoEntity.QuickCheckError,
		LastFullCheckAt:     repoEntity.LastFullCheckAt,
		FullCheckError:      repoEntity.FullCheckError,
		SizeOnDisk:          sizeOnDisk,
		TotalSize:           totalSize,
		DeduplicationRatio:  dedupRatio,
		CompressionRatio:    compressionRatio,
		SpaceSavingsPercent: spaceSavingsPercent,
		HasPassword:         s.keyring.HasRepositoryPassword(repoEntity.ID),
	}
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
		SetHasPassword(password != "").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository in database: %w", err)
	}

	// Store password in keyring
	if password != "" {
		if err := s.keyring.SetRepositoryPassword(repoEntity.ID, password); err != nil {
			// Rollback: delete the created repository
			if delErr := s.db.Repository.DeleteOneID(repoEntity.ID).Exec(ctx); delErr != nil {
				s.log.Warnf("Failed to rollback repository creation: %v", delErr)
			}
			return nil, fmt.Errorf("failed to store repository password in keyring: %w", err)
		}
	}

	s.eventEmitter.EmitEvent(ctx, types.EventRepositoryCreatedString())
	return s.entityToRepository(ctx, repoEntity), nil
}

// CreateCloudRepository creates a new ArcoCloud repository
func (s *Service) CreateCloudRepository(ctx context.Context, name, password string, location arcov1.RepositoryLocation) (*Repository, error) {
	// List existing cloud repositories to check if one already exists
	cloudRepos, err := s.cloudRepoClient.ListCloudRepositories(ctx)
	if err != nil {
		return nil, err
	}

	// Check if repository already exists
	var repo *arcov1.Repository
	for _, cloudRepo := range cloudRepos {
		if cloudRepo.Name == name {
			repo = cloudRepo
			s.log.Warnf("Repository '%s' already exists in ArcoCloud, proceeding with existing repository", name)
			break
		}
	}

	// If repository doesn't exist, create it
	if repo == nil {
		repo, err = s.cloudRepoClient.AddCloudRepository(ctx, name, location)
		if err != nil {
			return nil, err
		}

		// We need to wait a bit otherwise it can create errors when initializing the repository
		time.Sleep(500 * time.Millisecond)

		status := s.borgClient.Init(ctx, repo.RepoUrl, password, false)
		if status != nil && status.HasError() {
			s.log.Errorf("Failed to initialize repository during initialization: %s", status.GetError())
			return nil, fmt.Errorf("failed to initialize repository: %s", status.GetError())
		}
	}

	entRepo, err := database.WithTxData(ctx, s.db, func(tx *ent.Tx) (*ent.Repository, error) {
		// Create new local repository with cloud association
		entRepo, txErr := tx.Repository.
			Create().
			SetName(name).
			SetURL(repo.RepoUrl).
			SetHasPassword(password != "").
			Save(ctx)
		if txErr != nil {
			return nil, txErr
		}
		_, txErr = tx.CloudRepository.
			Create().
			SetCloudID(repo.Id).
			SetLocation(s.getLocationEnum(location)).
			SetRepository(entRepo).
			Save(ctx)
		if txErr != nil {
			return nil, txErr
		}
		return entRepo, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create repository in database: %w", err)
	}

	// Store password in keyring
	if password != "" {
		if err := s.keyring.SetRepositoryPassword(entRepo.ID, password); err != nil {
			return nil, fmt.Errorf("failed to store repository password in keyring: %w", err)
		}
	}

	s.eventEmitter.EmitEvent(ctx, types.EventRepositoryCreatedString())
	return s.entityToRepository(ctx, entRepo), nil
}

// SyncCloudRepositories syncs all cloud repositories from ArcoCloud to local database
func (si *ServiceInternal) SyncCloudRepositories(ctx context.Context) ([]*ent.Repository, error) {
	cloudRepos, err := si.cloudRepoClient.ListCloudRepositories(ctx)
	if err != nil {
		return nil, err
	}

	si.log.Debugf("Syncing %d cloud repositories", len(cloudRepos))

	var syncedRepos []*ent.Repository
	for _, cloudRepo := range cloudRepos {
		localRepo, err := si.syncSingleCloudRepository(ctx, cloudRepo)
		if err != nil {
			si.log.Errorf("Failed to sync cloud repository %s (%s): %v", cloudRepo.Name, cloudRepo.Id, err)
			return nil, err
		}
		if localRepo != nil {
			syncedRepos = append(syncedRepos, localRepo)
		}
	}
	return syncedRepos, nil
}

// syncSingleCloudRepository creates or updates a local repository entity with cloud metadata
func (s *Service) syncSingleCloudRepository(ctx context.Context, cloudRepo *arcov1.Repository) (*ent.Repository, error) {
	var result *ent.Repository
	err := database.WithTx(ctx, s.db, func(tx *ent.Tx) error {
		// Check if local repository already exists by ArcoCloud ID
		if cloudRepo.Id != "" {
			if localRepo, err := tx.Repository.Query().
				Where(repository.HasCloudRepositoryWith(
					cloudrepository.CloudIDEQ(cloudRepo.Id),
				)).
				First(ctx); err == nil {
				// Update existing repository
				updatedRepo, txErr := tx.Repository.UpdateOne(localRepo).
					SetName(cloudRepo.Name).
					SetURL(cloudRepo.RepoUrl).
					Save(ctx)
				if txErr != nil {
					return txErr
				}
				result = updatedRepo
				return nil
			}
		}

		// Check if repository exists by location (repo URL)
		if localRepo, err := tx.Repository.Query().
			Where(repository.URLEQ(cloudRepo.RepoUrl)).
			First(ctx); err == nil {
			// Create cloud repository association
			_, txErr := tx.CloudRepository.Create().
				SetCloudID(cloudRepo.Id).
				SetStorageUsedBytes(cloudRepo.StorageUsedBytes).
				SetLocation(s.getLocationEnum(cloudRepo.Location)).
				SetRepository(localRepo).
				Save(ctx)
			if txErr != nil {
				return txErr
			}

			// Update repository name if needed
			updatedRepo, txErr := tx.Repository.UpdateOne(localRepo).
				SetName(cloudRepo.Name).
				Save(ctx)
			if txErr != nil {
				return txErr
			}
			result = updatedRepo
			return nil
		}

		// Create new local repository with cloud association
		localRepo, txErr := tx.Repository.Create().
			SetName(cloudRepo.Name).
			SetURL(cloudRepo.RepoUrl).
			Save(ctx)
		if txErr != nil {
			return txErr
		}

		// Create cloud repository association
		_, txErr = tx.CloudRepository.Create().
			SetCloudID(cloudRepo.Id).
			SetStorageUsedBytes(cloudRepo.StorageUsedBytes).
			SetLocation(s.getLocationEnum(cloudRepo.Location)).
			SetRepository(localRepo).
			Save(ctx)
		if txErr != nil {
			return txErr
		}

		result = localRepo
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to sync cloud repository: %w", err)
	}

	return result, nil
}

// Update updates a repository with provided changes
func (s *Service) Update(ctx context.Context, repoId int, updateReq *UpdateRequest) (*Repository, error) {
	// Update the repository in the database
	updateQuery := s.db.Repository.UpdateOneID(repoId)

	// Apply updates based on provided fields
	if updateReq.Name != "" {
		updateQuery = updateQuery.SetName(updateReq.Name)
	}

	// Handle URL update with validation
	if updateReq.URL != "" {
		// Check repository is in Idle state before allowing path changes
		currentState := s.queueManager.GetRepositoryState(repoId)
		if statemachine.GetRepositoryStateType(currentState) != statemachine.RepositoryStateTypeIdle {
			return nil, errors.New("repository must be idle to change path")
		}

		result, err := s.ValidatePathChange(ctx, repoId, updateReq.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to validate path change: %w", err)
		}
		if !result.IsValid {
			return nil, errors.New(result.ErrorMessage)
		}

		// Update the URL
		updateQuery = updateQuery.SetURL(updateReq.URL)
	}

	// Execute the update
	_, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("repository with ID %d not found", repoId)
		}
		return nil, fmt.Errorf("failed to update repository %d: %w", repoId, err)
	}

	s.eventEmitter.EmitEvent(ctx, types.EventRepositoryUpdatedString())
	return s.Get(ctx, repoId)
}

// Remove removes a repository from database only (does not delete physical repo)
func (s *Service) Remove(ctx context.Context, id int) error {
	s.log.Debugw("Removing repository", "id", id)

	// 1. Cancel any active/queued operations for this repository
	queue := s.queueManager.GetQueue(id)
	operations := queue.GetOperations(nil)
	for _, op := range operations {
		err := s.queueManager.CancelOperation(id, op.ID)
		if err != nil {
			s.log.Warnw("Failed to cancel operation during repository removal", "repoID", id, "operationID", op.ID, "error", err)
		}
	}

	// 2. Get backup profiles that only belong to this repository
	backupProfiles, err := s.GetBackupProfilesThatHaveOnlyRepo(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get backup profiles for repository %d: %w", id, err)
	}

	// 3. Remove repository and backup profiles in a transaction
	err = database.WithTx(ctx, s.db, func(tx *ent.Tx) error {
		// Delete backup profiles that only have this repository
		if len(backupProfiles) > 0 {
			bpIds := make([]int, 0, len(backupProfiles))
			for _, bp := range backupProfiles {
				bpIds = append(bpIds, bp.ID)
			}

			_, err = tx.BackupProfile.Delete().
				Where(backupprofile.IDIn(bpIds...)).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to delete backup profiles: %w", err)
			}
		}

		// Delete the repository
		err = tx.Repository.
			DeleteOneID(id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete repository: %w", err)
		}

		return nil
	})
	if err == nil {
		s.eventEmitter.EmitEvent(ctx, types.EventRepositoryDeletedString())
	}
	return err
}

// Delete deletes a repository completely. This cancels all other operations
func (s *Service) Delete(ctx context.Context, id int) error {
	s.log.Debugw("Deleting repository", "id", id)

	// 1. Validate repository exists and get cloud repository info if needed
	repoEntity, err := s.db.Repository.Query().
		Where(repository.ID(id)).
		WithCloudRepository().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("repository %d not found: %w", id, err)
	}

	// 2. Cancel all queued operations for this repository
	queue := s.queueManager.GetQueue(id)
	operations := queue.GetOperations(nil)
	for _, op := range operations {
		err := s.queueManager.CancelOperation(id, op.ID)
		if err != nil {
			s.log.Warnw("Failed to cancel operation during repository deletion", "repoID", id, "operationID", op.ID, "error", err)
		}
	}

	// 3. Check if this is a cloud repository and handle accordingly
	if repoEntity.Edges.CloudRepository != nil {
		// For cloud repositories, delete directly via cloud service
		if err := s.cloudRepoClient.DeleteCloudRepository(ctx, repoEntity.Edges.CloudRepository.CloudID); err != nil {
			return err
		}
		s.eventEmitter.EmitEvent(ctx, types.EventRepositoryDeletedString())
		return nil
	}

	// 4. For local repositories, queue a delete operation
	operationID := uuid.New().String()

	// Create delete operation
	deleteOp := statemachine.NewOperationDelete(statemachine.Delete{
		RepositoryID: id,
	})

	// Create initial status (queued with position 0)
	initialStatus := NewOperationStatusQueued(Queued{
		Position: 0,
	})

	// Create the queued operation
	queuedOp := &QueuedOperation{
		ID:         operationID,
		RepoID:     id,
		Operation:  deleteOp,
		Status:     initialStatus,
		CreatedAt:  time.Now(),
		ValidUntil: nil, // no expiration
	}

	// Queue the delete operation
	_, err = s.queueManager.AddOperation(id, queuedOp)
	if err != nil {
		return fmt.Errorf("failed to queue delete operation: %w", err)
	}

	// Remove repository from database after successfully queuing the delete operation
	return s.Remove(ctx, id)
}

// RefreshArchives refreshes all archives of a repository
func (s *Service) RefreshArchives(ctx context.Context, repoId int) (string, error) {
	// Create archive refresh operation
	archiveRefreshOp := statemachine.NewOperationArchiveRefresh(statemachine.ArchiveRefresh{
		RepositoryID: repoId,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		archiveRefreshOp,
		repoId,
		nil,  // No backup profile for archive refresh (repository-wide operation)
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(repoId, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue archive refresh operation: %w", err)
	}

	return operationID, nil
}

// QueueCheck queues a repository integrity check operation
func (s *Service) QueueCheck(ctx context.Context, repoId int, quickVerification bool) (string, error) {
	// Create check operation
	checkOp := statemachine.NewOperationCheck(statemachine.Check{
		RepositoryID:      repoId,
		QuickVerification: quickVerification,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		checkOp,
		repoId,
		nil,   // No backup profile for check (repository-wide operation)
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(repoId, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue check operation: %w", err)
	}

	s.log.Infof("Queued %s check for repo %d", map[bool]string{true: "quick", false: "full"}[quickVerification], repoId)
	return operationID, nil
}

// isCloudRepository checks if a repository is an ArcoCloud repository
func (s *Service) isCloudRepository(ctx context.Context, repoID int) bool {
	exists, err := s.db.Repository.Query().
		Where(repository.And(
			repository.IDEQ(repoID),
			repository.HasCloudRepository(),
		)).
		Exist(ctx)
	if err != nil {
		s.log.Errorw("IsCloudRepository query error", "error", err)
	}
	return exists
}

// isRepositoryMountedOrMounting checks if a repository is in mounted or mounting state
func (s *Service) isRepositoryMountedOrMounting(repositoryId int) bool {
	repoState := s.queueManager.GetRepositoryState(repositoryId)
	repoStateUnion := statemachine.ToRepositoryStateUnion(repoState)
	return repoStateUnion.Type == statemachine.RepositoryStateTypeMounted ||
		repoStateUnion.Type == statemachine.RepositoryStateTypeMounting
}

// getLocationEnum converts arcov1.RepositoryLocation to cloudrepository.Location
func (s *Service) getLocationEnum(location arcov1.RepositoryLocation) cloudrepository.Location {
	switch location {
	case arcov1.RepositoryLocation_REPOSITORY_LOCATION_EU:
		return cloudrepository.LocationEU
	case arcov1.RepositoryLocation_REPOSITORY_LOCATION_US:
		return cloudrepository.LocationUS
	case arcov1.RepositoryLocation_REPOSITORY_LOCATION_UNSPECIFIED:
	}
	s.log.Errorw("Unknown repository location, defaulting to EU", "location", location)
	return cloudrepository.LocationEU
}

// GetBackupProfilesThatHaveOnlyRepo gets backup profiles that only have this repo
func (s *Service) GetBackupProfilesThatHaveOnlyRepo(ctx context.Context, repoId int) ([]*ent.BackupProfile, error) {
	backupProfiles, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.And(
			backupprofile.HasRepositoriesWith(repository.ID(repoId)),
		)).
		WithRepositories().
		All(ctx)
	if err != nil {
		return nil, err
	}
	var result []*ent.BackupProfile
	for _, bp := range backupProfiles {
		if len(bp.Edges.Repositories) == 1 {
			result = append(result, bp)
		}
	}
	return result, nil
}

// ============================================================================
// QUEUED OPERATIONS
// ============================================================================

// QueueBackup queues a backup operation
func (s *Service) QueueBackup(ctx context.Context, backupId types.BackupId) (string, error) {
	// Check if repository is mounted or mounting - cannot start backup in this state
	if s.isRepositoryMountedOrMounting(backupId.RepositoryId) {
		return "", fmt.Errorf("cannot start backup while repository is mounted or mounting - please unmount the repository first")
	}

	// Create backup operation
	backupOp := statemachine.NewOperationBackup(statemachine.Backup{
		BackupID: backupId,
		Progress: &borgtypes.BackupProgress{
			TotalFiles:     0,
			ProcessedFiles: 0,
		},
	})

	// Create queued operation
	queue := s.queueManager.GetQueue(backupId.RepositoryId)
	queuedOp := queue.CreateQueuedOperation(
		backupOp,
		backupId.RepositoryId,
		&backupId.BackupProfileId,
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(backupId.RepositoryId, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue backup operation: %w", err)
	}

	return operationID, nil
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
	// Check if repository is mounted or mounting - cannot start prune in this state
	if s.isRepositoryMountedOrMounting(backupId.RepositoryId) {
		return "", fmt.Errorf("cannot start prune while repository is mounted or mounting - please unmount the repository first")
	}

	// Create prune operation
	pruneOp := statemachine.NewOperationPrune(statemachine.Prune{
		BackupID: backupId,
	})

	// Create queued operation using factory method
	queue := s.queueManager.GetQueue(backupId.RepositoryId)
	queuedOp := queue.CreateQueuedOperation(
		pruneOp,
		backupId.RepositoryId,
		&backupId.BackupProfileId,
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(backupId.RepositoryId, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue prune operation: %w", err)
	}

	return operationID, nil
}

// QueueArchiveDelete queues an archive deletion operation
func (s *Service) QueueArchiveDelete(ctx context.Context, archiveId int) (string, error) {
	// Get archive to determine repository ID and backup profile ID
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		WithBackupProfile().
		Only(ctx)
	if err != nil {
		return "", fmt.Errorf("archive %d not found: %w", archiveId, err)
	}

	// Check if repository is mounted or mounting - cannot delete archives in this state
	if s.isRepositoryMountedOrMounting(archiveEntity.Edges.Repository.ID) {
		return "", fmt.Errorf("cannot delete archive while repository is mounted or mounting - please unmount the repository first")
	}

	// Create archive delete operation
	archiveDeleteOp := statemachine.NewOperationArchiveDelete(statemachine.ArchiveDelete{
		ArchiveID: archiveId,
	})

	// Get backup profile ID if available
	var backupProfileID *int
	if archiveEntity.Edges.BackupProfile != nil {
		backupProfileID = &archiveEntity.Edges.BackupProfile.ID
	}

	// Create queued operation using factory method
	queue := s.queueManager.GetQueue(archiveEntity.Edges.Repository.ID)
	queuedOp := queue.CreateQueuedOperation(
		archiveDeleteOp,
		archiveEntity.Edges.Repository.ID,
		backupProfileID,
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(archiveEntity.Edges.Repository.ID, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue archive delete operation: %w", err)
	}

	return operationID, nil
}

// QueueArchiveRename queues an archive rename operation
func (s *Service) QueueArchiveRename(ctx context.Context, archiveId int, name string) (string, error) {
	// Get archive to determine repository ID and backup profile ID
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		WithBackupProfile().
		Only(ctx)
	if err != nil {
		return "", fmt.Errorf("archive %d not found: %w", archiveId, err)
	}

	// Check if repository is mounted or mounting - cannot rename archives in this state
	if s.isRepositoryMountedOrMounting(archiveEntity.Edges.Repository.ID) {
		return "", fmt.Errorf("cannot rename archive while repository is mounted or mounting - please unmount the repository first")
	}

	// Get prefix from backup profile if available
	var prefix string
	if archiveEntity.Edges.BackupProfile != nil {
		prefix = archiveEntity.Edges.BackupProfile.Prefix
	}

	// Create archive rename operation
	archiveRenameOp := statemachine.NewOperationArchiveRename(statemachine.ArchiveRename{
		ArchiveID: archiveId,
		Prefix:    prefix,
		Name:      name,
	})

	// Get backup profile ID if available
	var backupProfileID *int
	if archiveEntity.Edges.BackupProfile != nil {
		backupProfileID = &archiveEntity.Edges.BackupProfile.ID
	}

	// Create queued operation using factory method
	queue := s.queueManager.GetQueue(archiveEntity.Edges.Repository.ID)
	queuedOp := queue.CreateQueuedOperation(
		archiveRenameOp,
		archiveEntity.Edges.Repository.ID,
		backupProfileID,
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(archiveEntity.Edges.Repository.ID, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue archive rename operation: %w", err)
	}

	return operationID, nil
}

// QueueArchiveComment queues an archive comment update operation
func (s *Service) QueueArchiveComment(ctx context.Context, archiveId int, comment string) (string, error) {
	// Get archive to determine repository ID
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return "", fmt.Errorf("archive %d not found: %w", archiveId, err)
	}

	// Check if repository is mounted or mounting - cannot update comments in this state
	if s.isRepositoryMountedOrMounting(archiveEntity.Edges.Repository.ID) {
		return "", fmt.Errorf("cannot update archive comment while repository is mounted or mounting - please unmount the repository first")
	}

	// Create archive comment operation
	archiveCommentOp := statemachine.NewOperationArchiveComment(statemachine.ArchiveComment{
		ArchiveID: archiveId,
		Comment:   comment,
	})

	// Create queued operation using factory method
	queue := s.queueManager.GetQueue(archiveEntity.Edges.Repository.ID)
	queuedOp := queue.CreateQueuedOperation(
		archiveCommentOp,
		archiveEntity.Edges.Repository.ID,
		nil,   // no backup profile ID
		nil,   // no expiration
		false, // will be queued
	)

	// Add to queue
	operationID, err := s.queueManager.AddOperation(archiveEntity.Edges.Repository.ID, queuedOp)
	if err != nil {
		return "", fmt.Errorf("failed to queue archive comment operation: %w", err)
	}

	return operationID, nil
}

// ============================================================================
// OPERATION MANAGEMENT
// ============================================================================

func (s *Service) GetActiveOperation(ctx context.Context, repoId int, operationType *statemachine.OperationType) (*SerializableQueuedOperation, error) {
	operation := s.queueManager.GetActiveOperation(repoId, operationType)
	if operation == nil {
		return nil, nil
	}

	// Convert to serializable format
	serialized := toSerializableQueuedOperation(operation)
	return &serialized, nil
}

// CancelOperation cancels a queued or running operation
func (s *Service) CancelOperation(ctx context.Context, repositoryId int, operationId string) error {
	return s.queueManager.CancelOperation(repositoryId, operationId)
}

// GetQueuedOperations returns all operations for a repository, optionally filtered by operation type
func (s *Service) GetQueuedOperations(ctx context.Context, repoId int, operationType *statemachine.OperationType) ([]*SerializableQueuedOperation, error) {
	operations, err := s.queueManager.GetQueuedOperations(repoId, operationType)
	if err != nil {
		return nil, err
	}

	// Convert to serializable format
	var serializableOps []*SerializableQueuedOperation
	for _, op := range operations {
		if op != nil {
			serialized := toSerializableQueuedOperation(op)
			serializableOps = append(serializableOps, &serialized)
		}
	}

	return serializableOps, nil
}

// ============================================================================
// IMMEDIATE OPERATIONS
// ============================================================================

// AbortBackup immediately aborts a running backup operation
func (s *Service) AbortBackup(ctx context.Context, backupId types.BackupId) error {
	// Find the operation ID for this backup
	operationID, err := s.findOperationIDByBackupID(backupId)
	if err != nil {
		return fmt.Errorf("cannot abort backup: %w", err)
	}

	// Cancel the operation using the queue manager
	err = s.queueManager.CancelOperation(backupId.RepositoryId, operationID)
	if err != nil {
		return fmt.Errorf("failed to cancel backup operation: %w", err)
	}

	s.log.Infow("Backup operation aborted",
		"backupId", backupId.String(),
		"operationId", operationID,
		"repositoryId", backupId.RepositoryId)

	return nil
}

// AbortBackups aborts multiple running backup operations
func (s *Service) AbortBackups(ctx context.Context, backupIds []types.BackupId) error {
	var errs []string
	var abortedCount int

	for _, backupId := range backupIds {
		err := s.AbortBackup(ctx, backupId)
		if err != nil {
			errs = append(errs, fmt.Sprintf("backup %s: %v", backupId.String(), err))
		} else {
			abortedCount++
		}
	}

	// Log summary
	s.log.Infow("Bulk backup abortion completed",
		"totalRequested", len(backupIds),
		"aborted", abortedCount,
		"failed", len(errs))

	// Return combined error if any operations failed
	if len(errs) > 0 {
		return fmt.Errorf("failed to abort %d out of %d backups: %s",
			len(errs), len(backupIds), strings.Join(errs, "; "))
	}

	return nil
}

// Mount mounts a repository
func (s *Service) Mount(ctx context.Context, repoId int) (*MountResult, error) {
	// Check if macFUSE is installed on macOS
	if !platform.IsMacFUSEInstalled() {
		return &MountResult{MacFUSENotInstalled: true}, nil
	}

	// Check if repository is already mounted
	repoState := s.queueManager.GetRepositoryState(repoId)
	repoStateUnion := statemachine.ToRepositoryStateUnion(repoState)

	if repoStateUnion.Type == statemachine.RepositoryStateTypeMounted {
		// Repository is already mounted, find the repository mount and open file manager
		if repoStateUnion.Mounted != nil {
			for _, mount := range repoStateUnion.Mounted.Mounts {
				if mount.MountType == statemachine.MountTypeRepository {
					// Found repository mount, open file manager
					go openFileManager(mount.MountPath, s.log)
					return &MountResult{}, nil
				}
			}
		}
		return nil, fmt.Errorf("repository is mounted but no repository mount found")
	}

	// Get repository from database to calculate mount path
	repoEntity, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Calculate mount path
	mountPath, err := getRepoMountPath(repoEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount path: %w", err)
	}

	// Create mount operation with mount path
	mountOp := statemachine.NewOperationMount(statemachine.Mount{
		RepositoryID: repoId,
		MountPath:    mountPath,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		mountOp,
		repoId,
		nil,  // No backup profile for mount operations
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue
	opId, err := s.queueManager.AddOperation(repoId, queuedOp)
	if err != nil {
		return nil, err
	}
	return &MountResult{OperationId: opId}, nil
}

// MountArchive mounts a specific archive
func (s *Service) MountArchive(ctx context.Context, archiveId int) (*MountResult, error) {
	// Check if macFUSE is installed on macOS
	if !platform.IsMacFUSEInstalled() {
		return &MountResult{MacFUSENotInstalled: true}, nil
	}

	// Get archive to determine repository ID and calculate mount path
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive %d not found: %w", archiveId, err)
	}

	// Get repository ID
	repoId := archiveEntity.Edges.Repository.ID

	// Check if repository is already mounted and this archive is mounted
	repoState := s.queueManager.GetRepositoryState(repoId)
	repoStateUnion := statemachine.ToRepositoryStateUnion(repoState)

	if repoStateUnion.Type == statemachine.RepositoryStateTypeMounted {
		// Repository is mounted, check if this specific archive is mounted
		if repoStateUnion.Mounted != nil {
			for _, mount := range repoStateUnion.Mounted.Mounts {
				if mount.MountType == statemachine.MountTypeArchive && mount.ArchiveID != nil && *mount.ArchiveID == archiveId {
					// Found archive mount, open file manager
					go openFileManager(mount.MountPath, s.log)
					return &MountResult{}, nil
				}
			}

			// If archive not separately mounted, check for repository mount
			for _, mount := range repoStateUnion.Mounted.Mounts {
				if mount.MountType == statemachine.MountTypeRepository {
					// Archive accessible via repository mount, open archive path within repository
					archivePath := filepath.Join(mount.MountPath, archiveEntity.Name)
					go openFileManager(archivePath, s.log)
					return &MountResult{}, nil
				}
			}
		}
		return nil, fmt.Errorf("repository is mounted but no mounts found")
	}

	// Calculate mount path for the archive
	mountPath, err := getArchiveMountPath(archiveEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get archive mount path: %w", err)
	}

	// Create mount archive operation with mount path
	mountOp := statemachine.NewOperationMountArchive(statemachine.MountArchive{
		ArchiveID: archiveId,
		MountPath: mountPath,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		mountOp,
		repoId,
		nil,  // No backup profile for mount operations
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue (will start immediately or fail)
	opId, err := s.queueManager.AddOperation(repoId, queuedOp)
	if err != nil {
		return nil, err
	}
	return &MountResult{OperationId: opId}, nil
}

// Unmount unmounts a repository
func (s *Service) Unmount(ctx context.Context, repoId int) (string, error) {
	// Get repository to calculate mount path
	repoEntity, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	// Calculate mount path
	mountPath, err := getRepoMountPath(repoEntity)
	if err != nil {
		return "", fmt.Errorf("failed to get mount path: %w", err)
	}

	// Create unmount operation with mount path
	unmountOp := statemachine.NewOperationUnmount(statemachine.Unmount{
		RepositoryID: repoId,
		MountPath:    mountPath,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		unmountOp,
		repoId,
		nil,  // No backup profile for unmount operations
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue (will start immediately or fail)
	return s.queueManager.AddOperation(repoId, queuedOp)
}

// UnmountArchive unmounts a specific archive
func (s *Service) UnmountArchive(ctx context.Context, archiveId int) (string, error) {
	// Get archive to determine repository ID and calculate mount path
	archiveEntity, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		Only(ctx)
	if err != nil {
		return "", fmt.Errorf("archive %d not found: %w", archiveId, err)
	}

	// Get repository ID
	repoId := archiveEntity.Edges.Repository.ID

	// Calculate mount path for the archive
	mountPath, err := getArchiveMountPath(archiveEntity)
	if err != nil {
		return "", fmt.Errorf("failed to get archive mount path: %w", err)
	}

	// Create unmount archive operation with mount path
	unmountOp := statemachine.NewOperationUnmountArchive(statemachine.UnmountArchive{
		ArchiveID: archiveId,
		MountPath: mountPath,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(repoId)
	queuedOp := queue.CreateQueuedOperation(
		unmountOp,
		repoId,
		nil,  // No backup profile for unmount operations
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue (will start immediately or fail)
	return s.queueManager.AddOperation(repoId, queuedOp)
}

// UnmountAllForRepos unmounts all mounts for specified repositories
func (s *Service) UnmountAllForRepos(ctx context.Context, repoIds []int) []error {
	var errorSlice []error
	for _, repoId := range repoIds {
		_, err := s.Unmount(ctx, repoId)
		if err != nil {
			errorSlice = append(errorSlice, fmt.Errorf("failed to unmount repository %d: %w", repoId, err))
		}
	}
	return errorSlice
}

// clearWillBePrunedFlags clears all WillBePruned flags for archives in the backup profile's repositories
func (s *Service) clearWillBePrunedFlags(ctx context.Context, backupProfile *ent.BackupProfile) error {
	// Extract repository IDs from the backup profile
	repositoryIDs := make([]int, 0, len(backupProfile.Edges.Repositories))
	for _, repo := range backupProfile.Edges.Repositories {
		repositoryIDs = append(repositoryIDs, repo.ID)
	}

	if len(repositoryIDs) == 0 {
		s.log.Errorw("No repositories found for backup profile, nothing to clear",
			"backupProfileId", backupProfile.ID)
		return nil
	}

	// Clear WillBePruned flags for all archives in these repositories that belong to this backup profile
	clearedCount, err := s.db.Debug().Archive.
		Update().
		Where(archive.And(
			archive.HasRepositoryWith(repository.IDIn(repositoryIDs...)),
			archive.HasBackupProfileWith(backupprofile.ID(backupProfile.ID)),
			archive.WillBePruned(true),
		)).
		SetWillBePruned(false).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to clear WillBePruned flags: %w", err)
	}

	s.log.Infow("Cleared WillBePruned flags",
		"backupProfileId", backupProfile.ID,
		"repositoryCount", len(repositoryIDs),
		"clearedArchiveCount", clearedCount)

	// Emit archive change events for all affected repositories
	for _, repo := range backupProfile.Edges.Repositories {
		s.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repo.ID))
	}

	return nil
}

// ExaminePrunes analyzes what would be pruned with given rules
func (s *Service) ExaminePrunes(ctx context.Context, backupProfileId int, pruningRule *backupprofileservice.PruningRule, saveResults bool) ([]ExaminePruningResult, error) {
	backupProfile, err := s.db.BackupProfile.
		Query().
		WithRepositories().
		Where(backupprofile.ID(backupProfileId)).
		Only(ctx)
	if err != nil {
		return []ExaminePruningResult{{Error: err}}, nil
	}

	// Convert custom type to ent type for internal use
	entPruningRule := pruningRule.ToEnt()

	// If pruning is disabled and we're saving results, just clear all WillBePruned flags
	if !entPruningRule.IsEnabled && saveResults {
		s.log.Infow("Pruning rule is disabled, clearing all WillBePruned flags",
			"backupProfileId", backupProfileId)
		err := s.clearWillBePrunedFlags(ctx, backupProfile)
		if err != nil {
			s.log.Errorw("Failed to clear WillBePruned flags",
				"backupProfileId", backupProfileId,
				"error", err)
			return []ExaminePruningResult{{Error: err}}, nil
		}
		return []ExaminePruningResult{}, nil
	}

	var wg sync.WaitGroup
	resultCh := make(chan ExaminePruningResult, len(backupProfile.Edges.Repositories))
	results := make([]ExaminePruningResult, 0, len(backupProfile.Edges.Repositories))

	for _, repo := range backupProfile.Edges.Repositories {
		wg.Add(1)
		bId := types.BackupId{BackupProfileId: backupProfileId, RepositoryId: repo.ID}
		go s.startExaminePrune(ctx, bId, entPruningRule, saveResults, &wg, resultCh)
	}

	// Wait for all examine prune jobs to finish
	wg.Wait()
	close(resultCh)

	// Collect results
	for result := range resultCh {
		results = append(results, result)
	}

	return results, nil
}

// startExaminePrune starts an examine prune operation for a single repository
func (s *Service) startExaminePrune(ctx context.Context, bId types.BackupId, pruningRule *ent.PruningRule, saveResults bool, wg *sync.WaitGroup, resultCh chan<- ExaminePruningResult) {
	defer wg.Done()

	repo, err := s.db.Repository.Query().
		Where(repository.ID(bId.RepositoryId)).
		Select(repository.FieldName).
		Only(ctx)
	if err != nil {
		resultCh <- ExaminePruningResult{BackupID: bId, Error: err, RepositoryName: ""}
		return
	}

	// Create result channel for this operation
	pruneResultCh := make(chan borgtypes.PruneResult, 1)

	// Create examine prune operation with result channel
	examinePruneOp := statemachine.NewOperationExaminePrune(statemachine.ExaminePrune{
		BackupID:    bId,
		PruningRule: pruningRule,
		SaveResults: saveResults,
		ResultCh:    pruneResultCh,
	})

	// Create queued operation with immediate flag
	queue := s.queueManager.GetQueue(bId.RepositoryId)
	queuedOp := queue.CreateQueuedOperation(
		examinePruneOp,
		bId.RepositoryId,
		&bId.BackupProfileId,
		nil,  // no expiration
		true, // will start immediately or fail
	)

	// Add to queue (will start immediately or fail)
	operationID, err := s.queueManager.AddOperation(bId.RepositoryId, queuedOp)
	if err != nil {
		s.log.Debugf("Failed to start examine prune operation: %s", err)
		resultCh <- ExaminePruningResult{BackupID: bId, Error: err, RepositoryName: repo.Name}
		return
	}

	s.log.Debugf("Started examine prune operation %s for repository %s", operationID, repo.Name)

	// Wait for result from the operation with timeout
	select {
	case pruneResult := <-pruneResultCh:
		resultCh <- ExaminePruningResult{
			BackupID:               bId,
			RepositoryName:         repo.Name,
			CntArchivesToBeDeleted: len(pruneResult.PruneArchives),
			Error:                  nil,
		}
	case <-time.After(60 * time.Second):
		resultCh <- ExaminePruningResult{
			BackupID:       bId,
			RepositoryName: repo.Name,
			Error:          fmt.Errorf("examination timeout"),
		}
	case <-ctx.Done():
		resultCh <- ExaminePruningResult{
			BackupID:       bId,
			RepositoryName: repo.Name,
			Error:          ctx.Err(),
		}
	}
}

// FixStoredPassword validates and updates the stored password for a repository
func (s *Service) FixStoredPassword(ctx context.Context, repoId int, password string) (FixStoredPasswordResult, error) {
	// Validate password is not empty
	if password == "" {
		return FixStoredPasswordResult{
			Success:      false,
			ErrorMessage: "password cannot be empty",
		}, nil
	}

	// Get repository from database
	repoEntity, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		return FixStoredPasswordResult{Success: false}, fmt.Errorf("failed to get repository: %w", err)
	}

	// Test the new password by calling borg info
	_, status := s.borgClient.Info(ctx, repoEntity.URL, password, false)

	// Check if password is wrong
	if status.HasError() {
		if status.Error.ExitCode == borgtypes.ErrorPassphraseWrong.ExitCode {
			return FixStoredPasswordResult{
				Success:      false,
				ErrorMessage: "Incorrect password",
			}, nil
		}
		return FixStoredPasswordResult{
			Success:      false,
			ErrorMessage: status.GetError(),
		}, nil
	}

	// Get current repository state
	currentState := s.queueManager.GetRepositoryState(repoId)

	// Check if repository has passphrase error and clear it
	if statemachine.GetRepositoryStateType(currentState) == statemachine.RepositoryStateTypeError {
		if errorVariant, ok := currentState.(statemachine.ErrorVariant); ok {
			errorData := errorVariant()
			if errorData.ErrorType == statemachine.ErrorTypePassphrase {
				// Create new idle state
				idleState := statemachine.NewRepositoryStateIdle(statemachine.Idle{})

				// Transition from Error to Idle state
				err = s.stateMachine.Transition(repoId, currentState, idleState)
				if err != nil {
					s.log.Warnf("Failed to transition repository %d from error to idle state: %v", repoId, err)
				} else {
					// Update the repository state
					s.queueManager.setRepositoryState(repoId, idleState)
				}
			}
		}
	}

	// Password is correct, update in keyring
	err = s.keyring.SetRepositoryPassword(repoId, password)
	if err != nil {
		return FixStoredPasswordResult{Success: false}, fmt.Errorf("failed to update password in keyring: %w", err)
	}

	// Update has_password flag
	err = s.db.Repository.UpdateOneID(repoId).SetHasPassword(true).Exec(ctx)
	if err != nil {
		s.log.Warnf("Failed to update has_password flag for repository %d: %v", repoId, err)
	}

	return FixStoredPasswordResult{
		Success:      true,
		ErrorMessage: "",
	}, nil
}

// ChangePassphrase changes the passphrase of a repository's encryption key
func (s *Service) ChangePassphrase(ctx context.Context, repoId int, currentPassword, newPassword string) (ChangePassphraseResult, error) {
	// Validate inputs
	if currentPassword == "" {
		return ChangePassphraseResult{
			Success:      false,
			ErrorMessage: "current password cannot be empty",
		}, nil
	}
	if newPassword == "" {
		return ChangePassphraseResult{
			Success:      false,
			ErrorMessage: "new password cannot be empty",
		}, nil
	}
	if currentPassword == newPassword {
		return ChangePassphraseResult{
			Success:      false,
			ErrorMessage: "new password must be different from current password",
		}, nil
	}

	// Get repository from database
	repoEntity, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		return ChangePassphraseResult{Success: false}, fmt.Errorf("failed to get repository: %w", err)
	}

	// Check repository is in Idle state
	currentState := s.queueManager.GetRepositoryState(repoId)
	if statemachine.GetRepositoryStateType(currentState) != statemachine.RepositoryStateTypeIdle {
		return ChangePassphraseResult{
			Success:      false,
			ErrorMessage: "repository must be idle to change passphrase",
		}, nil
	}

	// Call borg to change the passphrase
	status := s.borgClient.ChangePassphrase(ctx, repoEntity.URL, currentPassword, newPassword)

	// Check for errors
	if status.HasError() {
		if status.Error.ExitCode == borgtypes.ErrorPassphraseWrong.ExitCode {
			return ChangePassphraseResult{
				Success:      false,
				ErrorMessage: "incorrect current password",
			}, nil
		}
		return ChangePassphraseResult{
			Success:      false,
			ErrorMessage: status.GetError(),
		}, nil
	}

	// Passphrase changed successfully, update in keyring
	err = s.keyring.SetRepositoryPassword(repoId, newPassword)
	if err != nil {
		return ChangePassphraseResult{Success: false}, fmt.Errorf("failed to update password in keyring: %w", err)
	}

	// Update has_password flag
	err = s.db.Repository.UpdateOneID(repoId).SetHasPassword(true).Exec(ctx)
	if err != nil {
		s.log.Warnf("Failed to update has_password flag for repository %d: %v", repoId, err)
	}

	return ChangePassphraseResult{
		Success:      true,
		ErrorMessage: "",
	}, nil
}

// RegenerateSSHKey regenerates SSH key for ArcoCloud repositories
func (s *Service) RegenerateSSHKey(ctx context.Context) error {
	err := s.cloudRepoClient.AddOrReplaceSSHKey(ctx)
	if err != nil {
		return err
	}

	// Get all repositories and clear SSH error states for cloud repositories
	repos, err := s.All(ctx)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if s.isCloudRepository(ctx, repo.ID) {
			// Get current repository state
			currentState := s.queueManager.GetRepositoryState(repo.ID)

			// Check if repository has SSH key error
			if statemachine.GetRepositoryStateType(currentState) == statemachine.RepositoryStateTypeError {
				if errorVariant, ok := currentState.(statemachine.ErrorVariant); ok {
					errorData := errorVariant()
					if errorData.ErrorType == statemachine.ErrorTypeSSHKey {
						// Create new idle state
						idleState := statemachine.NewRepositoryStateIdle(statemachine.Idle{})

						// Transition from Error to Idle state
						err = s.stateMachine.Transition(repo.ID, currentState, idleState)
						if err != nil {
							return fmt.Errorf("failed to transition repository %d from error to idle state: %w", repo.ID, err)
						}

						// Update the repository state
						s.queueManager.setRepositoryState(repo.ID, idleState)
					}
				}
			}
		}
	}

	return nil
}

// BreakLock breaks a repository lock
func (s *Service) BreakLock(ctx context.Context, repoId int) error {
	// Get repository from database
	repoEntity, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		return fmt.Errorf("failed to get repository %d: %w", repoId, err)
	}

	// Get current state
	currentState := s.queueManager.GetRepositoryState(repoId)

	// Validate repository is in error state
	if statemachine.GetRepositoryStateType(currentState) != statemachine.RepositoryStateTypeError {
		return fmt.Errorf("repository %d is not in error state, cannot break lock", repoId)
	}

	// Get password from keyring
	password, err := s.keyring.GetRepositoryPassword(repoId)
	if err != nil {
		return fmt.Errorf("failed to get password for repository %d: %w", repoId, err)
	}

	// Break borg repository lock
	status := s.borgClient.BreakLock(ctx, repoEntity.URL, password)
	if !status.IsCompletedWithSuccess() {
		return fmt.Errorf("failed to break lock for repository %d: %s", repoId, status.GetError())
	}

	// Transition state from Error to Idle
	idleState := statemachine.NewRepositoryStateIdle(statemachine.Idle{})
	err = s.stateMachine.Transition(repoId, currentState, idleState)
	if err != nil {
		return fmt.Errorf("failed to transition repository %d from error to idle state: %w", repoId, err)
	}

	// Update the repository state
	s.queueManager.setRepositoryState(repoId, idleState)

	return nil
}

// ============================================================================
// ARCHIVE METHODS
// ============================================================================

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

// editStateData holds the accumulated edit state data for an archive
type editStateData struct {
	newName    *string
	newComment *string
	isActive   bool
}

// getArchiveOperationStates returns maps of archive IDs to their operation states
func (s *Service) getArchiveOperationStates(repoID int) (map[int]ArchiveEditState, map[int]ArchiveDeleteState) {
	editData := make(map[int]*editStateData) // Track edit data per archive
	deleteStates := make(map[int]ArchiveDeleteState)

	// Get the repository queue
	queue := s.queueManager.GetQueue(repoID)

	// Check active operation
	if active := queue.GetActive(); active != nil {
		switch statemachine.GetOperationType(active.Operation) {
		case statemachine.OperationTypeArchiveRename:
			renameOp := active.Operation.(statemachine.ArchiveRenameVariant)()
			newName := renameOp.Prefix + renameOp.Name
			if editData[renameOp.ArchiveID] == nil {
				editData[renameOp.ArchiveID] = &editStateData{}
			}
			editData[renameOp.ArchiveID].newName = &newName
			editData[renameOp.ArchiveID].isActive = true
		case statemachine.OperationTypeArchiveComment:
			commentOp := active.Operation.(statemachine.ArchiveCommentVariant)()
			if editData[commentOp.ArchiveID] == nil {
				editData[commentOp.ArchiveID] = &editStateData{}
			}
			editData[commentOp.ArchiveID].newComment = &commentOp.Comment
			editData[commentOp.ArchiveID].isActive = true
		case statemachine.OperationTypeArchiveDelete:
			deleteOp := active.Operation.(statemachine.ArchiveDeleteVariant)()
			deleteStates[deleteOp.ArchiveID] = NewArchiveDeleteStateDeleteActive(DeleteActive{})
		case statemachine.OperationTypeArchiveRefresh,
			statemachine.OperationTypeBackup,
			statemachine.OperationTypeCheck,
			statemachine.OperationTypeDelete,
			statemachine.OperationTypeExaminePrune,
			statemachine.OperationTypeMount,
			statemachine.OperationTypeMountArchive,
			statemachine.OperationTypePrune,
			statemachine.OperationTypeUnmount,
			statemachine.OperationTypeUnmountArchive:
			// These operations don't affect individual archive edit/delete states
		}
	}

	// Check queued rename operations
	archiveRenameType := statemachine.OperationTypeArchiveRename
	queuedRenames := queue.GetQueuedOperations(&archiveRenameType)
	for _, queuedOp := range queuedRenames {
		renameOp := queuedOp.Operation.(statemachine.ArchiveRenameVariant)()
		newName := renameOp.Prefix + renameOp.Name
		if editData[renameOp.ArchiveID] == nil {
			editData[renameOp.ArchiveID] = &editStateData{}
		}
		// Only set if not already active (active takes precedence)
		if editData[renameOp.ArchiveID].newName == nil {
			editData[renameOp.ArchiveID].newName = &newName
		}
	}

	// Check queued comment operations
	archiveCommentType := statemachine.OperationTypeArchiveComment
	queuedComments := queue.GetQueuedOperations(&archiveCommentType)
	for _, queuedOp := range queuedComments {
		commentOp := queuedOp.Operation.(statemachine.ArchiveCommentVariant)()
		if editData[commentOp.ArchiveID] == nil {
			editData[commentOp.ArchiveID] = &editStateData{}
		}
		// Only set if not already active (active takes precedence)
		if editData[commentOp.ArchiveID].newComment == nil {
			editData[commentOp.ArchiveID].newComment = &commentOp.Comment
		}
	}

	// Check queued delete operations
	archiveDeleteType := statemachine.OperationTypeArchiveDelete
	queuedDeletes := queue.GetQueuedOperations(&archiveDeleteType)
	for _, queuedOp := range queuedDeletes {
		deleteOp := queuedOp.Operation.(statemachine.ArchiveDeleteVariant)()
		deleteStates[deleteOp.ArchiveID] = NewArchiveDeleteStateDeleteQueued(DeleteQueued{})
	}

	// Convert editData to ArchiveEditState
	editStates := make(map[int]ArchiveEditState)
	for archiveID, data := range editData {
		if data.isActive {
			editStates[archiveID] = NewArchiveEditStateEditActive(EditActive{
				NewName:    data.newName,
				NewComment: data.newComment,
			})
		} else {
			editStates[archiveID] = NewArchiveEditStateEditQueued(EditQueued{
				NewName:    data.newName,
				NewComment: data.newComment,
			})
		}
	}

	return editStates, deleteStates
}

// GetPaginatedArchives retrieves paginated archives for a repository
func (s *Service) GetPaginatedArchives(ctx context.Context, req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error) {
	if req.RepositoryId <= 0 {
		return nil, fmt.Errorf("repositoryId is required")
	}
	if req.Page <= 0 {
		return nil, fmt.Errorf("page is required")
	}
	if req.PageSize <= 0 {
		return nil, fmt.Errorf("pageSize is required")
	}

	// Filter by repository
	archivePredicates := []predicate.Archive{
		archive.HasRepositoryWith(repository.ID(req.RepositoryId)),
	}

	// If a backup profile filter is specified, filter by it
	if req.BackupProfileFilter != nil {
		if req.BackupProfileFilter.Id != 0 {
			// First filter by BackupProfile.ID
			archivePredicates = append(archivePredicates, archive.HasBackupProfileWith(backupprofile.ID(req.BackupProfileFilter.Id)))
		} else if req.BackupProfileFilter.IsUnknownFilter {
			// If the unknown filter is specified, filter by archives that don't have a backup profile
			archivePredicates = append(archivePredicates, archive.Not(archive.HasBackupProfile()))
		}
		// Filter by BackupProfile.Name does not have to be supported
		// Filter all is implicit
	}

	// If a search term is specified, filter by name or comment
	if req.Search != "" {
		archivePredicates = append(archivePredicates, archive.Or(
			archive.NameContains(req.Search),
			archive.CommentContains(req.Search),
		))
	}

	// If start date is specified, filter by it
	if !req.StartDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtGTE(req.StartDate))
	}

	// If end date is specified, filter by it
	if !req.EndDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtLTE(req.EndDate))
	}

	total, err := s.db.Archive.
		Query().
		Where(archive.And(archivePredicates...)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	archives, err := s.db.Archive.
		Query().
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.Select(backupprofile.FieldName)
			q.Select(backupprofile.FieldPrefix)
		}).
		Where(archive.And(archivePredicates...)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Get pending operation states for this repository
	editStates, deleteStates := s.getArchiveOperationStates(req.RepositoryId)

	// Convert archives to enhanced archives with pending changes
	enhancedArchives := make([]*ArchiveWithPendingChanges, len(archives))
	for i, archiveEntity := range archives {
		enhancedArchive := &ArchiveWithPendingChanges{
			Archive: archiveEntity,
		}

		// Set edit state (default to none if not found)
		if editState, exists := editStates[archiveEntity.ID]; exists {
			enhancedArchive.EditStateUnion = ToArchiveEditStateUnion(editState)
		} else {
			noneState := NewArchiveEditStateEditNone(EditNone{})
			enhancedArchive.EditStateUnion = ToArchiveEditStateUnion(noneState)
		}

		// Set delete state (default to none if not found)
		if deleteState, exists := deleteStates[archiveEntity.ID]; exists {
			enhancedArchive.DeleteStateUnion = ToArchiveDeleteStateUnion(deleteState)
		} else {
			noneState := NewArchiveDeleteStateDeleteNone(DeleteNone{})
			enhancedArchive.DeleteStateUnion = ToArchiveDeleteStateUnion(noneState)
		}

		enhancedArchives[i] = enhancedArchive
	}

	return &PaginatedArchivesResponse{
		Archives: enhancedArchives,
		Total:    total,
	}, nil
}

// GetFilteredArchiveIds retrieves all archive IDs matching the filter criteria (without pagination)
// This is used for "select all across pages" functionality
func (s *Service) GetFilteredArchiveIds(ctx context.Context, req *PaginatedArchivesRequest) ([]int, error) {
	if req.RepositoryId <= 0 {
		return nil, fmt.Errorf("repositoryId is required")
	}

	// Build filter predicates (same logic as GetPaginatedArchives)
	archivePredicates := []predicate.Archive{
		archive.HasRepositoryWith(repository.ID(req.RepositoryId)),
	}

	if req.BackupProfileFilter != nil {
		if req.BackupProfileFilter.Id != 0 {
			archivePredicates = append(archivePredicates, archive.HasBackupProfileWith(backupprofile.ID(req.BackupProfileFilter.Id)))
		} else if req.BackupProfileFilter.IsUnknownFilter {
			archivePredicates = append(archivePredicates, archive.Not(archive.HasBackupProfile()))
		}
	}

	if req.Search != "" {
		archivePredicates = append(archivePredicates, archive.Or(
			archive.NameContains(req.Search),
			archive.CommentContains(req.Search),
		))
	}

	if !req.StartDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtGTE(req.StartDate))
	}

	if !req.EndDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtLTE(req.EndDate))
	}

	// Query only IDs
	ids, err := s.db.Archive.
		Query().
		Where(archive.And(archivePredicates...)).
		IDs(ctx)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// GetPruningDates retrieves pruning dates for specified archives
func (s *Service) GetPruningDates(ctx context.Context, archiveIds []int) (PruningDates, error) {
	var pruningDates PruningDates
	archives, err := s.db.Archive.
		Query().
		Where(archive.And(
			archive.IDIn(archiveIds...),
			archive.HasBackupProfile(),
			archive.WillBePruned(true),
		)).
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithPruningRule()
		}).
		All(ctx)
	if err != nil {
		return pruningDates, err
	}
	for _, arch := range archives {
		if arch.Edges.BackupProfile.Edges.PruningRule != nil {
			pruningDates.Dates = append(pruningDates.Dates, PruningDate{
				ArchiveId: arch.ID,
				Date:      arch.Edges.BackupProfile.Edges.PruningRule.NextRun,
			})
		}
	}
	return pruningDates, nil
}

// GetLastArchiveByBackupId gets last archive for backup profile
func (s *Service) GetLastArchiveByBackupId(ctx context.Context, backupId types.BackupId) (*ent.Archive, error) {
	first, err := s.db.Archive.
		Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(backupId.RepositoryId)),
			archive.HasBackupProfileWith(backupprofile.ID(backupId.BackupProfileId)),
		)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)
	if err != nil && ent.IsNotFound(err) {
		return &ent.Archive{}, nil
	}
	return first, err
}

// ============================================================================
// BACKUP MANAGEMENT
// ============================================================================

// GetBackupButtonStatus gets backup button status for given backup IDs
func (s *Service) GetBackupButtonStatus(ctx context.Context, backupIds []types.BackupId) (BackupButtonStatus, error) {
	if len(backupIds) == 0 {
		return BackupButtonStatusRunBackup, nil
	}

	if len(backupIds) == 1 {
		return s.getSingleBackupButtonStatus(ctx, backupIds[0])
	}

	return s.getCombinedBackupButtonStatus(ctx, backupIds)
}

// getSingleBackupButtonStatus gets backup button status for a single backup ID
func (s *Service) getSingleBackupButtonStatus(ctx context.Context, backupId types.BackupId) (BackupButtonStatus, error) {
	// Check repository state first
	repositoryState := s.queueManager.GetRepositoryState(backupId.RepositoryId)

	// Check repository state type
	switch statemachine.GetRepositoryStateType(repositoryState) {
	case statemachine.RepositoryStateTypeError:
		return BackupButtonStatusLocked, nil
	case statemachine.RepositoryStateTypeMounted:
		return BackupButtonStatusUnmount, nil
	case statemachine.RepositoryStateTypeQueued:
		// Check if this specific backup is queued
		if s.isBackupInQueue(backupId) {
			return BackupButtonStatusWaiting, nil
		}
		// Repository is busy with another operation
		return BackupButtonStatusBusy, nil
	case statemachine.RepositoryStateTypeBackingUp:
		// Check if this is our backup that's running
		backingUpVariant := repositoryState.(statemachine.BackingUpVariant)
		backingUpData := backingUpVariant()
		if backingUpData.Data.BackupID.String() == backupId.String() {
			return BackupButtonStatusAbort, nil
		}
		// Repository is busy with another backup
		return BackupButtonStatusBusy, nil
	case statemachine.RepositoryStateTypePruning,
		statemachine.RepositoryStateTypeDeleting,
		statemachine.RepositoryStateTypeRefreshing,
		statemachine.RepositoryStateTypeChecking,
		statemachine.RepositoryStateTypeMounting:
		// Repository is busy with other operations
		return BackupButtonStatusBusy, nil
	case statemachine.RepositoryStateTypeIdle:
		// Repository is idle, can run backup
		return BackupButtonStatusRunBackup, nil
	default:
		assert.Fail("Unhandled RepositoryStateType in getSingleBackupButtonStatus")
		return BackupButtonStatusRunBackup, nil
	}
}

// getCombinedBackupButtonStatus gets combined backup button status for multiple backup IDs
func (s *Service) getCombinedBackupButtonStatus(ctx context.Context, backupIds []types.BackupId) (BackupButtonStatus, error) {
	hasWaiting := false
	hasRunning := false

	for _, backupId := range backupIds {
		status, err := s.getSingleBackupButtonStatus(ctx, backupId)
		if err != nil {
			return BackupButtonStatusRunBackup, err
		}

		// High priority statuses that should return immediately
		switch status {
		case BackupButtonStatusLocked:
			return BackupButtonStatusLocked, nil
		case BackupButtonStatusUnmount:
			return BackupButtonStatusUnmount, nil
		case BackupButtonStatusBusy:
			return BackupButtonStatusBusy, nil
		case BackupButtonStatusWaiting:
			hasWaiting = true
		case BackupButtonStatusAbort:
			hasRunning = true
		case BackupButtonStatusRunBackup:

		}
	}

	// Return combined status based on what we found
	if hasRunning {
		return BackupButtonStatusAbort, nil
	}
	if hasWaiting {
		return BackupButtonStatusWaiting, nil
	}

	// All backups are idle
	return BackupButtonStatusRunBackup, nil
}

// isBackupInQueue checks if a backup operation is queued or active
func (s *Service) isBackupInQueue(backupId types.BackupId) bool {
	// Check active operations
	activeOps := s.queueManager.GetActiveOperations()
	for _, op := range activeOps {
		if backupVariant, isBackup := op.Operation.(statemachine.BackupVariant); isBackup {
			backupData := backupVariant()
			if backupData.BackupID.String() == backupId.String() {
				return true
			}
		}
	}

	// Check queued operations for the repository
	queuedOps, err := s.queueManager.GetQueuedOperations(backupId.RepositoryId, nil)
	if err != nil {
		return false
	}

	for _, op := range queuedOps {
		if backupVariant, isBackup := op.Operation.(statemachine.BackupVariant); isBackup {
			backupData := backupVariant()
			if backupData.BackupID.String() == backupId.String() {
				return true
			}
		}
	}

	return false
}

// findOperationIDByBackupID finds the operation ID for a given backup ID
func (s *Service) findOperationIDByBackupID(backupId types.BackupId) (string, error) {
	// Check active operations first
	activeOps := s.queueManager.GetActiveOperations()
	for _, op := range activeOps {
		if backupVariant, isBackup := op.Operation.(statemachine.BackupVariant); isBackup {
			backupData := backupVariant()
			if backupData.BackupID.String() == backupId.String() {
				return op.ID, nil
			}
		}
	}

	// Check queued operations for the repository
	queuedOps, err := s.queueManager.GetQueuedOperations(backupId.RepositoryId, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get queued operations: %w", err)
	}

	for _, op := range queuedOps {
		if backupVariant, isBackup := op.Operation.(statemachine.BackupVariant); isBackup {
			backupData := backupVariant()
			if backupData.BackupID.String() == backupId.String() {
				return op.ID, nil
			}
		}
	}

	return "", fmt.Errorf("backup operation not found for backup ID %s", backupId.String())
}

// GetCombinedBackupProgress gets backup progress for given backup IDs
func (s *Service) GetCombinedBackupProgress(ctx context.Context, backupIds []types.BackupId) (*borgtypes.BackupProgress, error) {
	// Get all active operations from queue manager
	activeOperations := s.queueManager.GetActiveOperations()

	var totalFiles, processedFiles int
	found := false

	// Iterate through provided backup IDs
	for _, targetBackupId := range backupIds {
		// Check active operations for matching backup operations
		for _, operation := range activeOperations {
			// Check if this operation is a backup operation
			if backupVariant, isBackup := operation.Operation.(statemachine.BackupVariant); isBackup {
				backupData := backupVariant()
				// Check if this backup operation matches our target backup ID
				if backupData.BackupID.String() == targetBackupId.String() {
					// Check if the operation has progress data
					if backupData.Progress != nil {
						found = true
						totalFiles += backupData.Progress.TotalFiles
						processedFiles += backupData.Progress.ProcessedFiles
					}
				}
			}
		}
	}

	// Return combined progress if any was found, otherwise nil
	if !found {
		return nil, nil
	}

	return &borgtypes.BackupProgress{
		TotalFiles:     totalFiles,
		ProcessedFiles: processedFiles,
	}, nil
}

// GetBackupState gets backup state for given backup ID
func (s *Service) GetBackupState(ctx context.Context, backupId types.BackupId) (*statemachine.Backup, error) {
	// Get active operations from queue manager
	activeOperations := s.queueManager.GetActiveOperations()

	// Search through active operations for matching backup ID
	for _, operation := range activeOperations {
		// Check if this operation is a backup operation
		if backupVariant, isBackup := operation.Operation.(statemachine.BackupVariant); isBackup {
			backupData := backupVariant()
			// Check if this backup operation matches our target backup ID
			if backupData.BackupID.String() == backupId.String() {
				return &backupData, nil
			}
		}
	}

	// No active backup found for this backup ID
	return nil, nil
}

// ============================================================================
// VALIDATION METHODS
// ============================================================================

// ValidateRepoName validates a repository name
func (s *Service) ValidateRepoName(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < schema.ValRepositoryMinNameLength {
		return fmt.Sprintf("Name must be at least %d characters long", schema.ValRepositoryMinNameLength), nil
	}
	if len(name) > schema.ValRepositoryMaxNameLength {
		return fmt.Sprintf("Name can not be longer than %d characters", schema.ValRepositoryMaxNameLength), nil
	}

	exist, err := s.db.Repository.
		Query().
		Where(repository.Name(name)).
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Repository name must be unique", nil
	}

	return "", nil
}

// ValidateRepoPath validates a repository path
func (s *Service) ValidateRepoPath(ctx context.Context, path string, isLocal bool) (string, error) {
	if path == "" {
		return "Path is required", nil
	}
	if isLocal {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~") {
			return "Path must start with / or ~", nil
		}
		expandedPath := util.ExpandPath(path)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return "Path does not exist", nil
		}
		if info, err := os.Stat(expandedPath); err == nil && !info.IsDir() {
			return "Path is not a folder", nil
		}
		if entries, err := os.ReadDir(expandedPath); err == nil && len(entries) > 0 {
			if !s.IsBorgRepository(expandedPath) {
				return "Folder must be empty", nil
			}
		}
	}

	exist, err := s.db.Repository.
		Query().
		Where(repository.URL(path)).
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Repository is already connected", nil
	}

	return "", nil
}

// ValidatePathChange validates path format and uniqueness (no connection test).
// Returns:
// - IsValid=false + ErrorMessage: Blocking errors (cannot proceed)
// - IsValid=true: Path is valid (connection not tested)
func (s *Service) ValidatePathChange(ctx context.Context, repoId int, newPath string) (*ValidatePathChangeResult, error) {
	// Get existing repository with cloud association
	repoEntity, err := s.db.Repository.Query().
		Where(repository.ID(repoId)).
		WithCloudRepository().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return &ValidatePathChangeResult{
				IsValid:      false,
				ErrorMessage: "Repository not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// 1. Check not ArcoCloud
	if repoEntity.Edges.CloudRepository != nil {
		return &ValidatePathChangeResult{
			IsValid:      false,
			ErrorMessage: "Cannot change path for ArcoCloud repositories",
		}, nil
	}

	// Determine if current repo is local (path starts with /)
	isCurrentLocal := strings.HasPrefix(repoEntity.URL, "/")
	isNewLocal := strings.HasPrefix(newPath, "/")

	// 2. Validate path format matches repo type
	if isCurrentLocal != isNewLocal {
		if isCurrentLocal {
			return &ValidatePathChangeResult{
				IsValid:      false,
				ErrorMessage: "New path must be a local path (starting with /)",
			}, nil
		}
		return &ValidatePathChangeResult{
			IsValid:      false,
			ErrorMessage: "New path must be a remote path (SSH format)",
		}, nil
	}

	// 3. For local paths, validate path exists
	if isNewLocal {
		expandedPath := util.ExpandPath(newPath)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return &ValidatePathChangeResult{
				IsValid:      false,
				ErrorMessage: "Path does not exist",
			}, nil
		}
		if info, err := os.Stat(expandedPath); err == nil && !info.IsDir() {
			return &ValidatePathChangeResult{
				IsValid:      false,
				ErrorMessage: "Path is not a folder",
			}, nil
		}
	}

	// 4. Check path uniqueness (excluding current repo)
	exists, err := s.db.Repository.Query().
		Where(
			repository.URL(newPath),
			repository.IDNEQ(repoId),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check path uniqueness: %w", err)
	}
	if exists {
		return &ValidatePathChangeResult{
			IsValid:      false,
			ErrorMessage: "Path is already used by another repository",
		}, nil
	}

	return &ValidatePathChangeResult{
		IsValid: true,
	}, nil
}

// TestPathConnection tests if a path can connect to a valid borg repository.
// Requires repository to be in Idle state (for borg call).
// Returns:
// - IsValid=false + ErrorMessage: Not idle (blocking)
// - IsValid=true + ConnectionWarning: Connection failed (can proceed with warning)
// - IsValid=true: Connection successful
func (s *Service) TestPathConnection(ctx context.Context, repoId int, newPath string, password string) (*ValidatePathChangeResult, error) {
	// Check repository exists
	_, err := s.db.Repository.Get(ctx, repoId)
	if err != nil {
		if ent.IsNotFound(err) {
			return &ValidatePathChangeResult{
				IsValid:      false,
				ErrorMessage: "Repository not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Check repository is Idle (required for borg call)
	currentState := s.queueManager.GetRepositoryState(repoId)
	if statemachine.GetRepositoryStateType(currentState) != statemachine.RepositoryStateTypeIdle {
		return &ValidatePathChangeResult{
			IsValid:      false,
			ErrorMessage: "Repository must be idle to test connection",
		}, nil
	}

	// Test connection with borg info
	expandedPath := util.ExpandPath(newPath)
	usePassword := password
	if usePassword == "" {
		var err error
		usePassword, err = s.keyring.GetRepositoryPassword(repoId)
		if err != nil {
			s.log.Warnf("Failed to get repository password from keyring: %v", err)
		}
	}
	_, status := s.borgClient.Info(ctx, expandedPath, usePassword, true)
	if status != nil && status.HasError() {
		// Check for common errors and return as warnings
		var warning string
		if errors.Is(status.Error, borgtypes.ErrorRepositoryDoesNotExist) ||
			errors.Is(status.Error, borgtypes.ErrorRepositoryInvalidRepository) {
			warning = "Path is not a valid Borg repository"
		} else if errors.Is(status.Error, borgtypes.ErrorPassphraseWrong) {
			warning = "Invalid password"
		} else {
			warning = fmt.Sprintf("Failed to connect: %s", status.Error.Error())
		}
		return &ValidatePathChangeResult{
			IsValid:           true,
			ConnectionWarning: warning,
		}, nil
	}

	// Connection successful
	return &ValidatePathChangeResult{
		IsValid: true,
	}, nil
}

// ValidateArchiveName validates an archive name
func (s *Service) ValidateArchiveName(ctx context.Context, archiveId int, name string) (string, error) {
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < 3 {
		return "Name must be at least 3 characters long", nil
	}
	if len(name) > 50 {
		return "Name can not be longer than 50 characters", nil
	}
	pattern := `^[a-zA-Z0-9-_]+$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return "", err
	}
	if !matched {
		return "Name can only contain letters, numbers, hyphens, and underscores", nil
	}

	// Get archive with repository and backup profile edges
	arch, err := s.db.Archive.Query().
		Where(archive.ID(archiveId)).
		WithRepository().
		WithBackupProfile().
		Only(ctx)
	if err != nil {
		return "", err
	}
	assert.NotNil(arch.Edges.Repository, "archive must have a repository")

	// Get prefix from backup profile
	var prefix string
	if arch.Edges.BackupProfile != nil {
		prefix = arch.Edges.BackupProfile.Prefix
	}

	// Check prefix requirements
	if arch.Edges.BackupProfile != nil {
		// For archives with backup profiles, the name will be prefixed automatically
		// so no additional validation needed for the prefix
	} else {
		if prefix != "" {
			err = fmt.Errorf("the archive can not have a prefix if it is not connected to a backup profile")
			assert.Error(err)
			return "", err
		}

		// If it is not connected to a backup profile,
		// it can not start with any prefix used by another backup profile of the repository
		backupProfiles, err := arch.Edges.Repository.QueryBackupProfiles().All(ctx)
		if err != nil {
			return "", err
		}
		for _, bp := range backupProfiles {
			prefixWithoutTrailingDash := strings.TrimSuffix(bp.Prefix, "-")
			if strings.HasPrefix(name, prefixWithoutTrailingDash) {
				return "The new name must not start with the prefix of another backup profile", nil
			}
		}
	}

	fullName := prefix + name
	exist, err := s.db.Archive.
		Query().
		Where(archive.Name(fullName)).
		Where(archive.HasRepositoryWith(repository.ID(arch.Edges.Repository.ID))).
		Where(archive.IDNEQ(archiveId)).
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Archive name must be unique", nil
	}

	return "", nil
}

// testRepoConnection performs the actual repository connection test
func (s *Service) testRepoConnection(ctx context.Context, path, password string) (testRepoConnectionResult, error) {
	// Expand ~ in local paths to user home directory
	path = util.ExpandPath(path)
	s.log.Debugf("Testing repository connection to %s", path)
	result := testRepoConnectionResult{
		Success:         false,
		IsPasswordValid: false,
		IsBorgRepo:      false,
	}

	_, status := s.borgClient.Info(ctx, path, password, false)
	if status == nil {
		result.Success = true
		result.IsPasswordValid = true
		result.IsBorgRepo = true
		return result, nil
	}

	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			return result, fmt.Errorf("repository info retrieval was cancelled")
		}
		if errors.Is(status.Error, borgtypes.ErrorPassphraseWrong) {
			result.IsBorgRepo = true
			return result, nil
		}
		if errors.Is(status.Error, borgtypes.ErrorRepositoryDoesNotExist) || errors.Is(status.Error, borgtypes.ErrorRepositoryInvalidRepository) {
			return result, nil
		}
		return result, fmt.Errorf("info command failed: %s", status.GetError())
	}

	if status.HasWarning() {
		s.log.Warnf("Repository info retrieval completed with warning: %s", status.GetWarning())
	}
	result.Success = true
	result.IsPasswordValid = true
	result.IsBorgRepo = true
	return result, nil
}

// TestRepoConnection tests connection to a repository
func (s *Service) TestRepoConnection(ctx context.Context, path, password string) (TestRepoConnectionResult, error) {
	// Expand ~ in local paths to user home directory
	path = util.ExpandPath(path)
	toTestRepoConnectionResult := func(t testRepoConnectionResult, err error, needsPassword bool) (TestRepoConnectionResult, error) {
		if err != nil {
			return TestRepoConnectionResult{}, err
		}
		result := TestRepoConnectionResult{}
		result.Success = t.Success
		result.IsPasswordValid = t.IsPasswordValid
		result.IsBorgRepo = t.IsBorgRepo
		result.NeedsPassword = needsPassword
		return result, nil
	}

	// First test with a random password
	// If the test is successful, we know that the repository is not password protected
	randPassword := uuid.New().String()
	tr, err := s.testRepoConnection(ctx, path, randPassword)
	if err != nil || tr.Success || !tr.IsBorgRepo {
		return toTestRepoConnectionResult(tr, err, false)
	}

	// If we are here it means we need a password
	if password != "" {
		tr, err = s.testRepoConnection(ctx, path, password)
		return toTestRepoConnectionResult(tr, err, true)
	}
	return toTestRepoConnectionResult(tr, nil, true)
}

// IsBorgRepository checks if a path contains a borg repository
func (s *Service) IsBorgRepository(path string) bool {
	// Check if path has a README file with the string borg in it
	fp := filepath.Join(path, "README")
	_, err := os.Stat(fp)
	if err != nil {
		return false
	}
	contents, err := os.ReadFile(fp)
	if err != nil {
		return false
	}

	if strings.Contains(string(contents), "borg") {
		return true
	}

	// Otherwise check if we have a config file
	fp = filepath.Join(path, "config")
	_, err = os.Stat(fp)
	if err != nil {
		return false
	}
	contents, err = os.ReadFile(fp)
	if err != nil {
		return false
	}
	return strings.Contains(string(contents), "[repository]")
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

// getLastErrorNotification returns the most recent error notification for a repository
func (s *Service) getLastErrorNotification(ctx context.Context, repoID int) *ent.Notification {
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
		if !ent.IsNotFound(err) {
			s.log.Errorw("Failed to query error notifications",
				"repoID", repoID,
				"error", err.Error())
		}
		return nil
	}
	return notificationEnt
}

// getLastBackup returns info about the last successful backup
func (s *Service) getLastBackup(ctx context.Context, repoID int) *types.LastBackup {
	lastArchive, err := s.db.Archive.Query().
		Where(archive.HasRepositoryWith(repository.ID(repoID))).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)

	if err != nil {
		if !ent.IsNotFound(err) {
			s.log.Errorw("Failed to query latest archive",
				"repoID", repoID,
				"error", err.Error())
		}
		return nil // No successful backups yet
	}

	result := &types.LastBackup{
		Timestamp: &lastArchive.CreatedAt,
	}
	if lastArchive.WarningMessage != nil {
		result.WarningMessage = *lastArchive.WarningMessage
	}
	return result
}

// getLastAttempt returns info about the most recent attempt (success, warning, or error)
func (s *Service) getLastAttempt(ctx context.Context, repoID int) *types.LastAttempt {
	// Get latest archive
	lastArchive, err := s.db.Archive.Query().
		Where(archive.HasRepositoryWith(repository.ID(repoID))).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)

	if err != nil && !ent.IsNotFound(err) {
		s.log.Errorw("Failed to query latest archive for attempt",
			"repoID", repoID,
			"error", err.Error())
	}

	// Get latest error notification
	errorNotification := s.getLastErrorNotification(ctx, repoID)

	// Determine which is more recent
	if errorNotification != nil {
		// Check if error is newer than archive (or no archive exists)
		if lastArchive == nil || errorNotification.CreatedAt.After(lastArchive.CreatedAt) {
			return &types.LastAttempt{
				Status:    types.BackupStatusError,
				Timestamp: &errorNotification.CreatedAt,
				Message:   errorNotification.Message,
			}
		}
	}

	if lastArchive == nil {
		return nil // No attempts yet
	}

	// Archive is the most recent attempt
	if lastArchive.WarningMessage != nil {
		return &types.LastAttempt{
			Status:    types.BackupStatusWarning,
			Timestamp: &lastArchive.CreatedAt,
			Message:   *lastArchive.WarningMessage,
		}
	}

	return &types.LastAttempt{
		Status:    types.BackupStatusSuccess,
		Timestamp: &lastArchive.CreatedAt,
	}
}

// GetConnectedRemoteHosts gets connected remote hosts
func (s *Service) GetConnectedRemoteHosts(ctx context.Context) ([]string, error) {
	repos, err := s.db.Repository.Query().
		Where(repository.And(
			repository.URLContains("@"),
			repository.Not(repository.HasCloudRepository()),
		)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// user@host:~/path/to/repo -> user@host:port
	// ssh://user@host:port/./path/to/repo -> user@host:port
	hosts := make(map[string]struct{})
	for _, repo := range repos {
		// Extract user, host and port from location
		parsedURL, err := url.Parse(repo.URL)
		if err != nil {
			continue
		}
		userInfo := ""
		if parsedURL.User != nil {
			userInfo = parsedURL.User.String() + "@"
		}
		host := parsedURL.Host
		// Add host to map
		hosts[userInfo+host] = struct{}{}
	}

	// Convert map to slice
	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result, nil
}
