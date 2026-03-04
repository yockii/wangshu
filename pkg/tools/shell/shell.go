package shell

import "github.com/yockii/wangshu/pkg/tools"

// RegisterShellTools registers all shell-related tools
func RegisterShellTools() {
	tools.GetDefaultToolRegistry().Register(NewExecTool())
	tools.GetDefaultToolRegistry().Register(NewProcessTool())
	tools.GetDefaultToolRegistry().Register(NewAutoInteractiveTool())
}
