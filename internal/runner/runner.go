package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yockii/wangshu/internal/agent"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/tools/configtool"
	"github.com/yockii/wangshu/internal/tools/message"
	"github.com/yockii/wangshu/internal/tools/system"
	"github.com/yockii/wangshu/internal/tools/task"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
	"github.com/yockii/wangshu/pkg/channel/feishu"
	"github.com/yockii/wangshu/pkg/channel/web"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/llm/claude"
	"github.com/yockii/wangshu/pkg/llm/ollama"
	"github.com/yockii/wangshu/pkg/llm/openai"
	"github.com/yockii/wangshu/pkg/skills"
	"github.com/yockii/wangshu/pkg/tools"
	"github.com/yockii/wangshu/pkg/tools/browser"
	"github.com/yockii/wangshu/pkg/tools/builtin"
	"github.com/yockii/wangshu/pkg/tools/filesystem"
	"github.com/yockii/wangshu/pkg/tools/memory"
	"github.com/yockii/wangshu/pkg/tools/network"
	"github.com/yockii/wangshu/pkg/tools/runtime"
	"github.com/yockii/wangshu/pkg/utils"
)

var defaultAgent *agent.Agent

func Initialize(isTUIMode bool) (*agent.Agent, error) {
	if err := config.DefaultCfg.ValidateWithMode(isTUIMode); err != nil {
		return nil, err
	}

	config.ReleaseSkills()

	usedProviders := make(map[string]bool)
	for _, agent := range config.DefaultCfg.Agents {
		if agent.Provider == "" {
			continue
		}
		usedProviders[agent.Provider] = true
	}

	providerCount := 0
	for providerName, providerCfg := range config.DefaultCfg.Providers {
		if !usedProviders[providerName] {
			continue
		}

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
		case "ollama":
			ollamaProvider := ollama.NewProvider(providerCfg.BaseURL)
			llm.RegisterProvider(providerName, ollamaProvider)
			providerCount++
		default:
			slog.Error("Unsupported LLM provider type", "type", providerCfg.Type)
		}
	}

	if providerCount == 0 {
		slog.Error("No LLM provider configured")
		return nil, fmt.Errorf("no LLM provider configured")
	}

	bus.Default().Start(context.Background())

	tools.GetDefaultToolRegistry().Register(&builtin.SleepTool{})
	tools.GetDefaultToolRegistry().Register(&builtin.GetTimeTool{})

	tools.GetDefaultToolRegistry().Register(filesystem.NewReadFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewWriteFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewListDirectoryTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewMoveFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewCopyFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewEditFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewFindFileTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewGrepTool())
	tools.GetDefaultToolRegistry().Register(filesystem.NewDeleteFileTool())

	tools.GetDefaultToolRegistry().Register(network.NewWebSearchTool())
	tools.GetDefaultToolRegistry().Register(network.NewWebFetchTool())

	tools.GetDefaultToolRegistry().Register(memory.NewMemoryTool())

	tools.GetDefaultToolRegistry().Register(runtime.NewPythonRunTool())
	tools.GetDefaultToolRegistry().Register(runtime.NewNodeRunTool())
	tools.GetDefaultToolRegistry().Register(runtime.NewNpmRunTool())
	tools.GetDefaultToolRegistry().Register(runtime.NewGitRunTool())
	tools.GetDefaultToolRegistry().Register(task.NewTaskTool())
	tools.GetDefaultToolRegistry().Register(task.NewCronTool())
	tools.GetDefaultToolRegistry().Register(message.NewMessageTool())
	tools.GetDefaultToolRegistry().Register(system.NewVersionTool())
	tools.GetDefaultToolRegistry().Register(system.NewVariableTool())
	configtool.SetReloadFunc(Reload)
	tools.GetDefaultToolRegistry().Register(configtool.NewConfigTool())
	tools.GetDefaultToolRegistry().Register(browser.NewBrowserTool())

	skills.InitializeSkillLoader()

	defaultAgent = agent.InitializeAgentManager(isTUIMode)

	return defaultAgent, nil
}

