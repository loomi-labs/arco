//go:build !integration

package types

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Event string

const (
	EventStartupStateChanged    Event = "startupStateChanged"
	EventNotificationAvailable  Event = "notificationAvailable"
	EventBackupStateChanged     Event = "backupStateChanged"
	EventPruneStateChanged      Event = "pruneStateChanged"
	EventRepoStateChanged       Event = "repoStateChanged"
	EventArchivesChanged        Event = "archivesChanged"
	EventBackupProfileCreated   Event = "backupProfileCreated"
	EventBackupProfileUpdated   Event = "backupProfileUpdated"
	EventBackupProfileDeleted   Event = "backupProfileDeleted"
	EventRepositoryCreated      Event = "repositoryCreated"
	EventRepositoryUpdated      Event = "repositoryUpdated"
	EventRepositoryDeleted      Event = "repositoryDeleted"
	EventAuthStateChanged       Event = "authStateChanged"
	EventCheckoutStateChanged   Event = "checkoutStateChanged"
	EventSubscriptionAdded      Event = "subscriptionAdded"
	EventSubscriptionCancelled  Event = "subscriptionCancelled"
	EventSettingsChanged        Event = "settingsChanged"
	EventOperationErrorOccurred Event = "operationErrorOccurred"
	EventNotificationDismissed  Event = "notificationDismissed"
	EventNotificationCreated    Event = "notificationCreated"
	EventWindowCloseRequested   Event = "windowCloseRequested"
)

var AllEvents = []Event{
	EventStartupStateChanged,
	EventNotificationAvailable,
	EventBackupStateChanged,
	EventPruneStateChanged,
	EventRepoStateChanged,
	EventArchivesChanged,
	EventBackupProfileCreated,
	EventBackupProfileUpdated,
	EventBackupProfileDeleted,
	EventRepositoryCreated,
	EventRepositoryUpdated,
	EventRepositoryDeleted,
	EventAuthStateChanged,
	EventCheckoutStateChanged,
	EventSubscriptionAdded,
	EventSubscriptionCancelled,
	EventSettingsChanged,
	EventOperationErrorOccurred,
	EventNotificationDismissed,
	EventNotificationCreated,
	EventWindowCloseRequested,
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

func EventBackupProfileCreatedString() string {
	return fmt.Sprintf("%s", EventBackupProfileCreated.String())
}

func EventBackupProfileUpdatedString() string {
	return fmt.Sprintf("%s", EventBackupProfileUpdated.String())
}

func EventRepositoryCreatedString() string {
	return fmt.Sprintf("%s", EventRepositoryCreated.String())
}

func EventRepositoryUpdatedString() string {
	return fmt.Sprintf("%s", EventRepositoryUpdated.String())
}

func EventRepositoryDeletedString() string {
	return fmt.Sprintf("%s", EventRepositoryDeleted.String())
}

func EventSettingsChangedString() string {
	return fmt.Sprintf("%s", EventSettingsChanged.String())
}

func EventNotificationDismissedString() string {
	return EventNotificationDismissed.String()
}

func EventNotificationCreatedString() string {
	return EventNotificationCreated.String()
}

func EventWindowCloseRequestedString() string {
	return EventWindowCloseRequested.String()
}

type RuntimeEventEmitter struct{}

func (r *RuntimeEventEmitter) EmitEvent(_ context.Context, event string, data ...string) {
	args := make([]any, len(data))
	for i, d := range data {
		args[i] = d
	}
	application.Get().Event.Emit(event, args...)
}
