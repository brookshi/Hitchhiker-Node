var SandboxRequest = (function () {
    function SandboxRequest() {}
    return SandboxRequest;
}());
var Sandbox = (function () {
    function Sandbox(projectId, vid, envId, envName, variables, allProjectJsFiles, dataFiles, record) {
        var _this = this;
        this.projectId = projectId;
        this.vid = vid;
        this.envId = envId;
        this.envName = envName;
        this._allProjectJsFiles = {};
        this._projectDataFiles = {};
        this.tests = {};
        this.exportObj = {
            content: Sandbox.defaultExport
        };
        this.variables = variables;
        this._allProjectJsFiles = allProjectJsFiles;
        this._projectDataFiles = dataFiles;
        if (record) {
            this.request = {
                url: record.url,
                method: record.method || 'GET',
                body: record.body,
                headers: {}
            };
            record.headers.filter(function (h) {
                return h.isActive;
            }).forEach(function (h) {
                _this.request.headers[h.key] = h.value;
            });
        }
    }
    Sandbox.prototype.getProjectFile = function (file, type) {
        return "./global_data/" + this.projectId + "/" + type + "/" + file;
    };
    Sandbox.prototype.require = function (lib) {
        if (!this._allProjectJsFiles[lib]) {
            throw new Error("no valid js lib named [" + lib + "], you should upload this lib first.");
        }
        var libPath = this._allProjectJsFiles[lib];
        if (!libPath) {
            throw new Error("[" + libPath + "] does not exist.");
        }
        return require(libPath);
    };
    Sandbox.prototype.readFile = function (file) {
        return ""; //this.readFileByReader(file, f => fs.readFileSync(f, 'utf8'));
    };
    Sandbox.prototype.readFileByReader = function (file, reader) {
        if (this._projectDataFiles[file]) {
            return reader(this._projectDataFiles[file]);
        }
        throw new Error(file + " not exists.");
    };
    Sandbox.prototype.saveFile = function (file, content, replaceIfExist) {
        if (replaceIfExist === void 0) {
            replaceIfExist = true;
        }
        //ProjectDataService.instance.saveDataFile(this.projectId, file, content, replaceIfExist);
    };
    Sandbox.prototype.removeFile = function (file) {
        //ProjectDataService.instance.removeFile(ProjectDataService.dataFolderName, this.projectId, file);
    };
    Sandbox.prototype.setEnvVariable = function (key, value) {
        this.variables[key] = value;
    };
    Sandbox.prototype.getEnvVariable = function (key) {
        return this.variables[key];
    };
    Sandbox.prototype.removeEnvVariable = function (key) {
        Reflect.deleteProperty(this.variables, key);
    };
    Sandbox.prototype.setRequest = function (r) {
        this.request = r;
    };
    Object.defineProperty(Sandbox.prototype, "environment", {
        get: function () {
            return this.envName;
        },
        enumerable: true,
        configurable: true
    });
    Sandbox.prototype.export = function (obj) {
        this.exportObj.content = obj;
    };;
    return Sandbox;
}());
Sandbox.defaultExport = 'export:impossiblethis:tropxe';
//# sourceMappingURL=sandbox_stress.js.map