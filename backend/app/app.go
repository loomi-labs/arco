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
	entsql "entgo.io/ent/dialect/sql"
	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v66/github"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/notification"
	"github.com/loomi-labs/arco/backend/app/plan"
	"github.com/loomi-labs/arco/backend/app/repository"
	appstate "github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/subscription"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/app/user"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	internalauth "github.com/loomi-labs/arco/backend/internal/auth"
	"github.com/loomi-labs/arco/backend/internal/keyring"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/pkg/browser"
	"github.com/pressly/goose/v3"
	"github.com/teamwork/reload"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

const (
	Name = "Arco"
)

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
	keyring              *keyring.Service
	userService          *user.ServiceInternal
	authService          *auth.ServiceInternal
	planService          *plan.ServiceInternal
	subscriptionService  *subscription.ServiceInternal
	repositoryService    *repository.ServiceInternal
	backupProfileService *backup_profile.ServiceInternal
	notificationService  *notification.Service
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
		userService:              user.NewService(log, state),
		authService:              auth.NewService(log, state),
		planService:              plan.NewService(log, state),
		subscriptionService:      subscription.NewService(log, state),
		repositoryService:        repository.NewService(log, config),
		backupProfileService:     backup_profile.NewService(log, state, config),
		notificationService:      notification.NewService(log),
	}
}

func (a *App) UserService() *user.Service {
	return a.userService.Service
}

func (a *App) BackupProfileService() *backup_profile.Service {
	return a.backupProfileService.Service
}

