export namespace app {
	
	export class BackupProfileFilter {
	    id?: number;
	    name: string;
	    isAllFilter: boolean;
	    isUnknownFilter: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupProfileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.isAllFilter = source["isAllFilter"];
	        this.isUnknownFilter = source["isUnknownFilter"];
	    }
	}
	export class Env {
	    debug: boolean;
	    startPage: string;
	
	    static createFrom(source: any = {}) {
	        return new Env(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.debug = source["debug"];
	        this.startPage = source["startPage"];
	    }
	}
	export class PaginatedArchivesRequest {
	    repositoryId: number;
	    page: number;
	    pageSize: number;
	    backupProfileFilter?: BackupProfileFilter;
	    search?: string;
	    // Go type: time
	    startDate?: any;
	    // Go type: time
	    endDate?: any;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedArchivesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositoryId = source["repositoryId"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.backupProfileFilter = this.convertValues(source["backupProfileFilter"], BackupProfileFilter);
	        this.search = source["search"];
	        this.startDate = this.convertValues(source["startDate"], null);
	        this.endDate = this.convertValues(source["endDate"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaginatedArchivesResponse {
	    archives: ent.Archive[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedArchivesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archives = this.convertValues(source["archives"], ent.Archive);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace backupprofile {
	
	export enum Icon {
	    home = "home",
	    briefcase = "briefcase",
	    book = "book",
	    envelope = "envelope",
	    camera = "camera",
	    fire = "fire",
	}

}

export namespace backupschedule {
	
	export enum Weekday {
	    monday = "monday",
	    tuesday = "tuesday",
	    wednesday = "wednesday",
	    thursday = "thursday",
	    friday = "friday",
	    saturday = "saturday",
	    sunday = "sunday",
	}

}

export namespace borg {
	
	export class BackupProgress {
	    totalFiles: number;
	    processedFiles: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalFiles = source["totalFiles"];
	        this.processedFiles = source["processedFiles"];
	    }
	}

}

export namespace ent {
	
	export class FailedBackupRunEdges {
	    backupProfile?: BackupProfile;
	    repository?: Repository;
	
	    static createFrom(source: any = {}) {
	        return new FailedBackupRunEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupProfile = this.convertValues(source["backupProfile"], BackupProfile);
	        this.repository = this.convertValues(source["repository"], Repository);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FailedBackupRun {
	    id?: number;
	    error: string;
	    // Go type: FailedBackupRunEdges
	    edges: any;
	
	    static createFrom(source: any = {}) {
	        return new FailedBackupRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.error = source["error"];
	        this.edges = this.convertValues(source["edges"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupScheduleEdges {
	    backupProfile?: BackupProfile;
	
	    static createFrom(source: any = {}) {
	        return new BackupScheduleEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupProfile = this.convertValues(source["backupProfile"], BackupProfile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupSchedule {
	    id?: number;
	    hourly: boolean;
	    // Go type: time
	    dailyAt?: any;
	    weekday?: backupschedule.Weekday;
	    // Go type: time
	    weeklyAt?: any;
	    monthday?: number;
	    // Go type: time
	    monthlyAt?: any;
	    // Go type: time
	    nextRun: any;
	    // Go type: time
	    lastRun?: any;
	    lastRunStatus?: string;
	    edges: BackupScheduleEdges;
	
	    static createFrom(source: any = {}) {
	        return new BackupSchedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hourly = source["hourly"];
	        this.dailyAt = this.convertValues(source["dailyAt"], null);
	        this.weekday = source["weekday"];
	        this.weeklyAt = this.convertValues(source["weeklyAt"], null);
	        this.monthday = source["monthday"];
	        this.monthlyAt = this.convertValues(source["monthlyAt"], null);
	        this.nextRun = this.convertValues(source["nextRun"], null);
	        this.lastRun = this.convertValues(source["lastRun"], null);
	        this.lastRunStatus = source["lastRunStatus"];
	        this.edges = this.convertValues(source["edges"], BackupScheduleEdges);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupProfileEdges {
	    repositories?: Repository[];
	    archives?: Archive[];
	    backupSchedule?: BackupSchedule;
	    failedBackupRuns?: FailedBackupRun[];
	
	    static createFrom(source: any = {}) {
	        return new BackupProfileEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositories = this.convertValues(source["repositories"], Repository);
	        this.archives = this.convertValues(source["archives"], Archive);
	        this.backupSchedule = this.convertValues(source["backupSchedule"], BackupSchedule);
	        this.failedBackupRuns = this.convertValues(source["failedBackupRuns"], FailedBackupRun);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupProfile {
	    id: number;
	    name: string;
	    prefix: string;
	    backupPaths: string[];
	    excludePaths: string[];
	    icon: backupprofile.Icon;
	    edges: BackupProfileEdges;
	
	    static createFrom(source: any = {}) {
	        return new BackupProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.prefix = source["prefix"];
	        this.backupPaths = source["backupPaths"];
	        this.excludePaths = source["excludePaths"];
	        this.icon = source["icon"];
	        this.edges = this.convertValues(source["edges"], BackupProfileEdges);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RepositoryEdges {
	    backupProfiles?: BackupProfile[];
	    archives?: Archive[];
	    failedBackupRuns?: FailedBackupRun[];
	
	    static createFrom(source: any = {}) {
	        return new RepositoryEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupProfiles = this.convertValues(source["backupProfiles"], BackupProfile);
	        this.archives = this.convertValues(source["archives"], Archive);
	        this.failedBackupRuns = this.convertValues(source["failedBackupRuns"], FailedBackupRun);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Repository {
	    id: number;
	    name: string;
	    location: string;
	    password: string;
	    stats_total_chunks: number;
	    stats_total_size: number;
	    stats_total_csize: number;
	    stats_total_unique_chunks: number;
	    stats_unique_size: number;
	    stats_unique_csize: number;
	    edges: RepositoryEdges;
	
	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.location = source["location"];
	        this.password = source["password"];
	        this.stats_total_chunks = source["stats_total_chunks"];
	        this.stats_total_size = source["stats_total_size"];
	        this.stats_total_csize = source["stats_total_csize"];
	        this.stats_total_unique_chunks = source["stats_total_unique_chunks"];
	        this.stats_unique_size = source["stats_unique_size"];
	        this.stats_unique_csize = source["stats_unique_csize"];
	        this.edges = this.convertValues(source["edges"], RepositoryEdges);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ArchiveEdges {
	    repository?: Repository;
	    backupProfile?: BackupProfile;
	
	    static createFrom(source: any = {}) {
	        return new ArchiveEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repository = this.convertValues(source["repository"], Repository);
	        this.backupProfile = this.convertValues(source["backupProfile"], BackupProfile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Archive {
	    id: number;
	    name: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    duration: any;
	    borgId: string;
	    edges: ArchiveEdges;
	
	    static createFrom(source: any = {}) {
	        return new Archive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.duration = this.convertValues(source["duration"], null);
	        this.borgId = source["borgId"];
	        this.edges = this.convertValues(source["edges"], ArchiveEdges);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	

}

export namespace state {
	
	export enum BackupStatus {
	    idle = "idle",
	    waiting = "waiting",
	    running = "running",
	    completed = "completed",
	    cancelled = "cancelled",
	    failed = "failed",
	}
	export enum RepoStatus {
	    idle = "idle",
	    backingUp = "backingUp",
	    pruning = "pruning",
	    deleting = "deleting",
	    mounted = "mounted",
	    performingOperation = "performingOperation",
	    locked = "locked",
	}
	export enum BackupButtonStatus {
	    runBackup = "runBackup",
	    waiting = "waiting",
	    abort = "abort",
	    locked = "locked",
	    unmount = "unmount",
	    busy = "busy",
	}
	export class BackupState {
	    status: BackupStatus;
	    progress?: borg.BackupProgress;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.progress = this.convertValues(source["progress"], borg.BackupProgress);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MountState {
	    is_mounted: boolean;
	    mount_path: string;
	
	    static createFrom(source: any = {}) {
	        return new MountState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_mounted = source["is_mounted"];
	        this.mount_path = source["mount_path"];
	    }
	}
	export class RepoState {
	    status: RepoStatus;
	
	    static createFrom(source: any = {}) {
	        return new RepoState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	    }
	}

}

export namespace types {
	
	export enum Event {
	    notificationAvailable = "notificationAvailable",
	    backupStateChanged = "backupStateChanged",
	    repoStateChanged = "repoStateChanged",
	}
	export class BackupId {
	    backupProfileId: number;
	    repositoryId: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupId(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupProfileId = source["backupProfileId"];
	        this.repositoryId = source["repositoryId"];
	    }
	}
	export class FrontendError {
	    message: string;
	    stack: string;
	
	    static createFrom(source: any = {}) {
	        return new FrontendError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.stack = source["stack"];
	    }
	}
	export class Notification {
	    message: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new Notification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.level = source["level"];
	    }
	}

}

