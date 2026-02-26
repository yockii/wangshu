package tasks

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterTaskTools registers all task-related tools
func RegisterTaskTools(registry *tools.Registry) {
	registry.Register(NewTaskTool())
}
