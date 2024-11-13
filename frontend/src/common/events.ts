import { types } from "../../wailsjs/go/models";


export function backupStateChangedEvent(bId: types.BackupId): string {
    return `${types.Event.backupStateChanged}:${bId.backupProfileId}-${bId.repositoryId}`;
}

export function repoStateChangedEvent(repositoryId: number): string {
    return `${types.Event.repoStateChanged}:${repositoryId}`;
}

export function archivesChanged(repositoryId: number): string {
    return `${types.Event.archivesChanged}:${repositoryId}`;
}

export function backupProfileDeletedEvent(): string {
    return `${types.Event.backupProfileDeleted}`;
}