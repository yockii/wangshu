package app

import (
	"embed"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/config"
)

var (
	app     *application.App
	once    sync.Once
	systray *application.SystemTray

	iconBytes []byte
)

func GetApp() *application.App {
	return app
}

func Run() error {
	if app == nil {
		return fmt.Errorf("app not initialized")
	}
	return app.Run()
}

func InitializeApp(assets embed.FS, services ...application.Service) {
	once.Do(func() {
		app = application.New(application.Options{
			Name:        "望舒",
			Description: "你的智能个人终端助理",
			Services:    services,
			Assets: application.AssetOptions{
				Handler:    application.AssetFileServerFS(assets),
				Middleware: localModelMiddleware,
			},
			Mac: application.MacOptions{
				ApplicationShouldTerminateAfterLastWindowClosed: true,
			},
			Server: application.ServerOptions{
				Host:        "localhost",
				Port:        8080,
				ReadTimeout: 300 * time.Second,
				IdleTimeout: 720 * time.Second,
			},
			// 单例运行
			SingleInstance: &application.SingleInstanceOptions{
				UniqueID: "com.xhnic.wangshu",
				OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
					ShowChatWindow()
				},
			},
		})

		iconBytes, _ = assets.ReadFile("frontend/dist/tray_icon.png")
		buildSystemTray()
	})
}

func buildSystemTray() {
	if systray == nil {
		systray = app.SystemTray.New()
		systray.SetLabel("望舒 - 个人AI助理")
		systray.SetIcon(iconBytes).SetTooltip("望舒")
		systray.OnClick(func() {
			ShowChatWindow()
		})
	}

	trayMenu := app.NewMenu()
	trayMenu.Add("打开聊天窗口").OnClick(func(ctx *application.Context) {
		ShowChatWindow()
	})
	trayMenu.Add("配置").OnClick(func(ctx *application.Context) {
		ShowConfigWindow()
	})

	if config.DefaultCfg.Live2D.Enabled {
		if Live2DVisible {
			if Live2DEditMode {
				trayMenu.Add("退出编辑").OnClick(func(ctx *application.Context) {
					ExitLive2DEditMode()
					RebuildTrayMenu()
				})
			} else {
				trayMenu.Add("隐藏桌宠").OnClick(func(ctx *application.Context) {
					HideLive2DWindow()
				})
				trayMenu.Add("编辑桌宠").OnClick(func(ctx *application.Context) {
					EnterLive2DEditMode()
					RebuildTrayMenu()
				})
			}
		} else {
			trayMenu.Add("桌宠").OnClick(func(ctx *application.Context) {
				ShowLive2DWindow()
			})
		}
	}

	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	systray.SetMenu(trayMenu)
}

func RebuildTrayMenu() {
	buildSystemTray()
}
