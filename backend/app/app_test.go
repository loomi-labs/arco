package app

import (
	"arco/backend/app/types"
	"arco/backend/borg/mockborg"
	"arco/backend/ent"
	"arco/backend/ent/enttest"
	"context"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

func NewTestApp(t *testing.T) (*App, *mockborg.MockBorg) {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build logger: %v", err))
	}

	a := NewApp(log.Sugar(), &types.Config{})
	a.ctx = context.Background()
	a.config = nil
	close(a.backupScheduleChangedCh)
	a.backupScheduleChangedCh = nil

	ctrl := gomock.NewController(t)
	mockBorg := mockborg.NewMockBorg(ctrl)
	a.borg = mockBorg

	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}
	a.db = enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1", opts...)
	return a, mockBorg
}
