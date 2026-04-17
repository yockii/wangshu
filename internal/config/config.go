package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/internal/store"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

var configFile string
var dbFile string
var DefaultCfg *Config

func Initialize(cfgFilePath string) error {
	// 检查是否以.json结尾
	if strings.HasSuffix(cfgFilePath, ".json") {
		configFile = cfgFilePath
	} else {
		dbFile = cfgFilePath
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	DefaultCfg = cfg

	ReleaseLive2dModels()

	return nil
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			if os.IsNotExist(err) {
				cfg = defaultConfig()
				// // 引导用户在控制台上填写内容
				// leadUserToFillConfig(cfg)
				// 写入文件
				err = os.MkdirAll(filepath.Dir(configFile), 0755)
				if err != nil {
					return nil, err
				}
				cfgJson, err := json.MarshalIndent(cfg, "", "\t")
				if err != nil {
					return nil, err
				}
				if err := os.WriteFile(configFile, cfgJson, 0644); err != nil {
					return nil, err
				}

				dealCfgPath(cfg)

				return cfg, nil
			}
			return nil, err
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		cfg = loadFromDB()
	}

	dealCfgPath(cfg)

	return cfg, nil
}

func loadFromDB() *Config {
	cfg := &Config{}
	agents, err := store.List[*types.AgentConfig](constant.StorePrefixAgent)
	if err != nil {
		return nil
	}
	cfg.Agents = make(map[string]*types.AgentConfig)
	for _, agent := range agents {
		cfg.Agents[agent.AgentName] = agent
	}

	providers, err := store.List[*types.ProviderConfig](constant.StorePrefixProvider)
	if err != nil {
		return nil
	}
	cfg.Providers = make(map[string]*types.ProviderConfig)
	for _, provider := range providers {
		cfg.Providers[provider.ProviderName] = provider
	}

	channels, err := store.List[*types.ChannelConfig](constant.StorePrefixChannel)
	if err != nil {
		return nil
	}
	cfg.Channels = make(map[string]*types.ChannelConfig)
	for _, channel := range channels {
		cfg.Channels[channel.ChannelName] = channel
	}

	skill, err := store.Get[*types.SkillConfig](constant.StoreSkill, constant.SkillID)
	if err != nil {
		skill = &types.SkillConfig{
			ID:         constant.SkillID,
			GlobalPath: "./skills",
		}
		err = nil
	}
	cfg.Skill = *skill

	browser, err := store.Get[*types.BrowserConfig](constant.StoreBrowser, constant.BrowserID)
	if err != nil {
		browser = &types.BrowserConfig{
			ID:      constant.BrowserID,
			DataDir: "./browser",
		}
		err = nil
	}
	cfg.Browser = *browser

	live2d, err := store.Get[*types.Live2DConfig](constant.StoreLive2D, constant.Live2DID)
	if err != nil {
		live2d = &types.Live2DConfig{
			ID:       constant.Live2DID,
			Enabled:  false,
			ModelDir: "./live2d",
			Width:    200,
			Height:   380,
		}
		err = nil
	}
	cfg.Live2D = *live2d

	mcpServers, err := store.List[*types.McpConfig](constant.StorePrefixMcpServer)
	if err != nil {
		return nil
	}
	cfg.McpServers = make(map[string]*types.McpConfig)
	for _, mcp := range mcpServers {
		cfg.McpServers[mcp.Name] = mcp
	}

	return cfg
}

func dealCfgPath(cfg *Config) {
	for _, agent := range cfg.Agents {
		if agent == nil {
			continue
		}
		agent.Workspace = utils.ExpandPath(agent.Workspace)
	}

	if cfg.Skill.GlobalPath != "" {
		cfg.Skill.GlobalPath = utils.ExpandPath(cfg.Skill.GlobalPath)
	}

	if cfg.Browser.DataDir != "" {
		cfg.Browser.DataDir = utils.ExpandPath(cfg.Browser.DataDir)
	}

	if cfg.Live2D.ModelDir != "" {
		cfg.Live2D.ModelDir = utils.ExpandPath(cfg.Live2D.ModelDir)
	}
}

