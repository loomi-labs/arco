package client

import (
	"arco/backend/borg/types"
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/repository"
	"context"
	"fmt"
	"os"
)

func (b *BorgClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.db.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetDirectories([]string{}).
		SetHasPeriodicBackups(true).
		//SetPeriodicBackupTime(time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)).
		Save(context.Background())
}

func (b *BorgClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BorgClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithRepositories().
		Where(backupprofile.ID(id)).Only(context.Background())
}

func (b *BorgClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.db.BackupProfile.Query().All(context.Background())
}

func (b *BorgClient) SaveBackupProfile(backup ent.BackupProfile) error {
	_, err := b.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetDirectories(backup.Directories).
		SetHasPeriodicBackups(backup.HasPeriodicBackups).
		//SetPeriodicBackupTime(backup.PeriodicBackupTime).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(context.Background())
	return err
}

func (b *BorgClient) RunBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repositoryId),
			repository.HasBackupprofilesWith(backupprofile.ID(backupProfileId)),
		)).
		WithBackupprofiles(func(q *ent.BackupProfileQuery) {
			q.Limit(1)
			q.Where(backupprofile.ID(backupProfileId))
		}).
		Only(context.Background())
	if err != nil {
		return err
	}
	if len(repo.Edges.Backupprofiles) != 1 {
		return fmt.Errorf("repository does not have the backup profile")
	}

	backupProfile := repo.Edges.Backupprofiles[0]
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not complete")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	b.inChan.StartBackup <- types.BackupJob{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
		RepoUrl:         repo.URL,
		RepoPassword:    repo.Password,
		Hostname:        hostname,
		Directories:     backupProfile.Directories,
		BinaryPath:      b.binaryPath,
	}
	return nil
}

func (b *BorgClient) RunBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		err := b.RunBackup(backupProfileId, repo.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
