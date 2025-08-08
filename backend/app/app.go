package app

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v66/github"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/plan"
	"github.com/loomi-labs/arco/backend/app/repository"
	appstate "github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/subscription"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	internalauth "github.com/loomi-labs/arco/backend/internal/auth"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/pressly/goose/v3"
	"github.com/teamwork/reload"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

const (
	Name = "Arco"
)

var (
	Version = "v0.0.0"
)

type EnvVar string

const (
	EnvVarDebug           EnvVar = "ARCO_DEBUG"
	EnvVarDevelopment     EnvVar = "ARCO_DEVELOPMENT"
	EnvVarStartPage       EnvVar = "ARCO_START_PAGE"
	EnvVarCloudRPCURL     EnvVar = "ARCO_CLOUD_RPC_URL"
	EnvVarEnableLoginBeta EnvVar = "ARCO_ENABLE_LOGIN_BETA"
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
	eventEmitter             types.EventEmitter
	shouldQuit               bool

	// Startup
	ctx                  context.Context
	cancel               context.CancelFunc
	db                   *ent.Client
	authService          *auth.ServiceInternal
	planService          *plan.ServiceInternal
	subscriptionService  *subscription.ServiceInternal
	repositoryService    *repository.ServiceInternal
	backupProfileService *backup_profile.ServiceInternal
}

func NewApp(
	log *zap.SugaredLogger,
	config *types.Config,
	eventEmitter types.EventEmitter,
) *App {
	state := appstate.NewState(log, eventEmitter)
	sshPrivateKeys := util.SearchSSHKeys(log, config.SSHDir)
	return &App{
		log:                      log,
		config:                   config,
		state:                    state,
		borg:                     borg.NewBorg(config.BorgPath, log, sshPrivateKeys, nil),
		backupScheduleChangedCh:  make(chan struct{}),
		pruningScheduleChangedCh: make(chan struct{}),
		eventEmitter:             eventEmitter,
		shouldQuit:               false,
		authService:              auth.NewService(log, state),
		planService:              plan.NewService(log, state),
		subscriptionService:      subscription.NewService(log, state),
		repositoryService:        repository.NewService(log, state),
		backupProfileService:     backup_profile.NewService(log, state, config),
	}
}

// These clients separate the different types of operations that can be performed with the Borg client
// This makes it easier to expose them in a clean way to the frontend

// AppClient is a client for application related operations
type AppClient App

func (a *App) BackupProfileService() *backup_profile.Service {
	return a.backupProfileService.Service
}

func (a *App) RepositoryService() *repository.Service {
	return a.repositoryService.Service
}

func (a *App) AppClient() *AppClient {
	return (*AppClient)(a)
}

func (a *App) AuthService() *auth.Service {
	return a.authService.Service
}

func (a *App) PlanService() *plan.Service {
	return a.planService.Service
}

func (a *App) SubscriptionService() *subscription.Service {
	return a.subscriptionService.Service
}

