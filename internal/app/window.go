package app

import (
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/variable"
	"github.com/yockii/wangshu/pkg/constant"
)

var (
	windowLocker sync.Mutex
	chatWindow   *application.WebviewWindow
	configWindow *application.WebviewWindow
	live2dWindow *application.WebviewWindow
	qrcodeWindow *application.WebviewWindow
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
				// HiddenOnTaskbar:                   true,
				ExStyle: getLive2DWindowExStyle(),
			},
			Frameless:         true,
			BackgroundColour:  application.NewRGBA(0, 0, 0, 0),
			BackgroundType:    application.BackgroundTypeTranslucent,
			URL:               "#/live2d",
			Width:             width,
			Height:            height,
			MinWidth:          50,
			MinHeight:         80,
			DisableResize:     true,
			AlwaysOnTop:       true,
			IgnoreMouseEvents: true,
			X:                 config.DefaultCfg.Live2D.X,
			Y:                 config.DefaultCfg.Live2D.Y,
			InitialPosition:   application.WindowXY,
		})

	}
	// 重载 live2d 页面
	live2dWindow.Reload()
	live2dWindow.Show()
	variable.Live2DVisible = true
	RebuildTrayMenu()
}

func HideLive2DWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if live2dWindow != nil {
		live2dWindow.Hide()
		variable.Live2DVisible = false
		RebuildTrayMenu()
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

	variable.Live2DEditMode = true
	SetLive2DIgnoresMouseEvents(false)
	live2dWindow.SetResizable(true)
	app.Event.Emit(constant.EventLive2DEditMode, true)
}

func ExitLive2DEditMode() {
	variable.Live2DEditMode = false
	SetLive2DIgnoresMouseEvents(true)
	live2dWindow.SetResizable(false)
	// 获取当前窗口 xy坐标
	x, y := live2dWindow.Position()
	// 保存当前窗口 xy坐标
	config.DefaultCfg.Live2D.X = x
	config.DefaultCfg.Live2D.Y = y

	// 当前窗口大小
	width, height := live2dWindow.Size()
	config.DefaultCfg.Live2D.Width = width
	config.DefaultCfg.Live2D.Height = height

	// 保存当前窗口 xy坐标
	config.SaveConfig(config.DefaultCfg)

	RebuildTrayMenu()

	app.Event.Emit(constant.EventLive2DEditMode, false)
}

func ToggleLive2DEditMode() {
	if variable.Live2DEditMode {
		ExitLive2DEditMode()
	} else {
		EnterLive2DEditMode()
	}
}

func IsLive2DEditMode() bool {
	return variable.Live2DEditMode
}

func ShowQRCodeWindow(qrURL string) {
	windowLocker.Lock()
	defer windowLocker.Unlock()

	if qrcodeWindow == nil {
		qrcodeWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title: "微信登录 - 扫码授权",
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 50,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			Windows: application.WindowsWindow{
				DisableFramelessWindowDecorations: true,
			},
			Frameless:        false,
			BackgroundColour: application.NewRGBA(255, 255, 255, 255),
			BackgroundType:   application.BackgroundTypeSolid,
			URL:              "#/qrcode",
			Width:            320,
			Height:           400,
			MinWidth:         320,
			MinHeight:        400,
			MaxWidth:         320,
			MaxHeight:        400,
			DisableResize:    true,
			AlwaysOnTop:      true,
			InitialPosition:  application.WindowCenter,
		})
	}

	qrcodeWindow.Show()
	qrcodeWindow.Focus()

	app.Event.Emit(constant.EventQrcodeUpdate, qrURL)
}

func HideQRCodeWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if qrcodeWindow != nil {
		qrcodeWindow.Hide()
	}
}

func UpdateQRCodeStatus(status string) {
	// app.Event.Emit("qrcode-status", status)
}

func CloseQRCodeWindow() {
	windowLocker.Lock()
	defer windowLocker.Unlock()
	if qrcodeWindow != nil {
		qrcodeWindow.Close()
		qrcodeWindow = nil
	}
}
