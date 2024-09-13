package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupschedule"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func parseX(timeStr string) time.Time {
	expected, err := time.ParseInLocation(time.DateTime, timeStr, time.Local)
	if err != nil {
		panic(err)
	}
	return expected
}

func hourMinute(date time.Time, hour int, minute int) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

func hourMinutePtr(date time.Time, hour int, minute int) *time.Time {
	result := hourMinute(date, hour, minute)
	return &result
}

func weekdayHourMinute(date time.Time, weekday time.Weekday, hour int, minute int) time.Time {
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, 1)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

func monthdayHourMinute(date time.Time, monthday uint8, hour int, minute int) time.Time {
	for uint8(date.Day()) != monthday {
		date = date.AddDate(0, 0, 1)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

func TestScheduler(t *testing.T) {
	var a *App
	var profile *ent.BackupProfile
	var now = time.Now()
	var firstOfJanuary2024 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)

	setup := func(t *testing.T) {
		a = NewTestApp(t)
		p, err := a.BackupClient().NewBackupProfile()
		assert.NoError(t, err, "Failed to create new backup profile")
		profile, err = a.BackupClient().SaveBackupProfile(*p)
		assert.NoError(t, err, "Failed to save backup profile")
		assert.NotNil(t, profile, "Expected backup profile, got nil")
		now = time.Now()
	}

	weekdayPtr := func(w backupschedule.Weekday) *backupschedule.Weekday {
		return &w
	}

	tests := []struct {
		name     string
		schedule ent.BackupSchedule
		fromTime time.Time
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:     "getNextBackupTime - hourly - from now",
			schedule: ent.BackupSchedule{Hourly: true},
			fromTime: now,
			wantTime: now.Add(time.Hour).Truncate(time.Hour),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime - hourly - from 2024-01-01 at 00:59",
			schedule: ent.BackupSchedule{Hourly: true},
			fromTime: firstOfJanuary2024.Add(time.Minute * 59),
			wantTime: parseX("2024-01-01 01:00:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime - hourly - from 2024-01-01 at 01:00",
			schedule: ent.BackupSchedule{Hourly: true},
			fromTime: firstOfJanuary2024.Add(time.Hour),
			wantTime: parseX("2024-01-01 02:00:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime daily at 10:15 - from today at 9:00",
			schedule: ent.BackupSchedule{DailyAt: hourMinutePtr(now, 10, 15)},
			fromTime: hourMinute(now, 9, 0),
			wantTime: hourMinute(now, 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime daily at 10:15 - from today at 11:00",
			schedule: ent.BackupSchedule{DailyAt: hourMinutePtr(now, 10, 15)},
			fromTime: hourMinute(now, 11, 0),
			wantTime: hourMinute(now.AddDate(0, 0, 1), 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime daily at 10:30 - from 2024-01-01 00:00",
			schedule: ent.BackupSchedule{DailyAt: hourMinutePtr(firstOfJanuary2024, 10, 30)},
			fromTime: firstOfJanuary2024,
			wantTime: parseX("2024-01-01 10:30:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 9:00",
			schedule: ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdayWednesday)},
			fromTime: weekdayHourMinute(now, time.Wednesday, 9, 0),
			wantTime: weekdayHourMinute(now, time.Wednesday, 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 11:00",
			schedule: ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdayWednesday)},
			fromTime: weekdayHourMinute(now, time.Wednesday, 11, 0),
			wantTime: weekdayHourMinute(now.AddDate(0, 0, 7), time.Wednesday, 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 9:00",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{5}[0]},
			fromTime: monthdayHourMinute(now, 5, 9, 0),
			wantTime: monthdayHourMinute(now, 5, 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 11:00",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{5}[0]},
			fromTime: monthdayHourMinute(now, 5, 11, 0),
			wantTime: monthdayHourMinute(now.AddDate(0, 1, 0), 5, 10, 15),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 1th - from 2024-01-01 00:00",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{1}[0]},
			fromTime: firstOfJanuary2024,
			wantTime: parseX("2024-01-01 10:15:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 30th - from 2024-01-01 00:00",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{30}[0]},
			fromTime: firstOfJanuary2024,
			wantTime: parseX("2024-01-30 10:15:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 29th - from 2024-02-01 00:00 (february has 29 days)",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{29}[0]},
			fromTime: parseX("2024-02-01 00:00:00"),
			wantTime: parseX("2024-02-29 10:15:00"),
			wantErr:  false,
		},
		{
			name:     "getNextBackupTime monthly at 10:15 on the 30th - from 2024-02-01 00:00 (february has 29 days in 2024)",
			schedule: ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{30}[0]},
			fromTime: parseX("2024-02-01 00:00:00"),
			wantTime: parseX("2024-02-29 10:15:00"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)

			// ARRANGE
			err := a.BackupClient().SaveBackupSchedule(profile.ID, tt.schedule)
			assert.NoError(t, err, "Failed to save backup schedule")

			// ACT
			nextTime, err := getNextBackupTime(&tt.schedule, tt.fromTime)

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoError(t, err, "Expected no error, got %v", err)
				assert.Equal(t, tt.wantTime, nextTime, "getNextBackupTime() = %v, want %v", nextTime, tt.wantTime)
			}
		})
	}

	t.Run("delete backup profile", func(t *testing.T) {
		setup(t)

		// ARRANGE
		schedule := ent.BackupSchedule{Hourly: true}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		assert.NoError(t, err, "Failed to save backup schedule")

		// ACT
		err = a.BackupClient().DeleteBackupProfile(profile.ID, false)

		// ASSERT
		assert.NoError(t, err, "DeleteBackupProfile() error = %v", err)
	})

	t.Run("backup schedule on incomplete backup profile", func(t *testing.T) {
		setup(t)

		// ARRANGE
		profile.Update().SetIsSetupComplete(false).SaveX(a.ctx)
		schedule := ent.BackupSchedule{
			Hourly: true,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		assert.NoError(t, err, "Failed to save backup schedule")

		// ACT
		schedules, err := a.getBackupSchedules()

		// ASSERT
		assert.NoError(t, err, "getBackupSchedules() error = %v", err)
		assert.Empty(t, schedules, "Expected no schedules")
	})
}
