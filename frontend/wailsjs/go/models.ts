export namespace borg {
	
	export class Archive {
	    archive: string;
	    barchive: string;
	    id: string;
	    name: string;
	    start: string;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new Archive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archive = source["archive"];
	        this.barchive = source["barchive"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.start = source["start"];
	        this.time = source["time"];
	    }
	}
	export class Encryption {
	    mode: string;
	
	    static createFrom(source: any = {}) {
	        return new Encryption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
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
	export class Repository {
	    id: string;
	    last_modified: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.last_modified = source["last_modified"];
	        this.location = source["location"];
	    }
	}
	export class ListResponse {
	    archives: Archive[];
	    encryption: Encryption;
	    repository: Repository;
	
	    static createFrom(source: any = {}) {
	        return new ListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archives = this.convertValues(source["archives"], Archive);
	        this.encryption = this.convertValues(source["encryption"], Encryption);
	        this.repository = this.convertValues(source["repository"], Repository);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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

export namespace ent {
	
	export class RepositoryEdges {
	    backupprofiles?: BackupProfile[];
	
	    static createFrom(source: any = {}) {
	        return new RepositoryEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupprofiles = this.convertValues(source["backupprofiles"], BackupProfile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
		    if (a.slice) {
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
	
	    static createFrom(source: any = {}) {
	        return new BackupProfileEdges(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositories = this.convertValues(source["repositories"], Repository);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
	    directories: string[];
	    hasPeriodicBackups: boolean;
	    // Go type: time
	    periodicBackupTime: any;
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
	        this.directories = source["directories"];
	        this.hasPeriodicBackups = source["hasPeriodicBackups"];
	        this.periodicBackupTime = this.convertValues(source["periodicBackupTime"], null);
	        this.isSetupComplete = source["isSetupComplete"];
	        this.edges = this.convertValues(source["edges"], BackupProfileEdges);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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

