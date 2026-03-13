package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

var DefaultCfg *Config

func Initialize(cfgFilePath string) error {
	cfg, err := LoadConfig(cfgFilePath)
	if err != nil {
		return err
	}

	DefaultCfg = cfg
	return nil
}

func LoadConfig(cfgFilePath string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(cfgFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// // 引导用户在控制台上填写内容
			// leadUserToFillConfig(cfg)
			// 写入文件
			err = os.MkdirAll(filepath.Dir(cfgFilePath), 0755)
			if err != nil {
				return nil, err
			}
			cfgJson, err := json.MarshalIndent(cfg, "", "\t")
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(cfgFilePath, cfgJson, 0644); err != nil {
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

	dealCfgPath(cfg)

	return cfg, nil
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
}

func (c *Config) Validate() error {
	return c.ValidateWithMode(false)
}

func (c *Config) ValidateWithMode(isTUIMode bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errors []string

	if errs := c.validateAgents(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := c.validateProviders(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := c.validateChannels(isTUIMode); len(errs) > 0 {
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
func (c *Config) validateChannels(isTUIMode bool) []string {
	var errors []string

	hasChannel := false

	for name, ch := range c.Channels {
		if !ch.Enabled {
			continue
		}
		hasChannel = true

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
		default:
			errors = append(errors, fmt.Sprintf("  - 渠道 '%s' 的类型 '%s' 不支持（目前仅支持：feishu、web）", name, ch.Type))
		}
	}

	if !hasChannel && !isTUIMode {
		errors = append(errors, "  - 未配置任何启用的渠道（请在channels中添加至少一个渠道配置）")
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
func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Create config directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

//go:embed workspace
var embeddedFiles embed.FS

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

//go:embed skills
var embeddedSkills embed.FS

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
