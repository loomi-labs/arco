// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Create as $Create } from "@wailsio/runtime";

export class BackupId {
    "backupProfileId": number;
    "repositoryId": number;

    /** Creates a new BackupId instance. */
    constructor($$source: Partial<BackupId> = {}) {
        if (!("backupProfileId" in $$source)) {
            this["backupProfileId"] = 0;
        }
        if (!("repositoryId" in $$source)) {
            this["repositoryId"] = 0;
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new BackupId instance from a string or object.
     */
    static createFrom($$source: any = {}): BackupId {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new BackupId($$parsedSource as Partial<BackupId>);
    }
}

export enum Event {
    /**
     * The Go zero value for the underlying type of the enum.
     */
    $zero = "",

    EventStartupStateChanged = "startupStateChanged",
    EventNotificationAvailable = "notificationAvailable",
    EventBackupStateChanged = "backupStateChanged",
    EventPruneStateChanged = "pruneStateChanged",
    EventRepoStateChanged = "repoStateChanged",
    EventArchivesChanged = "archivesChanged",
    EventBackupProfileDeleted = "backupProfileDeleted",
    EventAuthStateChanged = "authStateChanged",
    EventCheckoutStateChanged = "checkoutStateChanged",
    EventSubscriptionAdded = "subscriptionAdded",
    EventSubscriptionCancelled = "subscriptionCancelled",
};

/**
 * FrontendError is the error type that is received from the frontend
 */
export class FrontendError {
    "message": string;
    "stack": string;

    /** Creates a new FrontendError instance. */
    constructor($$source: Partial<FrontendError> = {}) {
        if (!("message" in $$source)) {
            this["message"] = "";
        }
        if (!("stack" in $$source)) {
            this["stack"] = "";
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new FrontendError instance from a string or object.
     */
    static createFrom($$source: any = {}): FrontendError {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new FrontendError($$parsedSource as Partial<FrontendError>);
    }
}

export class MountState {
    "isMounted": boolean;
    "mountPath": string;

    /** Creates a new MountState instance. */
    constructor($$source: Partial<MountState> = {}) {
        if (!("isMounted" in $$source)) {
            this["isMounted"] = false;
        }
        if (!("mountPath" in $$source)) {
            this["mountPath"] = "";
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new MountState instance from a string or object.
     */
    static createFrom($$source: any = {}): MountState {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new MountState($$parsedSource as Partial<MountState>);
    }
}

export class Notification {
    "message": string;
    "level": NotificationLevel;

    /** Creates a new Notification instance. */
    constructor($$source: Partial<Notification> = {}) {
        if (!("message" in $$source)) {
            this["message"] = "";
        }
        if (!("level" in $$source)) {
            this["level"] = NotificationLevel.$zero;
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new Notification instance from a string or object.
     */
    static createFrom($$source: any = {}): Notification {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new Notification($$parsedSource as Partial<Notification>);
    }
}

export enum NotificationLevel {
    /**
     * The Go zero value for the underlying type of the enum.
     */
    $zero = "",

    LevelInfo = "info",
    LevelWarning = "warning",
    LevelError = "error",
};
