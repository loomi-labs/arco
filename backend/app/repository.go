package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/archive"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/failedbackuprun"
	"arco/backend/ent/repository"
	"arco/backend/util"
	"strings"
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
		WithFailedBackupRuns(func(query *ent.FailedBackupRunQuery) {
			query.Where(failedbackuprun.And(
				failedbackuprun.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
				failedbackuprun.HasRepositoryWith(repository.ID(bId.RepositoryId)),
			))
		}).
		Where(repository.And(
			repository.ID(bId.RepositoryId),
			repository.HasBackupProfilesWith(backupprofile.ID(bId.BackupProfileId)),
		)).
		Only(r.ctx)
}

func (r *RepositoryClient) All() ([]*ent.Repository, error) {
	return r.db.Repository.
		Query().
		All(r.ctx)
}

func (r *RepositoryClient) GetNbrOfArchives(repoId int) (int, error) {
	return r.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		Count(r.ctx)
}

// TODO: remove this function or refactor it
func (r *RepositoryClient) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	// Check if we can connect to the repository
	if _, err := r.borg.Info(url, password); err != nil {
		return nil, err
	}

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetLocation(url).
		SetPassword(password).
		AddBackupProfileIDs(backupProfileId).
		Save(r.ctx)
}

func (r *RepositoryClient) AddBackupProfile(id int, backupProfileId int) (*ent.Repository, error) {
	return r.db.Repository.
		UpdateOneID(id).
		AddBackupProfileIDs(backupProfileId).
		Save(r.ctx)
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
	r.state.SetRepoStatus(id, state.RepoStatusIdle)
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
		user, hostAndPort, found := strings.Cut(repo.Location, "@")
		if !found {
			continue
		}
		host, remainder, found2 := strings.Cut(hostAndPort, ":")
		if !found2 {
			continue
		}

		port, _, found3 := strings.Cut(remainder, "/")
		if !found3 {
			port = ""
		}

		// Add host to map
		hosts[user+"@"+host+":"+port] = struct{}{}
	}

	// Convert map to slice
	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result, nil
}
