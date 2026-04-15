package configtool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/store"
	configtypes "github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	tooltypes "github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
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

const (
	RedactedAPIKey    = "API-KEY-REDACTED"
	RedactedAppSecret = "APP-SECRET-REDACTED"
	RedactedToken     = "TOKEN-REDACTED"
)

type ConfigTool struct {
	basic.SimpleTool
}

func NewConfigTool() *ConfigTool {
	tool := new(ConfigTool)
	tool.Name_ = constant.ToolNameConfig
	tool.Desc_ = `Manage application configuration. Provides safe and validated configuration operations.

IMPORTANT: Use this tool instead of edit_file for configuration changes because:
1. It validates configuration before saving to prevent invalid settings
2. It ensures proper JSON formatting
3. It supports granular operations (add/update/delete single items)
4. It can reload configuration without restarting the application

Available actions:
- get: Retrieve configuration (full, by section, or by specific item by ID)
- set: Update an existing item by ID, or update a singleton section (skill/browser/live2d)
- add: Add new configuration item (agent/provider/channel/mcp_server). The data JSON should include the name field (e.g. agent_name, provider_name, channel_name, name). ID will be auto-generated.
- delete: Delete configuration item by ID
- validate: Check if current configuration is valid
- reload: Reload configuration`
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
				"description": "Configuration section: 'agents', 'providers', 'channels', 'skill', 'browser', 'live2d', 'mcp_servers'. Required for get/set/add/delete actions.",
				"enum":        []string{"agents", "providers", "channels", "skill", "browser", "live2d", "mcp_servers"},
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Database ID of the specific item. For 'get' with section: returns specific item by ID. For 'set': updates the item with this ID. For 'delete': deletes the item with this ID. Not needed for 'add' (ID is auto-generated) or singleton sections (skill/browser/live2d).",
			},
			"data": map[string]any{
				"type":        "string",
				"description": "JSON string containing configuration data. For 'set' action: the complete item data including ID and name fields. For 'add' action: the new item's configuration (ID will be auto-generated if empty).",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *ConfigTool) execute(ctx context.Context, params map[string]string) *tooltypes.ToolResult {
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
		return tooltypes.NewToolResult().WithError(fmt.Errorf("invalid action: %s", action))
	}
}

func redactProvider(p *configtypes.ProviderConfig) {
	p.APIKey = RedactedAPIKey
}

func redactChannel(c *configtypes.ChannelConfig) {
	c.AppSecret = RedactedAppSecret
	c.Token = RedactedToken
}

func (t *ConfigTool) get(params map[string]string) *tooltypes.ToolResult {
	section := params["section"]
	id := params["id"]

	if section == "" {
		return t.getAll()
	}
	if id == "" {
		return t.getSection(section)
	}
	return t.getItemByID(section, id)
}

func (t *ConfigTool) getAll() *tooltypes.ToolResult {
	cfg := config.DefaultCfg
	if cfg == nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}
	cfgCopy := cfg.Copy()
	for _, provider := range cfgCopy.Providers {
		redactProvider(provider)
	}
	for _, channel := range cfgCopy.Channels {
		redactChannel(channel)
	}
	data, err := json.MarshalIndent(cfgCopy, "", "  ")
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal configuration: %w", err))
	}
	return tooltypes.NewToolResult().WithRaw(string(data)).
		WithStructured(actiontypes.NewActionOutput("success", "", cfgCopy, nil))
}

func (t *ConfigTool) getSection(section string) *tooltypes.ToolResult {
	switch section {
	case "agents":
		return listSection[*configtypes.AgentConfig](constant.StorePrefixAgent, "agents", false)
	case "providers":
		return listSection[*configtypes.ProviderConfig](constant.StorePrefixProvider, "providers", true)
	case "channels":
		return listSection[*configtypes.ChannelConfig](constant.StorePrefixChannel, "channels", true)
	case "skill":
		return getSingleton[*configtypes.SkillConfig](constant.StoreSkill, constant.SkillID, "skill", false)
	case "browser":
		return getSingleton[*configtypes.BrowserConfig](constant.StoreBrowser, constant.BrowserID, "browser", false)
	case "live2d":
		return getSingleton[*configtypes.Live2DConfig](constant.StoreLive2D, constant.Live2DID, "live2d", false)
	case "mcp_servers":
		return listSection[*configtypes.McpConfig](constant.StorePrefixMcpServer, "mcp_servers", false)
	default:
		return tooltypes.NewToolResult().WithError(fmt.Errorf("unknown section: %s", section))
	}
}

