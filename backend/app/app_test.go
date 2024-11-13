package app

import (
	"context"
	"github.com/loomi-labs/arco/backend/app/mockapp/mocktypes"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg/mockborg"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
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

	config := &types.Config{
		Dir:             t.TempDir(),
		BorgBinaries:    nil,
		BorgPath:        "",
		BorgVersion:     "",
		Icon:            nil,
		Migrations:      migrationsDir,
		GithubAssetName: "",
		Version:         "",
		ArcoPath:        "",
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

	db, err := a.initDb()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	a.db = db
	return a, mockBorg, mockEventEmitter
}
