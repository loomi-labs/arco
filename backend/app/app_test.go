package app

import (
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/enttest"
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "app test suite")
}

type TestingT interface {
	enttest.TestingT
	Log(args ...any)
}

func NewTestApp(t TestingT) *App {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build logger: %v", err))
	}

	a := NewApp(log.Sugar(), &types.Config{})
	a.ctx = context.Background()
	a.config = nil

	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}
	a.db = enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1", opts...)
	return a
}
