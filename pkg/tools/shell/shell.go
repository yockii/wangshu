package shell

import (
	"github.com/yockii/yoclaw/pkg/tools"
)

// RegisterShellTools registers all shell-related tools
func RegisterShellTools(registry *tools.Registry) {
	registry.Register(NewExecTool())
	registry.Register(NewProcessTool())
	registry.Register(NewAutoInteractiveTool())
}
