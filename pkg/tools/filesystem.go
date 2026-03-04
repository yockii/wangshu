package tools

import (
	"github.com/yockii/wangshu/pkg/tools/filesystem"
)

func RegisterFileSystemTools() {
	defaultToolRegistry.Register(filesystem.NewReadFileTool())
	defaultToolRegistry.Register(filesystem.NewWriteFileTool())
	defaultToolRegistry.Register(filesystem.NewListDirectoryTool())
	defaultToolRegistry.Register(filesystem.NewRenameFileTool())
	defaultToolRegistry.Register(filesystem.NewEditFileTool())
	defaultToolRegistry.Register(filesystem.NewFindFileTool())
	defaultToolRegistry.Register(filesystem.NewGrepTool())
}
