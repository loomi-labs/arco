package borg

import (
	"context"
	"fmt"
	"os/exec"
	"timebender/backend/ent"
	"timebender/backend/ent/repository"
)

func (b *Borg) GetRepository(id int) (*ent.Repository, error) {
	return b.client.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(context.Background())
}

func (b *Borg) GetRepositories() ([]*ent.Repository, error) {
	return b.client.Repository.Query().All(context.Background())
}

func (b *Borg) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(b.binaryPath, "info", "--json", url)
	cmd.Env = createEnv(password)
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	// Check if we can connect to the repository
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

	// Create a new repository entity
	return b.client.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(context.Background())
}