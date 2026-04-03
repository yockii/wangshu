package bundle

import "github.com/yockii/wangshu/internal/app"

type DialogBundle struct {
}

func (*DialogBundle) Info(title, msg string) {
	app.GetApp().Dialog.Info().SetTitle(title).SetMessage(msg).Show()
}

func (*DialogBundle) Warning(title, msg string) {
	app.GetApp().Dialog.Warning().SetTitle(title).SetMessage(msg).Show()
}

func (*DialogBundle) Error(title, msg string) {
	app.GetApp().Dialog.Error().SetTitle(title).SetMessage(msg).Show()
}
