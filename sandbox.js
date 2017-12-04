class SandboxRequest {}
class Sandbox {
    constructor(projectId, vid, envId, envName, variables, allProjectJsFiles, dataFiles, record) {
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
            record.headers.filter(h => h.isActive).forEach(h => {
                this.request.headers[h.key] = h.value;
            });
        }
    }
    getProjectFile(file, type) {
        return `./global_data/${this.projectId}/${type}/${file}`;
    }
    require(lib) {
        if (!this._allProjectJsFiles[lib]) {
            throw new Error(`no valid js lib named [${lib}], you should upload this lib first.`);
        }
        let libPath = this._allProjectJsFiles[lib];
        if (!libPath) {
            throw new Error(`[${libPath}] does not exist.`);
        }
        return require(libPath);
    }
    readFile(file) {
        return ""; //this.readFileByReader(file, f => fs.readFileSync(f, 'utf8'));
    }
    readFileByReader(file, reader) {
        if (this._projectDataFiles[file]) {
            return reader(this._projectDataFiles[file]);
        }
        throw new Error(`${file} not exists.`);
    }
    saveFile(file, content, replaceIfExist = true) {
        //ProjectDataService.instance.saveDataFile(this.projectId, file, content, replaceIfExist);
    }
    removeFile(file) {
        //ProjectDataService.instance.removeFile(ProjectDataService.dataFolderName, this.projectId, file);
    }
    setEnvVariable(key, value) {
        this.variables[key] = value;
    }
    getEnvVariable(key) {
        return this.variables[key];
    }
    removeEnvVariable(key) {
        Reflect.deleteProperty(this.variables, key);
    }
    setRequest(r) {
        this.request = r;
    }
    get environment() {
        return this.envName;
    }
    export (obj) {
        this.exportObj.content = obj;
    };
}
Sandbox.defaultExport = 'export:impossiblethis:tropxe';
//# sourceMappingURL=sandbox_stress.js.map