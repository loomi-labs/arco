//go:build !integration

package types

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

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

type RuntimeEventEmitter struct{}

func (r *RuntimeEventEmitter) EmitEvent(_ context.Context, event string) {
	application.Get().Event.Emit(event)
}
