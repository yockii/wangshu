import os
import sys
import builtins
import shutil
import pathlib
import functools
import base64
import subprocess

WORKSPACE_ROOT = os.environ.get('WANGSHU_WORKSPACE')
if not WORKSPACE_ROOT:
    raise RuntimeError("Security Error: WANGSHU_WORKSPACE not set")

WORKSPACE_ROOT = os.path.realpath(WORKSPACE_ROOT)

def check_write_permission(path):
    if path is None:
        return True
    
    try:
        real_path = os.path.realpath(path)
    except Exception:
        real_path = os.path.realpath(os.path.dirname(path) or '.')

    if not (real_path == WORKSPACE_ROOT or real_path.startswith(WORKSPACE_ROOT + os.sep)):
        raise PermissionError(
            f"Security Violation: Write access denied outside workspace.\n"
            f"Target: {path}\n"
            f"Resolved: {real_path}\n"
            f"Allowed Root: {WORKSPACE_ROOT}"
        )

original_open = builtins.open

def safe_open(file, mode='r', *args, **kwargs):
    if any(m in mode for m in ['w', 'a', 'x', '+']):
        check_write_permission(file)
    
    return original_open(file, mode, *args, **kwargs)

builtins.open = safe_open

def wrap_os_func(func_name, path_arg_index=0):
    original_func = getattr(os, func_name)
    
    @functools.wraps(original_func)
    def wrapper(*args, **kwargs):
        if len(args) > path_arg_index:
            check_write_permission(args[path_arg_index])
        
        return original_func(*args, **kwargs)
    
    setattr(os, func_name, wrapper)

os_write_funcs = [
    'open', 'mkdir', 'makedirs', 'remove', 'unlink', 'rename', 'replace', 
    'symlink', 'link', 'chown', 'chmod'
]

for func in os_write_funcs:
    if hasattr(os, func):
        idx = 0
        if func == 'rename' or func == 'replace':
            orig = getattr(os, func)
            def make_rename_wrapper(orig_func):
                def w(old, new, *args, **kwargs):
                    check_write_permission(new)
                    return orig_func(old, new, *args, **kwargs)
                return w
            setattr(os, func, make_rename_wrapper(orig))
            continue
            
        wrap_os_func(func, idx)

shutil_ops = ['copy', 'copy2', 'copytree', 'move', 'rmtree']
for op in shutil_ops:
    if hasattr(shutil, op):
        orig = getattr(shutil, op)
        @functools.wraps(orig)
        def make_shutil_wrapper(orig_func, op_name):
            def wrapper(*args, **kwargs):
                if op_name in ['copy', 'copy2', 'move']:
                    if len(args) >= 2:
                        check_write_permission(args[1])
                elif op_name in ['copytree', 'rmtree']:
                    if len(args) >= 1:
                        target = args[1] if op_name == 'copytree' else args[0]
                        check_write_permission(target)
                return orig_func(*args, **kwargs)
            return wrapper
        setattr(shutil, op, make_shutil_wrapper(orig, op))

pathlib_write_methods = ['write_text', 'write_bytes', 'mkdir', 'unlink', 'replace']
for method_name in pathlib_write_methods:
    if hasattr(pathlib.Path, method_name):
        orig_method = getattr(pathlib.Path, method_name)
        def make_pathlib_wrapper(orig_meth):
            def wrapper(self, *args, **kwargs):
                check_write_permission(str(self))
                return orig_meth(self, *args, **kwargs)
            return wrapper
        setattr(pathlib.Path, method_name, make_pathlib_wrapper(orig_method))


# --- 5. 拦截 os.system 和 os.popen (完全禁止) ---
def _blocked_shell_func(*args, **kwargs):
    raise PermissionError(
        "os.system() and os.popen() are blocked in sandbox mode. "
        "Use subprocess.run() with explicit command list instead."
    )

os.system = _blocked_shell_func
os.popen = _blocked_shell_func

# --- 6. 拦截 subprocess (白名单 + 环境变量注入) ---
SAFE_COMMANDS = {
    'git', 'npm', 'yarn', 'pip',
    'ls', 'dir', 'cat', 'type', 'head', 'tail', 'echo',
    'grep', 'find', 'where', 'which', 'curl', 'wget',
}

_original_subprocess_run = subprocess.run
_original_subprocess_call = subprocess.call
_original_subprocess_check_output = subprocess.check_output
_original_subprocess_check_call = subprocess.check_call
_original_subprocess_Popen = subprocess.Popen

def _get_command_name(cmd):
    if isinstance(cmd, list) and len(cmd) > 0:
        cmd_path = cmd[0]
        return os.path.basename(cmd_path).lower()
    elif isinstance(cmd, str):
        parts = cmd.split()
        if parts:
            return os.path.basename(parts[0]).lower()
    return ''

