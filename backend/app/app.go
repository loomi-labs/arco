package app

import (
	appstate "arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/borg"
	"arco/backend/ent"
	"arco/backend/util"
	"ariga.io/atlas-go-sdk/atlasexec"
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	Name = "Arco"
)

type EnvVar string

const (
	EnvVarDebug       EnvVar = "DEBUG"
	EnvVarDevelopment EnvVar = "DEVELOPMENT"
	EnvVarStartPage   EnvVar = "START_PAGE"
)

func (e EnvVar) String() string {
	return string(e)
}

type App struct {
	// Init
	log                      *zap.SugaredLogger
	config                   *types.Config
	state                    *appstate.State
	borg                     borg.Borg
	backupScheduleChangedCh  chan struct{}
	pruningScheduleChangedCh chan struct{}

	// Startup
	ctx    context.Context
	cancel context.CancelFunc
	db     *ent.Client
}

func NewApp(
	log *zap.SugaredLogger,
	config *types.Config,
) *App {
	state := appstate.NewState(log)
	return &App{
		log:                      log,
		config:                   config,
		state:                    state,
		borg:                     borg.NewBorg(config.BorgPath, log),
		backupScheduleChangedCh:  make(chan struct{}),
		pruningScheduleChangedCh: make(chan struct{}),
	}
}

// These clients separate the different types of operations that can be performed with the Borg client
// This makes it easier to expose them in a clean way to the frontend

// RepositoryClient is a client for repository related operations
type RepositoryClient App

// AppClient is a client for application related operations
type AppClient App

// BackupClient is a client for backup related operations
type BackupClient App

func (a *App) RepoClient() *RepositoryClient {
	return (*RepositoryClient)(a)
}

func (a *App) AppClient() *AppClient {
	return (*AppClient)(a)
}

func (a *App) BackupClient() *BackupClient {
	return (*BackupClient)(a)
}

func (r *RepositoryClient) backupClient() *BackupClient {
	return (*BackupClient)(r)
}

func (b *BackupClient) repoClient() *RepositoryClient {
	return (*RepositoryClient)(b)
}

func (a *App) Startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)

	// Initialize the database
	db, err := a.initDb()
	if err != nil {
		a.state.SetStartupError(err)
		a.log.Error(err)
		return
	}
	a.db = db

	// Initialize the systray
	if err := a.initSystray(); err != nil {
		a.state.SetStartupError(err)
		a.log.Error(err)
		return
	}

	// Register signal handler
	a.registerSignalHandler()

	// Save mount states
	a.RepoClient().setMountStates()

	// Ensure Borg binary is installed
	if err := a.ensureBorgBinary(); err != nil {
		a.state.SetStartupError(err)
		a.log.Error(err)
		return
	}

	// Schedule backups
	go a.startScheduleChangeListener()
	go a.startPruneScheduleChangeListener()
	a.backupScheduleChangedCh <- struct{}{}  // Trigger initial backup schedule check
	a.pruningScheduleChangedCh <- struct{}{} // Trigger initial pruning schedule check
}

func (a *App) Shutdown(_ context.Context) {
	a.log.Info(fmt.Sprintf("Shutting down %s", Name))
	a.cancel()
	err := a.db.Close()
	if err != nil {
		a.log.Error("Failed to close database connection")
	}
	os.Exit(0)
}

func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.log.Debug("Received beforeclose command")
	runtime.WindowHide(ctx)
	return true
}

func (a *App) Wakeup() {
	a.log.Debug("Received wakeup command")
	runtime.WindowShow(a.ctx)
}

func (a *App) applyMigrations(opts string) error {
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(
			a.config.Migrations,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to load working directory: %v", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer workdir.Close()

	// Initialize the atlasClient.
	atlasClient, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		return fmt.Errorf("failed to initialize atlasClient: %v", err)
	}

	// Run `atlas migrate apply`
	result, err := atlasClient.MigrateApply(a.ctx, &atlasexec.MigrateApplyParams{
		URL: fmt.Sprintf("sqlite:///%s%s", filepath.Join(a.config.Dir, "arco.db"), opts),
	})
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}
	a.log.Infof("Applied %d migrations", len(result.Applied))
	a.log.Infof("Current db version: %s", result.Current)
	if result.Current == "" && len(result.Applied) == 0 {
		return fmt.Errorf("could not apply migrations")
	}
	return nil
}

