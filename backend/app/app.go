package app

import (
	"archive/zip"
	"ariga.io/atlas-go-sdk/atlasexec"
	"bytes"
	"context"
	"fmt"
	"github.com/google/go-github/v66/github"
	appstate "github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/teamwork/reload"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	Name = "Arco"
)

var (
	Version = ""
)

type EnvVar string

const (
	EnvVarDebug       EnvVar = "DEBUG"
	EnvVarDevelopment EnvVar = "DEVELOPMENT"
	EnvVarStartPage   EnvVar = "START_PAGE"
)

func (e EnvVar) Name() string {
	return string(e)
}

func (e EnvVar) String() string {
	return os.Getenv(e.Name())
}

func (e EnvVar) Bool() bool {
	return os.Getenv(e.Name()) == "true"
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

	defer runtime.EventsEmit(a.ctx, types.EventAppReady.String())

	a.log.Infof("Running Arco version %s", Version)

	// Update Arco binary if necessary
	needsRestart, err := a.updateArco()
	if err != nil {
		a.state.SetStartupError(err)
		a.log.Error(err)
		return
	}
	// Restart if an updates has been performed
	if needsRestart {
		a.log.Info("Updated Arco binary... restarting")
		reload.Exec()
		return
	}

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
	if a.state.GetStartupError() != nil {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (a *App) Wakeup() {
	a.log.Debug("Received wakeup command")
	runtime.WindowShow(a.ctx)
}

func (a *App) updateArco() (bool, error) {
	if EnvVarDevelopment.Bool() {
		a.log.Info("Development mode enabled, skipping update check")
		return false, nil
	}

	client := github.NewClient(nil)

	release, err := a.getLatestRelease(client)
	if err != nil {
		return false, err
	}

	if *release.TagName == a.config.Version {
		a.log.Info("No updates available")
		return false, nil
	}

	a.log.Infof("Updating Arco binary to version %s", *release.TagName)

	releaseAsset, err := a.findReleaseAsset(release)
	if err != nil {
		return false, err
	}

	err = a.downloadReleaseAsset(client, releaseAsset)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) getLatestRelease(client *github.Client) (*github.RepositoryRelease, error) {
	release, _, err := client.Repositories.GetLatestRelease(a.ctx, "loomi-labs", "arco")
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	if release.TagName == nil {
		return nil, fmt.Errorf("could not find latest release")
	}
	return release, nil
}

func (a *App) findReleaseAsset(release *github.RepositoryRelease) (*github.ReleaseAsset, error) {
	for _, ra := range release.Assets {
		if ra.Name != nil && ra.BrowserDownloadURL != nil && *ra.Name == a.config.GithubAssetName {
			return ra, nil
		}
	}
	return nil, fmt.Errorf("could not find release asset for version %s", a.config.Version)
}

func (a *App) downloadReleaseAsset(client *github.Client, asset *github.ReleaseAsset) error {
	httpClient := &http.Client{Timeout: time.Second * 30}
	readCloser, _, err := client.Repositories.DownloadReleaseAsset(a.ctx, "loomi-labs", "arco", *asset.ID, httpClient)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	if readCloser == nil {
		return fmt.Errorf("failed to download release asset: readCloser is nil")
	}

	var buf bytes.Buffer
	size, err := io.Copy(&buf, readCloser)
	if err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}
	reader := bytes.NewReader(buf.Bytes())

	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return fmt.Errorf("failed to read zip zipReader: %w", err)
	}
	return a.extractBinary(zipReader)
}

func (a *App) extractBinary(zipReader *zip.Reader) error {
	open, err := zipReader.Open("arco")
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", "arco", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer open.Close()

	if err := os.Remove(a.config.ArcoPath); err == nil {
		a.log.Debugf("Removed old binary at %s", a.config.ArcoPath)
	}

	binFile, err := os.OpenFile(a.config.ArcoPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", a.config.ArcoPath, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer binFile.Close()

	if _, err := io.Copy(binFile, open); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", a.config.ArcoPath, err)
	}
	return nil
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

	// Set permissions on the database file
	if err := os.Chmod(filepath.Join(a.config.Dir, "arco.db"), 0600); err != nil {
		return nil, fmt.Errorf("failed to set permissions on database file: %v", err)
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

	binary, err := types.GetLatestBorgBinary(a.config.BorgBinaries)
	if err != nil {
		return err
	}

	// Download the binary
	return util.DownloadFile(a.config.BorgPath, binary.Url)
}

func (a *App) initSystray() error {
	//readyFunc := func() {
	//	systray.SetIcon(a.config.Icon)
	//	systray.SetTitle(Name)
	//	systray.SetTooltip(Name)
	//
	//	mOpen := systray.AddMenuItem(fmt.Sprintf("Open %s", Name), fmt.Sprintf("Open %s", Name))
	//	systray.AddSeparator()
	//	mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %s", Name), fmt.Sprintf("Quit %s", Name))
	//
	//	// Sets the icon of a menu item. Only available on Mac and Windows.
	//	mOpen.SetIcon(a.config.Icon)
	//	mQuit.SetIcon(a.config.Icon)
	//
	//	go func() {
	//		for {
	//			select {
	//			case <-mOpen.ClickedCh:
	//				runtime.WindowShow(a.ctx)
	//			case <-mQuit.ClickedCh:
	//				a.Shutdown(a.ctx)
	//			case <-a.ctx.Done():
	//				return
	//			}
	//		}
	//	}()
	//}
	//
	//exitFunc := func() {
	//	a.Shutdown(a.ctx)
	//}
	//
	//systray.Run(readyFunc, exitFunc)
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
