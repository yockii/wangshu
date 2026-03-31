package runtime

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "embed"

	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

//go:embed sandbox_wrapper.py
var wrapperScript string

type PythonRunTool struct {
	basic.SimpleTool
}

func NewPythonRunTool() *PythonRunTool {
	tool := new(PythonRunTool)
	tool.Name_ = "python_run"
	tool.Desc_ = "Execute Python code or script. **USE THIS TOOL INSTEAD OF SHELL COMMANDS FOR PYTHON TASKS.** Supports both inline code and script files. Automatically handles Python 3 detection and cross-platform execution. If you encounter ModuleNotFoundError, use the install_packages parameter to install required packages first."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Python code to execute (use this for short scripts)",
			},
			"script_path": map[string]any{
				"type":        "string",
				"description": "Path to Python script file (use this for existing scripts)",
			},
			"args": map[string]any{
				"type":        "string",
				"description": "Command line arguments for script (optional)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default: 30)",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory for execution (optional)",
			},
			"install_packages": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of Python packages to install before execution (e.g., 'pandas,numpy,requests')",
			},
		},
		"required": []string{},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *PythonRunTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	code := params["code"]
	scriptPath := params["script_path"]
	installPackages := params["install_packages"]

	if code == "" && scriptPath == "" {
		if installPackages != "" {
			return t.installPackages(installPackages)
		}
		return types.NewToolResult().WithError(fmt.Errorf("either 'code' or 'script_path' must be provided"))
	}

	if code != "" && scriptPath != "" {
		return types.NewToolResult().WithError(fmt.Errorf("cannot specify both 'code' and 'script_path'"))
	}

	timeout := 30 * time.Second
	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	workingDir := params["working_dir"]
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = os.TempDir()
		}
	}

	if installPackages != "" {
		result := t.installPackages(installPackages)
		if result.Err != nil {
			return result
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if code != "" {
		return t.executeCode(ctx, code, workingDir)
	}
	return t.executeScript(ctx, scriptPath, params["args"], workingDir)
}

func (t *PythonRunTool) buildSandboxScript(userCode string, scriptPath string) string {
	var sb strings.Builder
	sb.WriteString(wrapperScript)
	sb.WriteString("\n\n# === User Code ===\n")

	encodedCode := base64.StdEncoding.EncodeToString([]byte(userCode))

	if scriptPath != "" {
		sb.WriteString(fmt.Sprintf("run_user_code_from_base64(%q, %q)\n", encodedCode, scriptPath))
	} else {
		sb.WriteString(fmt.Sprintf("run_user_code_from_base64(%q)\n", encodedCode))
	}

	return sb.String()
}

func (t *PythonRunTool) executeCode(ctx context.Context, code, workingDir string) *types.ToolResult {
	pythonCmd, err := t.findPython()
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to find Python: %w", err))
	}

	fullScript := t.buildSandboxScript(code, "")

	cmd := exec.CommandContext(ctx, pythonCmd, "-c", fullScript)
	cmd.Dir = workingDir
	cmd.Env = t.buildEnv(workingDir)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("python execution timed out"))
	}

	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("python execution failed: %w\nOutput:\n%s", err, outputStr)).WithRaw(outputStr)
	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(
		actiontypes.NewRunData(outputStr, exitCode),
	)
}

func (t *PythonRunTool) executeScript(ctx context.Context, scriptPath, args, workingDir string) *types.ToolResult {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return types.NewToolResult().WithError(fmt.Errorf("script file not found: %s", scriptPath))
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read script file: %w", err))
	}

	pythonCmd, err := t.findPython()
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to find Python: %w", err))
	}

	fullScript := t.buildSandboxScript(string(scriptContent), scriptPath)

	cmd := exec.CommandContext(ctx, pythonCmd, "-c", fullScript)
	cmd.Dir = workingDir
	cmd.Env = t.buildEnv(workingDir)

	if args != "" {
		osArgs := strings.Fields(args)
		cmd.Env = append(cmd.Env, "PYTHON_ARGS="+strings.Join(osArgs, " "))
	}

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
	}
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("python execution timed out")).WithStructured(
			actiontypes.NewRunData(outputStr, exitCode),
		)
	}

	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("python execution failed: %w\nCommand: %s %s\nOutput:\n%s", err, pythonCmd, scriptPath, outputStr)).
			WithRaw(outputStr).WithStructured(
			actiontypes.NewRunData(outputStr, exitCode),
		)
	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(
		actiontypes.NewRunData(outputStr, exitCode),
	)
}

func (t *PythonRunTool) installPackages(packagesStr string) *types.ToolResult {
	packages := strings.Split(packagesStr, ",")
	var results []*types.ToolResult

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		result := t.installPackage(pkg)
		if result.Err != nil {
			return result
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return types.NewToolResult().WithError(fmt.Errorf("no valid package names provided")).WithStructured(
			actiontypes.NewRunData("", 0),
		)
	}

	tr := types.NewToolResult()
	var outputs []string
	for _, result := range results {
		outputs = append(outputs, result.Raw)
		if result.Err != nil {
			if tr.Err != nil {
				tr.WithError(errors.Join(tr.Err, result.Err))
			} else {
				tr.WithError(result.Err)
			}
		}
	}

	return tr.WithRaw(strings.Join(outputs, "\n")).WithStructured(
		actiontypes.NewRunData(strings.Join(outputs, "\n"), 0),
	)
}

func (t *PythonRunTool) installPackage(packageName string) *types.ToolResult {
	tr := types.NewToolResult()
	pythonCmd, err := t.findPython()
	if err != nil {
		return tr.WithError(err)
	}

	cmd := exec.Command(pythonCmd, "-m", "pip", "install", packageName)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return tr.WithError(fmt.Errorf("failed to install package %s: %w", packageName, err)).WithRaw(outputStr)
	}

	return tr.WithRaw(fmt.Sprintf("✅ Successfully installed %s\n%s", packageName, outputStr))
}

func (t *PythonRunTool) findPython() (string, error) {
	pythonCommands := []string{"python", "python3", "py"}

	for _, cmd := range pythonCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Python not found. Tried: %v. Please install Python 3 and ensure it's in your PATH", pythonCommands)
}

func (t *PythonRunTool) buildEnv(workingDir string) []string {
	env := os.Environ()

	env = append(env,
		"WANGSHU_WORKSPACE="+workingDir,
		"PYTHONUNBUFFERED=1",
	)

	return env
}