func (a *App) initDb() (*ent.Client, error) {
	// - Set WAL mode (not strictly necessary each time because it's persisted in the database, but good for first run)
	// - Set busy timeout, so concurrent writers wait on each other instead of erroring immediately
	// - Enable foreign key checks
	opts := "?_journal=WAL&_timeout=5000&_fk=1"

	// Apply migrations
	if err := a.applyMigrations(opts); err != nil {
		return nil, err
	}

	dbClient, err := ent.Open("sqlite3", fmt.Sprintf("file:%s%s", filepath.Join(a.config.Dir, "arco.db"), opts))
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %v", err)
	}
	return dbClient, nil
}

func (a *App) ensureBorgBinary() error {
	if !a.isTargetVersionInstalled(a.config.BorgVersion) {
		a.log.Info("Installing Borg binary")
		if err := a.installBorgBinary(); err != nil {
			return fmt.Errorf("failed to install Borg binary: %w", err)
		} else {
			// Check again to make sure the binary was installed correctly
			if !a.isTargetVersionInstalled(a.config.BorgVersion) {
				return fmt.Errorf("failed to install Borg binary: version mismatch")
			}
		}
	}
	return nil
}

func (a *App) isTargetVersionInstalled(targetVersion string) bool {
	// Check if the binary is installed
	if _, err := os.Stat(a.config.BorgPath); err == nil {
		version, err := a.version()
		// Check if the version is correct
		return err == nil && version == targetVersion
	}
	return false
}

func (a *App) version() (string, error) {
	cmd := exec.Command(a.config.BorgPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// Output is in the format "xxx 1.2.8\n"
	// We want to return "1.2.8"
	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return "", fmt.Errorf("unexpected output: %s", string(out))
	}
	return fields[1], nil
}

func (a *App) installBorgBinary() error {
	// Delete old binary if it exists
	if _, err := os.Stat(a.config.BorgPath); err == nil {
		if err := os.Remove(a.config.BorgPath); err != nil {
			return err
		}
	}

	binary, err := types.GetLatestBorgBinary(a.config.Binaries)
	if err != nil {
		return err
	}

	// Download the binary
	return util.DownloadFile(a.config.BorgPath, binary.Url)
}

func (a *App) initSystray() error {
	//	iconData, err := a.config.Icon.ReadFile("icon.png")
	//	if err != nil {
	//		return fmt.Errorf("failed to read icon: %v", err)
	//	}
	//
	//	readyFunc := func() {
	//		systray.SetIcon(iconData)
	//		systray.SetTitle(Name)
	//		systray.SetTooltip(Name)
	//
	//		mOpen := systray.AddMenuItem(fmt.Sprintf("Open %s", Name), fmt.Sprintf("Open %s", Name))
	//		systray.AddSeparator()
	//		mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %s", Name), fmt.Sprintf("Quit %s", Name))
	//
	//		// Sets the icon of a menu item. Only available on Mac and Windows.
	//		mOpen.SetIcon(iconData)
	//		mQuit.SetIcon(iconData)
	//
	//		go func() {
	//			for {
	//				select {
	//				case <-mOpen.ClickedCh:
	//					runtime.WindowShow(a.ctx)
	//				case <-mQuit.ClickedCh:
	//					a.Shutdown(a.ctx)
	//				}
	//			}
	//		}()
	//	}
	//
	//	exitFunc := func() {
	//		// TODO: check if there is a running backup and ask the user if they want to cancel it
	//		a.Shutdown(a.ctx)
	//	}
	//
	//	// TODO: not working right now -> fix this
	//	//systray.Run(readyFunc, exitFunc)
	//	_, _ = readyFunc, exitFunc
	return nil
}

// RegisterSignalHandler listens to interrupt signals and shuts down the application on receiving one
func (a *App) registerSignalHandler() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChan
		a.Shutdown(a.ctx)
	}()
}

// rollback calls to tx.Rollback and wraps the given error
// with the rollback error if occurred.
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}

// TODO: remove or move somewhere else
func (a *App) createSSHKeyPair() (string, error) {
	pair, err := util.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	a.log.Info(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}
