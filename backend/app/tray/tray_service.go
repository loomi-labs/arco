package tray

import (
	"context"
	"os/exec"

	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// BackupProfileServiceInterface defines the methods needed from backup profile service
type BackupProfileServiceInterface interface {
	GetBackupProfiles(ctx context.Context) ([]*backup_profile.BackupProfile, error)
}

// AppController defines the methods needed from the app for window and quit management
type AppController interface {
	ShowOrCreateMainWindow()
	Quit()
}

// Service manages the system tray menu
type Service struct {
	log                  *zap.SugaredLogger
	backupProfileService BackupProfileServiceInterface
	appController        AppController
	systray              *application.SystemTray
}

// NewService creates a new tray service
func NewService(log *zap.SugaredLogger) *Service {
	return &Service{log: log}
}

// Init initializes the service with its dependencies
func (s *Service) Init(
	backupProfileService BackupProfileServiceInterface,
	appController AppController,
	systray *application.SystemTray,
) {
	s.backupProfileService = backupProfileService
	s.appController = appController
	s.systray = systray
}

// getApp returns the Wails application instance
func (s *Service) getApp() *application.App {
	return application.Get()
}

// BuildMenu creates and sets the tray menu
func (s *Service) BuildMenu() {
	app := s.getApp()

	menu := app.NewMenu()

	// Open main window
	menu.Add("Open").OnClick(func(_ *application.Context) {
		s.appController.ShowOrCreateMainWindow()
	})

	menu.AddSeparator()

	// Add backup profiles
	profiles, err := s.backupProfileService.GetBackupProfiles(app.Context())
	if err != nil {
		s.log.Errorf("Failed to get backup profiles for tray menu: %v", err)
	} else {
		header := menu.Add("Backup Profiles")
		header.SetEnabled(false)

		if len(profiles) == 0 {
			empty := menu.Add("No profiles configured")
			empty.SetEnabled(false)
		} else {
			for _, profile := range profiles {
				s.addProfileSubmenu(menu, profile)
			}
		}

		menu.AddSeparator()
	}

	// Quit
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		s.appController.Quit()
	})

	// Update existing systray with the new menu
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
	}
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
		return
	}
	// Reap child process to prevent zombies
	go func() {
		_ = cmd.Wait()
	}()
}
