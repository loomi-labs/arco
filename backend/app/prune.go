package app

import (
	"arco/backend/borg"
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

	// Create go routine to save prune result
	ch := make(chan borg.PruneResult)
	go b.savePruneResult(bId, ch)

	err := b.borg.Prune(b.ctx, repoUrl, password, prefix, ch)
	if err != nil {
		b.state.AddNotification(err.Error(), LevelError)
	} else {
		b.state.AddNotification(fmt.Sprintf("Prune job completed"), LevelInfo)
	}
}

func (b *BackupClient) savePruneResult(bId BackupId, ch chan borg.PruneResult) {
	for {
		select {
		case <-b.ctx.Done():
			return
		case result, ok := <-ch:
			if !ok {
				// Channel is closed, break the loop
				return
			}
			b.state.SetPruneResult(bId, result)
		}
	}
}
