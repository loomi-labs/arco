package types

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/platform"
)

// ============================================================================
// BACKUP STATUS TYPES
// ============================================================================

// BackupStatus represents the status of the last backup
type BackupStatus string

const (
	BackupStatusSuccess BackupStatus = "success"
	BackupStatusWarning BackupStatus = "warning"
	BackupStatusError   BackupStatus = "error"
)

// LastBackup contains info about the last successful backup
type LastBackup struct {
	Timestamp      *time.Time `json:"timestamp,omitempty"`
	WarningMessage string     `json:"warningMessage,omitempty"`
}

// LastAttempt contains info about the last backup attempt (success, warning, or error)
type LastAttempt struct {
	Status    BackupStatus `json:"status"`
	Timestamp *time.Time   `json:"timestamp,omitempty"`
	Message   string       `json:"message,omitempty"`
}

const WindowTitle = "Arco"

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

type BackupId struct {
	BackupProfileId int `json:"backupProfileId"`
	RepositoryId    int `json:"repositoryId"`
}

func (b BackupId) String() string {
	return fmt.Sprintf("BackupProfileId: %d, RepositoryId: %d", b.BackupProfileId, b.RepositoryId)
}

type ExaminePruningResult struct {
	BackupID               BackupId
	RepositoryName         string
	CntArchivesToBeDeleted int
	Error                  error
}

// DeleteResult represents the result of a delete operation
type DeleteResult string

const (
	DeleteResultSuccess   DeleteResult = "success"
	DeleteResultCancelled DeleteResult = "cancelled"
	DeleteResultError     DeleteResult = "error"
)

type Icons struct {
	AppIconDark       []byte
	AppIconLight      []byte
	DarwinIcons       []byte
	DarwinMenubarIcon []byte
}

type Config struct {
	Dir              string
	SSHDir           string
	KeyringDir       string
	BorgBinaries     []platform.BorgBinary
	BorgPath         string // Base path for borg (directory for .tgz distributions, file for single binaries)
	BorgExePath      string // Actual executable path (same as BorgPath for single binaries, BorgPath/borg.exe for directories)
	BorgMountPath    string // Base path for mount binary (for cleanup tracking)
	BorgMountExePath string // Borg for mount operations (FUSE support) - may differ from BorgExePath on macOS
	BorgMountBinary  platform.BorgBinary // The mount binary info (for version checking and URL)
	BorgMountVersion string              // Version of the mount binary
	BorgVersion      string
	Icons            *Icons
	Migrations       fs.FS
	GithubAssetName  string
	Version          *semver.Version
	CheckForUpdates  bool
	CloudRPCURL      string
}

var AllIcons = []backupprofile.Icon{
	backupprofile.IconHome,
	backupprofile.IconBriefcase,
	backupprofile.IconBook,
	backupprofile.IconEnvelope,
	backupprofile.IconCamera,
	backupprofile.IconFire,
}
