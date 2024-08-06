package app

import (
	"arco/backend/app/types"
	"arco/backend/borg"
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/repository"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
)

/***********************************/
/********** Backup Profile *********/
/***********************************/

func (b *BackupClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.db.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetBackupPaths([]string{}).
		SetExcludePaths([]string{}).
		Save(b.ctx)
}

func (b *BackupClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BackupClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		Where(backupprofile.ID(id)).Only(b.ctx)
}

func (b *BackupClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithBackupSchedule().
		All(b.ctx)
}

func (b *BackupClient) SaveBackupProfile(backup ent.BackupProfile) error {
	b.log.Debug(fmt.Sprintf("Saving backup profile %d", backup.ID))
	_, err := b.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(b.ctx)
	return err
}

func (b *BackupClient) DeleteBackupProfile(id int, withBackups bool) error {
	if withBackups {
		backupProfile, err := b.GetBackupProfile(id)
		if err != nil {
			return err
		}
		for _, repo := range backupProfile.Edges.Repositories {
			bId := types.BackupId{
				BackupProfileId: id,
				RepositoryId:    repo.ID,
			}
			go b.runBorgDelete(bId, repo.URL, repo.Password, backupProfile.Prefix)
		}
	}
	err := b.db.BackupProfile.
		DeleteOneID(id).
		Exec(b.ctx)
	return err
}

func (b *BackupClient) getRepoWithCompletedBackupProfile(repoId int, backupProfileId int) (*ent.Repository, error) {
	repo, err := b.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repoId),
			repository.HasBackupProfilesWith(backupprofile.ID(backupProfileId)),
		)).
		WithBackupProfiles(func(q *ent.BackupProfileQuery) {
			q.Limit(1)
			q.Where(backupprofile.ID(backupProfileId))
		}).
		Only(b.ctx)
	if err != nil {
		return nil, err
	}
	if len(repo.Edges.BackupProfiles) != 1 {
		return nil, fmt.Errorf("repository does not have the backup profile")
	}
	if !repo.Edges.BackupProfiles[0].IsSetupComplete {
		return nil, fmt.Errorf("backup profile is not complete")
	}
	return repo, nil
}

func (b *BackupClient) startBackupJob(bId types.BackupId) error {
	repo, err := b.getRepoWithCompletedBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	go b.runBorgCreate(bId, repo.URL, repo.Password, backupProfile.Prefix, backupProfile.BackupPaths, backupProfile.ExcludePaths)
	return nil
}

// StartBackupJob starts a backup job for the given repository and backup profile.
func (b *BackupClient) StartBackupJob(bId types.BackupId) error {
	if canRun, reason := b.state.CanRunBackup(bId); !canRun {
		return fmt.Errorf(reason)
	}

	return b.startBackupJob(bId)
}

func (b *BackupClient) StartBackupJobs(backupProfileId int) ([]types.BackupId, error) {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return nil, err
	}
	if !backupProfile.IsSetupComplete {
		return nil, fmt.Errorf("backup profile is not setup")
	}

	var bIds []types.BackupId
	for _, repo := range backupProfile.Edges.Repositories {
		bId := types.BackupId{BackupProfileId: backupProfileId, RepositoryId: repo.ID}
		err := b.StartBackupJob(bId)
		if err != nil {
			return bIds, err
		}
		bIds = append(bIds, bId)
	}
	return bIds, nil
}

func (b *BackupClient) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(b.ctx, runtime.OpenDialogOptions{})
}

type BackupProgressResponse struct {
	BackupId types.BackupId      `json:"backupId"`
	Progress borg.BackupProgress `json:"progress"`
	Found    bool                `json:"found"`
}

func (b *BackupClient) GetBackupProgress(id types.BackupId) BackupProgressResponse {
	progress, found := b.state.GetBackupProgress(id)
	return BackupProgressResponse{
		BackupId: id,
		Progress: progress,
		Found:    found,
	}
}

func (b *BackupClient) GetBackupProgresses(ids []types.BackupId) []BackupProgressResponse {
	var progresses []BackupProgressResponse
	for _, id := range ids {
		progresses = append(progresses, b.GetBackupProgress(id))
	}
	return progresses
}

func (b *BackupClient) AbortBackupJob(id types.BackupId) error {
	b.state.RemoveRunningBackup(id)
	return nil
}

