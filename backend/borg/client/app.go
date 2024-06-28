package client

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"arco/backend/borg/worker"
	"arco/backend/ent"
	"arco/backend/ssh"
	"context"
	"fmt"
	"github.com/getlantern/systray"
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
	AppName = "Arco"
)

type EnvVar string

const (
	EnvVarDebug EnvVar = "DEBUG"
)

func (e EnvVar) String() string {
	return string(e)
}

type App struct {
	// Init
	log     *util.CmdLogger
	config  *types.Config
	inChan  *types.InputChannels
	outChan *types.OutputChannels
	worker  *worker.Worker

	// Startup
	ctx context.Context
	db  *ent.Client

	// State (runtime)
	runningBackups   []types.BackupIdentifier
	runningPruneJobs []types.BackupIdentifier
	occupiedRepos    []int
	startupErr       error
}

func NewApp(
	log *zap.SugaredLogger,
	config *types.Config,
) *App {
	inChan := types.NewInputChannels()
	outChan := types.NewOutputChannels()
	return &App{
		log:     util.NewCmdLogger(log),
		config:  config,
		inChan:  inChan,
		outChan: outChan,
		worker:  worker.NewWorker(log, config.BorgPath, inChan, outChan),
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

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize the database
	db, err := a.initDb()
	if err != nil {
		a.startupErr = err
		a.log.Error(err)
		return
	}
	a.db = db

	// Initialize the systray
	if err := a.initSystray(); err != nil {
		a.startupErr = err
		a.log.Error(err)
		return
	}

	// Register signal handler
	a.registerSignalHandler()

	// Ensure Borg binary is installed
	if err := a.ensureBorgBinary(); err != nil {
		a.startupErr = err
		a.log.Error(err)
		return
	}

	// Start the worker
	go a.worker.Run()
}

func (a *App) Shutdown(_ context.Context) {
	a.log.Info(fmt.Sprintf("Shutting down %s", AppName))
	a.worker.Stop()
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

func (a *App) initDb() (*ent.Client, error) {
	dbClient, err := ent.Open("sqlite3", fmt.Sprintf("file:%s?_fk=1", filepath.Join(a.config.Dir, "arco.db")))
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %v", err)
	}

	// Run the auto migration tool.
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		return nil, err
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
	// Output is in the format "borg 1.2.8\n"
	// We want to return "1.2.8"
	return strings.TrimSpace(strings.TrimPrefix(string(out), "borg ")), nil
}

func (a *App) installBorgBinary() error {
	// Delete old binary if it exists
	if _, err := os.Stat(a.config.BorgPath); err == nil {
		if err := os.Remove(a.config.BorgPath); err != nil {
			return err
		}
	}

	file, err := a.config.Binaries.ReadFile(util.GetBorgBinaryPathX())
	if err != nil {
		return err
	}
	return os.WriteFile(a.config.BorgPath, file, 0755)
}

func (a *App) initSystray() error {
	iconData, err := a.config.Icon.ReadFile("icon.png")
	if err != nil {
		return fmt.Errorf("failed to read icon: %v", err)
	}

	readyFunc := func() {
		systray.SetIcon(iconData)
		systray.SetTitle(AppName)
		systray.SetTooltip(AppName)

		mOpen := systray.AddMenuItem(fmt.Sprintf("Open %s", AppName), fmt.Sprintf("Open %s", AppName))
		systray.AddSeparator()
		mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %s", AppName), fmt.Sprintf("Quit %s", AppName))

		// Sets the icon of a menu item. Only available on Mac and Windows.
		mOpen.SetIcon(iconData)
		mQuit.SetIcon(iconData)

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					runtime.WindowShow(a.ctx)
				case <-mQuit.ClickedCh:
					a.Shutdown(a.ctx)
				}
			}
		}()
	}

	exitFunc := func() {
		// TODO: check if there is a running backup and ask the user if they want to cancel it
		a.Shutdown(a.ctx)
	}

	// TODO: not working right now -> fix this
	//systray.Run(readyFunc, exitFunc)
	_, _ = readyFunc, exitFunc
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

// TODO: remove or move somewhere else
func (a *App) createSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	a.log.Info(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}
