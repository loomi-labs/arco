package app

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/util"
	"net/url"
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
		Order(ent.Desc(repository.FieldName)).
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
	if err := r.borg.Init(util.ExpandPath(location), password, noPassword); err != nil {
		return nil, err
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

func (r *RepositoryClient) Delete(id int) error {
	r.log.Debugf("Deleting repository %d", id)
	return r.db.Repository.
		DeleteOneID(id).
		Exec(r.ctx)
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
