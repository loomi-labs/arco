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
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	// Check if we can connect to the repository
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

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
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	out, err := cmd.CombinedOutput()
	b.log.Info(fmt.Sprintf("Output: %s", out))
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

	// Create a new repository entity
	return b.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(b.ctx)
}