func listSection[T configtypes.BaseConfig](prefixKey, sectionName string, needRedact bool) *tooltypes.ToolResult {
	items, err := store.List[T](prefixKey)
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to list %s: %w", sectionName, err))
	}
	if needRedact {
		for _, item := range items {
			switch v := any(item).(type) {
			case *configtypes.ProviderConfig:
				redactProvider(v)
			case *configtypes.ChannelConfig:
				redactChannel(v)
			}
		}
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal %s: %w", sectionName, err))
	}
	return tooltypes.NewToolResult().WithRaw(string(data)).
		WithStructured(actiontypes.NewActionOutput("success", "", items, nil))
}

func getSingleton[T configtypes.BaseConfig](prefixKey, id, sectionName string, needRedact bool) *tooltypes.ToolResult {
	item, err := store.Get[T](prefixKey, id)
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to get %s config: %w", sectionName, err))
	}
	if needRedact {
		switch v := any(item).(type) {
		case *configtypes.ProviderConfig:
			redactProvider(v)
		case *configtypes.ChannelConfig:
			redactChannel(v)
		}
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal %s: %w", sectionName, err))
	}
	return tooltypes.NewToolResult().WithRaw(string(data)).
		WithStructured(actiontypes.NewActionOutput("success", "", item, nil))
}

func (t *ConfigTool) getItemByID(section, id string) *tooltypes.ToolResult {
	switch section {
	case "agents":
		return getItemByIDWithRedact[*configtypes.AgentConfig](constant.StorePrefixAgent, "agent", id, false)
	case "providers":
		return getItemByIDWithRedact[*configtypes.ProviderConfig](constant.StorePrefixProvider, "provider", id, true)
	case "channels":
		return getItemByIDWithRedact[*configtypes.ChannelConfig](constant.StorePrefixChannel, "channel", id, true)
	case "mcp_servers":
		return getItemByIDWithRedact[*configtypes.McpConfig](constant.StorePrefixMcpServer, "mcp_server", id, false)
	default:
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section '%s' does not support item-level access by ID", section))
	}
}

func getItemByIDWithRedact[T configtypes.BaseConfig](prefixKey, typeName, id string, needRedact bool) *tooltypes.ToolResult {
	item, err := store.Get[T](prefixKey, id)
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("%s with id '%s' not found: %w", typeName, id, err))
	}
	if needRedact {
		switch v := any(item).(type) {
		case *configtypes.ProviderConfig:
			r := *v
			redactProvider(&r)
			data, err := json.MarshalIndent(&r, "", "  ")
			if err != nil {
				return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal %s: %w", typeName, err))
			}
			return tooltypes.NewToolResult().WithRaw(string(data)).
				WithStructured(actiontypes.NewActionOutput("success", "", &r, nil))
		case *configtypes.ChannelConfig:
			r := *v
			redactChannel(&r)
			data, err := json.MarshalIndent(&r, "", "  ")
			if err != nil {
				return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal %s: %w", typeName, err))
			}
			return tooltypes.NewToolResult().WithRaw(string(data)).
				WithStructured(actiontypes.NewActionOutput("success", "", &r, nil))
		}
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to marshal %s: %w", typeName, err))
	}
	return tooltypes.NewToolResult().WithRaw(string(data)).
		WithStructured(actiontypes.NewActionOutput("success", "", item, nil))
}

