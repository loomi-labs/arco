package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/borg"
	"arco/backend/ent"
	"arco/backend/ent/archive"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/notification"
	"arco/backend/ent/pruningrule"
	"arco/backend/ent/repository"
	"errors"
	"fmt"
	"github.com/eminarican/safetypes"
	"sync"
	"time"
)

func (b *BackupClient) SavePruningRule(backupId int, rule ent.PruningRule) (*ent.PruningRule, error) {
	defer b.sendPruningRuleChanged()

	// We ignore the ID from the given rule and get it from the db directly
	pruingRuleId, err := b.db.PruningRule.
		Query().
		Where(pruningrule.HasBackupProfileWith(backupprofile.ID(backupId))).
		FirstID(b.ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	} else if ent.IsNotFound(err) {
		b.log.Debug(fmt.Sprintf("Creating pruning rule for backup profile %d", backupId))
		return b.db.PruningRule.
			Create().
			SetIsEnabled(rule.IsEnabled).
			SetKeepHourly(rule.KeepHourly).
			SetKeepDaily(rule.KeepDaily).
			SetKeepWeekly(rule.KeepWeekly).
			SetKeepMonthly(rule.KeepMonthly).
			SetKeepYearly(rule.KeepYearly).
			SetKeepWithinDays(rule.KeepWithinDays).
			SetBackupProfileID(backupId).
			Save(b.ctx)
	}
	b.log.Debug(fmt.Sprintf("Updating pruning rule %d for backup profile %d", rule.ID, backupId))
	return b.db.PruningRule.
		UpdateOneID(pruingRuleId).
		SetIsEnabled(rule.IsEnabled).
		SetKeepHourly(rule.KeepHourly).
		SetKeepDaily(rule.KeepDaily).
		SetKeepWeekly(rule.KeepWeekly).
		SetKeepMonthly(rule.KeepMonthly).
		SetKeepYearly(rule.KeepYearly).
		SetKeepWithinDays(rule.KeepWithinDays).
		Save(b.ctx)
}

func (b *BackupClient) sendPruningRuleChanged() {
	b.log.Debug("Sending pruning rule changed event")
	if b.pruningScheduleChangedCh == nil {
		return
	}
	b.pruningScheduleChangedCh <- struct{}{}
}

func (b *BackupClient) StartPruneJob(bId types.BackupId) error {
	if canRun, reason := b.state.CanRunPrune(bId); !canRun {
		return errors.New(reason)
	}

	go func() {
		_, err := b.runPruneJob(bId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Prune job failed: %s", err))
		}
	}()
	return nil
}

func (b *BackupClient) StartPruneJobs(bIds []types.BackupId) error {
	for _, bId := range bIds {
		err := b.StartPruneJob(bId)
		if err != nil {
			return err
		}
	}
	return nil
}

type ExaminePruningResult struct {
	BackupID               types.BackupId
	RepositoryName         string
	CntArchivesToBeDeleted int
	Error                  error
}

func (b *BackupClient) startExaminePrune(bId types.BackupId, pruningRule *ent.PruningRule, wg *sync.WaitGroup, resultCh chan<- ExaminePruningResult) {
	defer wg.Done()

	repo, err := b.db.Repository.Query().
		Where(repository.ID(bId.RepositoryId)).
		Select(repository.FieldName).
		Only(b.ctx)
	if err != nil {
		return
	}

	if canRun, reason := b.state.CanPerformRepoOperation(bId); !canRun {
		resultCh <- ExaminePruningResult{BackupID: bId, Error: errors.New(reason), RepositoryName: repo.Name}
		return
	}

	cntToBeDeleted, err := b.examinePrune(bId, safetypes.Some(pruningRule))
	if err != nil {
		b.log.Error(fmt.Sprintf("Failed to examine prune: %s", err))
		b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to examine prune: %s", err), types.LevelError)
		resultCh <- ExaminePruningResult{BackupID: bId, Error: err, RepositoryName: repo.Name}
		return
	}

	resultCh <- ExaminePruningResult{BackupID: bId, CntArchivesToBeDeleted: cntToBeDeleted, RepositoryName: repo.Name}
}

