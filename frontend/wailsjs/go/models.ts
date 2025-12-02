export namespace main {
	
	export class AppConfig {
	    DbUrl: string;
	    DbUsername: string;
	    DbPassword: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DbUrl = source["DbUrl"];
	        this.DbUsername = source["DbUsername"];
	        this.DbPassword = source["DbPassword"];
	    }
	}
	export class Ok {
	
	
	    static createFrom(source: any = {}) {
	        return new Ok(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

