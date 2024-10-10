package cmd

import (
	"arco/backend/app"
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/util"
	"context"
	"embed"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

var binaries = []types.Binary{
	{
		Name:    "borg_1.4.0",
		Version: "1.4.0",
		Os:      types.Linux,
		Url:     "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
	},
	{
		Name:    "borg_1.4.0",
		Version: "1.4.0",
		Os:      types.Darwin,
		Url:     "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-macos1012",
	},
}

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

func initConfig(configDir string, icon embed.FS) (*types.Config, error) {
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

	return &types.Config{
		Dir:         configDir,
		Binaries:    binaries,
		BorgPath:    filepath.Join(configDir, binaries[0].Name),
		BorgVersion: binaries[0].Version,
		Icon:        icon,
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

func startApp(log *zap.SugaredLogger, config *types.Config, assets embed.FS, startHidden bool, uniqueRunId string) {
	arco := app.NewApp(log, config)

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
		},
		EnumBind: []interface{}{
			toTsEnums(types.AllWeekdays),
			toTsEnums(types.AllIcons),
			toTsEnums(state.AvailableBackupStatuses),
			toTsEnums(state.AvailableRepoStatuses),
			toTsEnums(state.AvailableBackupButtonStatuses),
			toTsEnums(types.AllEvents),
		},
		LogLevel:    logLevel,
		Logger:      util.NewZapLogWrapper(log.Desugar()),
		StartHidden: startHidden,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "arco",
	Short: "Arco is a backup tool that uses Borg to create backups.",
	Run: func(cmd *cobra.Command, args []string) {
		log := initLogger()
		//goland:noinspection GoUnhandledErrorResult
		defer log.Sync() // flushes buffer, if any

		configDir, err := cmd.Flags().GetString(configFlag)
		if err != nil {
			log.Fatalf("failed to get config flag: %v", err)
		}
		if configDir != "" {
			log.Infof("using config directory: %s", configDir)
		}

		startHidden, err := cmd.Flags().GetBool(hiddenFlag)
		if err != nil {
			log.Fatalf("failed to get hidden flag: %v", err)
		}
		if startHidden {
			log.Info("starting hidden")
		}
		uniqueRunId, err := cmd.Flags().GetString(uniqueRunIdFlag)
		if err != nil {
			log.Fatalf("failed to get unique run id flag: %v", err)
		}
		if uniqueRunId != "" {
			log.Infof("using unique run id: %s", uniqueRunId)
		}

		assets := cmd.Context().Value("assets").(embed.FS)
		icon := cmd.Context().Value("icon").(embed.FS)

		// Initialize the configuration
		config, err := initConfig(configDir, icon)
		if err != nil {
			log.Fatal(err)
		}

		startApp(log, config, assets, startHidden, uniqueRunId)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(assets embed.FS, icon embed.FS) {
	ctx := context.WithValue(context.Background(), "assets", assets)
	ctx = context.WithValue(ctx, "icon", icon)

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

const configFlag = "config"
const hiddenFlag = "hidden"
const uniqueRunIdFlag = "unique-run-id"

func init() {
	rootCmd.PersistentFlags().StringP(configFlag, "c", "", "config path (default is $HOME/.config/arco/)")
	rootCmd.PersistentFlags().Bool(hiddenFlag, false, "start hidden")
	rootCmd.PersistentFlags().String(uniqueRunIdFlag, "", "unique run id. Only one instance of Arco can run with the same id")
}
