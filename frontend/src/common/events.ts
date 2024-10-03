import { types } from "../../wailsjs/go/models";


export function backupStateChangedEvent(bId: types.BackupId): string {
    return `${types.Event.backupStateChanged}:${bId.backupProfileId}-${bId.repositoryId}`;
}

export function repoStateChangedEvent(repositoryId: number): string {
    return `${types.Event.repoStateChanged}:${repositoryId}`;
}