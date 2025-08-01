package app

import (
	"context"
	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/app/mockapp/mocktypes"
	"github.com/loomi-labs/arco/backend/app/repository"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg/mockborg"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"testing"
)

func NewTestApp(t *testing.T) (*App, *mockborg.MockBorg, *mocktypes.MockEventEmitter) {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err := logConfig.Build()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	migrationsDir := os.DirFS("../ent/migrate/migrations")

	tempDir := t.TempDir()
	config := &types.Config{
		Dir:             tempDir,
		SSHDir:          filepath.Join(tempDir, "ssh"),
		BorgBinaries:    nil,
		BorgPath:        "",
		BorgVersion:     "",
		Icons:           nil,
		Migrations:      migrationsDir,
		GithubAssetName: "",
		Version:         semver.MustParse("0.0.0"),
	}

	mockEventEmitter := mocktypes.NewMockEventEmitter(gomock.NewController(t))
	a := NewApp(log.Sugar(), config, mockEventEmitter)
	a.ctx = context.Background()

	close(a.backupScheduleChangedCh)
	a.backupScheduleChangedCh = nil
	close(a.pruningScheduleChangedCh)
	a.pruningScheduleChangedCh = nil

	mockBorg := mockborg.NewMockBorg(gomock.NewController(t))
	a.borg = mockBorg

	mockEventEmitter.EXPECT().EmitEvent(gomock.Any(), types.EventStartupStateChanged.String())
	db, err := a.initDb()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	a.db = db

	// Initialize repository service with required dependencies for tests
	// Create a cloud repository service for testing
	cloudRepositoryServiceInternal := repository.NewCloudRepositoryService(a.log, a.state, config)
	cloudRepositoryServiceInternal.Init(db, nil) // Pass nil for RPC client in tests
	a.repositoryService.Init(
		db,
		mockBorg,
		config,
		mockEventEmitter,
		cloudRepositoryServiceInternal.CloudRepositoryService,
	)

	return a, mockBorg, mockEventEmitter
}
