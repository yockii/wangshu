package task

import (
	"github.com/yockii/wangshu/pkg/tools"
)

// RegisterTaskTools registers all task-related tools
func RegisterTaskTools() {
	tools.GetDefaultToolRegistry().Register(NewTaskTool())
}