func (a *App) RepositoryService() *repository.Service {
	return a.repositoryService.Service
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

func (a *App) Keyring() *keyring.Service {
	return a.keyring
}

func (a *App) NotificationService() *notification.Service {
	return a.notificationService
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
	jwtInterceptor := internalauth.NewJWTAuthInterceptor(a.log, a.authService, a.db, a.state, a.keyring)
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
	a.userService.Init(a.db, a.eventEmitter)
	a.notificationService.Init(a.db, a.eventEmitter)
	a.authService.Init(a.db, authRPCClient, a.keyring)
	a.planService.Init(a.db, planRPCClient)
	a.subscriptionService.Init(a.db, subscriptionRPCClient)

	cloudRepositoryService := repository.NewCloudRepositoryClient(a.log, a.state, a.config)
	cloudRepositoryService.Init(a.db, cloudRepositoryRPCClient)

	a.repositoryService.Init(a.ctx, a.db, a.eventEmitter, a.borg, cloudRepositoryService, a.keyring)

	// Initialize backup profile service with repository service dependency
	a.backupProfileService.Init(a.ctx, a.db, a.eventEmitter, a.backupScheduleChangedCh, a.pruningScheduleChangedCh, a.repositoryService)

	// Check for macFUSE on macOS
	if platform.IsMacOS() && !platform.IsMacFUSEInstalled() {
		a.log.Warn("macFUSE is not installed")
		a.state.SetStartupStatus(a.ctx, a.state.GetStartupState().Status,
			fmt.Errorf("macFUSE is required for Arco to function. Please install it from https://macfuse.github.io and restart Arco"))
		// Open download page
		_ = browser.OpenURL("https://macfuse.github.io")
		return // Stop startup but keep app open showing error
	}

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
	// Clean up any old app bundles from previous updates
	a.cleanupOldAppBundles()

	if types.EnvVarDevelopment.Bool() {
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

	// On macOS, resolve to .app bundle path instead of binary path
	// e.g., /Users/foo/Applications/arco.app/Contents/MacOS/arco -> /Users/foo/Applications/arco.app
	if platform.IsMacOS() {
		path = a.resolveAppBundlePath(path)
		a.log.Debugf("Resolved app bundle path: %s", path)
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

	release, _, err := client.Repositories.GetLatestRelease(ctx, "shifty11", "arco-test")
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
	httpClient := &http.Client{Timeout: time.Second * 60}
	readCloser, _, err := client.Repositories.DownloadReleaseAsset(a.ctx, "shifty11", "arco-test", *asset.ID, httpClient)
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

	if platform.IsMacOS() {
		return a.extractAppBundle(zipReader, path)
	}
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

// cleanupOldAppBundles removes any leftover .app.old bundles from previous updates.
// This is called at startup to clean up after a successful update.
func (a *App) cleanupOldAppBundles() {
	if !platform.IsMacOS() {
		return
	}
	execPath, err := os.Executable()
	if err != nil {
		a.log.Debugf("cleanupOldAppBundles: os.Executable failed: %v", err)
		return
	}
	path, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		a.log.Debugf("cleanupOldAppBundles: EvalSymlinks failed: %v", err)
		return
	}
	appBundlePath := a.resolveAppBundlePath(path)
	if !strings.HasSuffix(appBundlePath, ".app") {
		a.log.Debugf("cleanupOldAppBundles: skip; not an app bundle path: %s", appBundlePath)
		return
	}
	oldAppPath := appBundlePath + ".old"
	if _, err := os.Stat(oldAppPath); err == nil {
		a.log.Infof("Cleaning up old app bundle: %s", oldAppPath)
		if err := os.RemoveAll(oldAppPath); err != nil {
			a.log.Warnf("Failed to clean up old app bundle: %v", err)
		}
	}
}

// resolveAppBundlePath resolves the binary path to the .app bundle path on macOS.
// e.g., /Users/foo/Applications/arco.app/Contents/MacOS/arco -> /Users/foo/Applications/arco.app
func (a *App) resolveAppBundlePath(binaryPath string) string {
	// Walk up the path to find the .app bundle
	path := binaryPath
	for path != "/" && path != "." {
		if strings.HasSuffix(path, ".app") {
			return path
		}
		path = filepath.Dir(path)
	}
	// Fallback: return the original path (should not happen in a valid .app bundle)
	a.log.Warnf("Could not find .app bundle in path: %s", binaryPath)
	return binaryPath
}

// extractAppBundle extracts the entire .app bundle from the ZIP to replace the existing bundle.
// This is required on macOS to preserve code signatures.
func (a *App) extractAppBundle(zipReader *zip.Reader, appBundlePath string) error {
	// Extract to a temp directory first for atomic replacement
	tempDir, err := os.MkdirTemp(filepath.Dir(appBundlePath), "arco-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp dir on failure

	tempAppPath := filepath.Join(tempDir, "arco.app")

	// Extract all files from the ZIP
	for _, file := range zipReader.File {
		// The ZIP contains arco.app/... so we need to extract it to tempDir
		destPath := filepath.Join(tempDir, file.Name)

		// Validate path to prevent Zip Slip attacks
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(tempDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in zip: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
		}

		// Extract file
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file %s: %w", file.Name, err)
		}

		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create file %s: %w", destPath, err)
		}

		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
	}

	// Verify the extraction produced an app bundle
	if _, err := os.Stat(tempAppPath); os.IsNotExist(err) {
		return fmt.Errorf("extracted ZIP does not contain arco.app bundle")
	}

	// Rename old app bundle to .old (instead of deleting, which fails on macOS due to code signature protection)
	// The running app continues via inodes even after its path is renamed.
	// The .old bundle will be cleaned up on next startup.
	if !strings.HasSuffix(appBundlePath, ".app") {
		return fmt.Errorf("refusing to replace non-.app path on macOS: %s", appBundlePath)
	}
	oldAppPath := appBundlePath + ".old"
	if _, err := os.Stat(appBundlePath); err == nil {
		_ = os.RemoveAll(oldAppPath) // best-effort cleanup of previous .old
		a.log.Debugf("Renaming old app bundle from %s to %s", appBundlePath, oldAppPath)
		if err := os.Rename(appBundlePath, oldAppPath); err != nil {
			return fmt.Errorf("failed to rename old app bundle: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat existing app bundle: %w", err)
	}

	// Move new app bundle into place
	a.log.Debugf("Moving new app bundle from %s to %s", tempAppPath, appBundlePath)
	if err := os.Rename(tempAppPath, appBundlePath); err != nil {
		return fmt.Errorf("failed to move new app bundle: %w", err)
	}

	return nil
}

func (a *App) applyMigrations(dbSource string) error {
	db, err := entsql.Open(dialect.SQLite, dbSource)
	if err != nil {
		return fmt.Errorf("failed opening connection to sqlite: %v", err)
	}

	defer func(db *entsql.Driver) {
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

	// Phase 1: Migrate up to the version that adds has_password column
	// This ensures the column exists before we try to set it
	if err := goose.UpTo(db.DB(), ".", 20251217094826); err != nil {
		return fmt.Errorf("failed to apply migrations phase 1: %v", err)
	}

	// Phase 2: Migrate credentials to keyring (sets has_password = true)
	// This runs AFTER has_password column exists but BEFORE password columns are dropped
	if err := a.migrateCredentialsToKeyring(db.DB()); err != nil {
		// Log but don't fail - user may have no credentials yet or keyring may be unavailable
		a.log.Warnf("Credential migration to keyring: %v", err)
	}

	// Phase 3: Run remaining migrations
	if err := goose.Up(db.DB(), "."); err != nil {
		return fmt.Errorf("failed to apply migrations phase 2: %v", err)
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

	binary, err := platform.GetLatestBorgBinary(a.config.BorgBinaries)
	if err != nil {
		return err
	}

	// Download the binary
	return util.DownloadFile(a.config.BorgPath, binary.Url)
}
