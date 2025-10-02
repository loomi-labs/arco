package backup_profile

import (
	"context"
	"testing"
	"time"

	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/enttest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

/*
TEST CASES - backup_profile_service.go

TestBackupProfileService_SaveBackupSchedule
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

TestBackupProfileService_GetPrefixSuggestions
* GetPrefixSuggestions with empty prefix
* GetPrefixSuggestions with alphanumeric prefix
* GetPrefixSuggestions with only non-alphanumeric prefix
* GetPrefixSuggestions with existing prefix and non-alphanumeric chars
* GetPrefixSuggestions with existing prefix
* GetPrefixSuggestions with underscore and hyphen

TestBackupProfileService_DeleteBackupProfile
* DeleteBackupProfile
* DeleteBackupProfile with invalid ID
* DeleteBackupProfile and archives

TestBackupProfileService_RemoveRepositoryFromBackupProfile
* RemoveRepositoryFromBackupProfile
* RemoveRepositoryFromBackupProfile with invalid backup profile ID
* RemoveRepositoryFromBackupProfile with invalid repository ID
* RemoveRepositoryFromBackupProfile and delete archives

*/

// mockRepositoryService implements RepositoryServiceInterface for testing
type mockRepositoryService struct{}

func (m *mockRepositoryService) QueueBackup(ctx context.Context, backupId types.BackupId) (string, error) {
	return "mock-operation-id", nil
}

func (m *mockRepositoryService) QueueBackups(ctx context.Context, backupIds []types.BackupId) ([]string, error) {
	ids := make([]string, len(backupIds))
	for i := range backupIds {
		ids[i] = "mock-operation-id"
	}
	return ids, nil
}

func (m *mockRepositoryService) QueuePrune(ctx context.Context, backupId types.BackupId) (string, error) {
	return "mock-operation-id", nil
}

func (m *mockRepositoryService) QueueArchiveDelete(ctx context.Context, archiveId int) (string, error) {
	return "mock-operation-id", nil
}

func (m *mockRepositoryService) QueueArchiveRename(ctx context.Context, archiveId int, name string) (string, error) {
	return "mock-operation-id", nil
}

// mockEventEmitter implements EventEmitter interface for testing
type mockEventEmitter struct{}

func (m *mockEventEmitter) EmitEvent(ctx context.Context, event string, data ...string) {
	// Do nothing for tests
}

// Helper function to create a test setup for backup profile service
func newTestBackupProfileService(t *testing.T) (*Service, *ent.Client, context.Context) {
	// Create in-memory database
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { db.Close() })

	// Create logger
	log, _ := zap.NewDevelopment()
	sugarLog := log.Sugar()

	// Create st and config
	st := &state.State{}
	config := &types.Config{}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create mock services
	mockRepoService := &mockRepositoryService{}
	mockEmitter := &mockEventEmitter{}

	// Create service
	serviceInternal := NewService(sugarLog, st, config)
	serviceInternal.Init(ctx, db, mockEmitter, make(chan struct{}, 1), make(chan struct{}, 1), mockRepoService)

	return serviceInternal.Service, db, ctx
}

func TestBackupProfileService_SaveBackupSchedule(t *testing.T) {
	var service *Service
	var db *ent.Client
	var ctx context.Context
	var profile *ent.BackupProfile
	var bs *ent.BackupSchedule
	var now = time.Now()

	setup := func(t *testing.T) {
		service, db, ctx = newTestBackupProfileService(t)

		// Create a backup profile
		p, err := service.NewBackupProfile(ctx)
		assert.NoError(t, err, "Failed to create new backup profile")
		p.Name = "Test profile"
		p.Prefix = "test-"
		bs = p.Edges.BackupSchedule

		// Create a repository
		r, err := db.Repository.Create().
			SetName("TestRepo").
			SetURL("/tmp").
			SetPassword("test").
			Save(ctx)
		assert.NoError(t, err, "Failed to create new repository")

		profile, err = service.CreateBackupProfile(ctx, *p, []int{r.ID})
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
			err := service.SaveBackupSchedule(ctx, profile.ID, newBackupSchedule(tt.schedule))

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoError(t, err, "Expected no error, got %v", err)

				updatedSchedule := db.BackupSchedule.
					Query().
					Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).
					OnlyX(ctx)

				cnt := db.BackupSchedule.Query().CountX(ctx)

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

func TestBackupProfileService_GetPrefixSuggestion(t *testing.T) {
	service, _, ctx := newTestBackupProfileService(t)

	tests := []struct {
		name      string
		prefix    string
		wantErr   bool
		wantEmpty bool
	}{
		{"GetPrefixSuggestion with empty prefix", "", true, false},
		{"GetPrefixSuggestion with alphanumeric prefix", "test123", false, false},
		{"GetPrefixSuggestion with only non-alphanumeric prefix", "!@#", true, false},
		{"GetPrefixSuggestion with valid prefix", "test", false, false},
		{"GetPrefixSuggestion with uppercase prefix", "TEST123", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			suggestion, err := service.GetPrefixSuggestion(ctx, tt.prefix)

			// ASSERT
			if tt.wantErr {
				assert.Error(t, err, "Expected error, got nil")
			} else {
				assert.NoError(t, err, "Expected no error, got %v", err)
				if !tt.wantEmpty {
					assert.NotEmpty(t, suggestion, "Expected non-empty suggestion")
				}
			}
		})
	}
}
