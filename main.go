package main

import (
	"arco/backend/app"
	"arco/backend/types"
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
	if os.Getenv(app.EnvVarDebug.String()) == "true" {
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
		Dir:         configDir,
		Binaries:    binaries,
		BorgPath:    filepath.Join(configDir, "borg"),
		BorgVersion: borgVersion,
		Icon:        icon,
	}, nil
}

func startApp(log *zap.SugaredLogger, config *types.Config) {
	arco := app.NewApp(log, config)

	logLevel, err := logger.StringToLogLevel(log.Level().String())
	if err != nil {
		log.Fatalf("failed to convert log level: %v", err)
	}

	// Create arco with options
	err = wails.Run(&options.App{
		Title:  "Arco",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        arco.Startup,
		OnShutdown:       arco.Shutdown,
		OnBeforeClose:    arco.BeforeClose,
		Bind: []interface{}{
			arco.AppClient(),
			arco.BackupClient(),
			arco.RepoClient(),
		},
		LogLevel: logLevel,
		Logger:   NewZapLogWrapper(log.Desugar()),
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "4ffabbd3-334a-454e-8c66-dee8d1ff9afb",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				arco.Wakeup()
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log := initLogger()
	//goland:noinspection GoUnhandledErrorResult
	defer log.Sync() // flushes buffer, if any

	// Initialize the configuration
	config, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	startApp(log, config)
}
