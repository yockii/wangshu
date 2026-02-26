package memory

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterMemoryTools registers all memory-related tools
func RegisterMemoryTools() {
	tools.GetDefaultToolRegistry().Register(NewMemoryTool())
}
