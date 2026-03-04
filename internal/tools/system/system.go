package system

import (
	"github.com/yockii/wangshu/pkg/tools"
)

// RegisterSystemTools registers all system-related tools
func RegisterSystemTools() {
	tools.GetDefaultToolRegistry().Register(NewCronTool())
}
