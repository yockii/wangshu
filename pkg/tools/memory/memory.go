package memory

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterMemoryTools registers all memory-related tools
func RegisterMemoryTools(registry *tools.Registry) {
	registry.Register(NewMemoryTool())
}