func (a *App) Startup(ctx context.Context) {
	a.log.Infof("Running Arco version %s", a.config.Version.String())
	a.ctx, a.cancel = context.WithCancel(ctx)

	if a.config.CheckForUpdates {
		// Update Arco binary if necessary
		needsRestart, err := a.updateArco()
		if err != nil {
			a.state.SetStartupStatus(a.ctx, a.state.GetStartupState().Status, err)
			a.log.Error(err)
			return
		}
		// Restart if an update has been performed
		if needsRestart {
			a.log.Info("Updated Arco binary... restarting")
			a.state.SetStartupStatus(a.ctx, appstate.StartupStatusRestartingArco, nil)

			// Sleep for a few seconds to allow the frontend to show the update message
			time.Sleep(3 * time.Second)
			reload.Exec()
			return
		}
	}

	// Initialize the database
	db, err := a.initDb()
	if err != nil {
		a.state.SetStartupStatus(a.ctx, a.state.GetStartupState().Status, err)
		a.log.Error(err)
		return
	}
	a.db = db
	a.config.Migrations = nil // Free up memory

	// Create JWT interceptor and HTTP client for cloud services
	jwtInterceptor := internalauth.NewJWTAuthInterceptor(a.log, a.authService, a.db, a.state)
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create unauthenticated RPC clients
	authRPCClient := arcov1connect.NewAuthServiceClient(
		httpClient,
		a.config.CloudRPCURL,
	)

	// Create authenticated RPC clients
	planRPCClient := arcov1connect.NewPlanServiceClient(
		httpClient,
		a.config.CloudRPCURL,
		connect.WithInterceptors(jwtInterceptor.UnaryInterceptor()),
	)

	subscriptionRPCClient := arcov1connect.NewSubscriptionServiceClient(
		httpClient,
		a.config.CloudRPCURL,
		connect.WithInterceptors(jwtInterceptor.UnaryInterceptor()),
	)
	cloudRepositoryRPCClient := arcov1connect.NewRepositoryServiceClient(
		httpClient,
		a.config.CloudRPCURL,
		connect.WithInterceptors(jwtInterceptor.UnaryInterceptor()),
	)

	// Initialize services with database and authenticated RPC clients
	a.authService.Init(a.db, authRPCClient)
	a.planService.Init(a.db, planRPCClient)
	a.subscriptionService.Init(a.db, subscriptionRPCClient)

	// Create cloud repository service first
	cloudRepositoryService := repository.NewCloudRepositoryClient(a.log, a.state, a.config)
	cloudRepositoryService.Init(a.db, cloudRepositoryRPCClient)

	// Initialize repository service with full dependencies
	a.repositoryService.Init(
		a.db,
		a.borg,
		a.config,
		a.eventEmitter,
		cloudRepositoryService,
	)

	// Initialize backup profile service with repository service dependency
	a.backupProfileService.Init(a.ctx, a.db, a.eventEmitter, a.backupScheduleChangedCh, a.pruningScheduleChangedCh, a.repositoryService)

	// Ensure Borg binary is installed
	if err := a.ensureBorgBinary(); err != nil {
		a.state.SetStartupStatus(a.ctx, a.state.GetStartupState().Status, err)
		a.log.Error(err)
		return
	}

	// Set a general status for the rest of the startup process
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusInitializingApp, nil)

	// Recover any pending authentication sessions
	if err := a.authService.RecoverAuthSessions(a.ctx); err != nil {
		a.log.Errorf("Failed to recover authentication sessions: %v", err)
		// Don't fail startup for session recovery errors, just log them
	}

	// Validate and clean up stored JWT
	if err := a.authService.ValidateAndRenewStoredTokens(a.ctx); err != nil {
		a.log.Errorf("Failed to validate stored tokens: %v", err)
		// Don't fail startup for token validation errors, just log them
	}

	// Save mount states
	a.repositoryService.SetMountStates(a.ctx)

	// Start ArcoCloud sync listener
	go a.startArcoCloudSyncListener()

	// Schedule backups
	go a.backupProfileService.StartScheduleChangeListener()
	go a.backupProfileService.StartPruneScheduleChangeListener()
	a.backupScheduleChangedCh <- struct{}{}  // Trigger initial backup schedule check
	a.pruningScheduleChangedCh <- struct{}{} // Trigger initial pruning schedule check

	// Set the app as ready
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusReady, nil)
}

func (a *App) Shutdown() {
	a.log.Info(fmt.Sprintf("Shutting down %s", Name))
	a.cancel()
	err := a.db.Close()
	if err != nil {
		a.log.Error("Failed to close database connection")
	}
	//os.Exit(0)
}

func (a *App) SetQuit() {
	a.shouldQuit = true
}

func (a *App) ShouldQuit() bool {
	a.log.Debug("ShouldQuit called")
	return a.state.GetStartupState().Error != "" || a.shouldQuit
}

func (a *App) startArcoCloudSyncListener() {
	a.log.Debug("Starting ArcoCloud sync listener")

	syncArcoCloudData := func() {
		go func() {
			_, err := a.repositoryService.SyncCloudRepositories(a.ctx)
			if err != nil {
				a.log.Error(err)
			}
		}()
	}

	// Initial sync if authenticated
	if a.state.GetAuthState().IsAuthenticated {
		syncArcoCloudData()
	}

	// Listen for auth state changes using Wails event system
	application.Get().Event.On(types.EventAuthStateChanged.String(), func(event *application.CustomEvent) {
		isAuthenticated := a.state.GetAuthState().IsAuthenticated
		a.log.Debugf("Auth state changed, authenticated: %v", isAuthenticated)

		if isAuthenticated {
			// User became authenticated - sync data
			syncArcoCloudData()
		}
	})
}

