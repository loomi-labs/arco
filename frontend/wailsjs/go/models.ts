export namespace app {
	
	export class BackupProgressResponse {
	    backupId: types.BackupId;
	    progress: borg.BackupProgress;
	    found: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupProgressResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupId = this.convertValues(source["backupId"], types.BackupId);
	        this.progress = this.convertValues(source["progress"], borg.BackupProgress);
	        this.found = source["found"];
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
	
	export class BackupScheduleEdges {
	    backup_profile?: BackupProfile;
	
	    static createFrom(source: any = {}) {
	        return new BackupScheduleEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backup_profile = this.convertValues(source["backup_profile"], BackupProfile);
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
	    backup_schedule?: BackupSchedule;
	
	    static createFrom(source: any = {}) {
	        return new BackupProfileEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositories = this.convertValues(source["repositories"], Repository);
	        this.backup_schedule = this.convertValues(source["backup_schedule"], BackupSchedule);
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
	    isSetupComplete: boolean;
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
	        this.isSetupComplete = source["isSetupComplete"];
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
	    backup_profiles?: BackupProfile[];
	    archives?: Archive[];
	
	    static createFrom(source: any = {}) {
	        return new RepositoryEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backup_profiles = this.convertValues(source["backup_profiles"], BackupProfile);
	        this.archives = this.convertValues(source["archives"], Archive);
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
	    url: string;
	    password: string;
	    edges: RepositoryEdges;
	
	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.password = source["password"];
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
	
	    static createFrom(source: any = {}) {
	        return new ArchiveEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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

}

export namespace types {
	
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

