package main

import (
	"arco/backend/borg/client"
	"arco/backend/borg/types"
	"arco/backend/borg/worker"
	"arco/backend/ent"
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

func initDb() (*ent.Client, error) {
	dbClient, err := ent.Open("sqlite3", "file:sqlite.db?_fk=1")
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %v", err)
	}

	// Run the auto migration tool.
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		return nil, err
	}
	return dbClient, nil
}

func startApp(log *zap.SugaredLogger, inChan *types.InputChannels, outChan *types.OutputChannels, borgWorker *worker.Worker) {
	logLevel, err := logger.StringToLogLevel(log.Level().String())
	if err != nil {
		log.Fatalf("failed to convert log level: %v", err)
	}

	// Initialize the database
	dbClient, err := initDb()
	if err != nil {
		log.Fatal(err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer dbClient.Close()

	borgClient := client.NewBorgClient(log, dbClient, inChan, outChan)

	// Create an instance of the app structure
	app := NewApp(borgClient, borgWorker)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "arco",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app.BorgClient,
		},
		LogLevel: logLevel,
		Logger:   NewZapLogWrapper(log.Desugar()),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log := initLogger()
	//goland:noinspection GoUnhandledErrorResult
	defer log.Sync() // flushes buffer, if any

	inChan := &types.InputChannels{
		StartBackup: make(chan types.BackupJob),
	}
	outChan := &types.OutputChannels{
		FinishBackup: make(chan types.FinishBackupJob),
	}

	// Create a borg daemon
	borgWorker := worker.NewWorker(log, inChan, outChan)

	go borgWorker.Run()
	startApp(log, inChan, outChan, borgWorker)
}
