import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";


export function backupStateChangedEvent(bId: types.BackupId): string {
  return `${types.Event.EventBackupStateChanged}:${bId.backupProfileId}-${bId.repositoryId}`;
}

export function repoStateChangedEvent(repositoryId: number): string {
  return `${types.Event.EventRepoStateChanged}:${repositoryId}`;
}

export function archivesChanged(repositoryId: number): string {
  return `${types.Event.EventArchivesChanged}:${repositoryId}`;
}

export function backupProfileDeletedEvent(): string {
  return `${types.Event.EventBackupProfileDeleted}`;
}

export function checkoutStateChangedEvent(): string {
  return `${types.Event.EventCheckoutStateChanged}`;
}

export function subscriptionAddedEvent(): string {
  return `${types.Event.EventSubscriptionAdded}`;
}

export function subscriptionCancelledEvent(): string {
  return `${types.Event.EventSubscriptionCancelled}`;
}

export function backupProfileCreatedEvent(): string {
  return `${types.Event.EventBackupProfileCreated}`;
}

export function backupProfileUpdatedEvent(): string {
  return `${types.Event.EventBackupProfileUpdated}`;
}

export function repositoryCreatedEvent(): string {
  return `${types.Event.EventRepositoryCreated}`;
}

export function repositoryUpdatedEvent(): string {
  return `${types.Event.EventRepositoryUpdated}`;
}

export function repositoryDeletedEvent(): string {
  return `${types.Event.EventRepositoryDeleted}`;
}

export function notificationDismissedEvent(): string {
  return `${types.Event.EventNotificationDismissed}`;
}

export function notificationCreatedEvent(): string {
  return `${types.Event.EventNotificationCreated}`;
}

export function windowCloseRequestedEvent(): string {
  return `${types.Event.EventWindowCloseRequested}`;
}
