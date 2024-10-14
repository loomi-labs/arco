package types

import (
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/settings"
	"embed"
	"fmt"
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
	Dir         string
	Binaries    []Binary
	BorgPath    string
	BorgVersion string
	Icon        embed.FS
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

type Event string

const (
	EventNotificationAvailable Event = "notificationAvailable"
	EventBackupStateChanged    Event = "backupStateChanged"
	EventRepoStateChanged      Event = "repoStateChanged"
)

var AllEvents = []Event{
	EventNotificationAvailable,
	EventBackupStateChanged,
	EventRepoStateChanged,
}

func (e Event) String() string {
	return string(e)
}

func EventBackupStateChangedString(bId BackupId) string {
	return fmt.Sprintf("%s:%d-%d", EventBackupStateChanged.String(), bId.BackupProfileId, bId.RepositoryId)
}

func EventRepoStateChangedString(repoId int) string {
	return fmt.Sprintf("%s:%d", EventRepoStateChanged.String(), repoId)
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
