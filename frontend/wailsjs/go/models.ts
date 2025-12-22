export namespace app {
	
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
	export class Project {
	    Id: number[];
	    Creator: number[];
	    Label: string;
	    Description: string;
	    Visibility: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Creator = source["Creator"];
	        this.Label = source["Label"];
	        this.Description = source["Description"];
	        this.Visibility = source["Visibility"];
	    }
	}
	export class ProjectCreate {
	    Label: string;
	    Description: string;
	    Visibility: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Description = source["Description"];
	        this.Visibility = source["Visibility"];
	    }
	}
	export class SampleGroupRelation {
	    Parent: number[];
	    Child: number[];
	
	    static createFrom(source: any = {}) {
	        return new SampleGroupRelation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Parent = source["Parent"];
	        this.Child = source["Child"];
	    }
	}
	export class ProjectSampleGroup {
	    Id: number[];
	    Creator: number[];
	    Label: string;
	    Description: string;
	    Properties: Property[];
	    Samples: number[][];
	
	    static createFrom(source: any = {}) {
	        return new ProjectSampleGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Creator = source["Creator"];
	        this.Label = source["Label"];
	        this.Description = source["Description"];
	        this.Properties = this.convertValues(source["Properties"], Property);
	        this.Samples = source["Samples"];
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
	export class Property {
	    Key: string;
	    Type: string;
	    Value: any;
	
	    static createFrom(source: any = {}) {
	        return new Property(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Type = source["Type"];
	        this.Value = source["Value"];
	    }
	}
	export class ProjectSample {
	    Id: number[];
	    Creator: number[];
	    MembershipCreator: number[];
	    // Go type: time
	    MembershipCreated: any;
	    Label: string;
	    Tags: string[];
	    Properties: Property[];
	    NoteCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectSample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Creator = source["Creator"];
	        this.MembershipCreator = source["MembershipCreator"];
	        this.MembershipCreated = this.convertValues(source["MembershipCreated"], null);
	        this.Label = source["Label"];
	        this.Tags = source["Tags"];
	        this.Properties = this.convertValues(source["Properties"], Property);
	        this.NoteCount = source["NoteCount"];
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
	export class ProjectResources {
	    Project: Project;
	    ProjectTags: string[];
	    Samples: ProjectSample[];
	    SampleGroups: ProjectSampleGroup[];
	    SampleGroupRelations: SampleGroupRelation[];
	    ProjectNoteCount: number;
	    ProjectUserPermission: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectResources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Project = this.convertValues(source["Project"], Project);
	        this.ProjectTags = source["ProjectTags"];
	        this.Samples = this.convertValues(source["Samples"], ProjectSample);
	        this.SampleGroups = this.convertValues(source["SampleGroups"], ProjectSampleGroup);
	        this.SampleGroupRelations = this.convertValues(source["SampleGroupRelations"], SampleGroupRelation);
	        this.ProjectNoteCount = source["ProjectNoteCount"];
	        this.ProjectUserPermission = source["ProjectUserPermission"];
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
	
	export class ProjectSampleCreate {
	    Label: string;
	    Tags: string[];
	    Properties: Property[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectSampleCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Tags = source["Tags"];
	        this.Properties = this.convertValues(source["Properties"], Property);
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
	
	export class ProjectWithUserPermission {
	    Id: number[];
	    Creator: number[];
	    Label: string;
	    Description: string;
	    Visibility: string;
	    UserPermission: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectWithUserPermission(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Creator = source["Creator"];
	        this.Label = source["Label"];
	        this.Description = source["Description"];
	        this.Visibility = source["Visibility"];
	        this.UserPermission = source["UserPermission"];
	    }
	}
	
	
	export class User {
	    Id: number[];
	    AccountStatus: string;
	    Email: string;
	    Name: string;
	    Role: string;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.AccountStatus = source["AccountStatus"];
	        this.Email = source["Email"];
	        this.Name = source["Name"];
	        this.Role = source["Role"];
	    }
	}
	export class UserCreate {
	    Email: string;
	    Name: string;
	    Password: string;
	    Role: string;
	
	    static createFrom(source: any = {}) {
	        return new UserCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.Name = source["Name"];
	        this.Password = source["Password"];
	        this.Role = source["Role"];
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

