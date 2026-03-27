package configtool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yockii/wangshu/internal/config"
	actiontypes "github.com/yockii/wangshu/pkg/action/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	"github.com/yockii/wangshu/pkg/utils"
)

var reloadFunc func() error

func SetReloadFunc(fn func() error) {
	reloadFunc = fn
}

const (
	ConfigActionGet      = "get"
	ConfigActionSet      = "set"
	ConfigActionAdd      = "add"
	ConfigActionDelete   = "delete"
	ConfigActionValidate = "validate"
	ConfigActionReload   = "reload"
)

type ConfigTool struct {
	basic.SimpleTool
}

func NewConfigTool() *ConfigTool {
	tool := new(ConfigTool)
	tool.Name_ = constant.ToolNameConfig
	tool.Desc_ = `Manage application configuration file. Provides safe and validated configuration operations.

IMPORTANT: Use this tool instead of edit_file for configuration changes because:
1. It validates configuration before saving to prevent invalid settings
2. It ensures proper JSON formatting
3. It supports granular operations (add/update/delete single items)
4. It can reload configuration without restarting the application

Available actions:
- get: Retrieve configuration (full, by section, or by specific item)
- set: Update configuration (section or specific item)
- add: Add new configuration item (agent/provider/channel)
- delete: Delete configuration item (agent/provider/channel)
- validate: Check if current configuration is valid
- reload: Reload configuration from file`
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform",
				"enum":        []string{ConfigActionGet, ConfigActionSet, ConfigActionAdd, ConfigActionDelete, ConfigActionValidate, ConfigActionReload},
			},
			"section": map[string]any{
				"type":        "string",
				"description": "Configuration section: 'agents', 'providers', 'channels', 'skill', 'browser'. Required for get/set/add/delete actions.",
				"enum":        []string{"agents", "providers", "channels", "skill", "browser"},
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the specific item to get/set/add/delete (e.g., agent name 'default', provider name 'myProvider', channel name 'webTest'). For get action: returns specific item. For set action: updates specific item. For add action: name of new item. For delete action: name of item to delete.",
			},
			"data": map[string]any{
				"type":        "string",
				"description": "JSON string containing configuration data. For 'set' action on section: updates entire section. For 'set' with name: updates single item. For 'add' action: the new item's configuration. Examples: '{\"workspace\": \"/path\", \"provider\": \"myProvider\", \"model\": \"gpt-4\"}' for a single agent, or '{\"default\": {...}}' for section-level update.",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *ConfigTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	action := params["action"]

	switch action {
	case ConfigActionGet:
		return t.get(params)
	case ConfigActionSet:
		return t.set(params)
	case ConfigActionAdd:
		return t.add(params)
	case ConfigActionDelete:
		return t.delete(params)
	case ConfigActionValidate:
		return t.validate()
	case ConfigActionReload:
		return t.reload()
	default:
		return types.NewToolResult().WithError(fmt.Errorf("invalid action: %s", action))
	}
}

func (t *ConfigTool) getConfigPath() string {
	cfgPath := "~/.wangshu/config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	return utils.ExpandPath(cfgPath)
}

func (t *ConfigTool) get(params map[string]string) *types.ToolResult {
	cfgOrigin := config.DefaultCfg
	if cfgOrigin == nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	// 复制一份，不要直接获取原始配置
	cfg := cfgOrigin.Copy()

	// 出于安全考虑，所有的ak/sk等信息屏蔽掉
	for _, provider := range cfg.Providers {
		provider.APIKey = "API-KEY-REDACTED"
	}
	for _, channel := range cfg.Channels {
		channel.AppSecret = "APP-SECRET-REDACTED"
		channel.Token = "TOKEN-REDACTED"
	}

	// 处理参数
	section := params["section"]
	name := params["name"]

	if section == "" {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal configuration: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg, nil))
	}

	if name == "" {
		return t.getSection(cfg, section)
	}
	return t.getSectionItem(cfg, section, name)
}

func (t *ConfigTool) getSection(cfg *config.Config, section string) *types.ToolResult {
	switch section {
	case "agents":
		data, err := json.MarshalIndent(cfg.Agents, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal agents: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg.Agents, nil))
	case "providers":
		data, err := json.MarshalIndent(cfg.Providers, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal providers: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg.Providers, nil))
	case "channels":
		data, err := json.MarshalIndent(cfg.Channels, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal channels: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg.Channels, nil))
	case "skill":
		data, err := json.MarshalIndent(cfg.Skill, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal skill: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg.Skill, nil))
	case "browser":
		data, err := json.MarshalIndent(cfg.Browser, "", "  ")
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to marshal browser: %w", err))
		}
		return types.NewToolResult().WithRaw(string(data)).
			WithStructured(actiontypes.NewActionOutput("success", "", cfg.Browser, nil))
	default:
		return types.NewToolResult().WithError(fmt.Errorf("unknown section: %s", section))
	}
}

