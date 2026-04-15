package config

import (
	"sync"

	"github.com/jinzhu/copier"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
)

type Config struct {
	Agents     map[string]*types.AgentConfig    `json:"agents"`
	Providers  map[string]*types.ProviderConfig `json:"providers"`
	Channels   map[string]*types.ChannelConfig  `json:"channels"`
	Skill      types.SkillConfig                `json:"skill"`
	Browser    types.BrowserConfig              `json:"browser"`
	Live2D     types.Live2DConfig               `json:"live2d"`
	McpServers map[string]*types.McpConfig      `json:"mcp_servers"`

	mu sync.RWMutex
}

func defaultConfig() *Config {
	return &Config{
		Agents: map[string]*types.AgentConfig{
			constant.Default: {
				Workspace:              "./workspace",
				Provider:               "myProvider",
				Model:                  "qwen3-max",
				Temperature:            0.7,
				EnableImageRecognition: false,
				MemoryOrganizeTime:     "00:00",
			},
		},
		Providers: map[string]*types.ProviderConfig{
			"myProvider": {
				Type:    "openai",
				APIKey:  "",
				BaseURL: "",
			},
		},
		Channels: map[string]*types.ChannelConfig{},
		Skill: types.SkillConfig{
			GlobalPath: "./skills",
		},
		Browser: types.BrowserConfig{
			DataDir: "./browser_profile",
		},
		Live2D: types.Live2DConfig{
			Enabled:   true,
			ModelDir:  "./live2d_models",
			ModelName: "hiyori",
			Width:     200,
			Height:    380,
			X:         0,
			Y:         0,
		},
		McpServers: map[string]*types.McpConfig{},
	}
}

func (cfg *Config) Copy() *Config {
	// 复制一份自身配置信息
	cfgCopy := &Config{}
	copier.Copy(cfgCopy, cfg)
	return cfgCopy
}
