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
)

//go:embed sandbox_wrapper.js
var jsWrapperScript string

type NodeRunTool struct {
	basic.SimpleTool
}

func NewNodeRunTool() *NodeRunTool {
	tool := new(NodeRunTool)
	tool.Name_ = "node_run"
	tool.Desc_ = "Execute Node.js code or script. **USE THIS TOOL INSTEAD OF SHELL COMMANDS FOR NODE.JS TASKS.** Supports both inline code and script files. Automatically handles Node.js detection and cross-platform execution. If you encounter 'Cannot find module' error, use the install_npm_packages parameter to install required packages first."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Node.js code to execute (use this for short scripts)",
			},
			"script_path": map[string]any{
				"type":        "string",
				"description": "Path to Node.js script file (use this for existing scripts)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default: 30)",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory for execution (optional)",
			},
			"install_npm_packages": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of npm packages to install before execution (e.g., 'lodash,axios,express')",
			},
		},
		"required": []string{},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *NodeRunTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	code := params["code"]
	scriptPath := params["script_path"]
	installPackages := params["install_npm_packages"]

	if code == "" && scriptPath == "" {
		if installPackages != "" {
			return t.installNpmPackages(installPackages, "")
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
		result := t.installNpmPackages(installPackages, workingDir)
		if result.Err != nil {
			return result
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if code != "" {
		return t.executeCode(ctx, code, workingDir)
	}
	return t.executeScript(ctx, scriptPath, workingDir)
}

func (t *NodeRunTool) buildSandboxScript(userCode string, scriptPath string) string {
	var sb strings.Builder
	sb.WriteString(jsWrapperScript)
	sb.WriteString("\n\n// === User Code ===\n")

	encodedCode := base64.StdEncoding.EncodeToString([]byte(userCode))

	if scriptPath != "" {
		sb.WriteString(fmt.Sprintf("runUserCodeFromBase64(%q, %q);\n", encodedCode, scriptPath))
	} else {
		sb.WriteString(fmt.Sprintf("runUserCodeFromBase64(%q);\n", encodedCode))
	}

	return sb.String()
}

func (t *NodeRunTool) executeCode(ctx context.Context, code, workingDir string) *types.ToolResult {
	nodeCmd, err := t.findNode()
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to find Node.js: %w", err))
	}

	fullScript := t.buildSandboxScript(code, "")

	cmd := exec.CommandContext(ctx, nodeCmd, "-e", fullScript)
	cmd.Dir = workingDir
	cmd.Env = t.buildEnv(workingDir)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("node execution timed out"))
	}

	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("node execution failed: %w\nOutput:\n%s", err, outputStr))
	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(map[string]any{
		"output": outputStr,
	})
}

func (t *NodeRunTool) executeScript(ctx context.Context, scriptPath, workingDir string) *types.ToolResult {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return types.NewToolResult().WithError(fmt.Errorf("script file not found: %s", scriptPath))
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read script file: %w", err))
	}

	nodeCmd, err := t.findNode()
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to find Node.js: %w", err))
	}

	fullScript := t.buildSandboxScript(string(scriptContent), scriptPath)

	cmd := exec.CommandContext(ctx, nodeCmd, "-e", fullScript)
	cmd.Dir = workingDir
	cmd.Env = t.buildEnv(workingDir)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return types.NewToolResult().WithError(fmt.Errorf("node execution timed out"))
	}

	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("node execution failed: %w\nOutput:\n%s", err, outputStr)).WithRaw(outputStr)
	}

	return types.NewToolResult().WithRaw(outputStr).WithStructured(map[string]any{
		"output": outputStr,
	})
}

func (t *NodeRunTool) installNpmPackages(packagesStr string, workingDir string) *types.ToolResult {
	packages := strings.Split(packagesStr, ",")
	var results []*types.ToolResult

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		result := t.installNpmPackage(pkg, workingDir)
		if result.Err != nil {
			return result
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return types.NewToolResult().WithError(fmt.Errorf("no valid package names provided"))
	}

	tr := types.NewToolResult()

	outputs := make([]string, 0, len(results))
	for _, result := range results {
		outputs = append(outputs, result.Raw)
		if result.Err != nil {
			if tr.Err != nil {
				tr.Err = errors.Join(tr.Err, result.Err)
			} else {
				tr.Err = result.Err
			}
		}
	}

	return tr.WithRaw(strings.Join(outputs, "\n")).WithStructured(map[string]any{
		"outputs": outputs,
	})
}

func (t *NodeRunTool) installNpmPackage(packageName string, workingDir string) *types.ToolResult {
	npmCmd, err := t.findNpm()
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	cmd := exec.Command(npmCmd, "install", packageName)
	cmd.Env = os.Environ()
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to install package %s: %w", packageName, err)).WithRaw(outputStr)
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("✅ Successfully installed %s\n%s", packageName, outputStr))
}

func (t *NodeRunTool) findNode() (string, error) {
	nodeCommands := []string{"node", "node.exe"}

	for _, cmd := range nodeCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Node.js not found. Tried: %v. Please install Node.js and ensure it's in your PATH", nodeCommands)
}

func (t *NodeRunTool) findNpm() (string, error) {
	npmCommands := []string{"npm", "npm.cmd"}

	for _, cmd := range npmCommands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("npm not found. Please install Node.js and npm from https://nodejs.org/")
}

func (t *NodeRunTool) buildEnv(workingDir string) []string {
	env := os.Environ()

	env = append(env,
		"WANGSHU_WORKSPACE="+workingDir,
		"NODE_NO_WARNINGS=1",
	)

	return env
}
