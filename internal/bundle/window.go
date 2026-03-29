package bundle

import "github.com/yockii/wangshu/internal/app"

type WindowBundle struct{}

func (w *WindowBundle) ShowChatWindow() {
	app.ShowChatWindow()
}

func (w *WindowBundle) HideChatWindow() {
	app.HideChatWindow()
}

func (w *WindowBundle) ShowConfigWindow() {
	app.ShowConfigWindow()
}

func (w *WindowBundle) HideConfigWindow() {
	app.HideConfigWindow()
}

func (w *WindowBundle) ShowLive2DWindow() {
	app.ShowLive2DWindow()
}

func (w *WindowBundle) HideLive2DWindow() {
	app.HideLive2DWindow()
}
