package app

import (
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/util"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (r *RepositoryClient) Get(repoId int) (*ent.Repository, error) {
	return r.db.Repository.
		Query().
		Where(repository.ID(repoId)).
		Only(r.ctx)
}

func (r *RepositoryClient) GetByBackupId(bId types.BackupId) (*ent.Repository, error) {
	return r.db.Repository.
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
		Only(r.ctx)
}

func (r *RepositoryClient) All() ([]*ent.Repository, error) {
	return r.db.Repository.
		Query().
		Order(func(s *sql.Selector) {
			// Order by name, case-insensitive
			s.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", repository.FieldName)))
		}).
		All(r.ctx)
}

func (r *RepositoryClient) GetNbrOfArchives(repoId int) (int, error) {
	return r.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		Count(r.ctx)
}

func (r *RepositoryClient) GetLastBackupErrorMsg(repoId int) (string, error) {
	// Get the last notification for the backup profile and repository
	// that is a failed backup run or failed pruning run
	lastNotification, err := r.db.Notification.
		Query().
		Where(notification.And(
			notification.HasRepositoryWith(repository.ID(repoId)),
		)).
		Order(ent.Desc(notification.FieldCreatedAt)).
		First(r.ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}
	if lastNotification != nil {
		// Check if there is a new archive since the last notification
		// If there is, we don't show the error message
		exist, err := r.db.Archive.
			Query().
			Where(archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.CreatedAtGT(lastNotification.CreatedAt),
			)).Exist(r.ctx)
		if err != nil && !ent.IsNotFound(err) {
			return "", err
		}
		if !exist {
			return lastNotification.Message, nil
		}
	}
	return "", nil
}

func (r *RepositoryClient) GetLocked() ([]*ent.Repository, error) {
	all, err := r.All()
	if err != nil {
		return nil, err
	}

	locked := make([]*ent.Repository, 0)
	for _, repo := range all {
		if r.state.GetRepoState(repo.ID).Status == state.RepoStatusLocked {
			locked = append(locked, repo)
		}
	}
	return locked, nil
}

func (r *RepositoryClient) GetWithActiveMounts() ([]*ent.Repository, error) {
	all, err := r.All()
	if err != nil {
		return nil, err
	}

	active := make([]*ent.Repository, 0)
	for _, repo := range all {
		if r.state.GetRepoState(repo.ID).Status == state.RepoStatusMounted {
			active = append(active, repo)
		}
	}
	return active, nil
}

func (r *RepositoryClient) Create(name, location, password string, noPassword bool) (*ent.Repository, error) {
	r.log.Debugf("Creating repository %s at %s", name, location)
	result, err := r.testRepoConnection(location, password)
	if err != nil {
		return nil, err
	}
	if !result.Success && !result.IsBorgRepo {
		// Create the repository if it does not exist
		if err := r.borg.Init(r.ctx, util.ExpandPath(location), password, noPassword); err != nil {
			return nil, err
		}
	} else if !result.Success {
		return nil, fmt.Errorf("could not connect to repository")
	}

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetLocation(location).
		SetPassword(password).
		Save(r.ctx)
}

func (r *RepositoryClient) Update(repository *ent.Repository) (*ent.Repository, error) {
	r.log.Debugf("Updating repository %d", repository.ID)
	return r.db.Repository.
		UpdateOne(repository).
		SetName(repository.Name).
		Save(r.ctx)
}

