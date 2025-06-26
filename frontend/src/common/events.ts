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

