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
)

type GitRunTool struct {
	basic.SimpleTool
}

func NewGitRunTool() *GitRunTool {
	tool := new(GitRunTool)
	tool.Name_ = "git_run"
	tool.Desc_ = "Execute git commands. **USE THIS TOOL INSTEAD OF SHELL COMMANDS FOR GIT TASKS.** Supports common git operations like clone, pull, push, commit, status, log, branch, etc. Automatically handles git detection and provides better error messages."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "git command to execute (e.g., 'status', 'pull', 'push', 'commit -m \"message\"', 'clone https://repo.url')",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory for git repository (optional, defaults to workspace)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default: 60)",
			},
		},
		"required": []string{"command"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *GitRunTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	command := params["command"]
	if command == "" {
		return types.NewToolResult().WithError(fmt.Errorf("command is required"))
	}

	workingDir := params["working_dir"]

	timeout := 60 * time.Second
	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	gitCmd, err := t.findGit()
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	cmdArgs := strings.Fields(command)
	cmd := exec.CommandContext(ctx, gitCmd, cmdArgs...)

	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		cmd.Dir = t.findGitRepoDir()
	}
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("git command timed out after %v", timeout))
	}

	outputStr := string(output)
	if err != nil {
		exitCode := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
		return types.NewToolResult().WithError(fmt.Errorf("git command failed with exit code %d\n%s", exitCode, outputStr)).WithRaw(outputStr)

	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(map[string]any{
		"output": outputStr,
	})
}

func (t *GitRunTool) findGit() (string, error) {
	gitCommands := []string{"git", "git.exe"}

	for _, cmd := range gitCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("git not found. Please install Git from https://git-scm.com/")
}

func (t *GitRunTool) findGitRepoDir() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := currentDir
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
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
