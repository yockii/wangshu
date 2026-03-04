package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type ListDirectoryTool struct {
	basic.SimpleTool
}

func NewListDirectoryTool() *ListDirectoryTool {
	tool := new(ListDirectoryTool)
	tool.Name_ = "list_directory"
	tool.Desc_ = "List files and directories in a directory. Returns a list of file names."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the directory to list",
			},
		},
		"required": []string{"path"},
	}
	return tool
}

func (t *ListDirectoryTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	path := params["path"]
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var result strings.Builder
	fmt.Fprintf(&result, "Contents of %s:\n", path)
	for _, entry := range entries {
		info, _ := entry.Info()
		if info.IsDir() {
			fmt.Fprintf(&result, "[DIR]  %s\n", entry.Name())
		} else {
			fmt.Fprintf(&result, "[FILE] %s (%d bytes)\n", entry.Name(), info.Size())
		}
	}

	return result.String(), nil
}
