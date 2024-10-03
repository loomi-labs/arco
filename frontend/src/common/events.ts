import { types } from "../../wailsjs/go/models";


export function backupStateChangedEvent(bId: types.BackupId): string {
    return `${types.Event.backupStateChanged}:${bId.backupProfileId}-${bId.repositoryId}`;
}

export function repoStateChangedEvent(bId: types.BackupId): string {
    return `${types.Event.repoStateChanged}:${bId.repositoryId}`;
}