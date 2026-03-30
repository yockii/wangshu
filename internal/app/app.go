package app

import (
	"embed"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var app *application.App
var once sync.Once

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
		})

		iconBytes, _ := assets.ReadFile("frontend/dist/tray_icon.png")
		buildSystemTray(iconBytes)
	})
}

func buildSystemTray(iconBytes []byte) {
	systray := app.SystemTray.New()
	systray.SetLabel("望舒 - 个人AI助理")
	systray.SetIcon(iconBytes).SetTooltip("望舒")

	trayMenu := app.NewMenu()
	trayMenu.Add("打开聊天窗口").OnClick(func(ctx *application.Context) {
		ShowChatWindow()
	})
	trayMenu.Add("配置").OnClick(func(ctx *application.Context) {
		ShowConfigWindow()
	})
	trayMenu.Add("桌宠").OnClick(func(ctx *application.Context) {
		ShowLive2DWindow()
	})
	trayMenu.Add("编辑桌宠").OnClick(func(ctx *application.Context) {
		ShowLive2DWindow()
		EnterLive2DEditMode()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	systray.SetMenu(trayMenu)

	systray.OnClick(func() {
		ShowChatWindow()
	})
}
