package app

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/pruningrule"
	"github.com/negrel/assert"
	"time"
)

func (a *App) startScheduleChangeListener() {
	a.log.Debug("Starting schedule change listener")
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

/***********************************/
/********** Backup Scheduling ******/
/***********************************/

func (a *App) scheduleBackups() []*time.Timer {
	a.log.Info("Scheduling backups")

	allBs, err := a.getBackupSchedules()
	if err != nil {
		a.log.Errorf("Failed to get backup schedules: %s", err)
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to get backup schedules: %s", err), types.LevelError)
		return nil
	}

	var timers []*time.Timer
	for _, bs := range allBs {
		backupProfileId := bs.Edges.BackupProfile.ID

		assert.NotNil(bs.Edges.BackupProfile.Edges.Repositories, "repositories is nil")

		for _, r := range bs.Edges.BackupProfile.Edges.Repositories {
			backupId := types.BackupId{
				BackupProfileId: backupProfileId,
				RepositoryId:    r.ID,
			}
			timer := a.scheduleBackup(bs, backupId)
			timers = append(timers, timer)
		}
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
	updatedAt := bs.UpdatedAt
	timer := time.AfterFunc(durationUntilNextBackup, func() {
		a.runScheduledBackup(bs, backupId, updatedAt)
	})
	a.log.Info(fmt.Sprintf("Scheduled backup %s in %s", backupId, durationUntilNextBackup))
	return timer
}

func (a *App) runScheduledBackup(bs *ent.BackupSchedule, backupId types.BackupId, updatedAt time.Time) {
	// Check if the backup schedule still exists
	// This is necessary because the backup schedule might have been deleted or modified (modified -> deleted and recreated)
	exist, err := a.db.BackupSchedule.
		Query().
		Where(backupschedule.And(
			backupschedule.ID(bs.ID),
			backupschedule.UpdatedAtEQ(updatedAt),
		)).
		Exist(a.ctx)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to check if backup schedule exists: %s", err))
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to run scheduled backup: %s", err), types.LevelError)
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
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to run scheduled backup: %s", err), types.LevelError)
	} else {
		lastRunStatus = result.String()
	}
	updated, err := a.updateBackupSchedule(bs, lastRunStatus)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to save backup run: %s", err))
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to save backup run: %s", err), types.LevelError)
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
	a.log.Debugf("Next run in %s", time.Until(nextBackupTime))
	update.SetNextRun(nextBackupTime)
	return update.Save(a.ctx)
}

func (a *App) getBackupSchedules() ([]*ent.BackupSchedule, error) {
	return a.db.BackupSchedule.
		Query().
		Where(backupschedule.ModeNEQ(backupschedule.ModeDisabled)).
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
	// Make sure we are working with UTC time (we don't care about the timezone)
	fromTime = fromTime.In(time.UTC)
	switch bs.Mode {
	case backupschedule.ModeHourly:
		return fromTime.Truncate(time.Hour).Add(time.Hour), nil
	case backupschedule.ModeDaily:
		dailyAt := bs.DailyAt.In(time.UTC)
		// Calculate the wanted duration from the beginning of the day
		wantedDuration :=
			time.Duration(dailyAt.Hour())*time.Hour + // hours
				time.Duration(dailyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the day for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff <= 0 {
			// If the difference is negative, we already passed the time for today
			// so we return the time for tomorrow
			return fromTime.Add(diff).AddDate(0, 0, 1), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	case backupschedule.ModeWeekly:
		weeklyAt := bs.WeeklyAt.In(time.UTC)
		// Calculate the wanted duration from the beginning of the week
		wantedDuration :=
			time.Duration(weekdayToTimeWeekday(bs.Weekday))*24*time.Hour + // days
				time.Duration(weeklyAt.Hour())*time.Hour + // hours
				time.Duration(weeklyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the week for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Weekday())*24*time.Hour + // days
				time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff <= 0 {
			// If the difference is negative, we already passed the time for this week
			// so we return the time for next week
			return fromTime.Add(diff).AddDate(0, 0, 7), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	case backupschedule.ModeMonthly:
		monthday := bs.Monthday
		monthlyAt := bs.MonthlyAt.In(time.UTC)

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
				time.Duration(monthlyAt.Hour())*time.Hour + // hours
				time.Duration(monthlyAt.Minute())*time.Minute // minutes

		// Calculate the duration from the beginning of the month for the fromTime
		fromTimeDuration :=
			time.Duration(fromTime.Day()-1)*24*time.Hour + // days
				time.Duration(fromTime.Hour())*time.Hour + // hours
				time.Duration(fromTime.Minute())*time.Minute // minutes

		diff := wantedDuration - fromTimeDuration
		if diff <= 0 {
			// If the difference is negative, we already passed the time for this month
			// so we return the time for next month
			return fromTime.Add(diff).AddDate(0, 1, 0), nil
		}
		// Otherwise we just wait the difference
		return fromTime.Add(diff), nil
	default:
		return time.Time{}, fmt.Errorf("no valid schedule found")
	}
}

/***********************************/
/********** Prune Scheduling *******/
/***********************************/

func (a *App) startPruneScheduleChangeListener() {
	a.log.Debug("Starting prune schedule change listener")
	var timers []*time.Timer
	for {
		<-a.pruningScheduleChangedCh

		// Stop all scheduled prunes
		for _, t := range timers {
			t.Stop()
		}

		// Schedule all prunes
		timers = a.schedulePrunes()
	}
}

func (a *App) schedulePrunes() []*time.Timer {
	a.log.Info("Scheduling prunes")

	pruningRules, err := a.getPruningRules()
	if err != nil {
		a.log.Errorf("Failed to get pruning schedules: %s", err)
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to get pruning rules: %s", err), types.LevelError)
		return nil
	}

	var timers []*time.Timer
	for _, pruningRule := range pruningRules {
		backupProfileId := pruningRule.Edges.BackupProfile.ID

		assert.NotNil(pruningRule.Edges.BackupProfile.Edges.Repositories, "repositories is nil")

		for _, r := range pruningRule.Edges.BackupProfile.Edges.Repositories {
			pruneId := types.BackupId{
				BackupProfileId: backupProfileId,
				RepositoryId:    r.ID,
			}
			timer := a.schedulePrune(pruningRule, pruneId)
			timers = append(timers, timer)
		}
	}
	return timers
}

func (a *App) schedulePrune(ps *ent.PruningRule, backupId types.BackupId) *time.Timer {
	// Calculate the duration until the next prune
	durationUntilNextPrune := time.Until(ps.NextRun)
	if durationUntilNextPrune < 0 {
		// If the duration is negative, schedule the prune immediately
		durationUntilNextPrune = 0
	}

	// Schedule the prune
	updatedAt := ps.UpdatedAt
	timer := time.AfterFunc(durationUntilNextPrune, func() {
		a.runScheduledPrune(ps, backupId, updatedAt)
	})
	a.log.Info(fmt.Sprintf("Scheduled prune %s in %s", backupId, durationUntilNextPrune))
	return timer
}

func (a *App) runScheduledPrune(ps *ent.PruningRule, backupId types.BackupId, updatedAt time.Time) {
	// Check if the prune schedule still exists and has not been modified
	exist, err := a.db.PruningRule.
		Query().
		Where(pruningrule.And(
			pruningrule.ID(ps.ID),
			pruningrule.UpdatedAtEQ(updatedAt),
		)).
		Exist(a.ctx)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to check if prune schedule exists: %s", err))
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to run scheduled prune: %s", err), types.LevelError)
		return
	}
	if !exist {
		a.log.Infof("Prune schedule %d does not exist anymore or has been modified, skipping", ps.ID)
		return
	}

	// Run the prune
	a.log.Infof("Running scheduled prune for %s", backupId)
	var lastRunStatus string
	result, err := a.BackupClient().runPruneJob(backupId)
	if err != nil {
		lastRunStatus = fmt.Sprintf("error: %s", err)
		a.log.Error(fmt.Sprintf("Failed to run scheduled prune: %s", err))
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to run scheduled prune: %s", err), types.LevelError)
	} else {
		lastRunStatus = result.String()
	}
	updated, err := a.updatePruningRule(ps, lastRunStatus)
	if err != nil {
		a.log.Error(fmt.Sprintf("Failed to save prune run: %s", err))
		a.state.AddNotification(a.ctx, fmt.Sprintf("Failed to save prune run: %s", err), types.LevelError)
	} else {
		a.schedulePrune(updated, backupId)
	}
}

func (a *App) updatePruningRule(pruningRule *ent.PruningRule, lastRunStatus string) (*ent.PruningRule, error) {
	lastRunTime := time.Now()
	update := pruningRule.Update()
	update.SetLastRun(lastRunTime)
	if lastRunStatus != "" {
		update.SetNillableLastRunStatus(&lastRunStatus)
	}
	backupSchedule, err := pruningRule.QueryBackupProfile().QueryBackupSchedule().First(a.ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	nextPruneTime := getNextPruneTime(backupSchedule, lastRunTime)
	update.SetNextRun(nextPruneTime)
	return update.Save(a.ctx)
}

func (a *App) getPruningRules() ([]*ent.PruningRule, error) {
	return a.db.PruningRule.
		Query().
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithRepositories()
			q.WithBackupSchedule()
		}).
		Where(pruningrule.IsEnabledEQ(true)).
		All(a.ctx)
}

func getNextPruneTime(bs *ent.BackupSchedule, fromTime time.Time) time.Time {
	if bs == nil {
		// If we don't have a backup schedule, we run the prune once a week
		return fromTime.AddDate(0, 0, 7)
	}

	// Calculate the next prune time based on the backup schedule
	// If the backup run is in the past, we run the prune in 1 hour
	if bs.NextRun.Before(time.Now().Add(time.Hour)) {
		return fromTime.Add(time.Hour)
	}

	// Otherwise we run the prune 1 minute after the backup
	return bs.NextRun.Add(time.Minute)
}
