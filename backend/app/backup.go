package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/repository"
	"arco/backend/types"
	"arco/backend/util"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"os/exec"
	"time"
)

/***********************************/
/********** Backup Profile *********/
/***********************************/

func (b *BackupClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.db.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetDirectories([]string{}).
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
		SetDirectories(backup.Directories).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(b.ctx)
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

// runBorgCreate runs the actual backup job.
// It is long running and should be run in a goroutine.
func (b *BackupClient) runBorgCreate(backupJob types.BackupJob) {
	repoLock := b.state.GetRepoLock(backupJob.Id)
	repoLock.Lock()
	defer repoLock.Unlock()
	defer b.state.DeleteRepoLock(backupJob.Id)
	b.state.AddRunningBackup(b.ctx, backupJob.Id)
	defer b.state.RemoveRunningBackup(backupJob.Id)

	// Prepare backup command
	name := fmt.Sprintf("%s-%s", backupJob.Prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))
	cmd := exec.CommandContext(b.ctx, b.config.BorgPath, append([]string{
		"create",
		fmt.Sprintf("%s::%s", backupJob.RepoUrl, name)},
		backupJob.Directories...,
	)...)
	cmd.Env = util.BorgEnv{}.WithPassword(backupJob.RepoPassword).AsList()

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err)).Error()
		b.state.AddNotification(errMsg, LevelError)
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
		if !backupJob.IsQuiet {
			b.state.AddNotification(fmt.Sprintf("Backup job completed in %s", time.Since(startTime)), LevelInfo)
		}
	}
}

func (b *BackupClient) startBackupJob(bId types.BackupIdentifier, isQuiet bool) error {
	repo, err := b.getRepoWithCompletedBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	go b.runBorgCreate(types.BackupJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
		Directories:  backupProfile.Directories,
		IsQuiet:      isQuiet,
	})
	return nil
}

// StartBackupJob starts a backup job for the given repository and backup profile.
// TODO: rename to StartBackupJob
func (b *BackupClient) StartBackupJob(backupProfileId int, repositoryId int) error {
	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if canRun, reason := b.state.CanRunBackup(bId); !canRun {
		return fmt.Errorf(reason)
	}

	return b.startBackupJob(bId, false)
}

// TODO: do we need this?
func (b *BackupClient) StartBackupJobs(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		err := b.StartBackupJob(backupProfileId, repo.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BackupClient) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(b.ctx, runtime.OpenDialogOptions{})
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