func (b *BackupClient) ExaminePrunes(backupProfileId int, pruningRule *ent.PruningRule) []ExaminePruningResult {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return []ExaminePruningResult{{Error: err}}
	}

	var wg sync.WaitGroup
	resultCh := make(chan ExaminePruningResult, len(backupProfile.Edges.Repositories))
	results := make([]ExaminePruningResult, 0, len(backupProfile.Edges.Repositories))

	for _, repo := range backupProfile.Edges.Repositories {
		wg.Add(1)
		bId := types.BackupId{BackupProfileId: backupProfileId, RepositoryId: repo.ID}
		go b.startExaminePrune(bId, pruningRule, &wg, resultCh)
	}

	// Wait for all examine prune jobs to finish
	wg.Wait()
	close(resultCh)

	// Collect results
	for result := range resultCh {
		results = append(results, result)
	}

	return results
}

/***********************************/
/********** Borg Commands **********/
/***********************************/

type PruneResult string

const (
	PruneResultSuccess  PruneResult = "success"
	PruneResultCanceled PruneResult = "canceled"
	PruneResultError    PruneResult = "error"
)

func (p PruneResult) String() string {
	return string(p)
}

func (b *BackupClient) runPruneJob(bId types.BackupId) (PruneResult, error) {
	repo, err := b.getRepoWithBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		b.state.SetPruneError(b.ctx, bId, err, false, false)
		b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to get repository: %s", err), types.LevelError)
		return PruneResultError, err
	}
	backupProfile := repo.Edges.BackupProfiles[0]
	pruningRule := backupProfile.Edges.PruningRule
	if pruningRule == nil {
		err = errors.New("pruning rule not found")
		b.state.SetPruneError(b.ctx, bId, err, false, false)
		b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to get pruning rule: %s", err), types.LevelError)
		return PruneResultError, err
	}

	b.state.SetPruneWaiting(b.ctx, bId)

	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the prune as running
	b.state.SetPruneRunning(b.ctx, bId)

	// Get all archives from the database
	archives, err := b.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(bId.RepositoryId))).
		All(b.ctx)
	if err != nil {
		b.state.SetPruneError(b.ctx, bId, err, false, false)
		b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to get archives: %s", err), types.LevelError)
		return PruneResultError, err
	}

	// Create go routine to save prune result
	borgCh := make(chan borg.PruneResult)
	resultCh := make(chan state.PruneJobResult)
	go b.savePruneResult(bId, false, archives, borgCh, resultCh)

	cmd := pruneEntityToBorgCmd(pruningRule)
	err = b.borg.Prune(b.ctx, repo.Location, repo.Password, backupProfile.Prefix, cmd, false, borgCh)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.SetPruneCancelled(b.ctx, bId)
			return PruneResultCanceled, nil
		} else if errors.As(err, &borg.LockTimeout{}) {
			err = fmt.Errorf("repository is locked")
			saveErr := b.saveDbNotification(bId, err.Error(), notification.TypeFailedPruningRun, safetypes.Some(notification.ActionUnlockRepository))
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetPruneError(b.ctx, bId, err, false, true)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to prune repository: %s", err), types.LevelError)
			return PruneResultError, err
		} else {
			saveErr := b.saveDbNotification(bId, err.Error(), notification.TypeFailedPruningRun, safetypes.None[notification.Action]())
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetPruneError(b.ctx, bId, err, true, false)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to prune repository: %s", err), types.LevelError)
			return PruneResultError, err
		}
	} else {
		select {
		case pruneResult := <-resultCh:
			// Prune job completed successfully
			defer b.state.SetPruneCompleted(b.ctx, bId, pruneResult)

			err = b.refreshRepoInfo(bId.RepositoryId, repo.Location, repo.Password)
			if err != nil {
				b.log.Error(fmt.Sprintf("Failed to get info for backup-profile %d: %s", bId, err))
			}

			_, err := b.repoClient().refreshArchives(bId.RepositoryId)
			if err != nil {
				b.log.Error(fmt.Sprintf("Failed to refresh archives for backup-profile %d: %s", bId, err))
			}

			return PruneResultSuccess, nil
		case <-time.After(30 * time.Second):
			return PruneResultError, fmt.Errorf("timeout waiting for prune result")
		case <-b.ctx.Done():
			return PruneResultError, fmt.Errorf("context canceled")
		}
	}
}