func (t *ConfigTool) set(params map[string]string) *tooltypes.ToolResult {
	section := params["section"]
	if section == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section parameter is required for set action"))
	}

	dataStr := params["data"]
	if dataStr == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("data parameter is required for set action"))
	}

	id := params["id"]

	switch section {
	case "agents":
		var agent configtypes.AgentConfig
		if err := json.Unmarshal([]byte(dataStr), &agent); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse agent data: %w", err))
		}
		if id != "" {
			agent.SetID(id)
		}
		if agent.AgentName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("agent_name is required"))
		}
		if agent.Workspace != "" {
			agent.Workspace = utils.ExpandPath(agent.Workspace)
		}
		if err := store.Save(constant.StorePrefixAgent, &agent); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save agent: %w", err))
		}
		syncMemoryAgents()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Agent '%s' (id: %s) updated", agent.AgentName, agent.GetID()))

	case "providers":
		var provider configtypes.ProviderConfig
		if err := json.Unmarshal([]byte(dataStr), &provider); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse provider data: %w", err))
		}
		if id != "" {
			provider.SetID(id)
		}
		if provider.ProviderName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("provider_name is required"))
		}
		restoreProviderSecrets(&provider)
		if err := store.Save(constant.StorePrefixProvider, &provider); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save provider: %w", err))
		}
		syncMemoryProviders()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Provider '%s' (id: %s) updated", provider.ProviderName, provider.GetID()))

	case "channels":
		var channel configtypes.ChannelConfig
		if err := json.Unmarshal([]byte(dataStr), &channel); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse channel data: %w", err))
		}
		if id != "" {
			channel.SetID(id)
		}
		if channel.ChannelName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("channel_name is required"))
		}
		restoreChannelSecrets(&channel)
		if err := store.Save(constant.StorePrefixChannel, &channel); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save channel: %w", err))
		}
		syncMemoryChannels()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Channel '%s' (id: %s) updated", channel.ChannelName, channel.GetID()))

	case "skill":
		var skill configtypes.SkillConfig
		if err := json.Unmarshal([]byte(dataStr), &skill); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse skill data: %w", err))
		}
		skill.ID = constant.SkillID
		if skill.GlobalPath != "" {
			skill.GlobalPath = utils.ExpandPath(skill.GlobalPath)
		}
		if err := store.Save(constant.StoreSkill, &skill); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save skill: %w", err))
		}
		syncMemorySkill()
		return tooltypes.NewToolResult().WithRaw("Skill configuration updated")

	case "browser":
		var browser configtypes.BrowserConfig
		if err := json.Unmarshal([]byte(dataStr), &browser); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse browser data: %w", err))
		}
		browser.ID = constant.BrowserID
		if browser.DataDir != "" {
			browser.DataDir = utils.ExpandPath(browser.DataDir)
		}
		if err := store.Save(constant.StoreBrowser, &browser); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save browser: %w", err))
		}
		syncMemoryBrowser()
		return tooltypes.NewToolResult().WithRaw("Browser configuration updated")

	case "live2d":
		var live2d configtypes.Live2DConfig
		if err := json.Unmarshal([]byte(dataStr), &live2d); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse live2d data: %w", err))
		}
		live2d.ID = constant.Live2DID
		if live2d.ModelDir != "" {
			live2d.ModelDir = utils.ExpandPath(live2d.ModelDir)
		}
		if err := store.Save(constant.StoreLive2D, &live2d); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save live2d: %w", err))
		}
		syncMemoryLive2D()
		return tooltypes.NewToolResult().WithRaw("Live2D configuration updated")

	case "mcp_servers":
		var mcp configtypes.McpConfig
		if err := json.Unmarshal([]byte(dataStr), &mcp); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse mcp_server data: %w", err))
		}
		if id != "" {
			mcp.SetID(id)
		}
		if mcp.Name == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("name is required for mcp_server"))
		}
		if err := store.Save(constant.StorePrefixMcpServer, &mcp); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save mcp_server: %w", err))
		}
		syncMemoryMcpServers()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("MCP server '%s' (id: %s) updated", mcp.Name, mcp.GetID()))

	default:
		return tooltypes.NewToolResult().WithError(fmt.Errorf("unknown section: %s", section))
	}
}

func restoreProviderSecrets(provider *configtypes.ProviderConfig) {
	if provider.APIKey == RedactedAPIKey && provider.GetID() != "" {
		original, err := store.Get[*configtypes.ProviderConfig](constant.StorePrefixProvider, provider.GetID())
		if err == nil && original != nil {
			provider.APIKey = original.APIKey
		}
	}
}

