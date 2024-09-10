package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupschedule"
	"testing"
	"time"
)

/*

TEST CASES - scheduler.go

* getNextBackupTime - hourly - from now
* getNextBackupTime - hourly - from 2024-01-01 at 00:59
* getNextBackupTime - hourly - from 2024-01-01 at 01:00
* getNextBackupTime daily at 10:15 - from today at 9:00
* getNextBackupTime daily at 10:30 - from 2024-01-01 00:00
* getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 9:00
* getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 11:00
* getNextBackupTime weekly at 10:15 on Monday - from 2024-01-01 00:00
* getNextBackupTime weekly at 10:15 on Sunday - from 2024-01-01 00:00
* getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 9:00
* getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 11:00
* getNextBackupTime monthly at 10:15 on the 1th - from 2024-01-01 00:00
* getNextBackupTime monthly at 10:15 on the 30th - from 2024-01-01 00:00
* getNextBackupTime monthly at 10:15 on the 29th - from 2024-02-01 00:00 (february has 29 days in 2024)
* getNextBackupTime monthly at 10:15 on the 30th - from 2024-02-01 00:00 (february has 29 days in 2024)

* delete backup profile
* backup schedule on incomplete backup profile

*/

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
		if err != nil {
			t.Fatalf("Failed to create new backup profile: %v", err)
		}
		profile = p
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
		{"getNextBackupTime - hourly - from now", ent.BackupSchedule{Hourly: true}, now, now.Add(time.Hour).Truncate(time.Hour), false},
		{"getNextBackupTime - hourly - from 2024-01-01 at 00:59", ent.BackupSchedule{Hourly: true}, firstOfJanuary2024.Add(time.Minute * 59), parseX("2024-01-01 01:00:00"), false},
		{"getNextBackupTime - hourly - from 2024-01-01 at 01:00", ent.BackupSchedule{Hourly: true}, firstOfJanuary2024.Add(time.Hour), parseX("2024-01-01 02:00:00"), false},
		{"getNextBackupTime daily at 10:15 - from today at 9:00", ent.BackupSchedule{DailyAt: hourMinutePtr(now, 10, 15)}, hourMinute(now, 9, 0), hourMinute(now, 10, 15), false},
		{"getNextBackupTime daily at 10:15 - from today at 11:00", ent.BackupSchedule{DailyAt: hourMinutePtr(now, 10, 15)}, hourMinute(now, 11, 0), hourMinute(now.AddDate(0, 0, 1), 10, 15), false},
		{"getNextBackupTime daily at 10:30 - from 2024-01-01 00:00", ent.BackupSchedule{DailyAt: hourMinutePtr(firstOfJanuary2024, 10, 30)}, firstOfJanuary2024, parseX("2024-01-01 10:30:00"), false},
		{"getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 9:00", ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdayWednesday)}, weekdayHourMinute(now, time.Wednesday, 9, 0), weekdayHourMinute(now, time.Wednesday, 10, 15), false},
		{"getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 11:00", ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdayWednesday)}, weekdayHourMinute(now, time.Wednesday, 11, 0), weekdayHourMinute(now.AddDate(0, 0, 7), time.Wednesday, 10, 15), false},
		{"getNextBackupTime weekly at 10:15 on Monday - from 2024-01-01 00:00", ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdayMonday)}, firstOfJanuary2024, parseX("2024-01-01 10:15:00"), false},
		{"getNextBackupTime weekly at 10:15 on Sunday - from 2024-01-01 00:00", ent.BackupSchedule{WeeklyAt: hourMinutePtr(now, 10, 15), Weekday: weekdayPtr(backupschedule.WeekdaySunday)}, firstOfJanuary2024, parseX("2024-01-07 10:15:00"), false},
		{"getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 9:00", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{5}[0]}, monthdayHourMinute(now, 5, 9, 0), monthdayHourMinute(now, 5, 10, 15), false},
		{"getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 11:00", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{5}[0]}, monthdayHourMinute(now, 5, 11, 0), monthdayHourMinute(now.AddDate(0, 1, 0), 5, 10, 15), false},
		{"getNextBackupTime monthly at 10:15 on the 1th - from 2024-01-01 00:00", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{1}[0]}, firstOfJanuary2024, parseX("2024-01-01 10:15:00"), false},
		{"getNextBackupTime monthly at 10:15 on the 30th - from 2024-01-01 00:00", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{30}[0]}, firstOfJanuary2024, parseX("2024-01-30 10:15:00"), false},
		{"getNextBackupTime monthly at 10:15 on the 29th - from 2024-02-01 00:00 (february has 29 days)", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{29}[0]}, parseX("2024-02-01 00:00:00"), parseX("2024-02-29 10:15:00"), false},
		{"getNextBackupTime monthly at 10:15 on the 30th - from 2024-02-01 00:00 (february has 29 days in 2024)", ent.BackupSchedule{MonthlyAt: hourMinutePtr(now, 10, 15), Monthday: &[]uint8{30}[0]}, parseX("2024-02-01 00:00:00"), parseX("2024-02-29 10:15:00"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)

			// ARRANGE
			err := a.BackupClient().SaveBackupSchedule(profile.ID, tt.schedule)
			if err != nil {
				t.Fatalf("Failed to save backup schedule: %v", err)
			}

			// ACT
			nextTime, err := getNextBackupTime(&tt.schedule, tt.fromTime)

			// ASSERT
			if (err != nil) != tt.wantErr {
				t.Errorf("getNextBackupTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && nextTime != tt.wantTime {
				t.Errorf("getNextBackupTime() = %v, want %v", nextTime, tt.wantTime)
			}
		})
	}

	t.Run("delete backup profile", func(t *testing.T) {
		setup(t)

		// ARRANGE
		schedule := ent.BackupSchedule{Hourly: true}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		if err != nil {
			t.Fatalf("Failed to save backup schedule: %v", err)
		}

		// ACT
		err = a.BackupClient().DeleteBackupProfile(profile.ID, false)

		// ASSERT
		if err != nil {
			t.Errorf("DeleteBackupProfile() error = %v", err)
		}
	})

	t.Run("backup schedule on incomplete backup profile", func(t *testing.T) {
		setup(t)

		// ARRANGE
		profile.Update().SetIsSetupComplete(false).SaveX(a.ctx)
		schedule := ent.BackupSchedule{
			Hourly: true,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		if err != nil {
			t.Fatalf("Failed to save backup schedule: %v", err)
		}

		// ACT
		schedules, err := a.getBackupSchedules()

		// ASSERT
		if err != nil {
			t.Errorf("getBackupSchedules() error = %v", err)
		}
		if len(schedules) != 0 {
			t.Errorf("Expected no schedules")
		}
	})
}
