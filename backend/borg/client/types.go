package client

import "embed"

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

type Config struct {
	Binaries    embed.FS
	BorgPath    string
	BorgVersion string
}

// -------- borg types --------

type Archive struct {
	Archive  string `json:"archive"`
	Barchive string `json:"barchive"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Start    string `json:"start"`
	Time     string `json:"time"`
}

type Encryption struct {
	Mode string `json:"mode"`
}

type Repository struct {
	ID           string `json:"id"`
	LastModified string `json:"last_modified"`
	Location     string `json:"location"`
}

type ListResponse struct {
	Archives   []Archive  `json:"archives"`
	Encryption Encryption `json:"encryption"`
	Repository Repository `json:"repository"`
}
