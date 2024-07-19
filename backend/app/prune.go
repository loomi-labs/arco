package app

import (
	"arco/backend/types"
	"fmt"
)

func (b *BackupClient) runPruneJob(pruneJob types.PruneJob) {
	repoLock := b.state.GetRepoLock(pruneJob.Id.RepositoryId)
	repoLock.Lock()
	defer repoLock.Unlock()
	defer b.state.DeleteRepoLock(pruneJob.Id.RepositoryId)
	b.state.AddRunningPruneJob(b.ctx, pruneJob.Id)
	defer b.state.RemoveRunningBackup(pruneJob.Id)

	err := b.borg.Prune(b.ctx, pruneJob)
	if err != nil {
		b.state.AddNotification(err.Error(), LevelError)
	} else {
		b.state.AddNotification(fmt.Sprintf("Prune job completed"), LevelInfo)
	}
}

func (b *BackupClient) PruneBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if canRun, reason := b.state.CanRunPruneJob(bId); !canRun {
		return fmt.Errorf(reason)
	}

	go b.runPruneJob(types.PruneJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
	})
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
