package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ent/repository"
	"context"
	"fmt"
	"os/exec"
)

func (b *BorgClient) GetRepository(id int) (*ent.Repository, error) {
	return b.db.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(context.Background())
}

func (b *BorgClient) GetRepositories() ([]*ent.Repository, error) {
	return b.db.Repository.Query().All(context.Background())
}

func (b *BorgClient) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(b.binaryPath, "info", "--json", url)
	cmd.Env = util.CreateEnv(password)
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
		Save(context.Background())
}
