package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/app"
	"github.com/yockii/wangshu/internal/bundle"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/runner"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// application.RegisterEvent[string]("time")
	application.RegisterEvent[bus.Message](app.EventMessage)
}

func initConfig() {
	cfgPath := "./config.json"
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

func main() {
	if constant.Version == "dev" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	initConfig()

	defaultAgent, err := runner.Initialize()
	if err != nil {
		slog.Error("Initialization failed", "error", err)
		return
	}

	runner.InitializeChannels(defaultAgent)

	runner.FlagFileCheck()

	app.InitializeApp(
		assets,
		application.NewService(&bundle.WindowBundle{}),
		application.NewService(&bundle.ConfigBundle{}),
		application.NewService(bundle.NewChatBundle(defaultAgent)),
	)

	app.ShowChatWindow()

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
