package config

import (
	"sync"

	"github.com/jinzhu/copier"
	"github.com/yockii/wangshu/pkg/constant"
)

type Config struct {
	Agents     map[string]*AgentConfig    `json:"agents"`
	Providers  map[string]*ProviderConfig `json:"providers"`
	Channels   map[string]*ChannelConfig  `json:"channels"`
	Skill      SkillConfig                `json:"skill"`
	Browser    BrowserConfig              `json:"browser"`
	Live2D     Live2DConfig               `json:"live2d"`
	McpServers map[string]*McpConfig      `json:"mcp_servers"`

	mu sync.RWMutex
}

type McpConfig struct {
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	TransportType string            `json:"transport_type,omitempty"` // 通信协议，默认stdio，以后可能扩展到http、sse等
	URL           string            `json:"url,omitempty"`            // 通信地址，用于http、sse等
}

type SkillConfig struct {
	GlobalPath string `json:"global_path"`
}

type BrowserConfig struct {
	DataDir string `json:"data_dir"` // 浏览器用户数据目录，用于持久化cookies、登录状态等
}

type Live2DConfig struct {
	Enabled   bool   `json:"enabled"`
	ModelDir  string `json:"model_dir"`
	ModelName string `json:"model_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
}

type AgentConfig struct {
	Workspace              string  `json:"workspace"`
	Provider               string  `json:"provider"`
	Model                  string  `json:"model"`
	Temperature            float64 `json:"temperature"`
	MaxTokens              int64   `json:"max_tokens"`
	EnableImageRecognition bool    `json:"enable_image_recognition"`
	// 每日0点或配置的时间进行记忆整理
	MemoryOrganizeTime string `json:"memory_organize_time"`
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
				Workspace:              "./workspace",
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
				APIKey:  "",
				BaseURL: "",
			},
		},
		Channels: map[string]*ChannelConfig{},
		Skill: SkillConfig{
			GlobalPath: "./skills",
		},
		Browser: BrowserConfig{
			DataDir: "./browser_profile",
		},
		Live2D: Live2DConfig{
			Enabled:   true,
			ModelDir:  "./live2d_models",
			ModelName: "hiyori",
			Width:     200,
			Height:    380,
			X:         0,
			Y:         0,
		},
		McpServers: map[string]*McpConfig{},
	}
}

func (cfg *Config) Copy() *Config {
	// 复制一份自身配置信息
	cfgCopy := &Config{}
	copier.Copy(cfgCopy, cfg)
	return cfgCopy
}
