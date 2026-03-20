const fs = require('fs');
const path = require('path');
const child_process = require('child_process');

const WORKSPACE_ROOT = process.env.WANGSHU_WORKSPACE;
if (!WORKSPACE_ROOT) {
    console.error('Security Error: WANGSHU_WORKSPACE not set');
    process.exit(1);
}

const WORKSPACE_ABSOLUTE = path.resolve(WORKSPACE_ROOT);

function checkWritePermission(targetPath) {
    if (!targetPath) return true;
    
    try {
        const resolvedPath = path.resolve(targetPath);
        if (!resolvedPath.startsWith(WORKSPACE_ABSOLUTE + path.sep) && resolvedPath !== WORKSPACE_ABSOLUTE) {
            throw new Error(
                `Security Violation: Write access denied outside workspace.\n` +
                `Target: ${targetPath}\n` +
                `Resolved: ${resolvedPath}\n` +
                `Allowed Root: ${WORKSPACE_ABSOLUTE}`
            );
        }
    } catch (e) {
        if (e.message.includes('Security Violation')) {
            throw e;
        }
    }
}

const originalWriteFile = fs.writeFile;
const originalWriteFileSync = fs.writeFileSync;
const originalAppendFile = fs.appendFile;
const originalAppendFileSync = fs.appendFileSync;
const originalOpen = fs.open;
const originalOpenSync = fs.openSync;
const originalMkdir = fs.mkdir;
const originalMkdirSync = fs.mkdirSync;
const originalRm = fs.rm;
const originalRmSync = fs.rmSync;
const originalUnlink = fs.unlink;
const originalUnlinkSync = fs.unlinkSync;
const originalRename = fs.rename;
const originalRenameSync = fs.renameSync;
const originalCopyFile = fs.copyFile;
const originalCopyFileSync = fs.copyFileSync;