/***********************************/
/********** Backup Schedule ********/
/***********************************/

func (b *BackupClient) SaveBackupSchedule(backupProfileId int, schedule ent.BackupSchedule) error {
	doesExist, err := b.db.BackupSchedule.
		Query().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exist(b.ctx)
	if err != nil {
		return err
	}
	tx, err := b.db.Tx(b.ctx)
	if err != nil {
		return err
	}
	if doesExist {
		_, err := tx.BackupSchedule.
			Delete().
			Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
			Exec(b.ctx)
		if err != nil {
			return rollback(tx, fmt.Errorf("failed to delete existing schedule: %w", err))
		}
	}
	_, err = tx.BackupSchedule.
		Create().
		SetHourly(schedule.Hourly).
		SetNillableDailyAt(schedule.DailyAt).
		SetNillableWeeklyAt(schedule.WeeklyAt).
		SetNillableWeekday(schedule.Weekday).
		SetNillableMonthlyAt(schedule.MonthlyAt).
		SetNillableMonthday(schedule.Monthday).
		SetBackupProfileID(backupProfileId).
		Save(b.ctx)
	if err != nil {
		return rollback(tx, fmt.Errorf("failed to save schedule: %w", err))
	}
	return tx.Commit()
}

func (b *BackupClient) DeleteBackupSchedule(backupProfileId int) error {
	_, err := b.db.BackupSchedule.
		Delete().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exec(b.ctx)
	return err
}

// rollback calls to tx.Rollback and wraps the given error
// with the rollback error if occurred.
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}

/***********************************/
/********** Borg Commands **********/
/***********************************/

// runBorgCreate runs the actual backup job.
// It is long running and should be run in a goroutine.
func (b *BackupClient) runBorgCreate(bId types.BackupId, repoUrl, password, prefix string, backupPaths []string, excludePaths []string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()
	// Wait to acquire the lock and then set the repo as locked
	b.state.SetRepoLocked(bId.RepositoryId)
	defer b.state.UnlockRepo(bId.RepositoryId)
	ctx := b.state.AddRunningBackup(b.ctx, bId)
	defer b.state.RemoveRunningBackup(bId)

	// Create go routine to receive progress info
	ch := make(chan borg.BackupProgress)
	go b.saveProgressInfo(bId, ch)

	err := b.borg.Create(ctx, repoUrl, password, prefix, backupPaths, excludePaths, ch)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.AddNotification("Backup job cancelled", types.LevelWarning)
		} else if errors.As(err, &borg.LockTimeout{}) {
			b.state.AddBorgLock(bId.RepositoryId)
			b.state.AddNotification("Backup job failed: repository is locked", types.LevelError)
		} else {
			b.state.AddNotification(err.Error(), types.LevelError)
		}
	} else {
		b.state.AddNotification(fmt.Sprintf("Backup job completed"), types.LevelInfo)
	}
}

func (b *BackupClient) runBorgDelete(bId types.BackupId, repoUrl, password, prefix string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()
	// Wait to acquire the lock and then set the repo as locked
	b.state.SetRepoLocked(bId.RepositoryId)
	defer b.state.UnlockRepo(bId.RepositoryId)
	b.state.AddRunningDeleteJob(b.ctx, bId)
	defer b.state.RemoveRunningDeleteJob(bId)

	err := b.borg.DeleteArchives(b.ctx, repoUrl, password, prefix)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.AddNotification("Delete job cancelled", types.LevelWarning)
		} else if errors.As(err, &borg.LockTimeout{}) {
			b.state.AddBorgLock(bId.RepositoryId)
			b.state.AddNotification("Delete job failed: repository is locked", types.LevelError)
		} else {
			b.state.AddNotification(err.Error(), types.LevelError)
		}
	} else {
		b.state.AddNotification(fmt.Sprintf("Delete job completed"), types.LevelInfo)
	}
}

func (b *BackupClient) saveProgressInfo(id types.BackupId, ch chan borg.BackupProgress) {
	for {
		select {
		case <-b.ctx.Done():
			return
		case progress, ok := <-ch:
			if !ok {
				// Channel is closed, break the loop
				return
			}
			b.state.UpdateBackupProgress(id, progress)
		}
	}
}
