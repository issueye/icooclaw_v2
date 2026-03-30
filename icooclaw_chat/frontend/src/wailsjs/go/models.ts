export namespace services {
	
	export class AgentProcessStatus {
	    managed: boolean;
	    running: boolean;
	    healthy: boolean;
	    pid: number;
	    startedAt?: string;
	    binaryPath?: string;
	    configPath?: string;
	    workingDir?: string;
	    workspacePath?: string;
	    apiBase?: string;
	    lastError?: string;
	    lastExit?: string;
	    outputPreview?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentProcessStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.managed = source["managed"];
	        this.running = source["running"];
	        this.healthy = source["healthy"];
	        this.pid = source["pid"];
	        this.startedAt = source["startedAt"];
	        this.binaryPath = source["binaryPath"];
	        this.configPath = source["configPath"];
	        this.workingDir = source["workingDir"];
	        this.workspacePath = source["workspacePath"];
	        this.apiBase = source["apiBase"];
	        this.lastError = source["lastError"];
	        this.lastExit = source["lastExit"];
	        this.outputPreview = source["outputPreview"];
	    }
	}

}

