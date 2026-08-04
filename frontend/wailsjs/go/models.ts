export namespace agent {
	
	export class Segment {
	    start: number;
	    end: number;
	    text: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new Segment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.text = source["text"];
	        this.confidence = source["confidence"];
	    }
	}
	export class MeetingState {
	    audio_file_path: string;
	    language: string;
	    wav_file_path: string;
	    raw_transcript: string;
	    segments: Segment[];
	    audio_duration: number;
	    meeting_notes: string;
	    summary_content?: string;
	    summary_error?: string;
	    output_path: string;
	    transcript_path?: string;
	    html_output_path?: string;
	    markdown_path?: string;
	    summary_enabled: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MeetingState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.audio_file_path = source["audio_file_path"];
	        this.language = source["language"];
	        this.wav_file_path = source["wav_file_path"];
	        this.raw_transcript = source["raw_transcript"];
	        this.segments = this.convertValues(source["segments"], Segment);
	        this.audio_duration = source["audio_duration"];
	        this.meeting_notes = source["meeting_notes"];
	        this.summary_content = source["summary_content"];
	        this.summary_error = source["summary_error"];
	        this.output_path = source["output_path"];
	        this.transcript_path = source["transcript_path"];
	        this.html_output_path = source["html_output_path"];
	        this.markdown_path = source["markdown_path"];
	        this.summary_enabled = source["summary_enabled"];
	        this.error = source["error"];
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

export namespace hardware {
	
	export class Profile {
	    cpuThreads: number;
	    architecture: string;
	    gpus: string[];
	    hasNvidia: boolean;
	    hasAmd: boolean;
	    hasIntel: boolean;
	    vulkanRuntime: boolean;
	    recommendedEngine: string;
	    recommendedModel: string;
	    recommendation: string;
	    driverAdvice: string;
	    driverUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuThreads = source["cpuThreads"];
	        this.architecture = source["architecture"];
	        this.gpus = source["gpus"];
	        this.hasNvidia = source["hasNvidia"];
	        this.hasAmd = source["hasAmd"];
	        this.hasIntel = source["hasIntel"];
	        this.vulkanRuntime = source["vulkanRuntime"];
	        this.recommendedEngine = source["recommendedEngine"];
	        this.recommendedModel = source["recommendedModel"];
	        this.recommendation = source["recommendation"];
	        this.driverAdvice = source["driverAdvice"];
	        this.driverUrl = source["driverUrl"];
	    }
	}

}

export namespace main {
	
	export class AppInfo {
	    engine: string;
	    whisperModel: string;
	    llmModel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.whisperModel = source["whisperModel"];
	        this.llmModel = source["llmModel"];
	    }
	}
	export class DeviceInfo {
	    id: string;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class MeetingHistoryItem {
	    id: string;
	    title: string;
	    sourceAudio: string;
	    generatedAt: string;
	    summary: string;
	    htmlPath: string;
	    markdownPath: string;
	    transcriptPath: string;
	
	    static createFrom(source: any = {}) {
	        return new MeetingHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.sourceAudio = source["sourceAudio"];
	        this.generatedAt = source["generatedAt"];
	        this.summary = source["summary"];
	        this.htmlPath = source["htmlPath"];
	        this.markdownPath = source["markdownPath"];
	        this.transcriptPath = source["transcriptPath"];
	    }
	}
	export class ModelCatalogItem {
	    id: string;
	    kind: string;
	    name: string;
	    path: string;
	    size: number;
	    description: string;
	    installed: boolean;
	    active: boolean;
	    recommended: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelCatalogItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.description = source["description"];
	        this.installed = source["installed"];
	        this.active = source["active"];
	        this.recommended = source["recommended"];
	    }
	}
	export class ModelOption {
	    name: string;
	    path: string;
	    size: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.active = source["active"];
	    }
	}
	export class Preferences {
	    whisperModel: string;
	    ollamaModel: string;
	    language: string;
	    outputDir: string;
	    temperature: number;
	    maxTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whisperModel = source["whisperModel"];
	        this.ollamaModel = source["ollamaModel"];
	        this.language = source["language"];
	        this.outputDir = source["outputDir"];
	        this.temperature = source["temperature"];
	        this.maxTokens = source["maxTokens"];
	    }
	}

}

