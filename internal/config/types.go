package config

import (
	"sync"

	"github.com/yockii/wangshu/pkg/constant"
)

type Config struct {
	Agents    map[string]*AgentConfig    `json:"agents"`
	Providers map[string]*ProviderConfig `json:"providers"`
	Channels  map[string]*ChannelConfig  `json:"channels"`
	Skill     SkillConfig                `json:"skill"`
	mu        sync.RWMutex
}

type SkillConfig struct {
	GlobalPath string `json:"global_path"`
}

type AgentConfig struct {
	Workspace              string  `json:"workspace"`
	Provider               string  `json:"provider"`
	Model                  string  `json:"model"`
	Temperature            float64 `json:"temperature"`
	MaxTokens              int64   `json:"max_tokens"`
	EnableImageRecognition bool    `json:"enable_image_recognition"`
	// 每日0点或配置的时间进行记忆整理
	MemoryOrganizeTime string `json:"memory_orgnaize_time"`
}

type ProviderConfig struct {
	Type    string `json:"type"` // openai/anthropic/ollama/...
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
}

type ChannelConfig struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Agent   string `json:"agent"`
	// feishu
	AppID     string `json:"app_id,omitempty"`
	AppSecret string `json:"app_secret,omitempty"`
	// web
	HostAddress string `json:"host_address,omitempty"`
	Token       string `json:"token,omitempty"`
}

func defaultConfig() *Config {
	return &Config{
		Agents: map[string]*AgentConfig{
			constant.Default: {
				Workspace:              "~/.wangshu/workspace",
				Provider:               "myProvider",
				Model:                  "qwen3-max",
				Temperature:            0.7,
				EnableImageRecognition: false,
				MemoryOrganizeTime:     "00:00",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:    "openai",
				APIKey:  "sk-your-openai-api-key",
				BaseURL: "your custom base url, blank if use openai official",
			},
		},
		Channels: map[string]*ChannelConfig{
			"feishuTest": {
				Type:      "feishu",
				Enabled:   false,
				Agent:     constant.Default,
				AppID:     "your feishu app id",
				AppSecret: "your feishu app secret",
			},
			"webTest": {
				Type:        "web",
				Enabled:     false,
				Agent:       constant.Default,
				HostAddress: "localhost:8080",
				Token:       "custom defined token",
			},
		},
		Skill: SkillConfig{
			GlobalPath: "~/.wangshu/skills",
		},
	}
}
