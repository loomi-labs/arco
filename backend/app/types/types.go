package types

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/wailsapp/wails/v3/pkg/application"
	"io/fs"
)

const WindowTitle = "Arco"

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

type Icons struct {
	AppIconDark  []byte
	AppIconLight []byte
	DarwinIcons  []byte
}

type Config struct {
	Dir             string
	SSHDir          string
	BorgBinaries    []BorgBinary
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

type Event string

const (
	EventStartupStateChanged   Event = "startupStateChanged"
	EventNotificationAvailable Event = "notificationAvailable"
	EventBackupStateChanged    Event = "backupStateChanged"
	EventPruneStateChanged     Event = "pruneStateChanged"
	EventRepoStateChanged      Event = "repoStateChanged"
	EventArchivesChanged       Event = "archivesChanged"
	EventBackupProfileDeleted  Event = "backupProfileDeleted"
	EventAuthStateChanged      Event = "authStateChanged"
	EventCheckoutStateChanged  Event = "checkoutStateChanged"
	EventSubscriptionAdded     Event = "subscriptionAdded"
	EventSubscriptionCancelled Event = "subscriptionCancelled"
)

var AllEvents = []Event{
	EventStartupStateChanged,
	EventNotificationAvailable,
	EventBackupStateChanged,
	EventPruneStateChanged,
	EventRepoStateChanged,
	EventArchivesChanged,
	EventBackupProfileDeleted,
	EventAuthStateChanged,
	EventCheckoutStateChanged,
	EventSubscriptionAdded,
	EventSubscriptionCancelled,
}

func (e Event) String() string {
	return string(e)
}

func EventBackupStateChangedString(bId BackupId) string {
	return fmt.Sprintf("%s:%d-%d", EventBackupStateChanged.String(), bId.BackupProfileId, bId.RepositoryId)
}

func EventPruneStateChangedString(bId BackupId) string {
	return fmt.Sprintf("%s:%d-%d", EventPruneStateChanged.String(), bId.BackupProfileId, bId.RepositoryId)
}

func EventRepoStateChangedString(repoId int) string {
	return fmt.Sprintf("%s:%d", EventRepoStateChanged.String(), repoId)
}

func EventArchivesChangedString(repoId int) string {
	return fmt.Sprintf("%s:%d", EventArchivesChanged.String(), repoId)
}

func EventCheckoutStateChangedString() string {
	return fmt.Sprintf("%s", EventCheckoutStateChanged.String())
}

func EventSubscriptionAddedString() string {
	return fmt.Sprintf("%s", EventSubscriptionAdded.String())
}

func EventSubscriptionCancelledString() string {
	return fmt.Sprintf("%s", EventSubscriptionCancelled.String())
}

type EventEmitter interface {
	EmitEvent(ctx context.Context, event string)
}

type RuntimeEventEmitter struct{}

func (r *RuntimeEventEmitter) EmitEvent(_ context.Context, event string) {
	application.Get().Event.Emit(event)
}

type MountState struct {
	IsMounted bool   `json:"isMounted"`
	MountPath string `json:"mountPath"`
}
