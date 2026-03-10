package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type PythonRunTool struct {
	basic.SimpleTool
}

func NewPythonRunTool() *PythonRunTool {
	tool := new(PythonRunTool)
	tool.Name_ = "python_run"
	tool.Desc_ = "Execute Python code or script. **USE THIS TOOL INSTEAD OF SHELL COMMANDS FOR PYTHON TASKS.** Supports both inline code and script files. Automatically handles Python 3 detection, cross-platform execution, and installs missing Python packages (pandas, numpy, openpyxl, requests, etc.) when needed."
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
		},
		"required": []string{},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *PythonRunTool) execute(ctx context.Context, params map[string]string) (string, error) {
	code := params["code"]
	scriptPath := params["script_path"]

	if code == "" && scriptPath == "" {
		return "", fmt.Errorf("either 'code' or 'script_path' must be provided")
	}

	if code != "" && scriptPath != "" {
		return "", fmt.Errorf("cannot specify both 'code' and 'script_path'")
	}

	timeout := 30 * time.Second
	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	workingDir := params["working_dir"]

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if code != "" {
		return t.executeCode(ctx, code, workingDir, 0)
	}
	return t.executeScript(ctx, scriptPath, params["args"], workingDir, 0)
}

func (t *PythonRunTool) executeCode(ctx context.Context, code, workingDir string, attempt int) (string, error) {
	pythonCmd, err := t.findPython()
	if err != nil {
		return "", fmt.Errorf("failed to find Python: %w", err)
	}

	cmd := exec.CommandContext(ctx, pythonCmd, "-c", code)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("python execution timed out")
	}

	if err != nil {
		if attempt >= 1 {
			return outputStr, fmt.Errorf("python execution failed after attempting to install dependencies: %w\nOutput:\n%s", err, outputStr)
		}

		missingModule := t.extractMissingModule(outputStr)
		if missingModule != "" {
			_, installErr := t.installPackage(missingModule)
			if installErr != nil {
				return outputStr, fmt.Errorf("python execution failed (missing module: %s, installation failed: %w)\nOriginal error: %w\nOutput:\n%s", missingModule, installErr, err, outputStr)
			}
			return t.executeCode(ctx, code, workingDir, attempt+1)
		}

		return outputStr, fmt.Errorf("python execution failed: %w\nCommand: %s -c <code>\nOutput:\n%s", err, pythonCmd, outputStr)
	}

	return outputStr, nil
}

func (t *PythonRunTool) executeScript(ctx context.Context, scriptPath, args, workingDir string, attempt int) (string, error) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script file not found: %s", scriptPath)
	}

	pythonCmd, err := t.findPython()
	if err != nil {
		return "", fmt.Errorf("failed to find Python: %w", err)
	}

	cmdArgs := []string{scriptPath}
	if args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	cmd := exec.CommandContext(ctx, pythonCmd, cmdArgs...)
	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		cmd.Dir = filepath.Dir(scriptPath)
	}
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("python execution timed out")
	}

	if err != nil {
		if attempt >= 1 {
			return outputStr, fmt.Errorf("python execution failed after attempting to install dependencies: %w\nOutput:\n%s", err, outputStr)
		}

		missingModule := t.extractMissingModule(outputStr)
		if missingModule != "" {
			_, installErr := t.installPackage(missingModule)
			if installErr != nil {
				return outputStr, fmt.Errorf("python execution failed (missing module: %s, installation failed: %w)\nOriginal error: %w\nOutput:\n%s", missingModule, installErr, err, outputStr)
			}
			return t.executeScript(ctx, scriptPath, args, workingDir, attempt+1)
		}

		return outputStr, fmt.Errorf("python execution failed: %w\nCommand: %s %s\nOutput:\n%s", err, pythonCmd, strings.Join(cmdArgs, " "), outputStr)
	}

	return outputStr, nil
}

func (t *PythonRunTool) extractMissingModule(output string) string {
	patterns := []string{
		`ModuleNotFoundError: No module named ['"]([^'"]+)['"]`,
		`ImportError: cannot import name ['"]([^'"]+)['"] from ['"]([^'"]+)['"]`,
		`ImportError: No module named ['"]([^'"]+)['"]`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			moduleName := matches[1]
			moduleName = strings.TrimSpace(moduleName)
			if moduleName != "" {
				return moduleName
			}
		}
	}

	return ""
}

func (t *PythonRunTool) installPackage(packageName string) (string, error) {
	pythonCmd, err := t.findPython()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(pythonCmd, "-m", "pip", "install", packageName)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return outputStr, fmt.Errorf("failed to install package %s: %w", packageName, err)
	}

	return fmt.Sprintf("✅ Successfully installed %s\n%s", packageName, outputStr), nil
}

func (t *PythonRunTool) findPython() (string, error) {
	pythonCommands := []string{"python3", "python", "py"}

	for _, cmd := range pythonCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Python not found. Tried: %v. Please install Python 3 and ensure it's in your PATH", pythonCommands)
}

func (t *PythonRunTool) getPythonVersion(pythonCmd string) (string, error) {
	cmd := exec.Command(pythonCmd, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (t *PythonRunTool) checkPythonVersion(version string) bool {
	return strings.Contains(version, "Python 3")
}
