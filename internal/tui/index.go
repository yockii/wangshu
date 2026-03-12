package tui

import (
	"log/slog"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/runner"
)

func RunTui() {
	if err := config.DefaultCfg.Validate(); err != nil {
		// 配置缺失，启动配置向导
		err = runConfigWizard()
		if err != nil {
			slog.Error("配置向导执行失败", "error", err)
			return
		}
		slog.Info("配置向导执行完成")
	}

	runner.Run()
}
