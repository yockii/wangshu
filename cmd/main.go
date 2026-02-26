package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yockii/yoclaw/internal/agent"
	"github.com/yockii/yoclaw/internal/config"
	"github.com/yockii/yoclaw/internal/constant"
	"github.com/yockii/yoclaw/internal/tasks"
	systemTools "github.com/yockii/yoclaw/internal/tools/system"
	taskTools "github.com/yockii/yoclaw/internal/tools/tasks"
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

	// 初始化大模型
	if config.DefaultCfg.Providers.OpenAI.APIKey != "" {
		llm.RegisterProvider("openai", llm.NewOpenAIProvider(config.DefaultCfg.Providers.OpenAI.APIKey, config.DefaultCfg.Providers.OpenAI.BaseURL))
	} else {
		slog.Error("No LLM provider configured")
		return
	}

	bus.Default().Start(context.Background())
	defer bus.Close()

	// 初始化工具注册中心
	toolsRegistry := tools.NewRegistry()
	tools.RegisterBuiltinTools(toolsRegistry)
	tools.RegisterFileSystemTools(toolsRegistry)
	// Register shell tools
	shellTools.RegisterShellTools(toolsRegistry)
	// Register network tools
	networkTools.RegisterNetworkTools(toolsRegistry)
	// Register system tools
	systemTools.RegisterSystemTools(toolsRegistry)
	// Register memory tools
	memoryTools.RegisterMemoryTools(toolsRegistry)
	// Register task tools
	taskTools.RegisterTaskTools(toolsRegistry)
	// TODO 实现并注册更多工具

	// 确保各个agent的workspace完整性
	for name, agent := range config.DefaultCfg.Agents {
		if err := config.EnsureWorkspace(agent.Workspace); err != nil {
			slog.Error("Failed to ensure workspace", "agent", name, "error", err)
			return
		}
	}

	skillLoader := skills.NewLoader(config.DefaultCfg.Skill.GlobalPath, config.DefaultCfg.Skill.BuiltInPath)

	// 初始化agents
	agents := make(map[string]*agent.Agent)
	var defaultAgent *agent.Agent
	for name, ac := range config.DefaultCfg.Agents {
		agents[name] = agent.NewAgent(
			llm.GetProvider(ac.Provider),
			ac.Model,
			toolsRegistry,
			24*time.Hour,
			10,
			ac.Workspace,
			skillLoader,
		)
		// Set agent name and start cron manager
		agents[name].SetName(name)
		if name == constant.Default || defaultAgent == nil {
			defaultAgent = agents[name]
		}
	}

	// Initialize global agent manager
	agent.InitializeAgentManager(agents)

	// Initialize task manager (uses global agents)
	_, err = tasks.Initialize()
	if err != nil {
		slog.Error("Failed to initialize task manager", "error", err)
		return
	}

	// 初始化channel
	if config.DefaultCfg.Channels.Feishu.Enabled {
		if config.DefaultCfg.Channels.Feishu.AppID != "" && config.DefaultCfg.Channels.Feishu.AppSecret != "" {
			feishuChannel := channel.NewFeishuChannel("feishu", config.DefaultCfg.Channels.Feishu.AppID, config.DefaultCfg.Channels.Feishu.AppSecret)
			channel.RegisterChannel("feishu", feishuChannel)
			var feishuAgent *agent.Agent
			if config.DefaultCfg.Channels.Feishu.Agent != "" {
				a, has := agents[config.DefaultCfg.Channels.Feishu.Agent]
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

	// Stop all cron managers
	for name, ag := range agents {
		slog.Info("Stopping agent", "name", name)
		ag.Stop()
	}
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
