package app

import (
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/config"
)

var (
	windowLocker sync.Mutex
	chatWindow   *application.WebviewWindow
	configWindow *application.WebviewWindow
	live2dWindow *application.WebviewWindow
)

func ShowChatWindow() {
	if config.DefaultCfg.Validate() != nil {
		slog.Error("Configuration validation failed")
		ShowConfigWindow()
		return
	}
	windowLocker.Lock()
	defer windowLocker.Unlock()

	if chatWindow == nil {
		chatWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title: "望舒 - 个人AI助理",
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 50,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			Windows: application.WindowsWindow{
				DisableFramelessWindowDecorations: true,
			},
			Frameless:        true,
			BackgroundColour: application.NewRGBA(0, 0, 0, 0),
			BackgroundType:   application.BackgroundTypeTranslucent,
			URL:              "#/chat",
		})
		// chatWindow.SnapAssist()
	}
	chatWindow.Show()
	chatWindow.Focus()
}

func HideChatWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if chatWindow != nil {
		chatWindow.Hide()
	}
}

func ShowConfigWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if configWindow == nil {
		configWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title: "望舒 - 个人AI助理 - 配置",
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 50,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			Windows: application.WindowsWindow{
				DisableFramelessWindowDecorations: true,
			},
			Frameless:        true,
			BackgroundColour: application.NewRGBA(0, 0, 0, 0),
			BackgroundType:   application.BackgroundTypeTranslucent,
			URL:              "#/config",
		})
		// configWindow.SnapAssist()
	}
	configWindow.Show()
	configWindow.Focus()
}

func HideConfigWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if configWindow != nil {
		configWindow.Hide()
	}
}

func ShowLive2DWindow() {
	if config.DefaultCfg.ValidateLive2D() != nil {
		slog.Error("Live2D configuration validation failed")
		app.Dialog.Warning().SetTitle("警告").SetMessage("Live2D 配置无效，请检查配置文件").Show()
		return
	}

	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow == nil {
		width := 200
		height := 380
		if config.DefaultCfg.Live2D.Width > 0 {
			width = config.DefaultCfg.Live2D.Width
		}
		if config.DefaultCfg.Live2D.Height > 0 {
			height = config.DefaultCfg.Live2D.Height
		}
		live2dWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title: "望舒 - 个人AI助理 - 2D",
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 50,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			Windows: application.WindowsWindow{
				DisableFramelessWindowDecorations: true,
			},
			Frameless:         true,
			BackgroundColour:  application.NewRGBA(0, 0, 0, 0),
			BackgroundType:    application.BackgroundTypeTranslucent,
			URL:               "#/live2d",
			Width:             width,
			Height:            height,
			DisableResize:     true,
			AlwaysOnTop:       true,
			IgnoreMouseEvents: true,
			X:                 config.DefaultCfg.Live2D.X,
			Y:                 config.DefaultCfg.Live2D.Y,
		})

	}
	live2dWindow.Show()
	Live2DVisible = true
	rebuildTrayMenu()
}

func HideLive2DWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.Hide()
		Live2DVisible = false
		rebuildTrayMenu()
	}
}

func ChangeLive2DWindowSize(width, height int) {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.SetSize(width, height)
	}
}

func SetLive2DIgnoresMouseEvents(ignore bool) {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.SetIgnoreMouseEvents(ignore)
	}
}

func EnterLive2DEditMode() {
	if config.DefaultCfg.ValidateLive2D() != nil {
		return
	}

	Live2DEditMode = true
	SetLive2DIgnoresMouseEvents(false)
	live2dWindow.SetResizable(true)
	app.Event.Emit("live2d-edit-mode", true)
}

func ExitLive2DEditMode() {
	Live2DEditMode = false
	SetLive2DIgnoresMouseEvents(true)
	live2dWindow.SetResizable(false)
	// 获取当前窗口 xy坐标
	x, y := live2dWindow.Position()
	// 保存当前窗口 xy坐标
	config.DefaultCfg.Live2D.X = x
	config.DefaultCfg.Live2D.Y = y
	// 保存当前窗口 xy坐标
	config.SaveConfig(config.DefaultCfg)

	app.Event.Emit("live2d-edit-mode", false)
}

func ToggleLive2DEditMode() {
	if Live2DEditMode {
		ExitLive2DEditMode()
	} else {
		EnterLive2DEditMode()
	}
}

func IsLive2DEditMode() bool {
	return Live2DEditMode
}
