package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yockii/wangshu/internal/agent"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/tools/message"
	systemTools "github.com/yockii/wangshu/internal/tools/system"
	taskTools "github.com/yockii/wangshu/internal/tools/task"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
	"github.com/yockii/wangshu/pkg/channel/feishu"
	"github.com/yockii/wangshu/pkg/channel/web"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/llm/claude"
	"github.com/yockii/wangshu/pkg/llm/openai"
	"github.com/yockii/wangshu/pkg/skills"
	"github.com/yockii/wangshu/pkg/tools"
	memoryTools "github.com/yockii/wangshu/pkg/tools/memory"
	networkTools "github.com/yockii/wangshu/pkg/tools/network"
	"github.com/yockii/wangshu/pkg/tools/runtime"
	"github.com/yockii/wangshu/pkg/utils"
)

func main() {
	run()

	// 阻塞命令行
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func run() {
	cfgPath := "~/.wangshu/config.json"
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
			openaiProvider := openai.NewProvider(providerCfg.APIKey, providerCfg.BaseURL)
			llm.RegisterProvider(providerName, openaiProvider)
			providerCount++
		case "anthropic":
			claudeProvider := claude.NewProvider(providerCfg.APIKey, providerCfg.BaseURL)
			llm.RegisterProvider(providerName, claudeProvider)
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
	// Register shell tools - disabled to encourage use of specialized tools
	// shellTools.RegisterShellTools()
	// Register network tools
	networkTools.RegisterNetworkTools()
	// Register memory tools
	memoryTools.RegisterMemoryTools()
	// Register runtime tools
	tools.GetDefaultToolRegistry().Register(runtime.NewPythonRunTool())
	tools.GetDefaultToolRegistry().Register(runtime.NewNpmRunTool())
	tools.GetDefaultToolRegistry().Register(runtime.NewGitRunTool())

	tools.GetDefaultToolRegistry().Register(taskTools.NewTaskTool())
	tools.GetDefaultToolRegistry().Register(systemTools.NewCronTool())
	tools.GetDefaultToolRegistry().Register(message.NewMessageTool())
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
					webChannel := web.NewWebChannel(name, ch.HostAddress, ch.Token)
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
					feishuChannel := feishu.NewFeishuChannel(name, ch.AppID, ch.AppSecret)

					// 设置 workspace（从关联的 agent 配置中获取）
					var feishuAgent *agent.Agent
					if ch.Agent != "" {
						a, has := agent.GetAgent(ch.Agent)
						if has {
							feishuAgent = a
							// 设置 workspace
							feishuChannel.SetWorkspace(a.GetWorkspace())
						}
					}
					if feishuAgent == nil {
						feishuAgent = defaultAgent
						// 使用 defaultAgent 的 workspace
						feishuChannel.SetWorkspace(defaultAgent.GetWorkspace())
					}

					channel.RegisterChannel(name, feishuChannel)
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
