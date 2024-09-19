package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/borg"
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/failedbackuprun"
	"arco/backend/ent/repository"
	"arco/backend/util"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"time"
)

/***********************************/
/********** Backup Profile *********/
/***********************************/

func (b *BackupClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// Choose the first icon that is not already in use
	all, err := b.db.BackupProfile.
		Query().
		Select(backupprofile.FieldIcon).
		All(b.ctx)
	if err != nil {
		return nil, err
	}
	icons := make(map[backupprofile.Icon]bool)
	for _, p := range all {
		icons[p.Icon] = true
	}
	selectedIcon := backupprofile.IconHome
	for _, icon := range types.AllIcons {
		if !icons[icon] {
			selectedIcon = icon
			break
		}
	}

	return &ent.BackupProfile{
		ID:           0,
		Name:         "",
		Prefix:       hostname,
		BackupPaths:  make([]string, 0),
		ExcludePaths: make([]string, 0),
		// TODO: remove isSetupComplete completely
		IsSetupComplete: false,
		Icon:            selectedIcon,
		Edges:           ent.BackupProfileEdges{},
	}, nil
}

func (b *BackupClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BackupClient) DoesPathExist(path string) bool {
	_, err := os.Stat(util.ExpandPath(path))
	return err == nil
}

func (b *BackupClient) IsDirectory(path string) bool {
	info, err := os.Stat(util.ExpandPath(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (b *BackupClient) IsDirectoryEmpty(path string) bool {
	path = util.ExpandPath(path)
	if !b.IsDirectory(path) {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	//goland:noinspection GoUnhandledErrorResult
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err != nil
}

func (b *BackupClient) CreateDirectory(path string) error {
	return os.MkdirAll(util.ExpandPath(path), 0755)
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
		WithRepositories().
		All(b.ctx)
}

func (b *BackupClient) SaveBackupProfile(backup ent.BackupProfile) (*ent.BackupProfile, error) {
	b.log.Debug(fmt.Sprintf("Saving backup profile %d", backup.ID))
	if backup.ID == 0 {
		return b.db.BackupProfile.
			Create().
			SetName(backup.Name).
			SetPrefix(backup.Prefix).
			SetBackupPaths(backup.BackupPaths).
			SetExcludePaths(backup.ExcludePaths).
			SetIsSetupComplete(backup.IsSetupComplete).
			SetIcon(backup.Icon).
			Save(b.ctx)
	}
	return b.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(b.ctx)
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
			go b.runBorgDelete(bId, repo.Location, repo.Password, backupProfile.Prefix)
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

/***********************************/
/********** Backup Functions *******/
/***********************************/

// StartBackupJob starts a backup job for the given repository and backup profile.
func (b *BackupClient) StartBackupJob(bId types.BackupId) error {
	if canRun, reason := b.state.CanRunBackup(bId); !canRun {
		return errors.New(reason)
	}

	go func() {
		_, err := b.runBorgCreate(bId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Backup job failed: %s", err))
		}
	}()

	return nil
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

func (b *BackupClient) AbortBackupJob(id types.BackupId) error {
	b.state.SetBackupCancelled(id, true)
	return nil
}

func (b *BackupClient) AbortBackupJobs(bIds []types.BackupId) error {
	for _, bId := range bIds {
		if b.state.GetBackupState(bId).Status == state.BackupStatusRunning {
			err := b.AbortBackupJob(bId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *BackupClient) GetState(bId types.BackupId) state.BackupState {
	return b.state.GetBackupState(bId)
}

func (b *BackupClient) GetBackupButtonStatus(bId types.BackupId) state.BackupButtonStatus {
	return b.state.GetBackupButtonStatus(bId)
}

func (b *BackupClient) GetCombinedBackupProgress(bIds []types.BackupId) *borg.BackupProgress {
	return b.state.GetCombinedBackupProgress(bIds)
}

func (b *BackupClient) GetCombinedBackupButtonStatus(bIds []types.BackupId) state.BackupButtonStatus {
	return b.state.GetCombinedBackupButtonStatus(bIds)
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
	backupTime, err := getNextBackupTime(&schedule, time.Now())
	if err != nil {
		return rollback(tx, fmt.Errorf("failed to get next backup time: %w", err))
	}
	_, err = tx.BackupSchedule.
		Create().
		SetHourly(schedule.Hourly).
		SetNillableDailyAt(schedule.DailyAt).
		SetNillableWeeklyAt(schedule.WeeklyAt).
		SetNillableWeekday(schedule.Weekday).
		SetNillableMonthlyAt(schedule.MonthlyAt).
		SetNillableMonthday(schedule.Monthday).
		SetNextRun(backupTime).
		SetBackupProfileID(backupProfileId).
		Save(b.ctx)
	if err != nil {
		return rollback(tx, fmt.Errorf("failed to save schedule: %w", err))
	}
	defer b.sendBackupScheduleChanged()
	return tx.Commit()
}

func (b *BackupClient) DeleteBackupSchedule(backupProfileId int) error {
	_, err := b.db.BackupSchedule.
		Delete().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exec(b.ctx)
	return err
}

func (b *BackupClient) sendBackupScheduleChanged() {
	if b.backupScheduleChangedCh == nil {
		return
	}
	b.backupScheduleChangedCh <- struct{}{}
}

/***********************************/
/********** Other Functions ********/
/***********************************/

func (b *BackupClient) refreshRepoInfo(repoId int, url, password string) error {
	info, err := b.borg.Info(url, password)
	if err != nil {
		return err
	}
	_, err = b.db.Repository.
		UpdateOneID(repoId).
		SetStatsTotalSize(info.Cache.Stats.TotalSize).
		SetStatsTotalCsize(info.Cache.Stats.TotalCSize).
		SetStatsTotalChunks(info.Cache.Stats.TotalChunks).
		SetStatsTotalUniqueChunks(info.Cache.Stats.TotalUniqueChunks).
		SetStatsUniqueCsize(info.Cache.Stats.UniqueCSize).
		SetStatsUniqueSize(info.Cache.Stats.UniqueSize).
		Save(b.ctx)
	return err
}

func (b *BackupClient) addNewArchive(repoId int, url, password string) error {
	info, err := b.borg.Info(url, password)
	if err != nil {
		return err
	}
	if len(info.Archives) == 0 {
		return fmt.Errorf("no archives found")
	}

	_, err = b.db.Archive.
		Create().
		SetRepositoryID(repoId).
		SetBorgID(info.Archives[0].ID).
		SetName(info.Archives[0].Name).
		SetCreatedAt(time.Time(info.Archives[0].Start)).
		SetDuration(time.Time(info.Archives[0].Duration)).
		Save(b.ctx)
	return err
}

func (b *BackupClient) deleteFailedBackupRun(bId types.BackupId) error {
	_, err := b.db.FailedBackupRun.
		Delete().
		Where(failedbackuprun.And(
			failedbackuprun.HasRepositoryWith(repository.ID(bId.RepositoryId)),
			failedbackuprun.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
		)).
		Exec(b.ctx)
	return err
}

func (b *BackupClient) saveFailedBackupRun(bId types.BackupId, backupErr error) error {
	err := b.deleteFailedBackupRun(bId)
	if err != nil {
		return err
	}
	_, err = b.db.FailedBackupRun.
		Create().
		SetRepositoryID(bId.RepositoryId).
		SetBackupProfileID(bId.BackupProfileId).
		SetError(backupErr.Error()).
		Save(b.ctx)
	return err
}

/***********************************/
/********** Borg Commands **********/
/***********************************/

// runBorgCreate runs the actual backup job.
// It is long running and should be run in a goroutine.
func (b *BackupClient) runBorgCreate(bId types.BackupId) (result state.BackupResult, err error) {
	repo, err := b.getRepoWithCompletedBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		b.state.SetBackupError(bId, err, false, false)
		b.state.AddNotification(fmt.Sprintf("Failed to get repository: %s", err), types.LevelError)
		return state.BackupResultError, err
	}
	backupProfile := repo.Edges.BackupProfiles[0]
	b.state.SetBackupWaiting(bId)

	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the backup as running
	ctx := b.state.SetBackupRunning(b.ctx, bId)

	// Create go routine to receive progress info
	ch := make(chan borg.BackupProgress)
	defer close(ch)
	go b.saveProgressInfo(bId, ch)

	archiveName, err := b.borg.Create(ctx, repo.Location, repo.Password, backupProfile.Prefix, backupProfile.BackupPaths, backupProfile.ExcludePaths, ch)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.SetBackupCancelled(bId, true)
			return state.BackupResultCancelled, nil
		} else if errors.As(err, &borg.LockTimeout{}) {
			err = fmt.Errorf("repository is locked")
			saveErr := b.saveFailedBackupRun(bId, err)
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save failed backup run: %s", saveErr))
			}
			b.state.SetBackupError(bId, err, false, true)
			b.state.AddNotification(fmt.Sprintf("Backup job failed: repository %s is locked", repo.Name), types.LevelError)
			return state.BackupResultError, err
		} else {
			saveErr := b.saveFailedBackupRun(bId, err)
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save failed backup run: %s", saveErr))
			}
			b.state.SetBackupError(bId, err, true, false)
			b.state.AddNotification(fmt.Sprintf("Backup job failed: %s", err), types.LevelError)
			return state.BackupResultError, err
		}
	} else {
		// Backup completed successfully
		defer b.state.SetBackupCompleted(bId, true)

		err = b.deleteFailedBackupRun(bId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to delete failed backup run: %s", err))
		}

		err = b.refreshRepoInfo(bId.RepositoryId, repo.Location, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup %d: %s", bId, err))
		}

		err = b.addNewArchive(bId.RepositoryId, archiveName, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup %d: %s", bId, err))
		}

		return state.BackupResultSuccess, nil
	}
}

// TODO: do we need this function? Maybe refactor it to?
func (b *BackupClient) runBorgDelete(bId types.BackupId, repoUrl, password, prefix string) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as locked
	b.state.SetRepoStatus(bId.RepositoryId, state.RepoStatusDeleting)
	b.state.AddRunningDeleteJob(b.ctx, bId)
	defer b.state.RemoveRunningDeleteJob(bId)

	err := b.borg.DeleteArchives(b.ctx, repoUrl, password, prefix)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.AddNotification("Delete job cancelled", types.LevelWarning)
			b.state.SetRepoStatus(bId.RepositoryId, state.RepoStatusIdle)
		} else if errors.As(err, &borg.LockTimeout{}) {
			//b.state.AddBorgLock(bId.RepositoryId)
			b.state.AddNotification("Delete job failed: repository is locked", types.LevelError)
			b.state.SetRepoStatus(bId.RepositoryId, state.RepoStatusLocked)
		} else {
			b.state.AddNotification(err.Error(), types.LevelError)
			b.state.SetRepoStatus(bId.RepositoryId, state.RepoStatusIdle)
		}
	} else {
		b.state.AddNotification("Delete job completed", types.LevelInfo)
		b.state.SetRepoStatus(bId.RepositoryId, state.RepoStatusIdle)
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