func pruneEntityToBorgCmd(pruningRule *ent.PruningRule) []string {
	var cmd []string
	if pruningRule.KeepHourly > 0 {
		cmd = append(cmd, "--keep-hourly", fmt.Sprintf("%d", pruningRule.KeepHourly))
	}
	if pruningRule.KeepDaily > 0 {
		cmd = append(cmd, "--keep-daily", fmt.Sprintf("%d", pruningRule.KeepDaily))
	}
	if pruningRule.KeepWeekly > 0 {
		cmd = append(cmd, "--keep-weekly", fmt.Sprintf("%d", pruningRule.KeepWeekly))
	}
	if pruningRule.KeepMonthly > 0 {
		cmd = append(cmd, "--keep-monthly", fmt.Sprintf("%d", pruningRule.KeepMonthly))
	}
	if pruningRule.KeepYearly > 0 {
		cmd = append(cmd, "--keep-yearly", fmt.Sprintf("%d", pruningRule.KeepYearly))
	}
	if pruningRule.KeepWithinDays > 0 {
		cmd = append(cmd, "--keep-within", fmt.Sprintf("%dd", pruningRule.KeepWithinDays))
	}
	return cmd
}

func (b *BackupClient) savePruneResult(bId types.BackupId, isDryRun bool, archives []*ent.Archive, ch chan borg.PruneResult, resultCh chan state.PruneJobResult) {
	defer close(resultCh)
	for {
		select {
		case <-b.ctx.Done():
			return
		case result, ok := <-ch:
			if !ok {
				// Channel is closed, break the loop
				return
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

			resultCh <- pjr

			// TODO: remove this
			//if isDryRun {
			//	b.state.SetDryRunPruneResult(bId, pjr)
			//} else {
			//	b.state.SetPruneResult(bId, pjr)
			//}
		}
	}
}

func (b *BackupClient) examinePrune(bId types.BackupId, pruningRuleOpt safetypes.Option[*ent.PruningRule]) (int, error) {
	repo, err := b.getRepoWithBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository: %w", err)
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	pruningRule := pruningRuleOpt.UnwrapOr(backupProfile.Edges.PruningRule)
	if pruningRule == nil {
		return 0, fmt.Errorf("pruning rule not found")
	}

	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusPerformingOperation)

	// Get all archives from the database
	archives, err := b.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(bId.RepositoryId))).
		All(b.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get archives: %w", err)
	}

	// Create go routine to save prune result
	borgCh := make(chan borg.PruneResult)
	resultCh := make(chan state.PruneJobResult)
	go b.savePruneResult(bId, true, archives, borgCh, resultCh)

	cmd := pruneEntityToBorgCmd(pruningRule)
	err = b.borg.Prune(b.ctx, repo.Location, repo.Password, backupProfile.Prefix, cmd, true, borgCh)
	if err != nil {
		if errors.As(err, &borg.LockTimeout{}) {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusLocked)
			return 0, fmt.Errorf("repository is locked")
		} else {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return 0, fmt.Errorf("failed to examine prune: %w", err)
		}
	} else {
		select {
		case pruneResult := <-resultCh:
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)

			keepIds := make([]int, len(pruneResult.KeepArchives))
			for i, keep := range pruneResult.KeepArchives {
				keepIds[i] = keep.Id
			}

			tx, err := b.db.Tx(b.ctx)
			if err != nil {
				return 0, fmt.Errorf("failed to start transaction: %w", err)
			}

			err = tx.Archive.
				Update().
				Where(archive.And(
					archive.IDIn(pruneResult.PruneArchives...),
					archive.WillBePruned(false)),
				).
				SetWillBePruned(true).
				Exec(b.ctx)
			if err != nil {
				return 0, rollback(tx, fmt.Errorf("failed to update archives: %w", err))
			}

			err = tx.Archive.
				Update().
				Where(archive.And(
					archive.IDIn(keepIds...),
					archive.WillBePruned(true)),
				).
				SetWillBePruned(false).
				Exec(b.ctx)

			return len(pruneResult.PruneArchives), tx.Commit()
		case <-time.After(10 * time.Second):
			return 0, fmt.Errorf("timeout waiting for prune result")
		case <-b.ctx.Done():
			return 0, fmt.Errorf("context canceled")
		}
	}
}
