package main

import (
	"context"
	"embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"timebender/backend/ent"
)

//go:embed all:frontend/dist
var assets embed.FS

func initLogger() *zap.SugaredLogger {
	if os.Getenv("DEBUG") == "true" {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log, err := config.Build()
		if err != nil {
			panic(fmt.Sprintf("failed to build logger: %v", err))
		}
		return log.Sugar()
	} else {
		return zap.Must(zap.NewProduction()).Sugar()
	}
}

func initDb() error {
	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		return fmt.Errorf("failed opening connection to sqlite: %v", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer client.Close()
	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		return err
	}
	return nil
}

func main() {
	log := initLogger()
	//goland:noinspection GoUnhandledErrorResult
	defer log.Sync() // flushes buffer, if any
	logLevel, err := logger.StringToLogLevel(log.Level().String())
	if err != nil {
		log.Fatalf("failed to convert log level: %v", err)
	}

	// Create an instance of the app structure
	app := NewApp(log)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "timebender",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app.Borg,
		},
		LogLevel: logLevel,
		Logger:   NewZapLogWrapper(log.Desugar()),
	})

	if err != nil {
		log.Fatal(err)
	}

	err = initDb()
	if err != nil {
		log.Fatal(err)
	}
}
