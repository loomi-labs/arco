package app

import (
	"github.com/loomi-labs/arco/backend/borg/mockborg"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

/*
TEST CASES - backup.go

TestBackupClient_SaveBackupSchedule
* SaveBackupSchedule with invalid schedule
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

TestBackupClient_GetPrefixSuggestions
* GetPrefixSuggestions with empty prefix
* GetPrefixSuggestions with alphanumeric prefix
* GetPrefixSuggestions with only non-alphanumeric prefix
* GetPrefixSuggestions with existing prefix and non-alphanumeric chars
* GetPrefixSuggestions with existing prefix
* GetPrefixSuggestions with underscore and hyphen
*/

func TestBackupClient_SaveBackupSchedule(t *testing.T) {
	var a *App
	var mockBorg *mockborg.MockBorg
	var profile *ent.BackupProfile
	var bs *ent.BackupSchedule
	var now = time.Now()

	setup := func(t *testing.T) {
		a, mockBorg = NewTestApp(t)
		p, err := a.BackupClient().NewBackupProfile()
		assert.NoError(t, err, "Failed to create new backup profile")
		p.Name = "Test profile"
		p.Prefix = "test-"
		bs = p.Edges.BackupSchedule

		mockBorg.EXPECT().Init(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		r, err := a.RepoClient().Create("Test profile", "test-", "test", false)
		assert.NoError(t, err, "Failed to create new repository")

		profile, err = a.BackupClient().CreateBackupProfile(*p, []int{r.ID})
		assert.NoError(t, err, "Failed to save backup profile")
		assert.NotNil(t, profile, "Expected backup profile, got nil")
	}

	newBackupSchedule := func(overrides ent.BackupSchedule) ent.BackupSchedule {
		if overrides.Mode != "" {
			bs.Mode = overrides.Mode
		}
		if !overrides.DailyAt.IsZero() {
			bs.DailyAt = overrides.DailyAt
		}
		if overrides.Weekday != "" {
			bs.Weekday = overrides.Weekday
		}
		if !overrides.WeeklyAt.IsZero() {
			bs.WeeklyAt = overrides.WeeklyAt
		}
		if overrides.Monthday != 0 {
			bs.Monthday = overrides.Monthday
		}
		if !overrides.MonthlyAt.IsZero() {
			bs.MonthlyAt = overrides.MonthlyAt
		}
		return *bs
	}

	tests := []struct {
		name     string
		schedule ent.BackupSchedule
		wantErr  bool
	}{
		{
			name:     "SaveBackupSchedule with invalid schedule",
			schedule: ent.BackupSchedule{Mode: "invalid"},
			wantErr:  true,
		},
		{
			name:     "SaveBackupSchedule with hourly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeHourly},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with daily schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeDaily, DailyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with weekly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeWeekly, Weekday: backupschedule.WeekdayMonday, WeeklyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with invalid weekly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeWeekly, Weekday: "invalid", WeeklyAt: now},
			wantErr:  true,
		},
		{
			name:     "SaveBackupSchedule with monthly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeMonthly, Monthday: []uint8{1}[0], MonthlyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with invalid monthly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeMonthly, Monthday: []uint8{32}[0], MonthlyAt: now},
			wantErr:  true,
		},
		{
			name:     "SaveBackupSchedule with hourly and daily schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeHourly, DailyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with hourly and weekly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeHourly, Weekday: backupschedule.WeekdayMonday, WeeklyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with hourly and monthly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeHourly, Monthday: []uint8{1}[0], MonthlyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with daily and weekly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeDaily, DailyAt: now, Weekday: backupschedule.WeekdayMonday, WeeklyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with daily and monthly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeDaily, DailyAt: now, Monthday: []uint8{1}[0], MonthlyAt: now},
			wantErr:  false,
		},
		{
			name:     "SaveBackupSchedule with weekly and monthly schedule",
			schedule: ent.BackupSchedule{Mode: backupschedule.ModeWeekly, Weekday: backupschedule.WeekdayMonday, WeeklyAt: now, Monthday: []uint8{1}[0], MonthlyAt: now},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			// ACT
			err := a.BackupClient().SaveBackupSchedule(profile.ID, newBackupSchedule(tt.schedule))

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoError(t, err, "Expected no error, got %v", err)

				updatedSchedule := a.db.BackupSchedule.
					Query().
					Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).
					OnlyX(a.ctx)

				cnt := a.db.BackupSchedule.Query().CountX(a.ctx)

				assert.Equalf(t, newBackupSchedule(tt.schedule).Mode, updatedSchedule.Mode, "Expected mode %s, got %s", newBackupSchedule(tt.schedule).Mode, updatedSchedule.Mode)
				//assert.Equalf(t, newBackupSchedule(tt.schedule).DailyAt.Unix(), updatedSchedule.DailyAt.Unix(), "Expected daily at %v, got %v", newBackupSchedule(tt.schedule).DailyAt, updatedSchedule.DailyAt)
				assert.Equalf(t, newBackupSchedule(tt.schedule).Weekday, updatedSchedule.Weekday, "Expected weekday %s, got %s", newBackupSchedule(tt.schedule).Weekday, updatedSchedule.Weekday)
				//assert.Equalf(t, newBackupSchedule(tt.schedule).WeeklyAt.Unix(), updatedSchedule.WeeklyAt.Unix(), "Expected weekly at %v, got %v", newBackupSchedule(tt.schedule).WeeklyAt, updatedSchedule.WeeklyAt)
				assert.Equalf(t, newBackupSchedule(tt.schedule).Monthday, updatedSchedule.Monthday, "Expected monthday %d, got %d", newBackupSchedule(tt.schedule).Monthday, updatedSchedule.Monthday)
				//assert.Equalf(t, newBackupSchedule(tt.schedule).MonthlyAt.Unix(), updatedSchedule.MonthlyAt.Unix(), "Expected monthly at %v, got %v", newBackupSchedule(tt.schedule).MonthlyAt, updatedSchedule.MonthlyAt)
				assert.Equalf(t, 1, cnt, "Expected 1 backup schedule, got %d", cnt)
			}
		})
	}
}