func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errors []string

	if errs := c.validateAgents(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := c.validateProviders(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := c.validateChannels(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := c.validateReferences(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证失败，发现 %d 个问题：\n%s",
			len(errors),
			strings.Join(errors, "\n"))
	}

	return nil
}

func (c *Config) ValidateLive2D() error {
	if c.Live2D.Enabled {
		dir := c.Live2D.ModelDir
		modelPath := filepath.Join(dir, c.Live2D.ModelName)
		// 检查是否存在 .model3.json结尾的文件
		entries, err := os.ReadDir(modelPath)
		if err != nil {
			return fmt.Errorf("读取模型目录 %s 失败: %w", modelPath, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".model.json") || strings.HasSuffix(name, ".model3.json") {
				return nil
			}
		}
		return fmt.Errorf("模型目录 %s 中不存在 model.json 或 model3.json 文件", modelPath)
	}
	return fmt.Errorf("Live2D 未配置")
}

// validateAgents 验证Agent配置，返回所有错误
func (c *Config) validateAgents() []string {
	var errors []string

	for agentName, agent := range c.Agents {
		if agent.Workspace == "" {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 缺少工作空间配置（请添加 \"workspace\": \"/path/to/workspace\"）", agentName))
		}

		if agent.Provider == "" {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 缺少Provider配置（请添加 \"provider\": \"provider名称\"）", agentName))
		}

		if agent.Model == "" {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 缺少模型配置（请添加 \"model\": \"模型名称\"）", agentName))
		}

		if agent.Temperature < 0 || agent.Temperature > 2 {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 的Temperature值 %.2f 超出合理范围（应为 0-2）", agentName, agent.Temperature))
		}
	}

	return errors
}

// validateProviders 验证Provider配置，返回所有错误
// 验证所有Agent引用的Provider
func (c *Config) validateProviders() []string {
	var errors []string

	usedProviders := make(map[string]bool)
	for agentName, agent := range c.Agents {
		if agent.Provider == "" {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 缺少Provider配置（请添加 \"provider\": \"provider名称\"）", agentName))
			continue
		}
		usedProviders[agent.Provider] = true
	}

	if len(usedProviders) == 0 {
		errors = append(errors, "  - 未配置任何被使用的Provider")
	}

	for providerName, provider := range c.Providers {
		if !usedProviders[providerName] {
			// 未被任何智能体引用的Provider不需要验证
			continue
		}

		if provider.Type == "" {
			errors = append(errors, fmt.Sprintf("  - Provider '%s' 缺少类型配置（请添加 \"type\": \"openai/anthropic/ollama\"）", providerName))
		}

		if provider.Type != "ollama" && provider.APIKey == "" {
			errors = append(errors, fmt.Sprintf("  - Provider '%s' 缺少API密钥（请添加 \"api_key\": \"your-api-key\"）", providerName))
		}

		if provider.BaseURL != "" && !strings.HasPrefix(provider.BaseURL, "http://") && !strings.HasPrefix(provider.BaseURL, "https://") {
			errors = append(errors, fmt.Sprintf("  - Provider '%s' 的BaseURL格式错误（应以 http:// 或 https:// 开头）", providerName))
		}
	}

	return errors
}

// validateChannels 验证Channel配置，返回所有错误
func (c *Config) validateChannels() []string {
	var errors []string

	for name, ch := range c.Channels {
		if !ch.Enabled {
			continue
		}

		if ch.Type == "" {
			errors = append(errors, fmt.Sprintf("  - 渠道 '%s' 缺少类型配置（请添加 \"type\": \"feishu/web\"）", name))
		}

		if ch.Agent == "" {
			errors = append(errors, fmt.Sprintf("  - 渠道 '%s' 未指定绑定的智能体（请添加 \"agent\": \"agent名称\"）", name))
		}

		switch ch.Type {
		case "web":
			if ch.HostAddress == "" {
				errors = append(errors, fmt.Sprintf("  - Web渠道 '%s' 缺少主机地址配置（请添加 \"host_address\": \"host:port\"，例如 \"localhost:8080\"）", name))
			}
			if ch.Token == "" {
				errors = append(errors, fmt.Sprintf("  - Web渠道 '%s' 缺少访问令牌配置（请添加 \"token\": \"your-secret-token\"）", name))
			}
		case "feishu":
			if ch.AppID == "" {
				errors = append(errors, fmt.Sprintf("  - 飞书渠道 '%s' 缺少AppID配置（请添加 \"app_id\": \"your-app-id\"）", name))
			}
			if ch.AppSecret == "" {
				errors = append(errors, fmt.Sprintf("  - 飞书渠道 '%s' 缺少AppSecret配置（请添加 \"app_secret\": \"your-app-secret\"）", name))
			}
		case "wechat_ilink":
			// 微信 iLink 渠道无需验证
		default:
			errors = append(errors, fmt.Sprintf("  - 渠道 '%s' 的类型 '%s' 不支持（目前仅支持：feishu、web）", name, ch.Type))
		}
	}

	return errors
}