def _inject_sandbox_env(kwargs):
    env = kwargs.get('env')
    if env is None:
        env = os.environ.copy()
    else:
        env = env.copy()
    env['WANGSHU_WORKSPACE'] = WORKSPACE_ROOT
    kwargs['env'] = env
    return kwargs

def _sandboxed_subprocess_run(cmd, *args, **kwargs):
    if kwargs.get('shell', False):
        raise PermissionError(
            "subprocess with shell=True is blocked in sandbox mode. "
            "Use subprocess.run(['command', 'arg1', 'arg2']) instead."
        )
    
    cmd_name = _get_command_name(cmd)
    if cmd_name and cmd_name not in SAFE_COMMANDS:
        raise PermissionError(
            f"Command '{cmd_name}' is not allowed in sandbox mode.\n"
            f"Allowed commands: {', '.join(sorted(SAFE_COMMANDS))}"
        )
    
    kwargs = _inject_sandbox_env(kwargs)
    return _original_subprocess_run(cmd, *args, **kwargs)

def _sandboxed_subprocess_call(cmd, *args, **kwargs):
    if kwargs.get('shell', False):
        raise PermissionError(
            "subprocess with shell=True is blocked in sandbox mode. "
            "Use subprocess.call(['command', 'arg1', 'arg2']) instead."
        )
    
    cmd_name = _get_command_name(cmd)
    if cmd_name and cmd_name not in SAFE_COMMANDS:
        raise PermissionError(
            f"Command '{cmd_name}' is not allowed in sandbox mode.\n"
            f"Allowed commands: {', '.join(sorted(SAFE_COMMANDS))}"
        )
    
    kwargs = _inject_sandbox_env(kwargs)
    return _original_subprocess_call(cmd, *args, **kwargs)

def _sandboxed_subprocess_check_output(cmd, *args, **kwargs):
    if kwargs.get('shell', False):
        raise PermissionError(
            "subprocess with shell=True is blocked in sandbox mode. "
            "Use subprocess.check_output(['command', 'arg1', 'arg2']) instead."
        )
    
    cmd_name = _get_command_name(cmd)
    if cmd_name and cmd_name not in SAFE_COMMANDS:
        raise PermissionError(
            f"Command '{cmd_name}' is not allowed in sandbox mode.\n"
            f"Allowed commands: {', '.join(sorted(SAFE_COMMANDS))}"
        )
    
    kwargs = _inject_sandbox_env(kwargs)
    return _original_subprocess_check_output(cmd, *args, **kwargs)

def _sandboxed_subprocess_check_call(cmd, *args, **kwargs):
    if kwargs.get('shell', False):
        raise PermissionError(
            "subprocess with shell=True is blocked in sandbox mode. "
            "Use subprocess.check_call(['command', 'arg1', 'arg2']) instead."
        )
    
    cmd_name = _get_command_name(cmd)
    if cmd_name and cmd_name not in SAFE_COMMANDS:
        raise PermissionError(
            f"Command '{cmd_name}' is not allowed in sandbox mode.\n"
            f"Allowed commands: {', '.join(sorted(SAFE_COMMANDS))}"
        )
    
    kwargs = _inject_sandbox_env(kwargs)
    return _original_subprocess_check_call(cmd, *args, **kwargs)

def _sandboxed_subprocess_Popen(cmd, *args, **kwargs):
    if kwargs.get('shell', False):
        raise PermissionError(
            "subprocess with shell=True is blocked in sandbox mode. "
            "Use subprocess.Popen(['command', 'arg1', 'arg2']) instead."
        )
    
    cmd_name = _get_command_name(cmd)
    if cmd_name and cmd_name not in SAFE_COMMANDS:
        raise PermissionError(
            f"Command '{cmd_name}' is not allowed in sandbox mode.\n"
            f"Allowed commands: {', '.join(sorted(SAFE_COMMANDS))}"
        )
    
    kwargs = _inject_sandbox_env(kwargs)
    return _original_subprocess_Popen(cmd, *args, **kwargs)

subprocess.run = _sandboxed_subprocess_run
subprocess.call = _sandboxed_subprocess_call
subprocess.check_output = _sandboxed_subprocess_check_output
subprocess.check_call = _sandboxed_subprocess_check_call
subprocess.Popen = _sandboxed_subprocess_Popen


def run_user_code(code, script_path=None):
    if script_path:
        sys.path.insert(0, os.path.dirname(os.path.abspath(script_path)))
        globals()['__file__'] = script_path
    else:
        globals()['__file__'] = '<inline>'
    
    try:
        exec(code, globals(), locals())
    except PermissionError as e:
        print(f"\n🛡️ 沙箱拦截: {e}", file=sys.stderr)
        sys.exit(13)
    except SystemExit:
        raise
    except Exception as e:
        raise


def run_user_code_from_base64(encoded_code, script_path=None):
    try:
        code = base64.b64decode(encoded_code).decode('utf-8')
    except Exception as e:
        raise RuntimeError(f"Failed to decode user code: {e}")
    run_user_code(code, script_path)
