package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	"github.com/yockii/wangshu/pkg/utils"
)

const ToolNameVariable = "variable"

type VariableTool struct {
	basic.SimpleTool
}

func NewVariableTool() *VariableTool {
	tool := new(VariableTool)
	tool.Name_ = ToolNameVariable
	tool.Desc_ = "Get runtime variables and configuration values. Supports getting config file path, agent workspace, skill directories, application version, operating system info, and other runtime parameters. Use this tool when you need to access system or configuration information."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"names": map[string]any{
				"type":        "string",
				"description": "Variable names to retrieve, comma-separated. Available variables: 'config_file', 'workspace', 'skill_path', 'app_version', 'os', 'arch', 'exec_path', 'home_dir'. Example: 'config_file,workspace,app_version'",
			},
		},
		"required": []string{"names"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *VariableTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	namesStr := params["names"]
	if namesStr == "" {
		return types.NewToolResult().WithError(fmt.Errorf("names parameter is required"))
	}

	names := strings.Split(namesStr, ",")
	for i, name := range names {
		names[i] = strings.TrimSpace(name)
	}

	agentName := params[constant.ToolCallParamAgentName]

	result := make(map[string]string)
	errors := make([]string, 0)

	for _, name := range names {
		if name == "" {
			continue
		}
		value, err := t.getVariable(name, agentName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", name, err.Error()))
		} else {
			result[name] = value
		}
	}

	if len(result) == 0 && len(errors) > 0 {
		return types.NewToolResult().WithError(fmt.Errorf("failed to get variables: %s", strings.Join(errors, "; ")))
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to marshal result: %w", err))
	}

	output := string(jsonData)
	if len(errors) > 0 {
		output += "\n\nErrors:\n" + strings.Join(errors, "\n")
	}

	return types.NewToolResult().WithRaw(output).WithStructured(map[string]any{"data": result, "errors": errors})
}

func (t *VariableTool) getVariable(name, agentName string) (string, error) {
	switch name {
	case "config_file":
		cfgPath := "~/.wangshu/config.json"
		if len(os.Args) > 1 {
			cfgPath = os.Args[1]
		}
		return utils.ExpandPath(cfgPath), nil

	case "workspace":
		if agentName == "" {
			return "", fmt.Errorf("agent_name is required for workspace variable")
		}
		if config.DefaultCfg == nil {
			return "", fmt.Errorf("configuration not initialized")
		}
		agent, ok := config.DefaultCfg.Agents[agentName]
		if !ok {
			return "", fmt.Errorf("agent '%s' not found", agentName)
		}
		return utils.ExpandPath(agent.Workspace), nil

	case "skill_path":
		if config.DefaultCfg == nil {
			return "", fmt.Errorf("configuration not initialized")
		}
		return utils.ExpandPath(config.DefaultCfg.Skill.GlobalPath), nil

	case "app_version":
		return constant.Version, nil

	case "os":
		return runtime.GOOS, nil

	case "arch":
		return runtime.GOARCH, nil

	case "exec_path":
		execPath, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("failed to get executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlinks: %w", err)
		}
		return execPath, nil

	case "home_dir":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return homeDir, nil

	default:
		return "", fmt.Errorf("unknown variable: %s", name)
	}
}
