//go:build !integration

package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/app"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initLogger(configDir string) *zap.SugaredLogger {
	if types.EnvVarDevelopment.Bool() {
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
		if types.EnvVarDebug.Bool() {
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

// ensureSSHDir creates SSH directory with proper permissions if it doesn't exist,
// or ensures existing directory has correct permissions
func ensureSSHDir(configDir string) error {
	sshDir := filepath.Join(configDir, "ssh")

	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		// SSH directory doesn't exist, create it with strict permissions
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			return fmt.Errorf("failed to create SSH directory %q: %w", sshDir, err)
		}
	} else if err != nil {
		// Error accessing SSH directory
		return fmt.Errorf("failed to access SSH directory %q: %w", sshDir, err)
	} else {
		// SSH directory exists, ensure it has proper permissions
		if err := os.Chmod(sshDir, 0700); err != nil {
			return fmt.Errorf("failed to set SSH directory permissions: %w", err)
		}
	}

	return nil
}

func createConfigDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create config directory if it doesn't exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory %q: %w", configDir, err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to access config directory %q: %w", configDir, err)
	}

	// Ensure SSH directory exists with proper permissions
	if err := ensureSSHDir(configDir); err != nil {
		return "", err
	}

	return configDir, nil
}

func initConfig(configDir string, icons *types.Icons, migrations fs.FS, autoUpdate bool) (*types.Config, error) {
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

	version, err := semver.NewVersion(types.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	cloudRPCURL := types.EnvVarCloudRPCURL.String()
	if cloudRPCURL == "" {
		if types.EnvVarDevelopment.Bool() {
			cloudRPCURL = "http://localhost:8080"
		} else {
			cloudRPCURL = "https://api.arco-backup.com"
		}
	}

	return &types.Config{
		Dir:             configDir,
		SSHDir:          filepath.Join(configDir, "ssh"),
		BorgBinaries:    platform.Binaries,
		BorgPath:        filepath.Join(configDir, platform.Binaries[0].Name),
		BorgVersion:     platform.Binaries[0].Version.String(),
		Icons:           icons,
		Migrations:      migrations,
		GithubAssetName: platform.GithubAssetName(),
		Version:         version,
		CheckForUpdates: autoUpdate,
		CloudRPCURL:     cloudRPCURL,
	}, nil
}

func showOrCreateMainWindow(config *types.Config) {
	wailsApp := application.Get()
	window, ok := wailsApp.Window.GetByName(types.WindowTitle)
	if ok {
		window.Show()
		window.Focus()
		return
	}

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  types.WindowTitle,
		Title: types.WindowTitle,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Linux: application.LinuxWindow{
			Icon: config.Icons.AppIconLight,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		StartState:       application.WindowStateMinimised,
		MaxWidth:         3840,
		MaxHeight:        3840,
	})
}

func startApp(log *zap.SugaredLogger, config *types.Config, assets fs.FS, startHidden bool, uniqueRunId string) {
	arco := app.NewApp(log, config, &types.RuntimeEventEmitter{})

	if uniqueRunId == "" {
		uniqueRunId = "4ffabbd3-334a-454e-8c66-dee8d1ff9afb"
	}

	wailsApp := application.New(application.Options{
		Name:        app.Name,
		Description: "Arco is a backup tool.",
		Services: []application.Service{
			application.NewService(arco.UserService()),
			application.NewService(arco.BackupProfileService()),
			application.NewService(arco.RepositoryService()),
			application.NewService(arco.RepositoryServiceN()),
			application.NewService(arco.AuthService()),
			application.NewService(arco.PlanService()),
			application.NewService(arco.SubscriptionService()),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: uniqueRunId,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				log.Debug("Wake up %s", app.Name)
				showOrCreateMainWindow(config)
			},
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		LogLevel: slog.Level(log.Level() * 4), // slog uses a multiplier of 4
		ShouldQuit: func() bool {
			return arco.ShouldQuit()
		},
		OnShutdown: func() {
			arco.Shutdown()
		},
	})

	if !startHidden {
		showOrCreateMainWindow(config)
	}

	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		// TODO: what about the context?
		arco.Startup(context.TODO())
	})

	systray := wailsApp.SystemTray.New()
	systray.SetLabel(app.Name)
	if platform.IsMacOS() {
		// Support for template icons on macOS
		systray.SetTemplateIcon(config.Icons.DarwinIcons)
	} else {
		// Support for light/dark mode icons
		systray.SetDarkModeIcon(config.Icons.AppIconDark)
		systray.SetIcon(config.Icons.AppIconLight)
	}

	// Add menu
	menu := wailsApp.NewMenu()
	menu.Add("Open").OnClick(func(_ *application.Context) {
		log.Debugf("Opening %s", app.Name)
		showOrCreateMainWindow(config)
	})
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		log.Debugf("Quitting %s", app.Name)
		arco.SetQuit()
		wailsApp.Quit()
	})
	systray.SetMenu(menu)

	// Run the application. This blocks until the application has been exited.
	err := wailsApp.Run()
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
		// Check if version flag is set
		showVersion, err := cmd.Flags().GetBool(versionFlag)
		if err != nil {
			return fmt.Errorf("failed to get version flag: %w", err)
		}
		if showVersion {
			fmt.Printf("%s %s\n", app.Name, types.Version)
			return nil
		}

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
		icons := cmd.Context().Value(iconKey).(*types.Icons)
		migrations := cmd.Context().Value(migrationsKey).(fs.FS)

		log := initLogger(configDir)
		//goland:noinspection GoUnhandledErrorResult
		defer log.Sync() // flushes buffer, if any

		// Initialize the configuration
		config, err := initConfig(configDir, icons, migrations, autoUpdate)
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
func Execute(assets fs.FS, icons *types.Icons, migrations fs.FS) {
	ctx := context.WithValue(context.Background(), assetsKey, assets)
	ctx = context.WithValue(ctx, iconKey, icons)
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
const versionFlag = "version"

func init() {
	rootCmd.PersistentFlags().StringP(configFlag, "c", "", "config path (default is $HOME/.config/arco/)")
	rootCmd.PersistentFlags().Bool(hiddenFlag, false, "start hidden (default is false)")
	rootCmd.PersistentFlags().String(uniqueRunIdFlag, "", "unique run id. Only one instance of Arco can run with the same id")
	rootCmd.PersistentFlags().Bool(autoUpdateFlag, true, "enable auto update (default is true)")
	rootCmd.PersistentFlags().BoolP(versionFlag, "v", false, "print version information and exit")
}
