package tui

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/utils"
)

func runConfigWizard() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		slog.Error("配置向导运行出错", "error", err)
		return err
	}

	cfgPath := "~/.wangshu/config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfgPath = utils.ExpandPath(cfgPath)
	if err = config.SaveConfig(cfgPath, config.DefaultCfg); err != nil {
		slog.Error("配置向导保存配置出错", "error", err)
		return err
	}

	return nil
}
