package network

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterNetworkTools registers all network-related tools
func RegisterNetworkTools(registry *tools.Registry) {
	registry.Register(NewWebSearchTool())
	registry.Register(NewWebFetchTool())
	registry.Register(NewBrowserTool())
}
