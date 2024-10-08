package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/borg"
	"arco/backend/ent/archive"
	"arco/backend/ent/repository"
	"errors"
)

func (b *BackupClient) PruneBackup(bId types.BackupId) error {
	repo, err := b.getRepoWithCompletedBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	if canRun, reason := b.state.CanRunPruneJob(bId); !canRun {
		return errors.New(reason)
	}

	go b.runPruneJob(bId, repo.Location, repo.Password, backupProfile.Prefix)
	return nil
}

func (b *BackupClient) PruneBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}

	for _, repo := range backupProfile.Edges.Repositories {
		bId := types.BackupId{BackupProfileId: backupProfileId, RepositoryId: repo.ID}
		err := b.PruneBackup(bId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BackupClient) DryRunPruneBackup(bId types.BackupId) error {
	repo, err := b.getRepoWithCompletedBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	if canRun, reason := b.state.CanRunPruneJob(bId); !canRun {
		return errors.New(reason)
	}

	go b.dryRunPruneJob(bId, repo.Location, repo.Password, backupProfile.Prefix)
	return nil
}

func (b *BackupClient) DryRunPruneBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}

	for _, repo := range backupProfile.Edges.Repositories {
		bId := types.BackupId{BackupProfileId: backupProfileId, RepositoryId: repo.ID}
		err := b.DryRunPruneBackup(bId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BackupClient) runPruneJob(bId types.BackupId, repoUrl string, password string, prefix string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as locked
	b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusPruning)
	defer b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
	b.state.AddRunningPruneJob(b.ctx, bId)
	defer b.state.RemoveRunningPruneJob(bId)

	// Create go routine to save prune result
	ch := make(chan borg.PruneResult)
	go b.savePruneResult(bId, false, ch)

	err := b.borg.Prune(b.ctx, repoUrl, password, prefix, false, ch)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.AddNotification(b.ctx, "Prune job was canceled", types.LevelWarning)
		} else if errors.As(err, &borg.LockTimeout{}) {
			//b.state.AddBorgLock(bId.RepositoryId) 	// TODO: fix this
			b.state.AddNotification(b.ctx, "Repository is locked by another operation", types.LevelError)
		} else {
			b.state.AddNotification(b.ctx, err.Error(), types.LevelError)
		}
	} else {
		b.state.AddNotification(b.ctx, "Prune job completed", types.LevelInfo)
	}
}

func (b *BackupClient) savePruneResult(bId types.BackupId, isDryRun bool, ch chan borg.PruneResult) {
	for {
		select {
		case <-b.ctx.Done():
			return
		case result, ok := <-ch:
			if !ok {
				// Channel is closed, break the loop
				return
			}

			// Get all archives from the database
			archives, err := b.db.Archive.
				Query().
				Where(archive.HasRepositoryWith(repository.ID(bId.RepositoryId))).
				All(b.ctx)
			if err != nil {
				b.log.Error("Error querying archives: ", err)
				continue
			}

			// Merge the prune result with the archives
			var pjr state.PruneJobResult
			for _, arch := range archives {
				found := false
				for _, keep := range result.KeepArchives {
					if arch.Name == keep.Name {
						pjr.KeepArchives = append(pjr.KeepArchives, state.KeepArchive{
							Id:     arch.ID,
							Name:   arch.Name,
							Reason: keep.Reason,
						})
						found = true
						break
					}
				}
				if !found {
					for _, prune := range result.PruneArchives {
						if arch.Name == prune.Name {
							pjr.PruneArchives = append(pjr.PruneArchives, arch.ID)
							break
						}
					}
				}
			}

			if isDryRun {
				b.state.SetDryRunPruneResult(bId, pjr)
			} else {
				b.state.SetPruneResult(bId, pjr)
			}
		}
	}
}

func (b *BackupClient) dryRunPruneJob(bId types.BackupId, repoUrl string, password string, prefix string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusPerformingOperation)
	defer b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
	b.state.AddRunningDryRunPruneJob(b.ctx, bId)
	defer b.state.RemoveRunningDryRunPruneJob(bId)

	// Create go routine to save prune result
	ch := make(chan borg.PruneResult)
	go b.savePruneResult(bId, true, ch)

	err := b.borg.Prune(b.ctx, repoUrl, password, prefix, true, ch)
	if err != nil {
		b.state.AddNotification(b.ctx, err.Error(), types.LevelError)
	} else {
		b.state.AddNotification(b.ctx, "Dry-run prune job completed", types.LevelInfo)
	}
}
