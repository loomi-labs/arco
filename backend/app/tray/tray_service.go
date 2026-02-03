package tray

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/repository"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// BackupProfileServiceInterface defines the methods needed from backup profile service
type BackupProfileServiceInterface interface {
	GetBackupProfiles(ctx context.Context) ([]*backup_profile.BackupProfile, error)
}

// RepositoryServiceInterface defines the methods needed from repository service
type RepositoryServiceInterface interface {
	QueueBackups(ctx context.Context, backupIds []types.BackupId) ([]string, error)
	GetBackupState(ctx context.Context, backupId types.BackupId) (*statemachine.Backup, error)
	GetLastArchiveByBackupId(ctx context.Context, backupId types.BackupId) (*ent.Archive, error)
	MountArchive(ctx context.Context, archiveId int) (*repository.MountResult, error)
	GetBackupButtonStatus(ctx context.Context, backupIds []types.BackupId) (repository.BackupButtonStatus, error)
}

// AppController defines the methods needed from the app for window and quit management
type AppController interface {
	ShowOrCreateMainWindow()
	Quit()
}

// Service manages the system tray menu
type Service struct {
	log                  *zap.SugaredLogger
	systray              *application.SystemTray
	backupProfileService BackupProfileServiceInterface
	repositoryService    RepositoryServiceInterface
	appController        AppController
}

// NewService creates a new tray service
func NewService(log *zap.SugaredLogger) *Service {
	return &Service{log: log}
}

// Init initializes the service with its dependencies
func (s *Service) Init(
	backupProfileService BackupProfileServiceInterface,
	repositoryService RepositoryServiceInterface,
	appController AppController,
	systray *application.SystemTray,
) {
	s.backupProfileService = backupProfileService
	s.repositoryService = repositoryService
	s.appController = appController
	s.systray = systray
}

// getApp returns the Wails application instance
func (s *Service) getApp() *application.App {
	return application.Get()
}

// BuildMenu creates the full tray menu from current state
func (s *Service) BuildMenu() {
	app := s.getApp()
	menu := app.NewMenu()

	// Open main window
	menu.Add("Open").OnClick(func(_ *application.Context) {
		s.appController.ShowOrCreateMainWindow()
	})

	menu.AddSeparator()

	// Add backup profiles (use fresh context for menu build, not for handlers)
	profiles, err := s.backupProfileService.GetBackupProfiles(app.Context())
	if err != nil {
		s.log.Errorf("Failed to get backup profiles for tray menu: %v", err)
	} else {
		for _, profile := range profiles {
			s.addProfileSubmenu(menu, profile)
		}

		if len(profiles) > 0 {
			menu.AddSeparator()
		}
	}

	// Quit
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		s.appController.Quit()
	})

	s.systray.SetMenu(menu)
}

// addProfileSubmenu adds a submenu for a backup profile
func (s *Service) addProfileSubmenu(menu *application.Menu, profile *backup_profile.BackupProfile) {
	submenu := menu.AddSubmenu(profile.Name)

	// Source Folders header
	if len(profile.BackupPaths) > 0 {
		header := submenu.Add("Source Folders")
		header.SetEnabled(false)

		// Add each source folder as clickable item
		for _, path := range profile.BackupPaths {
			pathCopy := path // Capture for closure
			submenu.Add("  " + path).OnClick(func(_ *application.Context) {
				s.openFolder(pathCopy)
			})
		}

		submenu.AddSeparator()
	}

	// Check if actions are available (use fresh context for menu build)
	ctx := s.getApp().Context()
	canBackup := s.canStartBackup(ctx, profile)
	canBrowse := s.canBrowseBackup(ctx, profile)

	// Run Backup Now
	profileCopy := profile // Capture for closure
	runBackupItem := submenu.Add("Run Backup Now")
	if canBackup {
		runBackupItem.OnClick(func(_ *application.Context) {
			// Get fresh context at click time, not stale context from menu build
			s.handleRunBackupNow(s.getApp().Context(), profileCopy)
		})
	} else {
		runBackupItem.SetEnabled(false)
	}

	// Browse Latest Backup
	browseItem := submenu.Add("Browse Latest Backup")
	if canBrowse {
		browseItem.OnClick(func(_ *application.Context) {
			// Get fresh context at click time, not stale context from menu build
			s.handleBrowseLatestBackup(s.getApp().Context(), profileCopy)
		})
	} else {
		browseItem.SetEnabled(false)
	}

	// Status line
	statusText := s.getStatusText(ctx, profile)
	status := submenu.Add(statusText)
	status.SetEnabled(false)
}

// handleRunBackupNow queues backups for all repositories of a profile
func (s *Service) handleRunBackupNow(ctx context.Context, profile *backup_profile.BackupProfile) {
	if len(profile.Repositories) == 0 {
		s.log.Warnf("Profile %s has no repositories", profile.Name)
		return
	}

	backupIds := make([]types.BackupId, len(profile.Repositories))
	for i, repo := range profile.Repositories {
		backupIds[i] = types.BackupId{
			BackupProfileId: profile.ID,
			RepositoryId:    repo.ID,
		}
	}

	_, err := s.repositoryService.QueueBackups(ctx, backupIds)
	if err != nil {
		s.log.Errorf("Failed to queue backups for profile %s: %v", profile.Name, err)
	} else {
		s.log.Infof("Queued backups for profile %s", profile.Name)
	}
}

