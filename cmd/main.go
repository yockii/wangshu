package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/runner"
	"github.com/yockii/wangshu/internal/tui"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

func main() {
	if constant.Version == "dev" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	initConfig()

	if utils.IsInteractiveTerminal() {
		tui.RunTui()
		return
	}
	runner.Run()

	// 阻塞命令行
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func initConfig() {
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
}
