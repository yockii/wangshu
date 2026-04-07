package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

type NpmRunTool struct {
	basic.SimpleTool
}

func NewNpmRunTool() *NpmRunTool {
	tool := new(NpmRunTool)
	tool.Name_ = "npm_run"
	tool.Desc_ = "Execute npm commands. **USE THIS TOOL INSTEAD OF SHELL COMMANDS FOR NODE.JS TASKS.** Supports install, run scripts, build, test, and other npm operations. Automatically handles npm detection and cross-platform execution."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "npm command to execute (e.g., 'install', 'run build', 'run dev', 'test')",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory containing package.json (optional, defaults to workspace)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default: 300 for install, 60 for others)",
			},
			"flags": map[string]any{
				"type":        "string",
				"description": "Additional npm flags (e.g., '--save-dev', '--global')",
			},
		},
		"required": []string{"command"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *NpmRunTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	command := params["command"]
	if command == "" {
		return types.NewToolResult().WithError(fmt.Errorf("command is required"))
	}

	workingDir := params["working_dir"]
	flags := params["flags"]

	timeout := 60 * time.Second
	if strings.HasPrefix(command, "install") || strings.HasPrefix(command, "i ") {
		timeout = 300 * time.Second
	}

	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	npmCmd, err := t.findNpm()
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to find npm: %w", err))
	}

	cmdArgs := strings.Fields(command)
	if flags != "" {
		cmdArgs = append(cmdArgs, strings.Fields(flags)...)
	}

	cmd := NewCommandContext(ctx, npmCmd, cmdArgs...)
	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		cmd.Dir = t.findPackageJsonDir()
	}
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
	}
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("npm command timed out after %v. Try increasing timeout or using background mode", timeout)).WithStructured(
			actiontypes.NewRunData(outputStr, exitCode),
		)
	}

	if err != nil {
		exitCode := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
		return types.NewToolResult().WithError(fmt.Errorf("npm command failed with exit code %d\n%s", exitCode, outputStr)).WithRaw(outputStr).WithStructured(
			actiontypes.NewRunData(outputStr, exitCode),
		)
	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(
		actiontypes.NewRunData(outputStr, exitCode),
	)
}

func (t *NpmRunTool) findNpm() (string, error) {
	npmCommands := []string{"npm", "npm.cmd"}

	for _, cmd := range npmCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("npm not found. Please install Node.js and npm from https://nodejs.org/")
}

func (t *NpmRunTool) findPackageJsonDir() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := currentDir
	for {
		packageJson := filepath.Join(dir, "package.json")
		if _, err := os.Stat(packageJson); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return currentDir
}
