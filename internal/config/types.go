package config

import (
	"sync"

	"github.com/yockii/yoclaw/internal/constant"
)

type Config struct {
	Agents    map[string]AgentConfig `json:"agents"`
	Providers ProvidersConfig        `json:"providers"`
	Channels  ChannelsConfig         `json:"channels"`
	Skill     SkillConfig            `json:"skill"`
	mu        sync.RWMutex
}

type SkillConfig struct {
	GlobalPath  string `json:"global_path"`
	BuiltInPath string `json:"builtin_path"`
}

type AgentConfig struct {
	Workspace   string  `json:"workspace"`
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

type ProvidersConfig struct {
	OpenAI OpenAIProviderConfig `json:"openai"`
}

type OpenAIProviderConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
}

type ChannelsConfig struct {
	Feishu FeishuConfig `json:"feishu"`
}

type ChannelBaseConfig struct {
	Enabled bool   `json:"enabled"`
	Agent   string `json:"agent"`
}

type FeishuConfig struct {
	ChannelBaseConfig
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

func DefaultConfig() *Config {
	return &Config{
		Agents: map[string]AgentConfig{
			constant.Default: {
				Workspace:   "~/.yoClaw/workspace",
				Provider:    "openai",
				Model:       "qwen3-max",
				Temperature: 0.7,
			},
		},
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{
				APIKey: "",
			},
		},
		Channels: ChannelsConfig{
			Feishu: FeishuConfig{
				ChannelBaseConfig: ChannelBaseConfig{
					Enabled: true,
					Agent:   constant.Default,
				},
				AppID:     "",
				AppSecret: "",
			},
		},
		Skill: SkillConfig{
			GlobalPath:  "~/.yoClaw/skills",
			BuiltInPath: "./skills",
		},
	}
}