func (a *App) updateArco() (bool, error) {
	if EnvVarDevelopment.Bool() {
		a.log.Info("Development mode enabled, skipping update check")
		return false, nil
	}
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusCheckingForUpdates, nil)

	client := github.NewClient(nil)

	release, err := a.getLatestRelease(client)
	if err != nil {
		// We don't want to fail the startup process if the update check fails
		a.log.Errorf("Failed to check for updates: %v", err)
		return false, nil
	}

	releaseVersion, err := semver.NewVersion(release.GetTagName())
	if err != nil {
		return false, fmt.Errorf("failed to parse release version: %w", err)
	}

	if releaseVersion.LessThanEqual(a.config.Version) {
		a.log.Info("No updates available")
		return false, nil
	}

	a.log.Infof("Updating Arco binary to version %s", releaseVersion.String())
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusApplyingUpdates, nil)

	releaseAsset, err := a.findReleaseAsset(release)
	if err != nil {
		return false, err
	}

	// Get execution path
	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}
	path, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return false, err
	}

	err = a.downloadReleaseAsset(client, releaseAsset, path)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) getLatestRelease(client *github.Client) (*github.RepositoryRelease, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	release, _, err := client.Repositories.GetLatestRelease(ctx, "loomi-labs", "arco")
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

func (a *App) downloadReleaseAsset(client *github.Client, asset *github.ReleaseAsset, path string) error {
	httpClient := &http.Client{Timeout: time.Second * 30}
	readCloser, _, err := client.Repositories.DownloadReleaseAsset(a.ctx, "loomi-labs", "arco", *asset.ID, httpClient)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	if readCloser == nil {
		return fmt.Errorf("failed to download release asset: readCloser is nil")
	}
	defer readCloser.Close()

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
	defer buf.Reset()

	return a.extractBinary(zipReader, path)
}

func (a *App) extractBinary(zipReader *zip.Reader, path string) error {
	arcoFilePath := "arco"

	open, err := zipReader.Open(arcoFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", arcoFilePath, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer open.Close()

	if err := os.Remove(path); err == nil {
		a.log.Debugf("Removed old binary at %s", path)
	}

	binFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer binFile.Close()

	if _, err := io.Copy(binFile, open); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", path, err)
	}
	return nil
}

func (a *App) applyMigrations(dbSource string) error {
	db, err := sql.Open(dialect.SQLite, dbSource)
	if err != nil {
		return fmt.Errorf("failed opening connection to sqlite: %v", err)
	}

	defer func(db *sql.Driver) {
		err := db.Close()
		if err != nil {
			a.log.Error("failed to close database connection")
		}
	}(db)

	// Add a prefix and suffix to the migrations (required by goose)
	gooseMigrations := &util.CustomFS{
		FS:     a.config.Migrations,
		Prefix: "-- +goose Up\n-- +goose StatementBegin\n",
		Suffix: "\n-- +goose StatementEnd\n",
	}

	goose.SetLogger(util.NewGooseLogger(a.log))
	goose.SetBaseFS(gooseMigrations)

	if err := goose.SetDialect(dialect.SQLite); err != nil {
		return fmt.Errorf("failed to set dialect: %v", err)
	}

	if err := goose.Up(db.DB(), "."); err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}
	return nil
}

func (a *App) initDb() (*ent.Client, error) {
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusInitializingDatabase, nil)

	// - Set WAL mode (not strictly necessary each time because it's persisted in the database, but good for first run)
	// - Set busy timeout, so concurrent writers wait on each other instead of erroring immediately
	// - Enable foreign key checks
	opts := "?_journal=WAL&_timeout=5000&_fk=1"
	dbSource := fmt.Sprintf("file:%s%s", filepath.Join(a.config.Dir, "arco.db"), opts)

	// Apply migrations
	if err := a.applyMigrations(dbSource); err != nil {
		return nil, err
	}

	// Set restrictive file permissions (owner read/write only) on the database file
	if err := os.Chmod(filepath.Join(a.config.Dir, "arco.db"), 0600); err != nil {
		return nil, fmt.Errorf("failed to set permissions on database file: %v", err)
	}

	dbClient, err := ent.Open(dialect.SQLite, dbSource)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %v", err)
	}
	return dbClient, nil
}

func (a *App) ensureBorgBinary() error {
	a.state.SetStartupStatus(a.ctx, appstate.StartupStatusCheckingForBorgUpdates, nil)
	if !a.isTargetVersionInstalled(a.config.BorgVersion) {
		a.state.SetStartupStatus(a.ctx, appstate.StartupStatusUpdatingBorg, nil)
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

// rollback calls to tx.Rollback and wraps the given error
// with the rollback error if occurred.
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}
