package app

import (
	"fmt"
)

func (b *BackupClient) PruneBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	bId := BackupId{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if canRun, reason := b.state.CanRunPruneJob(bId); !canRun {
		return fmt.Errorf(reason)
	}

	go b.runPruneJob(bId, repo.URL, repo.Password, backupProfile.Prefix)
	return nil
}

func (b *BackupClient) PruneBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		err := b.PruneBackup(backupProfileId, repo.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BackupClient) runPruneJob(bId BackupId, repoUrl string, password string, prefix string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()
	defer repoLock.Unlock()
	defer b.state.DeleteRepoLock(bId.RepositoryId)
	b.state.AddRunningPruneJob(b.ctx, bId)
	defer b.state.RemoveRunningBackup(bId)

	err := b.borg.Prune(b.ctx, repoUrl, password, prefix)
	if err != nil {
		b.state.AddNotification(err.Error(), LevelError)
	} else {
		b.state.AddNotification(fmt.Sprintf("Prune job completed"), LevelInfo)
	}
}
