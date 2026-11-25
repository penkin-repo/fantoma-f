export namespace main {
	
	export class FolderInfo {
	    id: string;
	    path: string;
	    name: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new FolderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.status = source["status"];
	    }
	}
	export class FontStyle {
	    style: string;
	    filePath: string;
	    format: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FontStyle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.style = source["style"];
	        this.filePath = source["filePath"];
	        this.format = source["format"];
	        this.isActive = source["isActive"];
	    }
	}
	export class FontFamily {
	    name: string;
	    styles: FontStyle[];
	    stylesCount: number;
	    regularPath: string;
	    format: string;
	    isActive: boolean;
	    isFullyActive: boolean;
	    category: string;
	    weightClass: number;
	    hasItalic: boolean;
	    isManual: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FontFamily(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.styles = this.convertValues(source["styles"], FontStyle);
	        this.stylesCount = source["stylesCount"];
	        this.regularPath = source["regularPath"];
	        this.format = source["format"];
	        this.isActive = source["isActive"];
	        this.isFullyActive = source["isFullyActive"];
	        this.category = source["category"];
	        this.weightClass = source["weightClass"];
	        this.hasItalic = source["hasItalic"];
	        this.isManual = source["isManual"];
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

