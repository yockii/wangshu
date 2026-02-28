package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Validate workspace path
	for agentName, agent := range c.Agents {
		if agent.Workspace == "" {
			return fmt.Errorf("workspace path is required for agent %s", agentName)
		}

		if agent.Provider == "" {
			return fmt.Errorf("provider is required for agent %s", agentName)
		}

		providerCfg := c.Providers[agent.Provider]
		if providerCfg.APIKey == "" && providerCfg.Type != "ollama" {
			return fmt.Errorf("provider '%s' requires API key", agent.Provider)
		}
	}

	// Validate channel config
	for name, ch := range c.Channels {
		if !ch.Enabled {
			continue
		}
		switch ch.Type {
		case "web":
			if ch.HostAddress == "" {
				return fmt.Errorf("%s host_address is required when enabled", name)
			}
			if ch.Token == "" {
				return fmt.Errorf("%s token is required when enabled", name)
			}
		case "feishu":
			if ch.AppID == "" {
				return fmt.Errorf("%s app_id is required when enabled", name)
			}
			if ch.AppSecret == "" {
				return fmt.Errorf("%s app_secret is required when enabled", name)
			}
		}
	}

	return nil
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

		if relPath == "BOOTSTRAP.md" {
			targetLockFile := filepath.Join(workspaceDir, "BOOTSTRAP.lock")
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

		return os.WriteFile(targetPath, data, 0644)
	})
}