func restoreChannelSecrets(channel *configtypes.ChannelConfig) {
	needAppSecret := channel.AppSecret == RedactedAppSecret
	needToken := channel.Token == RedactedToken
	if !needAppSecret && !needToken {
		return
	}
	if channel.GetID() == "" {
		return
	}
	original, err := store.Get[*configtypes.ChannelConfig](constant.StorePrefixChannel, channel.GetID())
	if err == nil && original != nil {
		if needAppSecret {
			channel.AppSecret = original.AppSecret
		}
		if needToken {
			channel.Token = original.Token
		}
	}
}

func (t *ConfigTool) add(params map[string]string) *tooltypes.ToolResult {
	section := params["section"]
	if section == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section parameter is required for add action"))
	}

	dataStr := params["data"]
	if dataStr == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("data parameter is required for add action"))
	}

	switch section {
	case "agents":
		var agent configtypes.AgentConfig
		if err := json.Unmarshal([]byte(dataStr), &agent); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse agent data: %w", err))
		}
		if agent.AgentName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("agent_name is required"))
		}
		if agent.Workspace != "" {
			agent.Workspace = utils.ExpandPath(agent.Workspace)
		}
		if err := store.Save(constant.StorePrefixAgent, &agent); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save agent: %w", err))
		}
		syncMemoryAgents()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Agent '%s' added successfully (id: %s)", agent.AgentName, agent.GetID()))

	case "providers":
		var provider configtypes.ProviderConfig
		if err := json.Unmarshal([]byte(dataStr), &provider); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse provider data: %w", err))
		}
		if provider.ProviderName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("provider_name is required"))
		}
		if err := store.Save(constant.StorePrefixProvider, &provider); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save provider: %w", err))
		}
		syncMemoryProviders()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Provider '%s' added successfully (id: %s)", provider.ProviderName, provider.GetID()))

	case "channels":
		var channel configtypes.ChannelConfig
		if err := json.Unmarshal([]byte(dataStr), &channel); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse channel data: %w", err))
		}
		if channel.ChannelName == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("channel_name is required"))
		}
		if err := store.Save(constant.StorePrefixChannel, &channel); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save channel: %w", err))
		}
		syncMemoryChannels()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Channel '%s' added successfully (id: %s)", channel.ChannelName, channel.GetID()))

	case "mcp_servers":
		var mcp configtypes.McpConfig
		if err := json.Unmarshal([]byte(dataStr), &mcp); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to parse mcp_server data: %w", err))
		}
		if mcp.Name == "" {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("name is required for mcp_server"))
		}
		if err := store.Save(constant.StorePrefixMcpServer, &mcp); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to save mcp_server: %w", err))
		}
		syncMemoryMcpServers()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("MCP server '%s' added successfully (id: %s)", mcp.Name, mcp.GetID()))

	default:
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section '%s' does not support add action", section))
	}
}

func (t *ConfigTool) delete(params map[string]string) *tooltypes.ToolResult {
	section := params["section"]
	if section == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section parameter is required for delete action"))
	}

	id := params["id"]
	if id == "" {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("id parameter is required for delete action"))
	}

	switch section {
	case "agents":
		item, err := store.Get[*configtypes.AgentConfig](constant.StorePrefixAgent, id)
		if err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("agent with id '%s' not found: %w", id, err))
		}
		if err := store.Delete[*configtypes.AgentConfig](constant.StorePrefixAgent, id); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to delete agent: %w", err))
		}
		cfg := config.DefaultCfg
		if cfg != nil {
			cfg.DeleteAgent(item.AgentName)
		}
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Agent '%s' (id: %s) deleted successfully", item.AgentName, id))

	case "providers":
		item, err := store.Get[*configtypes.ProviderConfig](constant.StorePrefixProvider, id)
		if err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("provider with id '%s' not found: %w", id, err))
		}
		if err := store.Delete[*configtypes.ProviderConfig](constant.StorePrefixProvider, id); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to delete provider: %w", err))
		}
		cfg := config.DefaultCfg
		if cfg != nil {
			cfg.DeleteProvider(item.ProviderName)
		}
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Provider '%s' (id: %s) deleted successfully", item.ProviderName, id))

	case "channels":
		item, err := store.Get[*configtypes.ChannelConfig](constant.StorePrefixChannel, id)
		if err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("channel with id '%s' not found: %w", id, err))
		}
		if err := store.Delete[*configtypes.ChannelConfig](constant.StorePrefixChannel, id); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to delete channel: %w", err))
		}
		cfg := config.DefaultCfg
		if cfg != nil {
			cfg.DeleteChannel(item.ChannelName)
		}
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("Channel '%s' (id: %s) deleted successfully", item.ChannelName, id))

	case "mcp_servers":
		item, err := store.Get[*configtypes.McpConfig](constant.StorePrefixMcpServer, id)
		if err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("mcp_server with id '%s' not found: %w", id, err))
		}
		if err := store.Delete[*configtypes.McpConfig](constant.StorePrefixMcpServer, id); err != nil {
			return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to delete mcp_server: %w", err))
		}
		syncMemoryMcpServers()
		return tooltypes.NewToolResult().WithRaw(fmt.Sprintf("MCP server '%s' (id: %s) deleted successfully", item.Name, id))

	default:
		return tooltypes.NewToolResult().WithError(fmt.Errorf("section '%s' does not support delete action", section))
	}
}