// GetBackupProfilesThatHaveOnlyRepo returns all backup profiles that only have the given repository
func (r *RepositoryClient) GetBackupProfilesThatHaveOnlyRepo(repoId int) ([]*ent.BackupProfile, error) {
	backupProfiles, err := r.db.BackupProfile.
		Query().
		Where(backupprofile.And(
			backupprofile.HasRepositoriesWith(repository.ID(repoId)),
		)).
		WithRepositories().
		All(r.ctx)
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
func (r *RepositoryClient) Remove(id int) error {
	r.log.Debugf("Removing repository %d", id)
	backupProfiles, err := r.GetBackupProfilesThatHaveOnlyRepo(id)
	if err != nil {
		return err
	}
	tx, err := r.db.Tx(r.ctx)
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
			Exec(r.ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}

	err = tx.Repository.
		DeleteOneID(id).
		Exec(r.ctx)
	if err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

// Delete deletes the repository with the given ID and all its backup profiles if they only have this repository
// It also deletes the physical repository on disk
func (r *RepositoryClient) Delete(id int) error {
	r.log.Debugf("Deleting repository %d", id)
	repo, err := r.Get(id)
	if err != nil {
		return err
	}

	if canMount, reason := r.state.CanRunDeleteJob(repo.ID); !canMount {
		return fmt.Errorf("cannot delete repository: %s", reason)
	}

	repoLock := r.state.GetRepoLock(repo.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	err = r.borg.DeleteRepository(r.ctx, repo.Location, repo.Password)
	if err != nil && !errors.Is(err, borg.ErrorRepositoryDoesNotExist) {
		// If the repository does not exist, we can ignore the error
		return err
	}
	return r.Remove(id)
}

func endOfMonth(t time.Time) time.Time {
	// Add one month to the current time
	nextMonth := t.AddDate(0, 1, 0)
	// Set the day to the first day of the next month and subtract one day
	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, nextMonth.Location()).Add(-time.Nanosecond)
}

func (r *RepositoryClient) SaveIntegrityCheckSettings(repoId int, enabled bool) (*ent.Repository, error) {
	r.log.Debug(fmt.Sprintf("Setting integrity check for repository %d to %t", repoId, enabled))
	if enabled {
		nextRun := endOfMonth(time.Now())
		return r.db.Repository.
			UpdateOneID(repoId).
			SetNillableNextIntegrityCheck(&nextRun).
			Save(r.ctx)
	}
	return r.db.Repository.
		UpdateOneID(repoId).
		SetNillableNextIntegrityCheck(nil).
		Save(r.ctx)
}

func (r *RepositoryClient) GetState(id int) state.RepoState {
	return r.state.GetRepoState(id)
}

func (r *RepositoryClient) BreakLock(id int) error {
	repo, err := r.Get(id)
	if err != nil {
		return err
	}

	err = r.borg.BreakLock(r.ctx, repo.Location, repo.Password)
	if err != nil {
		return err
	}
	r.state.SetRepoStatus(r.ctx, id, state.RepoStatusIdle)
	return nil
}

func (r *RepositoryClient) GetConnectedRemoteHosts() ([]string, error) {
	repos, err := r.db.Repository.Query().
		Where(repository.LocationContains("@")).
		All(r.ctx)
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

func (r *RepositoryClient) IsBorgRepository(path string) bool {
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

func (r *RepositoryClient) testRepoConnection(path, password string) (testRepoConnectionResult, error) {
	r.log.Debugf("Testing repository connection to %s", path)
	result := testRepoConnectionResult{
		Success:         false,
		IsPasswordValid: false,
		IsBorgRepo:      false,
	}
	_, err := r.borg.Info(borg.NoErrorCtx(r.ctx), path, password)
	if err != nil {
		if errors.Is(err, borg.ErrorPassphraseWrong) {
			result.IsBorgRepo = true
			return result, nil
		}
		if errors.Is(err, borg.ErrorRepositoryDoesNotExist) {
			return result, nil
		}
		return result, err
	}
	result.Success = true
	result.IsPasswordValid = true
	result.IsBorgRepo = true
	return result, nil
}

func (r *RepositoryClient) TestRepoConnection(path, password string) (TestRepoConnectionResult, error) {
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
	tr, err := r.testRepoConnection(path, randPassword.String())
	if err != nil || tr.Success || !tr.IsBorgRepo {
		return toTestRepoConnectionResult(tr, err, false)
	}

	// If we are here it means we need a password
	if password != "" {
		tr, err = r.testRepoConnection(path, password)
		return toTestRepoConnectionResult(tr, err, true)
	}
	return toTestRepoConnectionResult(tr, nil, true)
}
