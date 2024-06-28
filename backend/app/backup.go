package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/repository"
	"arco/backend/types"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"slices"
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

func (b *BackupClient) RunBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if slices.Contains(b.runningBackups, bId) {
		return fmt.Errorf("backup is already running")
	}
	if slices.Contains(b.occupiedRepos, repositoryId) {
		return fmt.Errorf("repository is busy")
	}

	b.runningBackups = append(b.runningBackups, bId)
	b.occupiedRepos = append(b.occupiedRepos, repositoryId)

	b.inChan.StartBackup <- types.BackupJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
		Directories:  backupProfile.Directories,
		BinaryPath:   b.config.BorgPath,
	}
	return nil
}

func (b *BackupClient) RunBackups(backupProfileId int) error {
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
	if doesExist {
		_, err = b.db.BackupSchedule.
			Update().
			Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
			SetHourly(schedule.Hourly).
			SetNillableDailyAt(schedule.DailyAt).
			SetNillableWeeklyAt(schedule.WeeklyAt).
			SetNillableWeekday(schedule.Weekday).
			SetNillableMonthlyAt(schedule.MonthlyAt).
			SetNillableMonthday(schedule.Monthday).
			Save(b.ctx)
		return err
	}
	_, err = b.db.BackupSchedule.
		Create().
		SetHourly(schedule.Hourly).
		SetNillableDailyAt(schedule.DailyAt).
		SetNillableWeeklyAt(schedule.WeeklyAt).
		SetNillableWeekday(schedule.Weekday).
		SetNillableMonthlyAt(schedule.MonthlyAt).
		SetNillableMonthday(schedule.Monthday).
		SetBackupProfileID(backupProfileId).
		Save(b.ctx)
	return err
}

func (b *BackupClient) DeleteBackupSchedule(backupProfileId int) error {
	_, err := b.db.BackupSchedule.
		Delete().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exec(b.ctx)
	return err
}
