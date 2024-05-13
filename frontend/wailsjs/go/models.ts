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
	export class Directory {
	    path: string;
	    isAdded: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Directory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.isAdded = source["isAdded"];
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
	export class Repo {
	    id: string;
	    name: string;
	    prefix: string;
	    directories: Directory[];
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.prefix = source["prefix"];
	        this.directories = this.convertValues(source["directories"], Directory);
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

