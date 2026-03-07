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
	export class SavePath {
	    Excel: string;
	    Json: string;
	
	    static createFrom(source: any = {}) {
	        return new SavePath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Excel = source["Excel"];
	        this.Json = source["Json"];
	    }
	}

}

export namespace config {
	
	export class ReConf {
	    deepseek: conf.Deepseek;
	    savePath: conf.SavePath;
	
	    static createFrom(source: any = {}) {
	        return new ReConf(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deepseek = this.convertValues(source["deepseek"], conf.Deepseek);
	        this.savePath = this.convertValues(source["savePath"], conf.SavePath);
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

export namespace models {
	
	export class ProductSpec {
	    name: string;
	    weight: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.weight = source["weight"];
	    }
	}
	export class AddressInfo {
	    id: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: gorm
	    deletedAt: any;
	    VID: number;
	    "收件人": string;
	    "手机号": string;
	    "收货地址": string;
	    "商品规格": ProductSpec[];
	    "备注": string;
	
	    static createFrom(source: any = {}) {
	        return new AddressInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.deletedAt = this.convertValues(source["deletedAt"], null);
	        this.VID = source["VID"];
	        this["收件人"] = source["收件人"];
	        this["手机号"] = source["手机号"];
	        this["收货地址"] = source["收货地址"];
	        this["商品规格"] = this.convertValues(source["商品规格"], ProductSpec);
	        this["备注"] = source["备注"];
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

export namespace order {
	
	export class OrderQueryRequest {
	    limit: number;
	    page: number;
	    key: string;
	    order: string;
	
	    static createFrom(source: any = {}) {
	        return new OrderQueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.page = source["page"];
	        this.key = source["key"];
	        this.order = source["order"];
	    }
	}

}

export namespace ordertutil {
	
	export class ExceladdressInfo {
	    VID: number;
	    "收件人": string;
	    "手机号": string;
	    "收货地址": string;
	    "商品规格": string;
	    "备注": string;
	
	    static createFrom(source: any = {}) {
	        return new ExceladdressInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.VID = source["VID"];
	        this["收件人"] = source["收件人"];
	        this["手机号"] = source["手机号"];
	        this["收货地址"] = source["收货地址"];
	        this["商品规格"] = source["商品规格"];
	        this["备注"] = source["备注"];
	    }
	}

}

