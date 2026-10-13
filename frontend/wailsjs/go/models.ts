export namespace config {
	
	export class ScrcpyOptions {
	    size: string;
	    dpi: string;
	    flex_display: boolean;
	    no_vd_decorations: boolean;
	    keep_content_on_close: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScrcpyOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.size = source["size"];
	        this.dpi = source["dpi"];
	        this.flex_display = source["flex_display"];
	        this.no_vd_decorations = source["no_vd_decorations"];
	        this.keep_content_on_close = source["keep_content_on_close"];
	    }
	}
	export class Config {
	    folder_name: string;
	    scope: string;
	    include: string[];
	    exclude: string[];
	    prefer_transport: string;
	    audio: boolean;
	    device: string;
	    settings_entry: boolean;
	    scrcpy: ScrcpyOptions;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder_name = source["folder_name"];
	        this.scope = source["scope"];
	        this.include = source["include"];
	        this.exclude = source["exclude"];
	        this.prefer_transport = source["prefer_transport"];
	        this.audio = source["audio"];
	        this.device = source["device"];
	        this.settings_entry = source["settings_entry"];
	        this.scrcpy = this.convertValues(source["scrcpy"], ScrcpyOptions);
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

export namespace main {
	
	export class AppView {
	    package: string;
	    label: string;
	    system: boolean;
	    selected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.package = source["package"];
	        this.label = source["label"];
	        this.system = source["system"];
	        this.selected = source["selected"];
	    }
	}
	export class Result {
	    serial: string;
	    total: number;
	    added: string[];
	    removed: string[];
	    updated: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial = source["serial"];
	        this.total = source["total"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.updated = source["updated"];
	    }
	}
	export class Status {
	    connected: boolean;
	    devices: string[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.devices = source["devices"];
	    }
	}

}

