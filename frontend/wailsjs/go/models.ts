export namespace conf {
	
	export class Deepseek {
	    Apikey: string;
	
	    static createFrom(source: any = {}) {
	        return new Deepseek(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Apikey = source["Apikey"];
	    }
	}
	export class Email {
	    domain: string;
	    port: number;
	    sendEmail: string;
	    AuthCode: string;
	    sendNickname: string;
	    SSL: boolean;
	    TLS: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Email(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.port = source["port"];
	        this.sendEmail = source["sendEmail"];
	        this.AuthCode = source["AuthCode"];
	        this.sendNickname = source["sendNickname"];
	        this.SSL = source["SSL"];
	        this.TLS = source["TLS"];
	    }
	}
	export class DB {
	    Host: string;
	    Port: number;
	    User: string;
	    Password: string;
	    DBName: string;
	    Debug: boolean;
	    MaxIdleConns: number;
	    MaxOpenConns: number;
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Host = source["Host"];
	        this.Port = source["Port"];
	        this.User = source["User"];
	        this.Password = source["Password"];
	        this.DBName = source["DBName"];
	        this.Debug = source["Debug"];
	        this.MaxIdleConns = source["MaxIdleConns"];
	        this.MaxOpenConns = source["MaxOpenConns"];
	    }
	}
	export class Log {
	    Level: string;
	    App: string;
	    Dir: string;
	
	    static createFrom(source: any = {}) {
	        return new Log(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Level = source["Level"];
	        this.App = source["App"];
	        this.Dir = source["Dir"];
	    }
	}
	export class Config {
	    Log: Log;
	    DB: DB[];
	    Email: Email;
	    Deepseek: Deepseek;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Log = this.convertValues(source["Log"], Log);
	        this.DB = this.convertValues(source["DB"], DB);
	        this.Email = this.convertValues(source["Email"], Email);
	        this.Deepseek = this.convertValues(source["Deepseek"], Deepseek);
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

