package app

import (
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/backupschedule"
	"fmt"
	"time"
)

func (a *App) startScheduleChangeListener() {
	var timers []*time.Timer
	for {
		<-a.backupScheduleChangedCh

		// Stop all scheduled backups
		for _, t := range timers {
			t.Stop()
		}

		// Schedule all backups
		timers = a.scheduleBackups()
	}
}

func (a *App) scheduleBackups() []*time.Timer {
	a.log.Info("Scheduling backups")

	allBs, err := a.getBackupSchedules()
	if err != nil {
		a.log.Errorf("Failed to get backup schedules: %s", err)
		a.state.AddNotification(fmt.Sprintf("Failed to get backup schedules: %s", err), types.LevelError)
		return nil
	}

	var timers []*time.Timer
	for _, bs := range allBs {
		backupProfileId := bs.Edges.BackupProfile.ID
		repositoryId := bs.Edges.BackupProfile.Edges.Repositories[0].ID
		backupId := types.BackupId{
			BackupProfileId: backupProfileId,
			RepositoryId:    repositoryId,
		}
		timer := a.scheduleBackup(bs, backupId)
		timers = append(timers, timer)
	}
	return timers
}

func (a *App) scheduleBackup(bs *ent.BackupSchedule, backupId types.BackupId) *time.Timer {
	// Calculate the duration until the next backup
	durationUntilNextBackup := time.Until(bs.NextRun)
	if durationUntilNextBackup < 0 {
		// If the duration is negative, schedule the backup immediately
		durationUntilNextBackup = 0
	}

	// Schedule the backup
	timer := time.AfterFunc(durationUntilNextBackup, func() {
		a.runScheduledBackup(bs, backupId)
	})
	a.log.Info(fmt.Sprintf("Scheduled backup %s in %s", backupId, durationUntilNextBackup))
	return timer
}

func (a *App) runScheduledBackup(bs *ent.BackupSchedule, backupId types.BackupId) {
	// Check if the backup schedule still exists
	// This is necessary because the backup schedule might have been deleted or modified (modified -> deleted and recreated)
	exist, err := a.db.BackupSchedule.
		Query().
		Where(backupschedule.ID(bs.ID)).
		Exist(a.ctx)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to check if backup schedule exists: %s", err))
		a.state.AddNotification(fmt.Sprintf("Failed to run scheduled backup: %s", err), types.LevelError)
		return
	}
	if !exist {
		a.log.Infof("Backup schedule %d does not exist anymore, skipping", bs.ID)
		return
	}

	// Run the backup
	a.log.Infof("Running scheduled backup for %s", backupId)
	var lastRunStatus string
	result, err := a.BackupClient().runBorgCreate(backupId)
	if err != nil {
		lastRunStatus = fmt.Sprintf("error: %s", err)
		a.log.Error(fmt.Sprintf("Failed to run scheduled backup: %s", err))
		a.state.AddNotification(fmt.Sprintf("Failed to run scheduled backup: %s", err), types.LevelError)
	} else {
		lastRunStatus = result.String()
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
	nextBackupTime, err := getNextBackupTime(bs, lastRunTime)
	if err != nil {
		return nil, err
	}
	update.SetNextRun(nextBackupTime)
	return update.Save(a.ctx)
}

func (a *App) getBackupSchedules() ([]*ent.BackupSchedule, error) {
	return a.db.BackupSchedule.
		Query().
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
func getNextBackupTime(bs *ent.BackupSchedule, fromTime time.Time) (time.Time, error) {
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
		monthday := *bs.Monthday

		// If we are in February and the monthday is 29 or 30, we use the last day of the month
		if fromTime.Month() == time.February && monthday > 28 {
			// Check if the year is a leap year
			year := fromTime.Year()
			if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
				// If it is a leap year, we use 29
				if monthday > 29 {
					monthday = 29
				}
			} else {
				// If it is not a leap year, we use 28
				monthday = 28
			}
		}

		// Calculate the wanted duration from the beginning of the month
		wantedDuration :=
			time.Duration(monthday-1)*24*time.Hour + // days
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