func TestBackupClient_GetPrefixSuggestions(t *testing.T) {
	var a *App
	var mockBorg *mockborg.MockBorg
	var profile *ent.BackupProfile

	setup := func(t *testing.T) {
		a, mockBorg = NewTestApp(t)
		p, err := a.BackupClient().NewBackupProfile()
		assert.NoError(t, err, "Failed to create new backup profile")
		p.Name = "Test profile"
		p.Prefix = "test-"

		mockBorg.EXPECT().Init(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		r, err := a.RepoClient().Create("Test profile", "test-", "test", false)
		assert.NoError(t, err, "Failed to create new repository")

		profile, err = a.BackupClient().CreateBackupProfile(*p, []int{r.ID})
		assert.NoError(t, err, "Failed to save backup profile")
		assert.NotNil(t, profile, "Expected backup profile, got nil")
	}

	type expectedPrefix struct {
		prefix     string
		exactMatch bool
	}

	tests := []struct {
		name           string
		prefix         string
		expectedPrefix *expectedPrefix
		wantErr        bool
	}{
		{"GetPrefixSuggestions with empty prefix", "", nil, true},
		{"GetPrefixSuggestions with alphanumeric prefix", "test123", &expectedPrefix{"test123-", true}, false},
		{"GetPrefixSuggestions with only non-alphanumeric prefix", "!@#", nil, true},
		{"GetPrefixSuggestions with existing prefix and non-alphanumeric chars", "test!@#", &expectedPrefix{"test", false}, false},
		{"GetPrefixSuggestions with existing prefix", "test", &expectedPrefix{"test", false}, false},
		{"GetPrefixSuggestions with underscore and hyphen", "this-is_a.test", &expectedPrefix{"thisisatest-", true}, false},
		{"GetPrefixSuggestions with uppercase prefix", "TEST123", &expectedPrefix{"test123-", true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			// ACT
			suggestion, err := a.BackupClient().GetPrefixSuggestion(tt.prefix)

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoErrorf(t, err, "Expected no error, got %v", err)
				assert.NotNil(t, suggestion, "Expected suggestion, got nil")
				if tt.expectedPrefix.exactMatch {
					assert.Equalf(t, tt.expectedPrefix.prefix, suggestion, "Expected prefix %s, got %s", tt.expectedPrefix.prefix, suggestion)
				} else {
					assert.Containsf(t, suggestion, tt.expectedPrefix.prefix, "Expected prefix %s to contain %s", suggestion, tt.expectedPrefix.prefix)
					expectedLen := len(tt.expectedPrefix.prefix) + 5
					assert.Lenf(t, suggestion, expectedLen, "Expected prefix length %d, got %d", expectedLen, len(suggestion))
				}
			}
		})
	}
}