fs.writeFile = function(file, data, options, callback) {
    try {
        checkWritePermission(file);
    } catch (e) {
        if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalWriteFile.apply(this, arguments);
};

fs.writeFileSync = function(file, data, options) {
    checkWritePermission(file);
    return originalWriteFileSync.apply(this, arguments);
};

fs.appendFile = function(file, data, options, callback) {
    try {
        checkWritePermission(file);
    } catch (e) {
        if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalAppendFile.apply(this, arguments);
};

fs.appendFileSync = function(file, data, options) {
    checkWritePermission(file);
    return originalAppendFileSync.apply(this, arguments);
};

fs.open = function(path, flags, mode, callback) {
    if (typeof flags === 'string' && /[wax+]/.test(flags)) {
        try {
            checkWritePermission(path);
        } catch (e) {
            if (typeof mode === 'function') {
                mode(e);
            } else if (typeof callback === 'function') {
                callback(e);
            } else {
                throw e;
            }
            return;
        }
    }
    return originalOpen.apply(this, arguments);
};

fs.openSync = function(path, flags, mode) {
    if (typeof flags === 'string' && /[wax+]/.test(flags)) {
        checkWritePermission(path);
    }
    return originalOpenSync.apply(this, arguments);
};

fs.mkdir = function(path, options, callback) {
    try {
        checkWritePermission(path);
    } catch (e) {
        if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalMkdir.apply(this, arguments);
};

fs.mkdirSync = function(path, options) {
    checkWritePermission(path);
    return originalMkdirSync.apply(this, arguments);
};

fs.rm = function(path, options, callback) {
    try {
        checkWritePermission(path);
    } catch (e) {
        if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalRm.apply(this, arguments);
};

fs.rmSync = function(path, options) {
    checkWritePermission(path);
    return originalRmSync.apply(this, arguments);
};

fs.unlink = function(path, callback) {
    try {
        checkWritePermission(path);
    } catch (e) {
        if (callback) callback(e);
        return;
    }
    return originalUnlink.apply(this, arguments);
};

fs.unlinkSync = function(path) {
    checkWritePermission(path);
    return originalUnlinkSync.apply(this, arguments);
};

fs.rename = function(oldPath, newPath, callback) {
    try {
        checkWritePermission(newPath);
    } catch (e) {
        if (callback) callback(e);
        return;
    }
    return originalRename.apply(this, arguments);
};

fs.renameSync = function(oldPath, newPath) {
    checkWritePermission(newPath);
    return originalRenameSync.apply(this, arguments);
};

fs.copyFile = function(src, dest, mode, callback) {
    try {
        checkWritePermission(dest);
    } catch (e) {
        if (typeof mode === 'function') {
            mode(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalCopyFile.apply(this, arguments);
};

fs.copyFileSync = function(src, dest, mode) {
    checkWritePermission(dest);
    return originalCopyFileSync.apply(this, arguments);
};

const SAFE_COMMANDS = new Set([
    'git', 'npm', 'yarn',
    'ls', 'dir', 'cat', 'type', 'head', 'tail', 'echo',
    'grep', 'find', 'where', 'which', 'curl', 'wget',
]);

const originalExec = child_process.exec;
const originalExecSync = child_process.execSync;
const originalSpawn = child_process.spawn;
const originalSpawnSync = child_process.spawnSync;
const originalExecFile = child_process.execFile;
const originalExecFileSync = child_process.execFileSync;

function getCommandName(cmd) {
    if (typeof cmd === 'string') {
        const parts = cmd.split(/\s+/);
        return path.basename(parts[0]).toLowerCase();
    } else if (Array.isArray(cmd) && cmd.length > 0) {
        return path.basename(cmd[0]).toLowerCase();
    }
    return '';
}

function injectSandboxEnv(options) {
    options = options || {};
    options.env = { ...options.env, ...process.env };
    options.env.WANGSHU_WORKSPACE = WORKSPACE_ABSOLUTE;
    return options;
}

function checkCommandAllowed(cmd) {
    const cmdName = getCommandName(cmd);
    if (cmdName && !SAFE_COMMANDS.has(cmdName)) {
        throw new Error(
            `Command '${cmdName}' is not allowed in sandbox mode.\n` +
            `Allowed commands: ${Array.from(SAFE_COMMANDS).sort().join(', ')}`
        );
    }
}

child_process.exec = function(command, options, callback) {
    try {
        checkCommandAllowed(command);
        options = injectSandboxEnv(options);
    } catch (e) {
        if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalExec.apply(this, arguments);
};

child_process.execSync = function(command, options) {
    checkCommandAllowed(command);
    options = injectSandboxEnv(options);
    return originalExecSync.apply(this, arguments);
};

child_process.spawn = function(command, args, options) {
    try {
        checkCommandAllowed(command);
        options = injectSandboxEnv(options);
    } catch (e) {
        throw e;
    }
    return originalSpawn.apply(this, arguments);
};

child_process.spawnSync = function(command, args, options) {
    checkCommandAllowed(command);
    options = injectSandboxEnv(options);
    return originalSpawnSync.apply(this, arguments);
};

child_process.execFile = function(file, args, options, callback) {
    try {
        checkCommandAllowed(file);
        options = injectSandboxEnv(options);
    } catch (e) {
        if (typeof args === 'function') {
            args(e);
        } else if (typeof options === 'function') {
            options(e);
        } else if (typeof callback === 'function') {
            callback(e);
        } else {
            throw e;
        }
        return;
    }
    return originalExecFile.apply(this, arguments);
};

child_process.execFileSync = function(file, args, options) {
    checkCommandAllowed(file);
    options = injectSandboxEnv(options);
    return originalExecFileSync.apply(this, arguments);
};


function runUserCode(code, scriptPath) {
    if (scriptPath) {
        const scriptDir = path.dirname(path.resolve(scriptPath));
        module.paths.unshift(scriptDir);
        process.argv[1] = scriptPath;
    }
    
    try {
        const script = new require('vm').Script(code, {
            filename: scriptPath || '<inline>'
        });
        script.runInThisContext();
    } catch (e) {
        if (e.message && e.message.includes('Security Violation')) {
            console.error('\n🛡️ 沙箱拦截:', e.message);
            process.exit(13);
        }
        throw e;
    }
}

function runUserCodeFromBase64(encodedCode, scriptPath) {
    const code = Buffer.from(encodedCode, 'base64').toString('utf-8');
    runUserCode(code, scriptPath);
}

module.exports = { runUserCode, runUserCodeFromBase64 };
