export namespace models {
	
	export class AIProviderConfig {
	    provider: string;
	    api_key: string;
	    base_url: string;
	    model: string;
	    max_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new AIProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.max_tokens = source["max_tokens"];
	    }
	}
	export class Note {
	    id: number;
	    article_id: number;
	    file_path: string;
	    title: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.article_id = source["article_id"];
	        this.file_path = source["file_path"];
	        this.title = source["title"];
	        this.created_at = source["created_at"];
	    }
	}
	export class FilterRule {
	    id: number;
	    type: string;
	    value: string;
	    action: string;
	    enabled: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.action = source["action"];
	        this.enabled = source["enabled"];
	        this.created_at = source["created_at"];
	    }
	}
	export class Article {
	    id: number;
	    feed_id: number;
	    title: string;
	    link: string;
	    content: string;
	    summary: string;
	    author: string;
	    // Go type: time
	    published: any;
	    is_filtered: boolean;
	    is_saved: boolean;
	    status: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Article(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.feed_id = source["feed_id"];
	        this.title = source["title"];
	        this.link = source["link"];
	        this.content = source["content"];
	        this.summary = source["summary"];
	        this.author = source["author"];
	        this.published = this.convertValues(source["published"], null);
	        this.is_filtered = source["is_filtered"];
	        this.is_saved = source["is_saved"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class Feed {
	    id: number;
	    title: string;
	    url: string;
	    description: string;
	    icon_url: string;
	    // Go type: time
	    last_fetched: any;
	    is_dead: boolean;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Feed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.icon_url = source["icon_url"];
	        this.last_fetched = this.convertValues(source["last_fetched"], null);
	        this.is_dead = source["is_dead"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class AppState {
	    feeds: Feed[];
	    articles: Article[];
	    filter_rules: FilterRule[];
	    notes: Note[];
	    ai_config: AIProviderConfig;
	    filter_mode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feeds = this.convertValues(source["feeds"], Feed);
	        this.articles = this.convertValues(source["articles"], Article);
	        this.filter_rules = this.convertValues(source["filter_rules"], FilterRule);
	        this.notes = this.convertValues(source["notes"], Note);
	        this.ai_config = this.convertValues(source["ai_config"], AIProviderConfig);
	        this.filter_mode = source["filter_mode"];
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

