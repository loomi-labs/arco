package types

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/platform"
)

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
	AppIconDark  []byte
	AppIconLight []byte
	DarwinIcons  []byte
}

type Config struct {
	Dir             string
	SSHDir          string
	BorgBinaries    []platform.BorgBinary
	BorgPath        string
	BorgVersion     string
	Icons           *Icons
	Migrations      fs.FS
	GithubAssetName string
	Version         *semver.Version
	CheckForUpdates bool
	CloudRPCURL     string
}

var AllIcons = []backupprofile.Icon{
	backupprofile.IconHome,
	backupprofile.IconBriefcase,
	backupprofile.IconBook,
	backupprofile.IconEnvelope,
	backupprofile.IconCamera,
	backupprofile.IconFire,
}
