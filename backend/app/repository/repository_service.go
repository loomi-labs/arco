package repository

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/database"
	"github.com/loomi-labs/arco/backend/app/state"
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
	"github.com/loomi-labs/arco/backend/util"
	"github.com/negrel/assert"
	"go.uber.org/zap"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log          *zap.SugaredLogger
	db           *ent.Client
	state        *state.State
	borg         borg.Borg
	config       *types.Config
	eventEmitter types.EventEmitter
	cloudService *CloudRepositoryClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new repository service with minimal dependencies (two-phase initialization)
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the repository service with remaining dependencies
func (si *ServiceInternal) Init(db *ent.Client, borg borg.Borg, config *types.Config, eventEmitter types.EventEmitter, cloudService *CloudRepositoryClient) {
	si.db = db
	si.borg = borg
	si.config = config
	si.eventEmitter = eventEmitter
	si.cloudService = cloudService
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("RepositoryService: database client is nil")
	}
}

// isCloudRepository checks if a repository is an ArcoCloud repository
func (s *Service) isCloudRepository(ctx context.Context, repoId int) bool {
	exists, err := s.db.Repository.Query().
		Where(repository.And(
			repository.IDEQ(repoId),
			repository.HasCloudRepository(),
		)).
		Exist(ctx)
	if err != nil {
		s.log.Errorf("IsCloudRepository query error: %s", err)
	}
	return exists
}

// rollback helper function for transactions
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
	}
	return err
}

func (s *Service) Get(ctx context.Context, repoId int) (*ent.Repository, error) {
	s.mustHaveDB()
	return s.db.Repository.
		Query().
		WithCloudRepository().
		Where(repository.ID(repoId)).
		Only(ctx)
}

func (s *Service) GetByBackupId(ctx context.Context, bId types.BackupId) (*ent.Repository, error) {
	s.mustHaveDB()
	return s.db.Repository.
		Query().
		//WithNotifications(func(query *ent.NotificationQuery) {
		//	query.Where(notification.And(
		//		notification.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
		//		notification.HasRepositoryWith(repository.ID(bId.RepositoryId)),
		//		notification.Seen(false),
		//	))
		//}).
		Where(repository.And(
			repository.ID(bId.RepositoryId),
			repository.HasBackupProfilesWith(backupprofile.ID(bId.BackupProfileId)),
		)).
		Only(ctx)
}

func (s *Service) All(ctx context.Context) ([]*ent.Repository, error) {
	s.mustHaveDB()

	// Sync cloud repositories to ensure freshness
	//if _, err := s.cloudService.ListCloudRepositories(ctx); err != nil {
	//	s.log.Warnf("Failed to sync cloud repositories: %v", err)
	//}

	// Return all repositories (local + synced cloud)
	return s.db.Repository.
		Query().
		Order(func(sel *sql.Selector) {
			// Order by name, case-insensitive
			sel.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", repository.FieldName)))
		}).
		All(ctx)
}

func (s *Service) GetNbrOfArchives(ctx context.Context, repoId int) (int, error) {
	s.mustHaveDB()
	return s.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		Count(ctx)
}

func (s *Service) GetLastBackupErrorMsg(ctx context.Context, repoId int) (string, error) {
	s.mustHaveDB()
	// Get the last notification for the backup profile and repository
	// that is a failed backup run or failed pruning run
	lastNotification, err := s.db.Notification.
		Query().
		Where(notification.And(
			notification.HasRepositoryWith(repository.ID(repoId)),
		)).
		Order(ent.Desc(notification.FieldCreatedAt)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}
	if lastNotification != nil {
		// Check if there is a new archive since the last notification
		// If there is, we don't show the error message
		exist, err := s.db.Archive.
			Query().
			Where(archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.CreatedAtGT(lastNotification.CreatedAt),
			)).Exist(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return "", err
		}
		if !exist {
			return lastNotification.Message, nil
		}
	}
	return "", nil
}

func (s *Service) GetLocked(ctx context.Context) ([]*ent.Repository, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}

	locked := make([]*ent.Repository, 0)
	for _, repo := range all {
		if s.state.GetRepoState(repo.ID).Status == state.RepoStatusLocked {
			locked = append(locked, repo)
		}
	}
	return locked, nil
}

