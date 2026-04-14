package main

import (
	"context"
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
	"github.com/yockii/wangshu/internal/store"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// application.RegisterEvent[string]("time")
	application.RegisterEvent[bus.Message](constant.EventMessage)

	application.RegisterEvent[bool](constant.EventLive2DEditMode)
}

func initConfig() {
	cfgPath := "./data"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfgPath = utils.ExpandPath(cfgPath)

	err := config.Initialize(cfgPath)
	if err != nil {
		slog.Error("Failed to load data", "error", err)
		return
	}
}

func main() {
	if err := store.Initialize(); err != nil {
		slog.Error("Failed to initialize store", "error", err)
		return
	}
	defer store.Shutdown()

	if constant.Version == "dev" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	initConfig()

	bus.Default().Start(context.Background())

	runner.RegisterTools()

	if err := config.DefaultCfg.Validate(); err == nil {
		defaultAgent, err := runner.Initialize()
		if err != nil {
			slog.Error("Initialization failed", "error", err)
		}

		runner.FlagFileCheck()

		bundle.DefaultChatBundle.SetAgent(defaultAgent)
	}

	app.InitializeApp(
		assets,
		application.NewService(&bundle.WindowBundle{}),
		application.NewService(&bundle.ConfigBundle{}),
		application.NewService(bundle.DefaultChatBundle),
		application.NewService(&bundle.DialogBundle{}),

		application.NewService(&bundle.Live2dBundle{}),
	)

	app.ShowChatWindow()
	if config.DefaultCfg.ValidateLive2D() == nil {
		app.ShowLive2DWindow()
	}

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
