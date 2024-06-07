package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ent/repository"
	"fmt"
	"os/exec"
)

func (b *BorgClient) GetRepository(id int) (*ent.Repository, error) {
	return b.db.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(b.ctx)
}

func (b *BorgClient) GetRepositories() ([]*ent.Repository, error) {
	return b.db.Repository.Query().All(b.ctx)
}

func (b *BorgClient) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(b.binaryPath, "info", "--json", url)
	cmd.Env = util.BorgEnv{}.WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)

	// Create a new repository entity
	return b.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(b.ctx)
}

func (b *BorgClient) InitNewRepo(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(b.binaryPath, "init", "--encryption=repokey-blake2", url)
	cmd.Env = util.BorgEnv{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)

	// Create a new repository entity
	return b.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(b.ctx)
}
