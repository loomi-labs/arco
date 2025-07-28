package repository

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	types2 "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/util"
	"go.uber.org/zap"
	"net/url"
	"os"
	"path/filepath"
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
	cloudService *CloudRepositoryService
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new repository service
func NewService(log *zap.SugaredLogger, db *ent.Client, state *state.State, borg borg.Borg, config *types.Config, eventEmitter types.EventEmitter, cloudService *CloudRepositoryService) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:          log,
			db:           db,
			state:        state,
			borg:         borg,
			config:       config,
			eventEmitter: eventEmitter,
			cloudService: cloudService,
		},
	}
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("RepositoryService: database client is nil")
	}
}

// isCloudRepository checks if a repository is an ArcoCloud repository
func (s *Service) isCloudRepository(repo *ent.Repository) bool {
	return repo.ArcoCloudID != nil && *repo.ArcoCloudID != ""
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
	if _, err := s.cloudService.ListCloudRepositories(ctx); err != nil {
		s.log.Warnf("Failed to sync cloud repositories: %v", err)
	}
	
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
		if !status.IsCompletedWithSuccess() {
			if status.HasBeenCanceled {
				return nil, fmt.Errorf("repository initialization was cancelled")
			}
			return nil, fmt.Errorf("failed to initialize repository: %s", status.GetError())
		}
		if status.HasWarning() {
			// TODO(log-warning): log warning to user
			s.log.Warnf("Repository initialization completed with warning: %s", status.GetWarning())
		}
	} else if !result.Success {
		return nil, fmt.Errorf("could not connect to repository")
	}

	// Create a new repository entity
	return s.db.Repository.
		Create().
		SetName(name).
		SetLocation(location).
		SetPassword(password).
		Save(ctx)
}

// CreateCloudRepository creates a new ArcoCloud repository
func (s *Service) CreateCloudRepository(ctx context.Context, name, password string, location arcov1.RepositoryLocation) (*ent.Repository, error) {
	return s.cloudService.AddCloudRepository(ctx, name, password, location)
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
	if s.isCloudRepository(repo) {
		return s.cloudService.DeleteCloudRepository(ctx, *repo.ArcoCloudID)
	}

	// Handle local repository deletion
	if canMount, reason := s.state.CanRunDeleteJob(repo.ID); !canMount {
		return fmt.Errorf("cannot delete repository: %s", reason)
	}

	repoLock := s.state.GetRepoLock(repo.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	status := s.borg.DeleteRepository(ctx, repo.Location, repo.Password)
	if !status.IsCompletedWithSuccess() && !errors.Is(status.Error, types2.ErrorRepositoryDoesNotExist) {
		// If the repository does not exist, we can ignore the error
		if status.HasBeenCanceled {
			return fmt.Errorf("repository deletion was cancelled")
		}
		return fmt.Errorf("failed to delete repository: %s", status.GetError())
	}
	if status.HasWarning() {
		// TODO(log-warning): log warning to user
		s.log.Warnf("Repository deletion completed with warning: %s", status.GetWarning())
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

	status := s.borg.BreakLock(ctx, repo.Location, repo.Password)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			return fmt.Errorf("lock breaking was cancelled")
		}
		return fmt.Errorf("failed to break lock: %s", status.GetError())
	}
	if status.HasWarning() {
		// TODO(log-warning): log warning to user
		s.log.Warnf("Lock breaking completed with warning: %s", status.GetWarning())
	}
	s.state.SetRepoStatus(ctx, id, state.RepoStatusIdle)
	return nil
}

func (s *Service) GetConnectedRemoteHosts(ctx context.Context) ([]string, error) {
	s.mustHaveDB()
	repos, err := s.db.Repository.Query().
		Where(repository.LocationContains("@")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// user@host:~/path/to/repo -> user@host:port
	// ssh://user@host:port/./path/to/repo -> user@host:port
	hosts := make(map[string]struct{})
	for _, repo := range repos {
		// Extract user, host and port from location
		parsedURL, err := url.Parse(repo.Location)
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
		if errors.Is(status.Error, types2.ErrorPassphraseWrong) {
			result.IsBorgRepo = true
			return result, nil
		}
		if errors.Is(status.Error, types2.ErrorRepositoryDoesNotExist) {
			return result, nil
		}
		if errors.Is(status.Error, types2.ErrorRepositoryInvalidRepository) {
			return result, nil
		}
		return result, fmt.Errorf("info command failed: %s", status.GetError())
	}
	if status.HasWarning() {
		// TODO(log-warning): log warning to user
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

// setMountStates sets the mount states of all repositories and archives to the state
// This method is called during app startup and doesn't need to be exposed to frontend
// TODO: Implement proper mount state management - this is a temporary stub
func (si *ServiceInternal) setMountStates(ctx context.Context) {
	si.log.Debug("Setting mount states - functionality to be implemented")
	// For now, this is a stub to maintain compilation while the rest of the transformation is completed
	// The mount functionality should be properly integrated in a follow-up task
}