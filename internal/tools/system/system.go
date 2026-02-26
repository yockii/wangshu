package system

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterSystemTools registers all system-related tools
func RegisterSystemTools() {
	tools.GetDefaultToolRegistry().Register(NewCronTool())
}