// validateReferences 验证配置之间的引用关系，返回所有错误
func (c *Config) validateReferences() []string {
	var errors []string

	for agentName, agent := range c.Agents {
		if _, exists := c.Providers[agent.Provider]; !exists {
			errors = append(errors, fmt.Sprintf("  - 智能体 '%s' 引用的Provider '%s' 不存在（请在providers中添加该配置）", agentName, agent.Provider))
		}
	}

	for channelName, channel := range c.Channels {
		if !channel.Enabled {
			continue
		}
		if _, exists := c.Agents[channel.Agent]; !exists {
			errors = append(errors, fmt.Sprintf("  - 渠道 '%s' 引用的智能体 '%s' 不存在（请在agents中添加该配置）", channelName, channel.Agent))
		}
	}

	return errors
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config) error {
	for name, mcp := range cfg.McpServers {
		mcp.Name = name
		store.Save(constant.StorePrefixMcpServer, mcp)
	}

	for name, agent := range cfg.Agents {
		agent.AgentName = name
		agent.Workspace = utils.ExpandPath(agent.Workspace)
		store.Save(constant.StorePrefixAgent, agent)
	}

	for name, provider := range cfg.Providers {
		provider.ProviderName = name
		store.Save(constant.StorePrefixProvider, provider)
	}

	for name, channel := range cfg.Channels {
		channel.ChannelName = name
		store.Save(constant.StorePrefixChannel, channel)
	}

	cfg.Skill.ID = constant.SkillID
	cfg.Skill.GlobalPath = utils.ExpandPath(cfg.Skill.GlobalPath)
	store.Save(constant.StoreSkill, &cfg.Skill)

	cfg.Browser.ID = constant.BrowserID
	cfg.Browser.DataDir = utils.ExpandPath(cfg.Browser.DataDir)
	store.Save(constant.StoreBrowser, &cfg.Browser)

	cfg.Live2D.ID = constant.Live2DID
	cfg.Live2D.ModelDir = utils.ExpandPath(cfg.Live2D.ModelDir)
	store.Save(constant.StoreLive2D, &cfg.Live2D)

	return nil

	// data, err := json.MarshalIndent(cfg, "", "  ")
	// if err != nil {
	// 	return err
	// }

	// // Create config directory if needed
	// dir := filepath.Dir(configFile)
	// if err := os.MkdirAll(dir, 0755); err != nil {
	// 	return err
	// }

	// return os.WriteFile(configFile, data, 0644)
}

// UpdateAgents updates agents configuration with lock protection
func (c *Config) UpdateAgents(agents map[string]*types.AgentConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, agent := range agents {
		if agent != nil {
			agent.Workspace = utils.ExpandPath(agent.Workspace)
		}
		c.Agents[name] = agent
	}
}

// UpdateProviders updates providers configuration with lock protection
func (c *Config) UpdateProviders(providers map[string]*types.ProviderConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, provider := range providers {
		c.Providers[name] = provider
	}
}

// UpdateChannels updates channels configuration with lock protection
func (c *Config) UpdateChannels(channels map[string]*types.ChannelConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, channel := range channels {
		c.Channels[name] = channel
	}
}

// UpdateSkill updates skill configuration with lock protection
func (c *Config) UpdateSkill(skill types.SkillConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if skill.GlobalPath != "" {
		skill.GlobalPath = utils.ExpandPath(skill.GlobalPath)
	}
	c.Skill = skill
}

