package bundle

import (
	"fmt"

	"github.com/yockii/wangshu/internal/app"
	"github.com/yockii/wangshu/internal/variable"
)

type WindowBundle struct{}

func (w *WindowBundle) UpdateGeoLocation(latitude, longitude, altitude, accuracy, altitudeAccuracy, heading, speed string) {
	variable.Geolocation = fmt.Sprintf("latitude: %s, longitude: %s, altitude: %s, accuracy: %s, altitudeAccuracy: %s, heading: %s, speed: %s", latitude, longitude, altitude, accuracy, altitudeAccuracy, heading, speed)
}

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

func (w *WindowBundle) ShowEmotionMappingWindow() {
	app.ShowEmotionMappingWindow()
}

func (w *WindowBundle) HideEmotionMappingWindow() {
	app.HideEmotionMappingWindow()
}
