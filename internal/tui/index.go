package tui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yockii/wangshu/internal/agent"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/runner"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
	"github.com/yockii/wangshu/pkg/logger"
)

func RunTui() {
	cleanup, stdoutWriter := logger.Setup()
	defer cleanup()

	if err := config.DefaultCfg.ValidateWithMode(true); err != nil {
		err = runConfigWizard(stdoutWriter)
		if err != nil {
			slog.Error("配置向导执行失败", "error", err)
			return
		}
		slog.Info("配置向导执行完成")
	}

	defaultAgent, err := runner.Initialize(true)
	if err != nil {
		slog.Error("初始化失败", "error", err)
		return
	}

	if defaultAgent == nil {
		slog.Error("没有可用的 Agent")
		return
	}

	tuiChannel := NewTUIChannel()

	channel.RegisterChannel(TUIChannelName, tuiChannel)
	bus.Default().RegisterInboundHandler(TUIChannelName, defaultAgent.SubscribeInbound)
	bus.Default().RegisterOutboundHandler(tuiChannel.SubscribeOutbound)

	runner.InitializeChannels(defaultAgent)

	p := tea.NewProgram(
		newRuntimeModel(tuiChannel, defaultAgent.GetName()),
		tea.WithAltScreen(),
		tea.WithOutput(stdoutWriter),
	)

	_, err = p.Run()
	if err != nil {
		slog.Error("运行时 TUI 错误", "error", err)
		return
	}

	bus.Default().Close()
	channel.StopAllChannel()
	agent.StopAllAgents()
}
