package cmd

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/app"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

var binaries = []types.BorgBinary{
	{
		Name:    "borg_1.4.0",
		Version: "1.4.0",
		Os:      util.Linux,
		Url:     "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
	},
	{
		Name:    "borg_1.4.0",
		Version: "1.4.0",
		Os:      util.Darwin,
		Url:     "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-macos1012",
	},
}

func initLogger(configDir string) *zap.SugaredLogger {
	if app.EnvVarDevelopment.Bool() {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zap.Must(config.Build()).Sugar()
	} else {
		logDir := filepath.Join(util.ExpandPath(configDir), "logs")
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err = os.MkdirAll(logDir, 0755)
			if err != nil {
				panic(fmt.Errorf("failed to create log directory: %w", err))
			}
		}

		// Create a new log file with the current date
		logFileName := filepath.Join(logDir, fmt.Sprintf("arco-%s.log", time.Now().Format("2006-01-02")))
		logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(fmt.Errorf("failed to open log file: %w", err))
		}
		fileWriter := zapcore.AddSync(logFile)

		// Create a production config
		config := zap.NewProductionConfig()
		if app.EnvVarDebug.Bool() {
			config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		}

		// Create a core that writes to the file
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(config.EncoderConfig),
			fileWriter,
			config.Level,
		)

		// Create a logger with the core
		return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Sugar()
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

func initConfig(configDir string, iconData []byte, migrations fs.FS, autoUpdate bool) (*types.Config, error) {
	if configDir == "" {
		var err error
		configDir, err = createConfigDir()
		if err != nil {
			return nil, err
		}
	} else {
		configDir = util.ExpandPath(configDir)
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("config directory %s does not exist", configDir)
		}
	}

	version, err := semver.NewVersion(app.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	return &types.Config{
		Dir:             configDir,
		BorgBinaries:    binaries,
		BorgPath:        filepath.Join(configDir, binaries[0].Name),
		BorgVersion:     binaries[0].Version,
		Icon:            iconData,
		Migrations:      migrations,
		GithubAssetName: util.GithubAssetName(),
		Version:         version,
		CheckForUpdates: autoUpdate,
	}, nil
}

type Stringer interface {
	String() string
}

func toTsEnums[T Stringer](states []T) []struct {
	Value  T
	TSName string
} {
	var allBs = make([]struct {
		Value  T
		TSName string
	}, len(states))

	for i, bs := range states {
		allBs[i] = struct {
			Value  T
			TSName string
		}{
			Value:  bs,
			TSName: bs.String(),
		}
	}
	return allBs
}

func startApp(log *zap.SugaredLogger, config *types.Config, assets fs.FS, startHidden bool, uniqueRunId string) {
	arco := app.NewApp(log, config, &types.RuntimeEventEmitter{})

	logLevel, err := logger.StringToLogLevel(log.Level().String())
	if err != nil {
		log.Fatalf("failed to convert log level: %v", err)
	}

	if uniqueRunId == "" {
		uniqueRunId = "4ffabbd3-334a-454e-8c66-dee8d1ff9afb"
	}

	// Create arco with options
	err = wails.Run(&options.App{
		Title: "Arco",
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        arco.Startup,
		OnShutdown:       arco.Shutdown,
		OnBeforeClose:    arco.BeforeClose,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: uniqueRunId,
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				arco.Wakeup()
			},
		},
		Bind: []interface{}{
			arco.AppClient(),
			arco.BackupClient(),
			arco.RepoClient(),
			arco.ValidationClient(),
		},
		EnumBind: []interface{}{
			toTsEnums(types.AllWeekdays),
			toTsEnums(types.AllIcons),
			toTsEnums(state.AvailableBackupStatuses),
			toTsEnums(state.AvailableRepoStatuses),
			toTsEnums(state.AvailableBackupButtonStatuses),
			toTsEnums(types.AllEvents),
			toTsEnums(types.AllBackupScheduleModes),
			toTsEnums(state.AvailableStartupStatuses),
		},
		LogLevel:    logLevel,
		Logger:      util.NewZapLogWrapper(log),
		MaxWidth:    3840,
		MaxHeight:   3840,
		StartHidden: startHidden,
		Linux: &linux.Options{
			Icon: config.Icon,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

type contextKey string

const (
	assetsKey     contextKey = "assets"
	iconKey       contextKey = "icon"
	migrationsKey contextKey = "migrations"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "arco",
	Short: "Arco is a backup tool that uses Borg to create backups.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := cmd.Flags().GetString(configFlag)
		if err != nil {
			return fmt.Errorf("failed to get config flag: %w", err)
		}
		startHidden, err := cmd.Flags().GetBool(hiddenFlag)
		if err != nil {
			return fmt.Errorf("failed to get hidden flag: %w", err)
		}
		uniqueRunId, err := cmd.Flags().GetString(uniqueRunIdFlag)
		if err != nil {
			return fmt.Errorf("failed to get unique run id flag: %w", err)
		}
		autoUpdate, err := cmd.Flags().GetBool(autoUpdateFlag)
		if err != nil {
			return fmt.Errorf("failed to get auto update flag: %w", err)
		}

		assets := cmd.Context().Value(assetsKey).(fs.FS)
		iconData := cmd.Context().Value(iconKey).([]byte)
		migrations := cmd.Context().Value(migrationsKey).(fs.FS)

		log := initLogger(configDir)
		//goland:noinspection GoUnhandledErrorResult
		defer log.Sync() // flushes buffer, if any

		// Initialize the configuration
		config, err := initConfig(configDir, iconData, migrations, autoUpdate)
		if err != nil {
			log.Errorf("failed to initialize config: %v", err)
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		if startHidden {
			log.Info("starting hidden")
		}
		if configDir != "" {
			log.Infof("using config directory: %s", configDir)
		}
		if uniqueRunId != "" {
			log.Infof("using unique run id: %s", uniqueRunId)
		}

		startApp(log, config, assets, startHidden, uniqueRunId)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(assets fs.FS, iconData []byte, migrations fs.FS) {
	ctx := context.WithValue(context.Background(), assetsKey, assets)
	ctx = context.WithValue(ctx, iconKey, iconData)
	ctx = context.WithValue(ctx, migrationsKey, migrations)

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const configFlag = "config"
const hiddenFlag = "hidden"
const uniqueRunIdFlag = "unique-run-id"
const autoUpdateFlag = "auto-update"

func init() {
	rootCmd.PersistentFlags().StringP(configFlag, "c", "", "config path (default is $HOME/.config/arco/)")
	rootCmd.PersistentFlags().Bool(hiddenFlag, false, "start hidden (default is false)")
	rootCmd.PersistentFlags().String(uniqueRunIdFlag, "", "unique run id. Only one instance of Arco can run with the same id")
	rootCmd.PersistentFlags().Bool(autoUpdateFlag, true, "enable auto update (default is true)")
}
