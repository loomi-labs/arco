package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ent/repository"
	"fmt"
	"os/exec"
)

func (r *RepositoryClient) Get(id int) (*ent.Repository, error) {
	return r.db.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(r.ctx)
}

func (r *RepositoryClient) All() ([]*ent.Repository, error) {
	return r.db.Repository.Query().All(r.ctx)
}

// TODO: remove this function or refactor it
func (r *RepositoryClient) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(r.config.BorgPath, "info", "--json", url)
	cmd.Env = util.BorgEnv{}.WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := r.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, r.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	r.log.LogCmdEnd(cmd.String(), startTime)

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(r.ctx)
}

func (r *RepositoryClient) Create(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(r.config.BorgPath, "init", "--encryption=repokey-blake2", url)
	cmd.Env = util.BorgEnv{}.WithPassword(password).AsList()

	startTime := r.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, r.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	r.log.LogCmdEnd(cmd.String(), startTime)

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(r.ctx)
}
