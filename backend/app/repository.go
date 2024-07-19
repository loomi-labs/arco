package app

import (
	"arco/backend/app/borg"
	"arco/backend/ent"
	"arco/backend/ent/repository"
	"os/exec"
)

func (r *RepositoryClient) Get(id int) (*ent.Repository, error) {
	return r.db.Repository.
		Query().
		WithBackupProfiles().
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
	cmd.Env = borg.Env{}.WithPassword(password).AsList()

	// Check if we can connect to the repository
	if err := r.borg.Info(url, password); err != nil {
		return nil, err
	}

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
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

func (r *RepositoryClient) Create(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(r.config.BorgPath, "init", "--encryption=repokey-blake2", url)
	cmd.Env = borg.Env{}.WithPassword(password).AsList()

	if err := r.borg.Init(url, password); err != nil {
		return nil, err
	}

	// Create a new repository entity
	return r.db.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupProfileIDs(backupProfileId).
		Save(r.ctx)
}
