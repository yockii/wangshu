package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
	"github.com/yockii/yoclaw/pkg/utils"
)

func main() {
	run()

	// 阻塞命令行
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func run() {
	cfgPath := "~/.yoClaw/config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfgPath = utils.ExpandPath(cfgPath)

	err := config.Initialize(cfgPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	if err = config.DefaultCfg.Validate(); err != nil {
		slog.Error("Config validation failed", "error", err)
		return
	}

	// 释放技能
	config.ReleaseSkills()

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

	skills.InitializeSkillLoader()

	// 初始化agents
	defaultAgent := agent.InitializeAgentManager()

	noChannelFound := true
	// 初始化channel
	for name, ch := range config.DefaultCfg.Channels {
		if ch.Enabled {
			switch ch.Type {
			case "web":
				if ch.HostAddress != "" && ch.Token != "" {
					noChannelFound = false
					webChannel := channel.NewWebChannel(name, ch.HostAddress, ch.Token)
					channel.RegisterChannel(name, webChannel)
					var webAgent *agent.Agent
					if ch.Agent != "" {
						a, has := agent.GetAgent(ch.Agent)
						if has {
							webAgent = a
						}
					}
					if webAgent == nil {
						webAgent = defaultAgent
					}
					bus.Default().RegisterInboundHandler(name, webAgent.SubscribeInbound)
					bus.Default().RegisterOutboundHandler(webChannel.SubscribeOutbound)
				} else {
					slog.Warn("Web channel enabled but hostAddress or token not configured")
				}
			case "feishu":
				if ch.AppID != "" && ch.AppSecret != "" {
					noChannelFound = false
					feishuChannel := channel.NewFeishuChannel(name, ch.AppID, ch.AppSecret)
					channel.RegisterChannel(name, feishuChannel)
					var feishuAgent *agent.Agent
					if ch.Agent != "" {
						a, has := agent.GetAgent(ch.Agent)
						if has {
							feishuAgent = a
						}
					}
					if feishuAgent == nil {
						feishuAgent = defaultAgent
					}
					bus.Default().RegisterInboundHandler(name, feishuAgent.SubscribeInbound)
					bus.Default().RegisterOutboundHandler(feishuChannel.SubscribeOutbound)
				}
			}
		}
	}

	if noChannelFound {
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
