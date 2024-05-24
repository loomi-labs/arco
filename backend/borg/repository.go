package borg

import (
	"arco/backend/ent"
	"arco/backend/ent/repository"
	"context"
	"fmt"
	"os/exec"
)

func (c *Client) GetRepository(id int) (*ent.Repository, error) {
	return c.client.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(context.Background())
}

func (c *Client) GetRepositories() ([]*ent.Repository, error) {
	return c.client.Repository.Query().All(context.Background())
}

func (c *Client) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(c.binaryPath, "info", "--json", url)
	cmd.Env = createEnv(password)
	c.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	// Check if we can connect to the repository
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

	// Create a new repository entity
	return c.client.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(context.Background())
}
