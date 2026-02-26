package network

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterNetworkTools registers all network-related tools
func RegisterNetworkTools() {
	tools.GetDefaultToolRegistry().Register(NewWebSearchTool())
	tools.GetDefaultToolRegistry().Register(NewWebFetchTool())
	tools.GetDefaultToolRegistry().Register(NewBrowserTool())
}
