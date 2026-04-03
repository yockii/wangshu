//go:build windows

package app

import (
	"github.com/wailsapp/wails/v3/pkg/w32"
)

func getLive2DWindowExStyle() int {
	return w32.WS_EX_TOOLWINDOW | w32.WS_EX_NOREDIRECTIONBITMAP | w32.WS_EX_TOPMOST | w32.WS_EX_LAYERED
}
