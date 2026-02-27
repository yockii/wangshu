package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/yockii/yoclaw/internal/agent"
	"github.com/yockii/yoclaw/internal/config"
	systemTools "github.com/yockii/yoclaw/internal/tools/system"
	taskTools "github.com/yockii/yoclaw/internal/tools/task"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/channel"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/skills"
	"github.com/yockii/yoclaw/pkg/tools"
	memoryTools "github.com/yockii/yoclaw/pkg/tools/memory"
	networkTools "github.com/yockii/yoclaw/pkg/tools/network"
	shellTools "github.com/yockii/yoclaw/pkg/tools/shell"
)

func main() {
	cfgPath := "~/.yoClaw/config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfgPath = expandPath(cfgPath)

	err := config.Initialize(cfgPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	if err = config.DefaultCfg.Validate(); err != nil {
		slog.Error("Config validation failed", "error", err)
		return
	}

	// 初始化大模型
	providerCount := 0
	for providerName, providerCfg := range config.DefaultCfg.Providers {
		if providerCfg.Type == "" {
			slog.Error("LLM provider type is empty", "provider", providerName)
			continue
		}
		if providerCfg.APIKey == "" && providerCfg.Type != "ollama" {
			slog.Error("LLM provider API key is empty", "provider", providerName)
			continue
		}
		switch providerCfg.Type {
		case "openai":
			llm.RegisterProvider(providerName, llm.NewOpenAIProvider(providerCfg.APIKey, providerCfg.BaseURL))
			providerCount++
		// case "ollama":
		// 	llm.RegisterProvider(providerName, llm.NewOllamaProvider(providerCfg.APIKey, providerCfg.BaseURL))
		default:
			slog.Error("Unsupported LLM provider type", "type", providerCfg.Type)
		}
	}

	if providerCount == 0 {
		slog.Error("No LLM provider configured")
		return
	}

	bus.Default().Start(context.Background())
	defer bus.Close()

	// 初始化工具注册中心
	tools.RegisterBuiltinTools()
	tools.RegisterFileSystemTools()
	// Register shell tools
	shellTools.RegisterShellTools()
	// Register network tools
	networkTools.RegisterNetworkTools()
	// Register system tools
	systemTools.RegisterSystemTools()
	// Register memory tools
	memoryTools.RegisterMemoryTools()
	taskTools.RegisterTaskTools()
	// TODO 实现并注册更多工具

	skills.InitializeSkillLoader(config.DefaultCfg.Skill.GlobalPath, config.DefaultCfg.Skill.BuiltInPath)

	// 初始化agents
	defaultAgent := agent.InitializeAgentManager()

	// 初始化channel
	if config.DefaultCfg.Channels.Feishu.Enabled {
		if config.DefaultCfg.Channels.Feishu.AppID != "" && config.DefaultCfg.Channels.Feishu.AppSecret != "" {
			feishuChannel := channel.NewFeishuChannel("feishu", config.DefaultCfg.Channels.Feishu.AppID, config.DefaultCfg.Channels.Feishu.AppSecret)
			channel.RegisterChannel("feishu", feishuChannel)
			var feishuAgent *agent.Agent
			if config.DefaultCfg.Channels.Feishu.Agent != "" {
				a, has := agent.GetAgent(config.DefaultCfg.Channels.Feishu.Agent)
				if has {
					feishuAgent = a
				}
			}
			if feishuAgent == nil {
				feishuAgent = defaultAgent
			}
			bus.Default().RegisterInboundHandler(feishuAgent.SubscribeInbound)
			bus.Default().RegisterOutboundHandler(feishuChannel.SubscribeOutbound)
		} else {
			slog.Warn("Feishu channel enabled but appId or appSecret not configured")
		}
	} else {
		slog.Error("No channel configured")
		return
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	channel.StopAllChannel()
	// Stop all agent
	agent.StopAllAgents()
	slog.Info("All agents stopped")
}

// expandPath expands ~ to user's home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		// Handle both / and \ as path separators
		if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
			return filepath.Join(home, path[2:])
		}
		return home
	}
	return path
}
