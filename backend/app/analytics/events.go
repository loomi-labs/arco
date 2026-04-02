package analytics

// EventName is a typed analytics event name.
type EventName string

const (
	// EventAppStarted is emitted once when the app finishes startup.
	EventAppStarted EventName = "app_started"

	// EventPageView is emitted on each frontend page navigation.
	EventPageView EventName = "page_view"

	// EventBackupCompleted is emitted when a backup finishes successfully.
	EventBackupCompleted EventName = "backup_completed"

	// EventBackupFailed is emitted when a backup fails.
	EventBackupFailed EventName = "backup_failed"

	// EventRepositoryCreated is emitted when a repository is created.
	EventRepositoryCreated EventName = "repository_created"

	// EventRepositoryDeleted is emitted when a repository is deleted.
	EventRepositoryDeleted EventName = "repository_deleted"

	// EventProfileCreated is emitted when a backup profile is created.
	EventProfileCreated EventName = "profile_created"

	// EventProfileUpdated is emitted when a backup profile is updated.
	EventProfileUpdated EventName = "profile_updated"

	// EventProfileDeleted is emitted when a backup profile is deleted.
	EventProfileDeleted EventName = "profile_deleted"

	// EventSettingsChanged is emitted when user settings are saved.
	EventSettingsChanged EventName = "settings_changed"

	// EventLoginCompleted is emitted when a user completes authentication.
	EventLoginCompleted EventName = "login_completed"
)

// Property keys for event properties.
const (
	PropPage            = "page"
	PropDurationSeconds = "duration_seconds"
	PropErrorType       = "error_type"
	PropLocationType    = "location_type"
)
