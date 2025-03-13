package app

import (
	"errors"
	"fmt"
	"github.com/eminarican/safetypes"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"sync"
	"time"
)

type PruningOptionName string

const (
	PruningOptionNone   PruningOptionName = "none"
	PruningOptionFew    PruningOptionName = "few"
	PruningOptionMany   PruningOptionName = "many"
	PruningOptionCustom PruningOptionName = "custom"
)

func (p PruningOptionName) String() string {
	return string(p)
}

type PruningOption struct {
	Name        PruningOptionName `json:"name"`
	KeepHourly  int               `json:"keepHourly"`
	KeepDaily   int               `json:"keepDaily"`
	KeepWeekly  int               `json:"keepWeekly"`
	KeepMonthly int               `json:"keepMonthly"`
	KeepYearly  int               `json:"keepYearly"`
}

var PruningOptions = []PruningOption{
	{Name: PruningOptionNone},
	{Name: PruningOptionFew, KeepHourly: 8, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 3, KeepYearly: 1},
	{Name: PruningOptionMany, KeepHourly: 24, KeepDaily: 14, KeepWeekly: 8, KeepMonthly: 6, KeepYearly: 2},
	{Name: PruningOptionCustom},
}

var defaultPruningOption = PruningOptions[2]

type GetPruningOptionsResponse struct {
	Options []PruningOption `json:"options"`
}

func (b *BackupClient) GetPruningOptions() GetPruningOptionsResponse {
	return GetPruningOptionsResponse{Options: PruningOptions}
}

func (b *BackupClient) SavePruningRule(backupId int, rule ent.PruningRule) (*ent.PruningRule, error) {
	defer b.sendPruningRuleChanged()

	backupProfile, err := b.GetBackupProfile(backupId)
	if err != nil {
		return nil, err
	}

	nextRun := getNextPruneTime(backupProfile.Edges.BackupSchedule, time.Now())

	if backupProfile.Edges.PruningRule != nil {
		b.log.Debug(fmt.Sprintf("Updating pruning rule %d for backup profile %d", rule.ID, backupId))
		return b.db.PruningRule.
			// We ignore the ID from the given rule and get it from the db directly
			UpdateOneID(backupProfile.Edges.PruningRule.ID).
			SetIsEnabled(rule.IsEnabled).
			SetKeepHourly(rule.KeepHourly).
			SetKeepDaily(rule.KeepDaily).
			SetKeepWeekly(rule.KeepWeekly).
			SetKeepMonthly(rule.KeepMonthly).
			SetKeepYearly(rule.KeepYearly).
			SetKeepWithinDays(rule.KeepWithinDays).
			SetNextRun(nextRun).
			Save(b.ctx)
	}
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
		SetNextRun(nextRun).
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

type ExaminePruningResult struct {
	BackupID               types.BackupId
	RepositoryName         string
	CntArchivesToBeDeleted int
	Error                  error
}

func (b *BackupClient) startExaminePrune(bId types.BackupId, pruningRule *ent.PruningRule, saveResults bool, wg *sync.WaitGroup, resultCh chan<- ExaminePruningResult) {
	defer wg.Done()

	repo, err := b.db.Repository.Query().
		Where(repository.ID(bId.RepositoryId)).
		Select(repository.FieldName).
		Only(b.ctx)
	if err != nil {
		return
	}

	cntToBeDeleted, err := b.examinePrune(bId, safetypes.Some(pruningRule), saveResults, false)
	if err != nil {
		b.log.Debugf("Failed to examine prune: %s", err)
		resultCh <- ExaminePruningResult{BackupID: bId, Error: err, RepositoryName: repo.Name}
		return
	}

	resultCh <- ExaminePruningResult{BackupID: bId, CntArchivesToBeDeleted: cntToBeDeleted, RepositoryName: repo.Name}
}

func (b *BackupClient) ExaminePrunes(backupProfileId int, pruningRule *ent.PruningRule, saveResults bool) []ExaminePruningResult {
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
		go b.startExaminePrune(bId, pruningRule, saveResults, &wg, resultCh)
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
	borgCh := make(chan borgtypes.PruneResult)
	resultCh := make(chan state.PruneJobResult)
	go b.savePruneResult(archives, borgCh, resultCh)

	cmd := pruneEntityToBorgCmd(pruningRule)
	err = b.borg.Prune(b.ctx, repo.Location, repo.Password, backupProfile.Prefix, cmd, false, borgCh)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.SetPruneCancelled(b.ctx, bId)
			return PruneResultCanceled, nil
		} else if errors.Is(err, borg.ErrorLockTimeout) {
			err = fmt.Errorf("repository %s is locked", repo.Name)
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

func (b *BackupClient) savePruneResult(archives []*ent.Archive, ch chan borgtypes.PruneResult, resultCh chan state.PruneJobResult) {
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
		}
	}
}