func (t *ConfigTool) getSectionItem(cfg *config.Config, section, name string) *types.ToolResult {
	switch section {
	case "agents":
		if agent, ok := cfg.Agents[name]; ok {
			data, err := json.MarshalIndent(agent, "", "  ")
			if err != nil {
				return types.NewToolResult().WithError(fmt.Errorf("failed to marshal agent: %w", err))
			}
			return types.NewToolResult().WithRaw(string(data)).
				WithStructured(actiontypes.NewActionOutput("success", "", agent, nil))
		}
		return types.NewToolResult().WithError(fmt.Errorf("agent '%s' not found", name))
	case "providers":
		if provider, ok := cfg.Providers[name]; ok {
			data, err := json.MarshalIndent(provider, "", "  ")
			if err != nil {
				return types.NewToolResult().WithError(fmt.Errorf("failed to marshal provider: %w", err))
			}
			return types.NewToolResult().WithRaw(string(data)).
				WithStructured(actiontypes.NewActionOutput("success", "", provider, nil))
		}
		return types.NewToolResult().WithError(fmt.Errorf("provider '%s' not found", name))
	case "channels":
		if channel, ok := cfg.Channels[name]; ok {
			data, err := json.MarshalIndent(channel, "", "  ")
			if err != nil {
				return types.NewToolResult().WithError(fmt.Errorf("failed to marshal channel: %w", err))
			}
			return types.NewToolResult().WithRaw(string(data)).
				WithStructured(actiontypes.NewActionOutput("success", "", channel, nil))
		}
		return types.NewToolResult().WithError(fmt.Errorf("channel '%s' not found", name))
	default:
		return types.NewToolResult().WithError(fmt.Errorf("section '%s' does not support item-level access", section))
	}
}

func (t *ConfigTool) set(params map[string]string) *types.ToolResult {
	section := params["section"]
	if section == "" {
		return types.NewToolResult().WithError(fmt.Errorf("section parameter is required for set action"))
	}

	dataStr := params["data"]
	if dataStr == "" {
		return types.NewToolResult().WithError(fmt.Errorf("data parameter is required for set action"))
	}

	cfg := config.DefaultCfg
	if cfg == nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	name := params["name"]
	var result string

	if name == "" {
		result = t.setSection(cfg, section, dataStr)
	} else {
		result = t.setSectionItem(cfg, section, name, dataStr)
	}

	if err := cfg.Validate(); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration validation failed: %w", err))
	}

	cfgPath := t.getConfigPath()
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to save configuration: %w", err))
	}

	return types.NewToolResult().WithRaw(result)
}

func (t *ConfigTool) setSection(cfg *config.Config, section, dataStr string) string {
	switch section {
	case "agents":
		var agents map[string]*config.AgentConfig
		if err := json.Unmarshal([]byte(dataStr), &agents); err == nil {
			cfg.UpdateAgents(agents)
			return fmt.Sprintf("Agents section updated with %d items", len(agents))
		}
	case "providers":
		var providers map[string]*config.ProviderConfig
		if err := json.Unmarshal([]byte(dataStr), &providers); err == nil {
			cfg.UpdateProviders(providers)
			return fmt.Sprintf("Providers section updated with %d items", len(providers))
		}
	case "channels":
		var channels map[string]*config.ChannelConfig
		if err := json.Unmarshal([]byte(dataStr), &channels); err == nil {
			cfg.UpdateChannels(channels)
			return fmt.Sprintf("Channels section updated with %d items", len(channels))
		}
	case "skill":
		var skill config.SkillConfig
		if err := json.Unmarshal([]byte(dataStr), &skill); err == nil {
			cfg.UpdateSkill(skill)
			return "Skill configuration updated"
		}
	case "browser":
		var browser config.BrowserConfig
		if err := json.Unmarshal([]byte(dataStr), &browser); err == nil {
			cfg.UpdateBrowser(browser)
			return "Browser configuration updated"
		}
	}
	return fmt.Sprintf("Configuration section '%s' updated", section)
}

