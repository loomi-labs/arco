package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/app/repository"
	"github.com/loomi-labs/arco/backend/app/types"
	typesmocks "github.com/loomi-labs/arco/backend/app/types/mocks"
	borgmocks "github.com/loomi-labs/arco/backend/borg/mocks"
	"github.com/loomi-labs/arco/backend/internal/keyring"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewTestApp(t *testing.T) (*App, *borgmocks.MockBorg, *typesmocks.MockEventEmitter) {
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

	mockEventEmitter := typesmocks.NewMockEventEmitter(gomock.NewController(t))
	a := NewApp(log.Sugar(), config, mockEventEmitter)

	// Create context for tests that can be cancelled during cleanup
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel

	close(a.backupScheduleChangedCh)
	a.backupScheduleChangedCh = nil
	close(a.pruningScheduleChangedCh)
	a.pruningScheduleChangedCh = nil

	mockBorg := borgmocks.NewMockBorg(gomock.NewController(t))
	a.borg = mockBorg

	mockEventEmitter.EXPECT().EmitEvent(gomock.Any(), types.EventStartupStateChanged.String())
	db, err := a.initDb()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	a.db = db

	// Initialize repository service with required dependencies for tests
	// Create a cloud repository client for testing
	cloudRepositoryClient := repository.NewCloudRepositoryClient(a.log, a.state, config)
	cloudRepositoryClient.Init(db, nil) // Pass nil for RPC client in tests

	// Create test keyring for tests
	testKeyring := keyring.NewTestService(a.log)
	a.keyring = testKeyring

	a.repositoryService.Init(
		a.ctx,
		db,
		mockEventEmitter,
		mockBorg,
		cloudRepositoryClient,
		testKeyring,
	)

	// Initialize backup profile service with repository service dependency
	a.backupProfileService.Init(a.ctx, db, mockEventEmitter, a.backupScheduleChangedCh, a.pruningScheduleChangedCh, a.repositoryService)

	// Add cleanup function to test to ensure context is cancelled
	t.Cleanup(func() {
		if a.cancel != nil {
			a.cancel()
		}
		if a.db != nil {
			a.db.Close()
		}
	})

	return a, mockBorg, mockEventEmitter
}
