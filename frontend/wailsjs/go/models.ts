export namespace main {
	
	export class AppConfig {
	    DbUrl: string;
	    DbUsername: string;
	    DbPassword: string;
	    DbName: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DbUrl = source["DbUrl"];
	        this.DbUsername = source["DbUsername"];
	        this.DbPassword = source["DbPassword"];
	        this.DbName = source["DbName"];
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
	export class Sample {
	    Id: number[];
	    Owner: number[];
	    Label: string;
	
	    static createFrom(source: any = {}) {
	        return new Sample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Owner = source["Owner"];
	        this.Label = source["Label"];
	    }
	}
	export class SampleCreate {
	    Label: string;
	
	    static createFrom(source: any = {}) {
	        return new SampleCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	    }
	}
	export class SampleGroupCreate {
	    Label: string;
	    Description: string;
	    Parents: number[][];
	    Samples: SampleCreate[];
	
	    static createFrom(source: any = {}) {
	        return new SampleGroupCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Description = source["Description"];
	        this.Parents = source["Parents"];
	        this.Samples = this.convertValues(source["Samples"], SampleCreate);
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
	export class User {
	    Id: number[];
	    Email: string;
	    Name: string;
	    PermissionRoles: string[];
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Email = source["Email"];
	        this.Name = source["Name"];
	        this.PermissionRoles = source["PermissionRoles"];
	    }
	}
	export class UserCreate {
	    Email: string;
	    Name: string;
	    Password: string;
	    PermissionRoles: string[];
	
	    static createFrom(source: any = {}) {
	        return new UserCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.Name = source["Name"];
	        this.Password = source["Password"];
	        this.PermissionRoles = source["PermissionRoles"];
	    }
	}
	export class UserCredentials {
	    Email: string;
	    Password: string;
	
	    static createFrom(source: any = {}) {
	        return new UserCredentials(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.Password = source["Password"];
	    }
	}

}

