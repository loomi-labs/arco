package types

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/settings"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
)

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

type Config struct {
	Dir             string
	BorgBinaries    []BorgBinary
	BorgPath        string
	BorgVersion     string
	Icon            []byte
	Migrations      fs.FS
	GithubAssetName string
	Version         *semver.Version
	ArcoPath        string
}

var AllWeekdays = []backupschedule.Weekday{
	backupschedule.WeekdayMonday,
	backupschedule.WeekdayTuesday,
	backupschedule.WeekdayWednesday,
	backupschedule.WeekdayThursday,
	backupschedule.WeekdayFriday,
	backupschedule.WeekdaySaturday,
	backupschedule.WeekdaySunday,
}

var AllIcons = []backupprofile.Icon{
	backupprofile.IconHome,
	backupprofile.IconBriefcase,
	backupprofile.IconBook,
	backupprofile.IconEnvelope,
	backupprofile.IconCamera,
	backupprofile.IconFire,
}

var AllBackupScheduleModes = []backupschedule.Mode{
	backupschedule.ModeDisabled,
	backupschedule.ModeHourly,
	backupschedule.ModeDaily,
	backupschedule.ModeWeekly,
	backupschedule.ModeMonthly,
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
)

var AllEvents = []Event{
	EventStartupStateChanged,
	EventNotificationAvailable,
	EventBackupStateChanged,
	EventPruneStateChanged,
	EventRepoStateChanged,
	EventArchivesChanged,
	EventBackupProfileDeleted,
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

type EventEmitter interface {
	EmitEvent(ctx context.Context, event string)
}

type RuntimeEventEmitter struct{}

func (r *RuntimeEventEmitter) EmitEvent(ctx context.Context, event string) {
	runtime.EventsEmit(ctx, event)
}

type MountState struct {
	IsMounted bool   `json:"isMounted"`
	MountPath string `json:"mountPath"`
}

var AllThemes = []settings.Theme{
	settings.ThemeSystem,
	settings.ThemeDark,
	settings.ThemeLight,
}