func (t *ConfigTool) setSectionItem(cfg *config.Config, section, name, dataStr string) string {
	switch section {
	case "agents":
		var agent config.AgentConfig
		if err := json.Unmarshal([]byte(dataStr), &agent); err == nil {
			cfg.SetAgent(name, &agent)
			return fmt.Sprintf("Agent '%s' updated", name)
		}
	case "providers":
		var provider config.ProviderConfig
		if err := json.Unmarshal([]byte(dataStr), &provider); err == nil {
			cfg.SetProvider(name, &provider)
			return fmt.Sprintf("Provider '%s' updated", name)
		}
	case "channels":
		var channel config.ChannelConfig
		if err := json.Unmarshal([]byte(dataStr), &channel); err == nil {
			cfg.SetChannel(name, &channel)
			return fmt.Sprintf("Channel '%s' updated", name)
		}
	}
	return fmt.Sprintf("Item '%s' in section '%s' updated", name, section)
}

func (t *ConfigTool) add(params map[string]string) *types.ToolResult {
	section := params["section"]
	if section == "" {
		return types.NewToolResult().WithError(fmt.Errorf("section parameter is required for add action"))
	}

	name := params["name"]
	if name == "" {
		return types.NewToolResult().WithError(fmt.Errorf("name parameter is required for add action"))
	}

	dataStr := params["data"]
	if dataStr == "" {
		return types.NewToolResult().WithError(fmt.Errorf("data parameter is required for add action"))
	}

	cfg := config.DefaultCfg
	if cfg == nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	switch section {
	case "agents":
		if _, exists := cfg.Agents[name]; exists {
			return types.NewToolResult().WithError(fmt.Errorf("agent '%s' already exists, use 'set' action to update", name))
		}
		var agent config.AgentConfig
		if err := json.Unmarshal([]byte(dataStr), &agent); err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to parse agent data: %w", err))
		}
		cfg.SetAgent(name, &agent)

	case "providers":
		if _, exists := cfg.Providers[name]; exists {
			return types.NewToolResult().WithError(fmt.Errorf("provider '%s' already exists, use 'set' action to update", name))
		}
		var provider config.ProviderConfig
		if err := json.Unmarshal([]byte(dataStr), &provider); err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to parse provider data: %w", err))
		}
		cfg.SetProvider(name, &provider)

	case "channels":
		if _, exists := cfg.Channels[name]; exists {
			return types.NewToolResult().WithError(fmt.Errorf("channel '%s' already exists, use 'set' action to update", name))
		}
		var channel config.ChannelConfig
		if err := json.Unmarshal([]byte(dataStr), &channel); err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to parse channel data: %w", err))
		}
		cfg.SetChannel(name, &channel)

	default:
		return types.NewToolResult().WithError(fmt.Errorf("section '%s' does not support add action", section))
	}

	if err := cfg.Validate(); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration validation failed: %w", err))
	}

	cfgPath := t.getConfigPath()
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to save configuration: %w", err))
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("%s '%s' added successfully", section[:len(section)-1], name))
}

func (t *ConfigTool) delete(params map[string]string) *types.ToolResult {
	section := params["section"]
	if section == "" {
		return types.NewToolResult().WithError(fmt.Errorf("section parameter is required for delete action"))
	}

	name := params["name"]
	if name == "" {
		return types.NewToolResult().WithError(fmt.Errorf("name parameter is required for delete action"))
	}

	cfg := config.DefaultCfg
	if cfg == nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	switch section {
	case "agents":
		if _, exists := cfg.Agents[name]; !exists {
			return types.NewToolResult().WithError(fmt.Errorf("agent '%s' not found", name))
		}
		cfg.DeleteAgent(name)

	case "providers":
		if _, exists := cfg.Providers[name]; !exists {
			return types.NewToolResult().WithError(fmt.Errorf("provider '%s' not found", name))
		}
		cfg.DeleteProvider(name)

	case "channels":
		if _, exists := cfg.Channels[name]; !exists {
			return types.NewToolResult().WithError(fmt.Errorf("channel '%s' not found", name))
		}
		cfg.DeleteChannel(name)

	default:
		return types.NewToolResult().WithError(fmt.Errorf("section '%s' does not support delete action", section))
	}

	if err := cfg.Validate(); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration validation failed: %w", err))
	}

	cfgPath := t.getConfigPath()
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to save configuration: %w", err))
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("%s '%s' deleted successfully", section[:len(section)-1], name))
}

func (t *ConfigTool) validate() *types.ToolResult {
	cfg := config.DefaultCfg
	if cfg == nil {
		return types.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	if err := cfg.Validate(); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("Configuration validation failed:\n%s", err.Error()))
	}

	return types.NewToolResult().WithRaw("Configuration is valid")
}

func (t *ConfigTool) reload() *types.ToolResult {
	if reloadFunc == nil {
		return types.NewToolResult().WithError(fmt.Errorf("reload function not initialized"))
	}

	if err := reloadFunc(); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to reload configuration: %w", err))
	}

	return types.NewToolResult().WithRaw("Configuration reloaded successfully. All agents, providers, and channels have been reinitialized.")
}