// handleBrowseLatestBackup finds and mounts the newest archive across all repositories
func (s *Service) handleBrowseLatestBackup(ctx context.Context, profile *backup_profile.BackupProfile) {
	archive, err := s.getNewestArchive(ctx, profile)
	if err != nil {
		s.log.Errorf("Failed to get newest archive for profile %s: %v", profile.Name, err)
		return
	}

	if archive == nil {
		s.log.Warnf("No archives found for profile %s", profile.Name)
		return
	}

	_, err = s.repositoryService.MountArchive(ctx, archive.ID)
	if err != nil {
		s.log.Errorf("Failed to mount archive for profile %s: %v", profile.Name, err)
	}
}

// getNewestArchive finds the newest archive across all repositories for a profile
func (s *Service) getNewestArchive(ctx context.Context, profile *backup_profile.BackupProfile) (*ent.Archive, error) {
	var newestArchive *ent.Archive

	for _, repo := range profile.Repositories {
		backupId := types.BackupId{
			BackupProfileId: profile.ID,
			RepositoryId:    repo.ID,
		}
		archive, err := s.repositoryService.GetLastArchiveByBackupId(ctx, backupId)
		if err != nil || archive == nil {
			continue
		}
		if newestArchive == nil || archive.CreatedAt.After(newestArchive.CreatedAt) {
			newestArchive = archive
		}
	}

	return newestArchive, nil
}

// canStartBackup checks if a backup can be started for a profile
// Returns true only if ALL repositories are idle and ready
func (s *Service) canStartBackup(ctx context.Context, profile *backup_profile.BackupProfile) bool {
	if len(profile.Repositories) == 0 {
		return false
	}

	for _, repo := range profile.Repositories {
		backupId := types.BackupId{
			BackupProfileId: profile.ID,
			RepositoryId:    repo.ID,
		}
		status, err := s.repositoryService.GetBackupButtonStatus(ctx, []types.BackupId{backupId})
		if err != nil {
			return false
		}
		// Only allow if status is "runBackup" (repository idle and ready)
		if status != repository.BackupButtonStatusRunBackup {
			return false
		}
	}
	return true
}

// canBrowseBackup checks if browsing/mounting is possible for a profile
// Returns true if at least one repository can be mounted (idle, queued, or already mounted)
func (s *Service) canBrowseBackup(ctx context.Context, profile *backup_profile.BackupProfile) bool {
	if len(profile.Repositories) == 0 {
		return false
	}

	// First check if there are any archives to browse
	archive, _ := s.getNewestArchive(ctx, profile)
	if archive == nil {
		return false
	}

	// Check if at least one repository is available for mounting
	for _, repo := range profile.Repositories {
		backupId := types.BackupId{
			BackupProfileId: profile.ID,
			RepositoryId:    repo.ID,
		}
		status, err := s.repositoryService.GetBackupButtonStatus(ctx, []types.BackupId{backupId})
		if err != nil {
			continue
		}
		// Can browse if repo is idle, already mounted, or just queued
		switch status {
		case repository.BackupButtonStatusRunBackup, // Idle
			repository.BackupButtonStatusUnmount, // Already mounted
			repository.BackupButtonStatusWaiting: // Queued but can still mount
			return true
		case repository.BackupButtonStatusAbort, // Backup running - can't mount
			repository.BackupButtonStatusBusy,   // Other operation running
			repository.BackupButtonStatusLocked: // Error state
			// Continue to next repository
		}
	}
	return false
}

// getStatusText returns the status text for a profile
func (s *Service) getStatusText(ctx context.Context, profile *backup_profile.BackupProfile) string {
	// Check if any backup is running
	for _, repo := range profile.Repositories {
		backupId := types.BackupId{
			BackupProfileId: profile.ID,
			RepositoryId:    repo.ID,
		}
		state, err := s.repositoryService.GetBackupState(ctx, backupId)
		if err == nil && state != nil && state.Progress != nil {
			progress := state.Progress
			if progress.TotalFiles > 0 {
				percentage := (progress.ProcessedFiles * 100) / progress.TotalFiles
				return fmt.Sprintf("Running... %d%%", percentage)
			}
			return "Running..."
		}
	}

	// Show last attempt result
	if profile.LastAttempt != nil && profile.LastAttempt.Timestamp != nil {
		timeAgo := formatTimeAgo(*profile.LastAttempt.Timestamp)
		return fmt.Sprintf("%s (%s)", profile.LastAttempt.Status, timeAgo)
	}

	// Show next scheduled run
	if profile.BackupSchedule != nil && !profile.BackupSchedule.NextRun.IsZero() {
		return fmt.Sprintf("Next: %s", profile.BackupSchedule.NextRun.Format("3:04 PM"))
	}

	return "Not scheduled"
}

// formatTimeAgo formats a timestamp as a human-readable "time ago" string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	return fmt.Sprintf("%dd ago", days)
}

// openFolder opens a folder in the system file manager
func (s *Service) openFolder(path string) {
	openCmd, err := platform.GetOpenFileManagerCmd()
	if err != nil {
		s.log.Errorf("Error getting open file manager command: %v", err)
		return
	}

	cmd := exec.Command(openCmd, path)
	if err := cmd.Start(); err != nil {
		s.log.Errorf("Error opening folder %s: %v", path, err)
	}
}

// SubscribeToEvents subscribes to events that should trigger menu refresh
func (s *Service) SubscribeToEvents() {
	events := []string{
		types.EventBackupProfileCreatedString(),
		types.EventBackupProfileUpdatedString(),
		types.EventBackupProfileDeleted.String(),
		types.EventBackupStateChanged.String(),
		types.EventRepoStateChanged.String(),
		types.EventArchivesChanged.String(),
	}

	app := s.getApp()
	for _, event := range events {
		eventCopy := event // Capture for closure
		app.Event.On(eventCopy, func(_ *application.CustomEvent) {
			s.log.Debugf("Tray menu refresh triggered by event: %s", eventCopy)
			s.BuildMenu()
		})
	}
}