func (s *Service) GetWithActiveMounts(ctx context.Context) ([]*ent.Repository, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}

	active := make([]*ent.Repository, 0)
	for _, repo := range all {
		if s.state.GetRepoState(repo.ID).Status == state.RepoStatusMounted {
			active = append(active, repo)
		}
	}
	return active, nil
}

func (s *Service) Create(ctx context.Context, name, location, password string, noPassword bool) (*ent.Repository, error) {
	s.mustHaveDB()
	s.log.Debugf("Creating repository %s at %s", name, location)
	result, err := s.testRepoConnection(ctx, location, password)
	if err != nil {
		return nil, err
	}
	if !result.Success && !result.IsBorgRepo {
		// Create the repository if it does not exist
		status := s.borg.Init(ctx, util.ExpandPath(location), password, noPassword)
		if err := s.handleBorgStatus(ctx, nil, status, "initialize repository"); err != nil {
			return nil, err
		}
	} else if !result.Success {
		return nil, fmt.Errorf("could not connect to repository")
	}

	// Create a new repository entity
	return s.db.Repository.
		Create().
		SetName(name).
		SetURL(location).
		SetPassword(password).
		Save(ctx)
}

func (si *ServiceInternal) SyncCloudRepositories(ctx context.Context) ([]*ent.Repository, error) {
	return si.syncCloudRepositories(ctx)
}

