package main

import (
	"arco/backend/borg/client"
	"arco/backend/borg/types"
	"arco/backend/ent"
	"context"
	"embed"
	"fmt"
	"github.com/godbus/dbus/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed bin
var binaries embed.FS

//go:embed icon.png
var icon embed.FS

const borgVersion = "1.2.8"

func initLogger() *zap.SugaredLogger {
	if os.Getenv(client.EnvVarDebug.String()) == "true" {
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

func getConfigDir() (path string, err error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	return filepath.Join(dir, ".config", "arco"), nil
}

func createConfigDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(configDir); os.IsNotExist(err) {
		return configDir, os.MkdirAll(configDir, 0755)
	} else if err != nil {
		return "", err
	}
	return configDir, nil
}

func initConfig() (*types.Config, error) {
	configDir, err := createConfigDir()
	if err != nil {
		return nil, err
	}

	return &types.Config{
		Binaries:    binaries,
		BorgPath:    filepath.Join(configDir, "borg"),
		BorgVersion: borgVersion,
		Icon:        icon,
	}, nil
}

func initDb() (*ent.Client, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	dbClient, err := ent.Open("sqlite3", fmt.Sprintf("file:%s?_fk=1", filepath.Join(configDir, "arco.db")))
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %v", err)
	}

	// Run the auto migration tool.
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		return nil, err
	}
	return dbClient, nil
}

func initDbus(log *zap.SugaredLogger) (*dbus.Conn, error) {
	// Check if another instance is running
	// If another instance is running, send a WakeUp message to the other instance and exit
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %v", err)
	}

	// Check if another instance is already running
	var running bool
	err = conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, client.DbusInterface).Store(&running)
	if err != nil {
		return nil, fmt.Errorf("failed to check if another instance is running: %v", err)
	}

	if running {
		// Another instance is running, send it a wakeup command
		log.Debug("Send wakeup command to other instance and exit")
		err = conn.Object(client.DbusInterface, client.DbusPath).Call(client.DbusInterface+".Wakeup", 0).Err
		if err != nil {
			return nil, fmt.Errorf("failed to send wakeup command to other instance: %v", err)
		}
		os.Exit(0)
	}

	return conn, nil
}

func startApp(
	log *zap.SugaredLogger,
	borgClient *client.BorgClient,
) {
	logLevel, err := logger.StringToLogLevel(log.Level().String())
	if err != nil {
		log.Fatalf("failed to convert log level: %v", err)
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "Arco",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        borgClient.Startup,
		OnShutdown:       borgClient.Shutdown,
		OnBeforeClose:    borgClient.BeforeClose,
		Bind: []interface{}{
			borgClient.AppClient(),
			borgClient.BackupClient(),
			borgClient.RepoClient(),
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

	// Check if another instance is running
	// If another instance is running, send a message to the other instance to open the window and exit
	dbusConn, err := initDbus(log)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the configuration
	config, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the database
	dbClient, err := initDb()
	if err != nil {
		log.Fatal(err)
	}

	borgClient := client.NewBorgClient(log, config, dbClient, dbusConn)

	startApp(log, borgClient)
}