func (t *ConfigTool) validate() *tooltypes.ToolResult {
	cfg := config.DefaultCfg
	if cfg == nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("configuration not initialized"))
	}

	if err := cfg.Validate(); err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("Configuration validation failed:\n%s", err.Error()))
	}

	return tooltypes.NewToolResult().WithRaw("Configuration is valid")
}

func (t *ConfigTool) reload() *tooltypes.ToolResult {
	if reloadFunc == nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("reload function not initialized"))
	}

	if err := reloadFunc(); err != nil {
		return tooltypes.NewToolResult().WithError(fmt.Errorf("failed to reload configuration: %w", err))
	}

	return tooltypes.NewToolResult().WithRaw("Configuration reloaded successfully. All agents, providers, and channels have been reinitialized.")
}

func syncMemoryAgents() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	agents, err := store.List[*configtypes.AgentConfig](constant.StorePrefixAgent)
	if err != nil {
		return
	}
	m := make(map[string]*configtypes.AgentConfig, len(agents))
	for _, a := range agents {
		if a.Workspace != "" {
			a.Workspace = utils.ExpandPath(a.Workspace)
		}
		m[a.AgentName] = a
	}
	cfg.UpdateAgents(m)
}

func syncMemoryProviders() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	providers, err := store.List[*configtypes.ProviderConfig](constant.StorePrefixProvider)
	if err != nil {
		return
	}
	m := make(map[string]*configtypes.ProviderConfig, len(providers))
	for _, p := range providers {
		m[p.ProviderName] = p
	}
	cfg.UpdateProviders(m)
}

func syncMemoryChannels() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	channels, err := store.List[*configtypes.ChannelConfig](constant.StorePrefixChannel)
	if err != nil {
		return
	}
	m := make(map[string]*configtypes.ChannelConfig, len(channels))
	for _, c := range channels {
		m[c.ChannelName] = c
	}
	cfg.UpdateChannels(m)
}

func syncMemorySkill() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	skill, err := store.Get[*configtypes.SkillConfig](constant.StoreSkill, constant.SkillID)
	if err != nil {
		return
	}
	cfg.UpdateSkill(*skill)
}

func syncMemoryBrowser() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	browser, err := store.Get[*configtypes.BrowserConfig](constant.StoreBrowser, constant.BrowserID)
	if err != nil {
		return
	}
	cfg.UpdateBrowser(*browser)
}

func syncMemoryLive2D() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	live2d, err := store.Get[*configtypes.Live2DConfig](constant.StoreLive2D, constant.Live2DID)
	if err != nil {
		return
	}
	cfg.Live2D = *live2d
}

func syncMemoryMcpServers() {
	cfg := config.DefaultCfg
	if cfg == nil {
		return
	}
	mcpServers, err := store.List[*configtypes.McpConfig](constant.StorePrefixMcpServer)
	if err != nil {
		return
	}
	m := make(map[string]*configtypes.McpConfig, len(mcpServers))
	for _, s := range mcpServers {
		m[s.Name] = s
	}
	cfg.McpServers = m
}