func (b *BackupClient) examinePrune(bId types.BackupId, pruningRuleOpt safetypes.Option[*ent.PruningRule], saveResults, skipAcquiringRepoLock bool) (int, error) {
	repo, err := b.getRepoWithBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository: %w", err)
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	pruningRule := pruningRuleOpt.UnwrapOr(backupProfile.Edges.PruningRule)
	if pruningRule == nil {
		return 0, fmt.Errorf("no pruning rule found")
	}

	// If the pruning rule is not enabled, we don't need to call borg
	if !pruningRule.IsEnabled {
		if saveResults {
			defer b.eventEmitter.EmitEvent(b.ctx, types.EventArchivesChangedString(bId.RepositoryId))
			err = b.db.Archive.
				Update().
				Where(archive.And(
					archive.HasRepositoryWith(repository.ID(bId.RepositoryId)),
					archive.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
					archive.WillBePruned(true)),
				).
				SetWillBePruned(false).
				Exec(b.ctx)
			if err != nil {
				return 0, fmt.Errorf("failed to update archives: %w", err)
			}
		}
		return 0, nil
	}

	if !skipAcquiringRepoLock {
		// We do not wait for other operations to finish
		// Either we can run the operation or we return an error
		if canRun, reason := b.state.CanPerformRepoOperation(bId.RepositoryId); !canRun {
			return 0, fmt.Errorf("can not examine prune: %s", reason)
		}

		repoLock := b.state.GetRepoLock(bId.RepositoryId)
		repoLock.Lock()         // We should not have to wait here
		defer repoLock.Unlock() // Unlock at the end

		b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusPerformingOperation)
	}

	// Get all archives from the database
	archives, err := b.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(bId.RepositoryId))).
		All(b.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get archives: %w", err)
	}

	// Create go routine to save prune result
	borgCh := make(chan borgtypes.PruneResult)
	resultCh := make(chan state.PruneJobResult)
	go b.savePruneResult(archives, borgCh, resultCh)

	cmd := pruneEntityToBorgCmd(pruningRule)
	err = b.borg.Prune(b.ctx, repo.Location, repo.Password, backupProfile.Prefix, cmd, true, borgCh)
	if err != nil {
		if errors.Is(err, borg.ErrorLockTimeout) {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusLocked)
			return 0, fmt.Errorf("repository %s is locked", repo.Name)
		} else {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return 0, fmt.Errorf("failed to examine prune: %w", err)
		}
	} else {
		select {
		case pruneResult := <-resultCh:
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)

			if saveResults {
				keepIds := make([]int, len(pruneResult.KeepArchives))
				for i, keep := range pruneResult.KeepArchives {
					keepIds[i] = keep.Id
				}

				tx, err := b.db.Tx(b.ctx)
				if err != nil {
					return 0, fmt.Errorf("failed to start transaction: %w", err)
				}

				cntToTrue, err := tx.Archive.
					Update().
					Where(archive.And(
						archive.IDIn(pruneResult.PruneArchives...),
						archive.WillBePruned(false)),
					).
					SetWillBePruned(true).
					Save(b.ctx)
				if err != nil {
					return 0, rollback(tx, fmt.Errorf("failed to update asdfasdfasdfasdf: %w", err))
				}

				cntToFalse, err := tx.Archive.
					Update().
					Where(archive.And(
						archive.IDIn(keepIds...),
						archive.WillBePruned(true)),
					).
					SetWillBePruned(false).
					Save(b.ctx)
				if err != nil {
					return 0, rollback(tx, fmt.Errorf("failed to update archives: %w", err))
				}
				err = tx.Commit()
				if err != nil {
					return 0, fmt.Errorf("failed to commit transaction: %w", err)
				}
				if cntToTrue+cntToFalse > 0 {
					b.eventEmitter.EmitEvent(b.ctx, types.EventArchivesChangedString(bId.RepositoryId))
				}

				return len(pruneResult.PruneArchives), nil
			}
			return len(pruneResult.PruneArchives), nil
		case <-time.After(10 * time.Second):
			return 0, fmt.Errorf("timeout waiting for prune result")
		case <-b.ctx.Done():
			return 0, fmt.Errorf("context canceled")
		}
	}
}
