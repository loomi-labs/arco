package app

import "fmt"

// FrontendError is the error type that is received from the frontend
type FrontendError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "info"
	LevelWarning NotificationLevel = "warning"
	LevelError   NotificationLevel = "error"
)

type Notification struct {
	Message string            `json:"message"`
	Level   NotificationLevel `json:"level"`
}

type MountState struct {
	IsMounted bool   `json:"is_mounted"`
	MountPath string `json:"mount_path"`
}

type BackupIdentifier struct {
	BackupProfileId int
	RepositoryId    int
}

func (b BackupIdentifier) String() string {
	return fmt.Sprintf("BackupProfileId: %d, RepositoryId: %d", b.BackupProfileId, b.RepositoryId)
}
