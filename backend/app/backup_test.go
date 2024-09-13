package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

/*

TEST CASES - backup.go

* SaveBackupSchedule with default values
* SaveBackupSchedule with hourly schedule
* SaveBackupSchedule with daily schedule
* SaveBackupSchedule with weekly schedule
* SaveBackupSchedule with invalid weekly schedule
* SaveBackupSchedule with monthly schedule
* SaveBackupSchedule with invalid monthly schedule
* SaveBackupSchedule with hourly and daily schedule
* SaveBackupSchedule with hourly and weekly schedule
* SaveBackupSchedule with hourly and monthly schedule
* SaveBackupSchedule with daily and weekly schedule
* SaveBackupSchedule with daily and monthly schedule
* SaveBackupSchedule with weekly and monthly schedule

* SaveBackupSchedule with an updated daily schedule
* SaveBackupSchedule with an updated weekly schedule (to hourly)

*/

func TestBackupClient_SaveBackupSchedule(t *testing.T) {
	var a *App
	var profile *ent.BackupProfile
	var now = time.Time{}

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
		wantErr  bool
	}{
		{"SaveBackupSchedule with default values", ent.BackupSchedule{}, true},
		{"SaveBackupSchedule with hourly schedule", ent.BackupSchedule{Hourly: true}, false},
		{"SaveBackupSchedule with daily schedule", ent.BackupSchedule{DailyAt: &now}, false},
		{"SaveBackupSchedule with weekly schedule", ent.BackupSchedule{Weekday: weekdayPtr(backupschedule.WeekdayMonday), WeeklyAt: &now}, false},
		{"SaveBackupSchedule with invalid weekly schedule", ent.BackupSchedule{Weekday: weekdayPtr("invalid"), WeeklyAt: &now}, true},
		{"SaveBackupSchedule with monthly schedule", ent.BackupSchedule{Monthday: &[]uint8{1}[0], MonthlyAt: &now}, false},
		{"SaveBackupSchedule with invalid monthly schedule", ent.BackupSchedule{Monthday: &[]uint8{32}[0], MonthlyAt: &now}, true},
		{"SaveBackupSchedule with hourly and daily schedule", ent.BackupSchedule{Hourly: true, DailyAt: &now}, true},
		{"SaveBackupSchedule with hourly and weekly schedule", ent.BackupSchedule{Hourly: true, Weekday: weekdayPtr(backupschedule.WeekdayMonday), WeeklyAt: &now}, true},
		{"SaveBackupSchedule with hourly and monthly schedule", ent.BackupSchedule{Hourly: true, Monthday: &[]uint8{1}[0], MonthlyAt: &now}, true},
		{"SaveBackupSchedule with daily and weekly schedule", ent.BackupSchedule{DailyAt: &now, Weekday: weekdayPtr(backupschedule.WeekdayMonday), WeeklyAt: &now}, true},
		{"SaveBackupSchedule with daily and monthly schedule", ent.BackupSchedule{DailyAt: &now, Monthday: &[]uint8{1}[0], MonthlyAt: &now}, true},
		{"SaveBackupSchedule with weekly and monthly schedule", ent.BackupSchedule{Weekday: weekdayPtr(backupschedule.WeekdayMonday), WeeklyAt: &now, Monthday: &[]uint8{1}[0], MonthlyAt: &now}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			// ACT
			err := a.BackupClient().SaveBackupSchedule(profile.ID, tt.schedule)

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoError(t, err, "Expected no error, got %v", err)
			}
		})
	}

	t.Run("SaveBackupSchedule with an updated schedule", func(t *testing.T) {
		setup(t)
		// ARRANGE
		schedule := ent.BackupSchedule{DailyAt: &now}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		assert.NoError(t, err, "Expected no error, got %v", err)
		bsId1 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		// ACT
		updatedHour := schedule.DailyAt.Add(time.Hour)
		schedule.DailyAt = &updatedHour
		err = a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		profile = a.db.BackupProfile.Query().Where(backupprofile.ID(profile.ID)).WithBackupSchedule().OnlyX(a.ctx)
		bsId2 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		assert.NoError(t, err, "Expected no error, got %v", err)
		assert.Equal(t, 1, a.db.BackupSchedule.Query().CountX(a.ctx), "Expected 1 backup schedule, got %d", a.db.BackupSchedule.Query().CountX(a.ctx))
		assert.NotEqual(t, bsId1, bsId2, "Expected different backup schedule IDs, got the same")
		assert.Equal(t, bsId2, profile.Edges.BackupSchedule.ID, "Expected backup schedule ID %d, got %d", bsId2, profile.Edges.BackupSchedule.ID)
		assert.Equal(t, updatedHour.Unix(), profile.Edges.BackupSchedule.DailyAt.Unix(), "Expected updated hour %v, got %v", updatedHour, profile.Edges.BackupSchedule.DailyAt)
	})

	t.Run("SaveBackupSchedule with an updated weekly schedule (to hourly)", func(t *testing.T) {
		setup(t)
		// ARRANGE
		weekday := backupschedule.WeekdayWednesday
		schedule := ent.BackupSchedule{Weekday: &weekday, WeeklyAt: &now}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		assert.NoError(t, err, "Expected no error, got %v", err)
		bsId1 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		// ACT
		schedule.Hourly = true
		schedule.WeeklyAt = nil
		schedule.Weekday = nil
		err = a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		profile = a.db.BackupProfile.Query().Where(backupprofile.ID(profile.ID)).WithBackupSchedule().OnlyX(a.ctx)
		bsId2 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		assert.NoError(t, err, "Expected no error, got %v", err)
		assert.Equal(t, 1, a.db.BackupSchedule.Query().CountX(a.ctx), "Expected 1 backup schedule, got %d", a.db.BackupSchedule.Query().CountX(a.ctx))
		assert.NotEqual(t, bsId1, bsId2, "Expected different backup schedule IDs, got the same")
		assert.Equal(t, bsId2, profile.Edges.BackupSchedule.ID, "Expected backup schedule ID %d, got %d", bsId2, profile.Edges.BackupSchedule.ID)
		assert.True(t, profile.Edges.BackupSchedule.Hourly, "Expected hourly schedule to be true, got false")
		assert.Nil(t, profile.Edges.BackupSchedule.WeeklyAt, "Expected weekly schedule to be nil, got %v", profile.Edges.BackupSchedule.WeeklyAt)
	})
}
