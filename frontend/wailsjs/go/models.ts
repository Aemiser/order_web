export namespace checkout {
	
	export class Spec {
	    name: string;
	    pricePerKg: number;
	    pricePerItem: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Spec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pricePerKg = source["pricePerKg"];
	        this.pricePerItem = source["pricePerItem"];
	        this.total = source["total"];
	    }
	}
	export class ProSPecs {
	    SID: number;
	    Name: string;
	    Specs: Spec[];
	
	    static createFrom(source: any = {}) {
	        return new ProSPecs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SID = source["SID"];
	        this.Name = source["Name"];
	        this.Specs = this.convertValues(source["Specs"], Spec);
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
	export class CheckoutResponse {
	    FieldList: string[];
	    PBMap: Record<string, Array<string>>;
	    Count: number;
	    ProSPecs: ProSPecs[];
	    TableData: string[];
	    TotalRow: number[];
	    PricePerKgRow: number[];
	    PricePerItemRow: number[];
	    TotalAmountRow: number[];
	
	    static createFrom(source: any = {}) {
	        return new CheckoutResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FieldList = source["FieldList"];
	        this.PBMap = source["PBMap"];
	        this.Count = source["Count"];
	        this.ProSPecs = this.convertValues(source["ProSPecs"], ProSPecs);
	        this.TableData = source["TableData"];
	        this.TotalRow = source["TotalRow"];
	        this.PricePerKgRow = source["PricePerKgRow"];
	        this.PricePerItemRow = source["PricePerItemRow"];
	        this.TotalAmountRow = source["TotalAmountRow"];
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
	export class ProPriceRequest {
	    productName: string;
	    spec: string;
	    total: number;
	    pricePerKg: number;
	    pricePerItem: number;
	    totalAmount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProPriceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productName = source["productName"];
	        this.spec = source["spec"];
	        this.total = source["total"];
	        this.pricePerKg = source["pricePerKg"];
	        this.pricePerItem = source["pricePerItem"];
	        this.totalAmount = source["totalAmount"];
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
	    createdAt: time.Time;
	    updatedAt: time.Time;
	    // Go type: gorm
	    deletedAt: any;
	    VID: number;
	    "收件人": string;
	    "手机号": string;
	    "收货地址": string;
	    "商品规格": ProductSpec[];
	    "备注": string;
	    Isstorage: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AddressInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.updatedAt = this.convertValues(source["updatedAt"], time.Time);
	        this.deletedAt = this.convertValues(source["deletedAt"], null);
	        this.VID = source["VID"];
	        this["收件人"] = source["收件人"];
	        this["手机号"] = source["手机号"];
	        this["收货地址"] = source["收货地址"];
	        this["商品规格"] = this.convertValues(source["商品规格"], ProductSpec);
	        this["备注"] = source["备注"];
	        this.Isstorage = source["Isstorage"];
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
	    Addresssinfo: models.AddressInfo;
	    startTime: time.Time;
	    endTime: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OrderQueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.page = source["page"];
	        this.key = source["key"];
	        this.order = source["order"];
	        this.Addresssinfo = this.convertValues(source["Addresssinfo"], models.AddressInfo);
	        this.startTime = this.convertValues(source["startTime"], time.Time);
	        this.endTime = this.convertValues(source["endTime"], time.Time);
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
	export class OrderQueryResponse {
	    list: ordertutil.ExceladdressInfo[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new OrderQueryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.list = this.convertValues(source["list"], ordertutil.ExceladdressInfo);
	        this.count = source["count"];
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

export namespace ordertutil {
	
	export class ExceladdressInfo {
	    createdAt: time.Time;
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
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.VID = source["VID"];
	        this["收件人"] = source["收件人"];
	        this["手机号"] = source["手机号"];
	        this["收货地址"] = source["收货地址"];
	        this["商品规格"] = source["商品规格"];
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

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

