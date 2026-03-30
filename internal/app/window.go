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
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow == nil {
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
			Frameless:        true,
			BackgroundColour: application.NewRGBA(0, 0, 0, 0),
			BackgroundType:   application.BackgroundTypeTranslucent,
			URL:              "/live2d",
			Width:            200,
			Height:           380,
			DisableResize:    true,
			AlwaysOnTop:      true,
		})
	}
	live2dWindow.Show()
}

func HideLive2DWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.Hide()
	}
}

func ChangeLive2DWindowSize(width, height int) {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.SetSize(width, height)
	}
}