// UpdateBrowser updates browser configuration with lock protection
func (c *Config) UpdateBrowser(browser types.BrowserConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if browser.DataDir != "" {
		browser.DataDir = utils.ExpandPath(browser.DataDir)
	}
	c.Browser = browser
}

// SetAgent sets a single agent configuration with lock protection
func (c *Config) SetAgent(name string, agent *types.AgentConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if agent != nil {
		agent.Workspace = utils.ExpandPath(agent.Workspace)
	}
	if c.Agents == nil {
		c.Agents = make(map[string]*types.AgentConfig)
	}
	c.Agents[name] = agent
}

// SetProvider sets a single provider configuration with lock protection
func (c *Config) SetProvider(name string, provider *types.ProviderConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Providers == nil {
		c.Providers = make(map[string]*types.ProviderConfig)
	}
	c.Providers[name] = provider
}

// SetChannel sets a single channel configuration with lock protection
func (c *Config) SetChannel(name string, channel *types.ChannelConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Channels == nil {
		c.Channels = make(map[string]*types.ChannelConfig)
	}
	c.Channels[name] = channel
}

// DeleteAgent deletes an agent configuration with lock protection
func (c *Config) DeleteAgent(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Agents, name)
}

// DeleteProvider deletes a provider configuration with lock protection
func (c *Config) DeleteProvider(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Providers, name)
}

// DeleteChannel deletes a channel configuration with lock protection
func (c *Config) DeleteChannel(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Channels, name)
}

func EnsureWorkspace(workspaceDir string, noloop ...bool) error {
	// 确保workspace目录存在
	if _, err := os.Stat(workspaceDir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(workspaceDir, 0755); err != nil {
				return fmt.Errorf("Failed to create workspace directory: %w", err)
			}
			if len(noloop) == 0 || !noloop[0] {
				return EnsureWorkspace(workspaceDir, true)
			}
			return nil
		}
		return err
	}
	return fs.WalkDir(embeddedFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil // 根目录跳过
		}
		if path == "workspace" {
			return nil // workspace目录跳过
		}

		relPath := path
		if strings.HasPrefix(path, "workspace/") {
			relPath = path[len("workspace/"):]
		}

		if relPath == "profile/BOOTSTRAP.md" {
			targetLockFile := filepath.Join(workspaceDir, constant.DirProfile, constant.ProfileLockBootstrap)
			if _, err := os.Stat(targetLockFile); err == nil {
				return nil // lock文件存在，跳过
			}
		}

		targetPath := filepath.Join(workspaceDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 复制文件
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)

		return os.WriteFile(targetPath, data, 0644)
	})
}

func ReleaseSkills() error {
	skillsDir := DefaultCfg.Skill.GlobalPath
	skillsDir = utils.ExpandPath(skillsDir)

	return fs.WalkDir(embeddedSkills, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil // 根目录跳过
		}
		if path == "skills" {
			return nil // skills目录跳过
		}

		relPath := path
		if strings.HasPrefix(path, "skills/") {
			relPath = path[len("skills/"):]
		}

		targetPath := filepath.Join(skillsDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 如果已存在，跳过
		if _, err = os.Stat(targetPath); err == nil {
			return nil
		}

		// 复制文件
		data, err := embeddedSkills.ReadFile(path)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)

		return os.WriteFile(targetPath, data, 0644)
	})
}

func ReleaseLive2dModels() error {
	if !DefaultCfg.Live2D.Enabled {
		return nil
	}
	modelsDir := DefaultCfg.Live2D.ModelDir
	modelsDir = utils.ExpandPath(modelsDir)

	return fs.WalkDir(embeddedLive2DModels, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil // 根目录跳过
		}
		if path == "live2d_models" {
			return nil // live2d_models目录跳过
		}

		relPath := path
		if strings.HasPrefix(path, "live2d_models/") {
			relPath = path[len("live2d_models/"):]
		}

		targetPath := filepath.Join(modelsDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 如果已存在，跳过
		if _, err = os.Stat(targetPath); err == nil {
			return nil
		}

		// 复制文件
		data, err := embeddedLive2DModels.ReadFile(path)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)

		return os.WriteFile(targetPath, data, 0644)
	})
}