func (s *Service) syncCloudRepositories(ctx context.Context) ([]*ent.Repository, error) {
	cloudRepos, err := s.cloudService.ListCloudRepositories(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debugf("Syncing %d cloud repositories", len(cloudRepos))

	var syncedRepos []*ent.Repository
	for _, cloudRepo := range cloudRepos {
		localRepo, err := s.syncSingleCloudRepository(ctx, cloudRepo)
		if err != nil {
			s.log.Errorf("Failed to sync cloud repository %s (%s): %v", cloudRepo.Name, cloudRepo.Id, err)
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
	s.mustHaveDB()

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
				SetLocation(getLocationEnum(cloudRepo.Location)).
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
			SetLocation(getLocationEnum(cloudRepo.Location)).
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

// RegenerateSSHKey regenerates SSH key for ArcoCloud repositories
func (s *Service) RegenerateSSHKey(ctx context.Context) error {
	return s.cloudService.AddOrReplaceSSHKey(ctx)
}

// CreateCloudRepository creates a new ArcoCloud repository
func (s *Service) CreateCloudRepository(ctx context.Context, name, password string, location arcov1.RepositoryLocation) (*ent.Repository, error) {
	// List existing cloud repositories to check if one already exists
	cloudRepos, err := s.cloudService.ListCloudRepositories(ctx)
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
		repo, err = s.cloudService.AddCloudRepository(ctx, name, location)
		if err != nil {
			return nil, err
		}

		status := s.borg.Init(ctx, repo.RepoUrl, password, false)
		if err := s.handleBorgStatus(ctx, nil, status, "initialize repository"); err != nil {
			return nil, err
		}
	}

	return database.WithTxData(ctx, s.db, func(tx *ent.Tx) (*ent.Repository, error) {
		// Create new local repository with cloud association
		entRepo, txErr := tx.Repository.
			Create().
			SetName(name).
			SetURL(repo.RepoUrl).
			SetPassword(password).
			Save(ctx)
		if txErr != nil {
			return nil, txErr
		}
		_, txErr = tx.CloudRepository.
			Create().
			SetCloudID(repo.Id).
			SetLocation(getLocationEnum(location)).
			SetRepository(entRepo).
			Save(ctx)
		if txErr != nil {
			return nil, txErr
		}
		return entRepo, nil
	})
}

func (s *Service) Update(ctx context.Context, repository *ent.Repository) (*ent.Repository, error) {
	s.mustHaveDB()
	s.log.Debugf("Updating repository %d", repository.ID)
	return s.db.Repository.
		UpdateOne(repository).
		SetName(repository.Name).
		Save(ctx)
}

// GetBackupProfilesThatHaveOnlyRepo returns all backup profiles that only have the given repository
func (s *Service) GetBackupProfilesThatHaveOnlyRepo(ctx context.Context, repoId int) ([]*ent.BackupProfile, error) {
	s.mustHaveDB()
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

// Remove deletes the repository with the given ID and all its backup profiles if they only have this repository
// It does not delete the physical repository on disk
func (s *Service) Remove(ctx context.Context, id int) error {
	s.mustHaveDB()
	s.log.Debugf("Removing repository %d", id)
	backupProfiles, err := s.GetBackupProfilesThatHaveOnlyRepo(ctx, id)
	if err != nil {
		return err
	}
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return err
	}

	if len(backupProfiles) > 0 {
		bpIds := make([]int, 0, len(backupProfiles))
		for _, bp := range backupProfiles {
			bpIds = append(bpIds, bp.ID)
		}

		_, err = tx.BackupProfile.Delete().
			Where(backupprofile.IDIn(bpIds...)).
			Exec(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}

	err = tx.Repository.
		DeleteOneID(id).
		Exec(ctx)
	if err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

// Delete deletes the repository with the given ID and all its backup profiles if they only have this repository
// It also deletes the physical repository on disk or cloud
func (s *Service) Delete(ctx context.Context, id int) error {
	s.mustHaveDB()
	s.log.Debugf("Deleting repository %d", id)
	repo, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Check if this is a cloud repository and route accordingly
	if s.isCloudRepository(ctx, repo.ID) {
		return s.cloudService.DeleteCloudRepository(ctx, repo.Edges.CloudRepository.CloudID)
	}

	// Handle local repository deletion
	if canMount, reason := s.state.CanRunDeleteJob(repo.ID); !canMount {
		return fmt.Errorf("cannot delete repository: %s", reason)
	}

	repoLock := s.state.GetRepoLock(repo.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	status := s.borg.DeleteRepository(ctx, repo.URL, repo.Password)
	if !status.IsCompletedWithSuccess() && !errors.Is(status.Error, borgtypes.ErrorRepositoryDoesNotExist) {
		// If the repository does not exist, we can ignore the error
		if status.HasBeenCanceled {
			return fmt.Errorf("repository deletion was cancelled")
		}
		return fmt.Errorf("failed to delete repository: %s", status.GetError())
	}
	return s.Remove(ctx, id)
}

func endOfMonth(t time.Time) time.Time {
	// Add one month to the current time
	nextMonth := t.AddDate(0, 1, 0)
	// Set the day to the first day of the next month and subtract one day
	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, nextMonth.Location()).Add(-time.Nanosecond)
}

func (s *Service) SaveIntegrityCheckSettings(ctx context.Context, repoId int, enabled bool) (*ent.Repository, error) {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Setting integrity check for repository %d to %t", repoId, enabled))
	if enabled {
		nextRun := endOfMonth(time.Now())
		return s.db.Repository.
			UpdateOneID(repoId).
			SetNillableNextIntegrityCheck(&nextRun).
			Save(ctx)
	}
	return s.db.Repository.
		UpdateOneID(repoId).
		SetNillableNextIntegrityCheck(nil).
		Save(ctx)
}

func (s *Service) GetState(ctx context.Context, id int) state.RepoState {
	return s.state.GetRepoState(id)
}

func (s *Service) BreakLock(ctx context.Context, id int) error {
	repo, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	status := s.borg.BreakLock(ctx, repo.URL, repo.Password)
	if err := s.handleBorgStatus(ctx, repo, status, "break lock"); err != nil {
		return err
	}
	s.state.SetRepoStatus(ctx, id, state.RepoStatusIdle)
	return nil
}

func (s *Service) GetConnectedRemoteHosts(ctx context.Context) ([]string, error) {
	s.mustHaveDB()
	repos, err := s.db.Repository.Query().
		Where(repository.URLContains("@")).
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

type TestRepoConnectionResult struct {
	Success         bool `json:"success"`
	NeedsPassword   bool `json:"needsPassword"`
	IsPasswordValid bool `json:"isPasswordValid"`
	IsBorgRepo      bool `json:"isBorgRepo"`
}

type testRepoConnectionResult struct {
	Success         bool `json:"success"`
	IsPasswordValid bool `json:"isPasswordValid"`
	IsBorgRepo      bool `json:"isBorgRepo"`
}

func (s *Service) testRepoConnection(ctx context.Context, path, password string) (testRepoConnectionResult, error) {
	s.log.Debugf("Testing repository connection to %s", path)
	result := testRepoConnectionResult{
		Success:         false,
		IsPasswordValid: false,
		IsBorgRepo:      false,
	}
	_, status := s.borg.Info(borg.NoErrorCtx(ctx), path, password)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			return result, fmt.Errorf("repository info retrieval was cancelled")
		}
		if errors.Is(status.Error, borgtypes.ErrorPassphraseWrong) {
			result.IsBorgRepo = true
			return result, nil
		}
		if errors.Is(status.Error, borgtypes.ErrorRepositoryDoesNotExist) {
			return result, nil
		}
		if errors.Is(status.Error, borgtypes.ErrorRepositoryInvalidRepository) {
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

func (s *Service) TestRepoConnection(ctx context.Context, path, password string) (TestRepoConnectionResult, error) {
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
	randPassword, err := uuid.NewRandom()
	if err != nil {
		return TestRepoConnectionResult{}, err
	}
	tr, err := s.testRepoConnection(ctx, path, randPassword.String())
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

// SetMountStates sets the mount states of all repositories and archives to the state
// This method is called during app startup and doesn't need to be exposed to frontend
func (si *ServiceInternal) SetMountStates(ctx context.Context) {
	repos, err := si.All(ctx)
	if err != nil {
		si.log.Error("Error getting all repositories: ", err)
		return
	}
	for _, repo := range repos {
		// Save the mount state for the repository
		path, err := getRepoMountPath(repo)
		if err != nil {
			return
		}
		mountState, err := getMountState(path)
		if err != nil {
			si.log.Error("Error getting mount state: ", err)
			continue
		}
		si.state.SetRepoMount(ctx, repo.ID, &mountState)

		// Save the mount states for all archives of the repository
		archives, err := repo.QueryArchives().All(ctx)
		if err != nil {
			si.log.Error("Error getting all archives: ", err)
			continue
		}
		var paths = make(map[int]string)
		for _, arch := range archives {
			archivePath, err := getArchiveMountPath(arch)
			if err != nil {
				si.log.Error("Error getting archive mount path: ", err)
				continue
			}
			paths[arch.ID] = archivePath
		}

		states, err := types.GetMountStates(paths)
		if err != nil {
			si.log.Error("Error getting mount states: ", err)
			continue
		}
		si.state.SetArchiveMounts(ctx, repo.ID, states)
	}
}

// Archive management methods

func (s *Service) RefreshArchives(ctx context.Context, repoId int) ([]*ent.Archive, error) {
	s.mustHaveDB()
	if s.state.GetRepoState(repoId).Status != state.RepoStatusIdle {
		return nil, fmt.Errorf("can not refresh archives: the repository is busy")
	}

	repoLock := s.state.GetRepoLock(repoId)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	return s.refreshArchivesWithoutLock(ctx, repoId)
}

func (si *ServiceInternal) RefreshArchivesWithoutLock(ctx context.Context, repoId int) ([]*ent.Archive, error) {
	return si.refreshArchivesWithoutLock(ctx, repoId)
}

// refreshArchivesWithoutLock fetches the archives from the borg repository and saves them to the database.
// It also deletes the archives that don't exist anymore.
// Precondition: the caller must have acquired the lock for the repository
func (s *Service) refreshArchivesWithoutLock(ctx context.Context, repoId int) ([]*ent.Archive, error) {
	repo, err := s.db.Repository.
		Query().
		Where(repository.ID(repoId)).
		WithBackupProfiles().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// Set the repo as performing operation
	s.state.SetRepoStatus(ctx, repoId, state.RepoStatusPerformingOperation)
	defer s.state.SetRepoStatus(ctx, repoId, state.RepoStatusIdle)

	listResponse, status := s.borg.List(ctx, repo.URL, repo.Password)
	if err := s.handleBorgStatus(ctx, repo, status, "get archives"); err != nil {
		return nil, err
	}

	// Get all the borg ids
	borgIds := make([]string, len(listResponse.Archives))
	for i, arch := range listResponse.Archives {
		borgIds[i] = arch.ID
	}

	// Delete the archives that don't exist anymore
	cntDeletedArchives, err := s.db.Archive.
		Delete().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.BorgIDNotIn(borgIds...),
			)).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if cntDeletedArchives > 0 {
		s.log.Info(fmt.Sprintf("Deleted %d archives", cntDeletedArchives))
	}

	// Check which archives are already saved
	archives, err := s.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		All(ctx)
	if err != nil {
		return nil, err
	}
	savedBorgIds := make([]string, len(archives))
	for i, arch := range archives {
		savedBorgIds[i] = arch.BorgID
	}

	// Save the new archives
	cntNewArchives := 0
	for _, arch := range listResponse.Archives {
		if !slices.Contains(savedBorgIds, arch.ID) {
			createdAt := time.Time(arch.Start)
			duration := time.Time(arch.End).Sub(createdAt)
			createQuery := s.db.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(createdAt).
				SetDuration(duration.Seconds()).
				SetRepositoryID(repoId)

			// Find the backup profile that has the same prefix as the archive
			for _, backupProfile := range repo.Edges.BackupProfiles {
				if strings.HasPrefix(arch.Name, backupProfile.Prefix) {
					createQuery = createQuery.SetBackupProfileID(backupProfile.ID)
					break
				}
			}

			newArchive, err := createQuery.Save(ctx)
			if err != nil {
				return nil, err
			}
			archives = append(archives, newArchive)
			cntNewArchives++
		}
	}
	if cntNewArchives > 0 {
		s.log.Info(fmt.Sprintf("Saved %d new archives", cntNewArchives))
	}

	if cntDeletedArchives > 0 || cntNewArchives > 0 {
		defer s.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(repoId))
	}

	return archives, nil
}

func (s *Service) DeleteArchive(ctx context.Context, id int) error {
	s.mustHaveDB()
	arch, err := s.db.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(ctx)
	if err != nil {
		return err
	}
	if canRun, reason := s.state.CanRunDeleteJob(arch.Edges.Repository.ID); !canRun {
		return fmt.Errorf("can not delete archive: %s", reason)
	}

	repoLock := s.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as locked
	s.state.SetRepoStatus(ctx, arch.Edges.Repository.ID, state.RepoStatusPerformingOperation)
	defer s.state.SetRepoStatus(ctx, arch.Edges.Repository.ID, state.RepoStatusIdle)

	status := s.borg.DeleteArchive(ctx, arch.Edges.Repository.URL, arch.Name, arch.Edges.Repository.Password)
	if err := s.handleBorgStatus(ctx, arch.Edges.Repository, status, "delete archive"); err != nil {
		return err
	}
	err = s.db.Archive.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventArchivesChangedString(arch.Edges.Repository.ID))
	return nil
}

type BackupProfileFilter struct {
	Id              int    `json:"id,omitempty"`
	Name            string `json:"name"`
	IsAllFilter     bool   `json:"isAllFilter"`
	IsUnknownFilter bool   `json:"isUnknownFilter"`
}

type PaginatedArchivesRequest struct {
	// Required
	RepositoryId int `json:"repositoryId"`
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	// Optional
	BackupProfileFilter *BackupProfileFilter `json:"backupProfileFilter,omitempty"`
	Search              string               `json:"search,omitempty"`
	StartDate           time.Time            `json:"startDate,omitempty"`
	EndDate             time.Time            `json:"endDate,omitempty"`
}

type PaginatedArchivesResponse struct {
	Archives []*ent.Archive `json:"archives"`
	Total    int            `json:"total"`
}

func (s *Service) GetPaginatedArchives(ctx context.Context, req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error) {
	s.mustHaveDB()
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

	// If a search term is specified, filter by it
	if req.Search != "" {
		archivePredicates = append(archivePredicates, archive.NameContains(req.Search))
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

	return &PaginatedArchivesResponse{
		Archives: archives,
		Total:    total,
	}, nil
}

type PruningDates struct {
	Dates []PruningDate `json:"dates"`
}

type PruningDate struct {
	ArchiveId int       `json:"archiveId"`
	NextRun   time.Time `json:"nextRun"`
}

func (s *Service) GetPruningDates(ctx context.Context, archiveIds []int) (response PruningDates, err error) {
	s.mustHaveDB()
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
		return
	}
	for _, arch := range archives {
		if arch.Edges.BackupProfile.Edges.PruningRule != nil {
			response.Dates = append(response.Dates, PruningDate{
				ArchiveId: arch.ID,
				NextRun:   arch.Edges.BackupProfile.Edges.PruningRule.NextRun,
			})
		}
	}
	return
}

func (s *Service) GetLastArchiveByRepoId(ctx context.Context, repoId int) (*ent.Archive, error) {
	s.mustHaveDB()
	first, err := s.db.Archive.
		Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(repoId)),
		)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)
	if err != nil && ent.IsNotFound(err) {
		return &ent.Archive{}, nil
	}
	return first, err
}

func (s *Service) GetArchive(ctx context.Context, id int) (*ent.Archive, error) {
	return s.db.Archive.
		Query().
		WithRepository().
		WithBackupProfile().
		Where(archive.ID(id)).
		Only(ctx)
}

func (s *Service) GetLastArchiveByBackupId(ctx context.Context, backupId types.BackupId) (*ent.Archive, error) {
	s.mustHaveDB()

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

// ValidateRepoName validates the name of a repository.
// The rules are enforced by the database.
func (s *Service) ValidateRepoName(ctx context.Context, name string) (string, error) {
	s.mustHaveDB()
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < schema.ValRepositoryMinNameLength {
		return fmt.Sprintf("Name must be at least %d characters long", schema.ValRepositoryMinNameLength), nil
	}
	if len(name) > schema.ValRepositoryMaxNameLength {
		return fmt.Sprintf("Name can not be longer than %d characters", schema.ValRepositoryMaxNameLength), nil
	}
	matched := schema.ValRepositoryNamePattern.MatchString(name)
	if !matched {
		return "Name can only contain letters, numbers, hyphens, and underscores", nil
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

// ValidateRepoPath validates the path of a repository.
func (s *Service) ValidateRepoPath(ctx context.Context, path string, isLocal bool) (string, error) {
	s.mustHaveDB()
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

// ValidateArchiveName validates the name of an archive.
// The rules are not enforced by the database because we import them from borg repositories which have different rules.
func (s *Service) ValidateArchiveName(ctx context.Context, archiveId int, prefix, name string) (string, error) {
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

	arch, err := s.GetArchive(ctx, archiveId)
	if err != nil {
		return "", err
	}
	assert.NotNil(arch.Edges.Repository, "archive must have a repository")

	// Check if the new name starts with the backup profile prefix
	if arch.Edges.BackupProfile != nil {
		if !strings.HasPrefix(prefix, arch.Edges.BackupProfile.Prefix) {
			return "The new name must start with the backup profile prefix", nil
		}
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
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Archive name must be unique", nil
	}

	return "", nil
}

// RenameArchive requires access to validation client
func (s *Service) RenameArchive(ctx context.Context, id int, prefix, name string) error {
	s.mustHaveDB()

	s.log.Debugf("Renaming archive %d to %s", id, name)
	validationError, err := s.ValidateArchiveName(ctx, id, prefix, name)
	if err != nil {
		return err
	}
	if validationError != "" {
		return fmt.Errorf("can not rename archive: %s", validationError)
	}

	newName := prefix + name

	arch, err := s.GetArchive(ctx, id)
	if err != nil {
		return err
	}

	if s.state.GetRepoState(arch.Edges.Repository.ID).Status != state.RepoStatusIdle {
		return fmt.Errorf("can not rename archive: the repository is busy")
	}

	repoLock := s.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	status := s.borg.Rename(ctx, arch.Edges.Repository.URL, arch.Name, arch.Edges.Repository.Password, newName)
	if err := s.handleBorgStatus(ctx, arch.Edges.Repository, status, "rename archive"); err != nil {
		return err
	}

	return s.db.Archive.
		UpdateOneID(id).
		SetName(newName).
		Exec(ctx)
}

// Mount management methods

func (s *Service) MountRepository(ctx context.Context, repoId int) (mountState types.MountState, err error) {
	s.mustHaveDB()
	repo, err := s.Get(ctx, repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	err = ensurePathExists(path)
	if err != nil {
		return
	}

	status := s.borg.MountRepository(ctx, repo.URL, repo.Password, path)
	if err = s.handleBorgStatus(ctx, repo, status, "mount repository"); err != nil {
		return
	}

	// Update the mount mountState
	mountState, err = getMountState(path)
	if err != nil {
		return
	}
	s.state.SetRepoMount(ctx, repoId, &mountState)

	// Open the file manager and forget about it
	go s.openFileManager(path)
	return
}

func (s *Service) MountArchive(ctx context.Context, archiveId int) (state types.MountState, err error) {
	s.mustHaveDB()
	arch, err := s.GetArchive(ctx, archiveId)
	if err != nil {
		return
	}

	if canMount, reason := s.state.CanMountRepo(arch.Edges.Repository.ID); !canMount {
		err = fmt.Errorf("can not mount arch: %s", reason)
		return
	}
	repoLock := s.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	path, err := getArchiveMountPath(arch)
	if err != nil {
		return
	}

	err = ensurePathExists(path)
	if err != nil {
		return
	}

	// Check current mount state
	state, err = getMountState(path)
	if err != nil {
		return
	}
	if !state.IsMounted {
		// If not mounted, mount it
		status := s.borg.MountArchive(ctx, arch.Edges.Repository.URL, arch.Name, arch.Edges.Repository.Password, path)
		if err = s.handleBorgStatus(ctx, arch.Edges.Repository, status, "mount archive"); err != nil {
			return
		}

		// Update the mount state
		state, err = getMountState(path)
		if err != nil {
			return
		}
		s.state.SetArchiveMount(ctx, arch.Edges.Repository.ID, archiveId, &state)
	}

	// Open the file manager and forget about it
	go s.openFileManager(path)
	return
}

func (s *Service) UnmountAllForRepos(ctx context.Context, repoIds []int) error {
	s.mustHaveDB()
	var unmountErrors []error
	for _, repoId := range repoIds {
		mount := s.GetRepoMountState(repoId)
		if mount.IsMounted {
			if _, err := s.UnmountRepository(ctx, repoId); err != nil {
				unmountErrors = append(unmountErrors, fmt.Errorf("error unmounting repository %d: %w", repoId, err))
			}
		}
		if states, err := s.GetArchiveMountStates(ctx, repoId); err != nil {
			unmountErrors = append(unmountErrors, fmt.Errorf("error getting archive mount states for repository %d: %w", repoId, err))
		} else {
			for archiveId, mountState := range states {
				if mountState.IsMounted {
					if _, err = s.UnmountArchive(ctx, archiveId); err != nil {
						unmountErrors = append(unmountErrors, fmt.Errorf("error unmounting archive %d: %w", archiveId, err))
					}
				}
			}
		}
	}
	if len(unmountErrors) > 0 {
		return fmt.Errorf("unmount errors: %v", unmountErrors)
	}
	return nil
}

func (s *Service) UnmountRepository(ctx context.Context, repoId int) (state types.MountState, err error) {
	s.mustHaveDB()
	repo, err := s.Get(ctx, repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	status := s.borg.Umount(ctx, path)
	if err = s.handleBorgStatus(ctx, repo, status, "unmount repository"); err != nil {
		return
	}

	// Update the mount state
	mountState, err := getMountState(path)
	if err != nil {
		return
	}
	s.state.SetRepoMount(ctx, repoId, &mountState)
	return
}

func (s *Service) UnmountArchive(ctx context.Context, archiveId int) (state types.MountState, err error) {
	s.mustHaveDB()
	arch, err := s.GetArchive(ctx, archiveId)
	if err != nil {
		return
	}

	path, err := getArchiveMountPath(arch)
	if err != nil {
		return
	}

	status := s.borg.Umount(ctx, path)
	if err = s.handleBorgStatus(ctx, arch.Edges.Repository, status, "unmount archive"); err != nil {
		return
	}

	// Update the mount state
	mountState, err := getMountState(path)
	if err != nil {
		return
	}
	s.state.SetArchiveMount(ctx, arch.Edges.Repository.ID, archiveId, &mountState)
	return
}

func (s *Service) GetRepoMountState(repoId int) types.MountState {
	return s.state.GetRepoMount(repoId)
}

func (s *Service) GetArchiveMountStates(ctx context.Context, repoId int) (states map[int]types.MountState, err error) {
	s.mustHaveDB()
	repo, err := s.Get(ctx, repoId)
	if err != nil {
		return
	}
	return s.state.GetArchiveMounts(repo.ID), nil
}

func (s *Service) openFileManager(path string) {
	openCmd, err := types.GetOpenFileManagerCmd()
	if err != nil {
		s.log.Error("Error getting open file manager command: ", err)
		return
	}
	cmd := exec.Command(openCmd, path)
	err = cmd.Run()
	if err != nil {
		s.log.Error("Error opening file manager: ", err)
	}
}

// Utility functions for mount paths

func getMountPath(name string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	mountPath, err := types.GetMountPath()
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
	// Check if the directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		//Create the directory
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func getMountState(path string) (state types.MountState, err error) {
	states, err := types.GetMountStates(map[int]string{0: path})
	if err != nil {
		return
	}
	if len(states) == 0 {
		return
	}
	return *states[0], nil
}

// handleBorgStatus handles common Borg status errors and updates repository error state accordingly
func (s *Service) handleBorgStatus(ctx context.Context, repo *ent.Repository, status *borgtypes.Status, operationName string) error {
	if status == nil {
		return nil
	}

	// Handle special case where repo is nil (initialization errors)
	if repo == nil {
		if status.HasError() {
			s.log.Errorf("Failed to %s during initialization: %s", operationName, status.GetError())
			return fmt.Errorf("failed to %s: %s", operationName, status.GetError())
		}
		return nil
	}

	repoId := repo.ID
	if status.HasError() {
		if status.HasBeenCanceled {
			s.log.Warnf("Operation %s was cancelled for repository %d", operationName, repoId)
			return fmt.Errorf("%s was cancelled", operationName)
		}

		// Check for specific error types and update state accordingly
		if errors.Is(status.Error, borgtypes.ErrorConnectionClosedWithHint) {
			s.log.Errorf("SSH key authentication failed for repository %d during %s: %s", repoId, operationName, status.GetError())

			// Determine error action based on repository type
			var errorAction state.RepoErrorAction
			if s.isCloudRepository(ctx, repo.ID) {
				errorAction = state.RepoErrorActionRegenerateSSH
			} else {
				errorAction = state.RepoErrorActionNone
			}

			s.state.SetRepoErrorState(ctx, repoId, state.RepoErrorTypeSSHKey, "SSH key authentication failed", errorAction)
			return fmt.Errorf("SSH key authentication failed: %s", status.GetError())
		}
		if errors.Is(status.Error, borgtypes.ErrorPassphraseWrong) {
			s.log.Errorf("Incorrect passphrase for repository %d during %s: %s", repoId, operationName, status.GetError())
			s.state.SetRepoErrorState(ctx, repoId, state.RepoErrorTypePassphrase, "Incorrect passphrase", state.RepoErrorActionNone)
			return fmt.Errorf("incorrect passphrase: %s", status.GetError())
		}

		s.log.Errorf("Failed to %s for repository %d: %s", operationName, repoId, status.GetError())
		return fmt.Errorf("failed to %s: %s", operationName, status.GetError())
	} else if status.HasWarning() {
		// Set warning state and log warning
		s.log.Warnf("Operation %s completed with warning for repository %d: %s", operationName, repoId, status.GetWarning())
		s.state.SetRepoWarningState(ctx, repoId, status.GetWarning())
		// Clear error state since operation completed successfully with warning
		s.state.ClearRepoErrorState(ctx, repoId)
	} else if status.IsCompletedWithSuccess() {
		// Clear any previous error and warning states on success
		s.state.ClearRepoErrorState(ctx, repoId)
		s.state.ClearRepoWarningState(ctx, repoId)
	} else {
		assert.Failf("Unexpected status %s", status.GetError())
	}
	return nil
}
