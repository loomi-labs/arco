package app

import (
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"fmt"
	"time"
)

func (a *App) scheduleBackups() {
	a.log.Info("Scheduling backups")

	// Get the duration until the next backup
	allBs, err := a.getBackupSchedules()
	if err != nil {
		a.log.Errorf("Failed to get backup schedules: %s", err)
		a.state.AddNotification(fmt.Sprintf("Failed to get backup schedules: %s", err), types.LevelError)
		return
	}

	for _, bs := range allBs {
		backupProfileId := bs.Edges.BackupProfile.ID
		repositoryId := bs.Edges.BackupProfile.Edges.Repositories[0].ID
		backupId := types.BackupId{
			BackupProfileId: backupProfileId,
			RepositoryId:    repositoryId,
		}
		a.scheduleBackup(bs, backupId)
	}
}

func (a *App) scheduleBackup(bs *ent.BackupSchedule, backupId types.BackupId) {
	// Calculate the duration until the next backup
	durationUntilNextBackup := bs.NextRun.Sub(time.Now())
	if durationUntilNextBackup < 0 {
		// If the duration is negative, schedule the backup immediately
		durationUntilNextBackup = 0
	}

	// Schedule the backup
	time.AfterFunc(durationUntilNextBackup, func() {
		a.runScheduledBackup(bs, backupId)
	})
	a.log.Info(fmt.Sprintf("Scheduled backup %s in %s", backupId, durationUntilNextBackup))
}

func (a *App) runScheduledBackup(bs *ent.BackupSchedule, backupId types.BackupId) {
	a.log.Infof("Running scheduled backup for %s", backupId)
	var lastRunStatus string
	err := a.BackupClient().startBackupJob(backupId)
	if err != nil {
		lastRunStatus = err.Error()
		a.log.Error(fmt.Sprintf("Failed to run scheduled backup: %s", err))
		a.state.AddNotification(fmt.Sprintf("Failed to run scheduled backup: %s", err), types.LevelError)
	}
	updated, err := a.updateBackupSchedule(bs, lastRunStatus)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to save backup run: %s", err))
		a.state.AddNotification(fmt.Sprintf("Failed to save backup run: %s", err), types.LevelError)
	} else {
		a.scheduleBackup(updated, backupId)
	}
}

func (a *App) updateBackupSchedule(bs *ent.BackupSchedule, lastRunStatus string) (*ent.BackupSchedule, error) {
	lastRunTime := time.Now()
	update := bs.Update()
	update.SetLastRun(lastRunTime)
	if lastRunStatus != "" {
		update.SetNillableLastRunStatus(&lastRunStatus)
	}
	nextBackupTime, err := a.getNextBackupTime(bs, lastRunTime)
	if err != nil {
		return nil, err
	}
	update.SetNextRun(nextBackupTime)
	return update.Save(a.ctx)
}

func (a *App) getBackupSchedules() ([]*ent.BackupSchedule, error) {
	return a.db.BackupSchedule.
		Query().
		Where(
			backupschedule.HasBackupProfileWith(
				backupprofile.IsSetupCompleteEQ(true))).
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithRepositories()
		}).
		All(a.ctx)
}

func weekdayToTimeWeekday(weekday backupschedule.Weekday) time.Weekday {
	switch weekday {
	case backupschedule.WeekdayMonday:
		return time.Monday
	case backupschedule.WeekdayTuesday:
		return time.Tuesday
	case backupschedule.WeekdayWednesday:
		return time.Wednesday
	case backupschedule.WeekdayThursday:
		return time.Thursday
	case backupschedule.WeekdayFriday:
		return time.Friday
	case backupschedule.WeekdaySaturday:
		return time.Saturday
	case backupschedule.WeekdaySunday:
		return time.Sunday
	}
	return time.Monday
}

// getNextBackupTime calculates the next time a backup should run based on the schedule
func (a *App) getNextBackupTime(bs *ent.BackupSchedule, fromTime time.Time) (time.Time, error) {
	if bs.Hourly {
		return fromTime.Truncate(time.Hour).Add(time.Hour), nil
	}
	if bs.DailyAt != nil {
		// Calculate the wanted duration from the beginning of the day
		wantedDuration :=
			time.Duration(bs.DailyAt.Hour())*time.Hour + // hours
				time.Duration(bs.DailyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the day for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff < 0 {
			// If the difference is negative, we already passed the time for today
			// so we return the time for tomorrow
			return fromTime.Add(diff).AddDate(0, 0, 1), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	}
	if bs.WeeklyAt != nil && bs.Weekday != nil {
		// Calculate the wanted duration from the beginning of the week
		wantedDuration :=
			time.Duration(weekdayToTimeWeekday(*bs.Weekday))*24*time.Hour + // days
				time.Duration(bs.WeeklyAt.Hour())*time.Hour + // hours
				time.Duration(bs.WeeklyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the week for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Weekday())*24*time.Hour + // days
				time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff < 0 {
			// If the difference is negative, we already passed the time for this week
			// so we return the time for next week
			return fromTime.Add(diff).AddDate(0, 0, 7), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	}
	if bs.MonthlyAt != nil && bs.Monthday != nil {
		// Calculate the wanted duration from the beginning of the month
		wantedDuration :=
			time.Duration(*bs.Monthday-1)*24*time.Hour + // days
				time.Duration(bs.MonthlyAt.Hour())*time.Hour + // hours
				time.Duration(bs.MonthlyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the month for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Day()-1)*24*time.Hour + // days
				time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff < 0 {
			// If the difference is negative, we already passed the time for this month
			// so we return the time for next month
			return fromTime.Add(diff).AddDate(0, 1, 0), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	}
	return time.Time{}, fmt.Errorf("no valid schedule found")
}
