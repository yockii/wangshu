package tools

import (
	"github.com/yockii/yoclaw/pkg/tools/filesystem"
)

func RegisterFileSystemTools(registry *Registry) {
	registry.Register(filesystem.NewReadFileTool())
	registry.Register(filesystem.NewWriteFileTool())
	registry.Register(filesystem.NewListDirectoryTool())
	registry.Register(filesystem.NewRenameFileTool())
	registry.Register(filesystem.NewEditFileTool())
	registry.Register(filesystem.NewFindFileTool())
	registry.Register(filesystem.NewGrepTool())
}