func InitializeChannels(defaultAgent *agent.Agent) bool {
	noChannelFound := true
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

					var feishuAgent *agent.Agent
					if ch.Agent != "" {
						a, has := agent.GetAgent(ch.Agent)
						if has {
							feishuAgent = a
							feishuChannel.SetWorkspace(a.GetWorkspace())
						}
					}
					if feishuAgent == nil {
						feishuAgent = defaultAgent
						feishuChannel.SetWorkspace(defaultAgent.GetWorkspace())
					}

					channel.RegisterChannel(name, feishuChannel)
					bus.Default().RegisterInboundHandler(name, feishuAgent.SubscribeInbound)
					bus.Default().RegisterOutboundHandler(feishuChannel.SubscribeOutbound)
				}
			}
		}
	}
	return noChannelFound
}

func Run() {
	defaultAgent, err := Initialize(false)
	if err != nil {
		slog.Error("Initialization failed", "error", err)
		return
	}

	noChannelFound := InitializeChannels(defaultAgent)
	if noChannelFound {
		slog.Error("No channel configured")
		return
	}

	flagFileCheck()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	channel.StopAllChannel()
	agent.StopAllAgents()
	slog.Info("All agents stopped")
}

func flagFileCheck() {
	exePath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path", "error", err)
		return
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		slog.Error("Failed to resolve symlinks", "error", err)
		return
	}

	restartFlagPath := filepath.Join(filepath.Dir(exePath), ".restart_flag")

	if _, err := os.Stat(restartFlagPath); os.IsNotExist(err) {
		return
	}

	flagData, err := os.ReadFile(restartFlagPath)
	if err != nil {
		slog.Error("Failed to read restart flag", "error", err)
		return
	}

	if err := os.Remove(restartFlagPath); err != nil {
		slog.Error("Failed to remove restart flag", "error", err)
	}

	parts := strings.Split(string(flagData), "|")
	if len(parts) != 4 {
		slog.Error("Invalid restart flag data", "data", string(flagData))
		return
	}

	agentName := parts[0]
	channelName := parts[1]
	chatID := parts[2]
	senderID := parts[3]

	slog.Info("Restart detected", "agent", agentName, "channel", channelName, "chatID", chatID, "senderID", senderID)

	ag, has := agent.GetAgent(agentName)
	if !has {
		slog.Error("Agent not found", "agent", agentName)
		return
	}

	err = ag.RestartMessage(context.Background(), bus.InboundMessage{
		Message: bus.Message{
			Metadata: bus.MessageMetadata{
				Channel:  channelName,
				ChatID:   chatID,
				SenderID: senderID,
			},
		},
	})

	if err != nil {
		slog.Error("Restart notification error", "error", err)
	}
}

func GetDefaultAgent() *agent.Agent {
	return defaultAgent
}

func Reload() error {
	cfgPath := "~/.wangshu/config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfgPath = utils.ExpandPath(cfgPath)

	newCfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load new configuration: %w", err)
	}

	if err := newCfg.Validate(); err != nil {
		return fmt.Errorf("new configuration is invalid: %w", err)
	}

	_, isTUIMode := channel.GetChannel(constant.TUIChannelName)

	if isTUIMode {
		channel.ClearChannelsExcept([]string{constant.TUIChannelName})
		bus.Default().ClearHandlersExcept([]string{constant.TUIChannelName})
	} else {
		channel.ClearChannels()
		bus.Default().ClearHandlers()
	}
	agent.ClearAgents()

	llm.ClearProviders()

	config.DefaultCfg = newCfg

	if err := initializeProviders(); err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	defaultAgent = agent.InitializeAgentManager(isTUIMode)

	noChannelFound := InitializeChannels(defaultAgent)
	if noChannelFound && !isTUIMode {
		slog.Warn("No channel configured after reload")
	}

	slog.Info("Configuration reloaded successfully")
	return nil
}

func initializeProviders() error {
	usedProviders := make(map[string]bool)
	for _, agent := range config.DefaultCfg.Agents {
		if agent.Provider == "" {
			continue
		}
		usedProviders[agent.Provider] = true
	}

	providerCount := 0
	for providerName, providerCfg := range config.DefaultCfg.Providers {
		if !usedProviders[providerName] {
			continue
		}

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
		case "ollama":
			ollamaProvider := ollama.NewProvider(providerCfg.BaseURL)
			llm.RegisterProvider(providerName, ollamaProvider)
			providerCount++
		default:
			slog.Error("Unsupported LLM provider type", "type", providerCfg.Type)
		}
	}

	if providerCount == 0 {
		return fmt.Errorf("no LLM provider configured")
	}

	return nil
}
