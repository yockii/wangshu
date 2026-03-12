package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type RenameFileTool struct {
	basic.SimpleTool
}

func NewRenameFileTool() *RenameFileTool {
	tool := new(RenameFileTool)
	tool.Name_ = "rename_file"
	tool.Desc_ = "Rename a file or directory. Returns success message. It can also used for MOVE file or directory. Note: If the new_path already exists, it will be overwritten."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"old_path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file or directory to rename",
			},
			"new_path": map[string]any{
				"type":        "string",
				"description": "The new absolute or relative path for the file or directory",
			},
		},
		"required": []string{"old_path", "new_path"},
	}
	return tool
}
func (t *RenameFileTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	oldPath := params["old_path"]
	newPath := params["new_path"]
	if oldPath == "" || newPath == "" {
		return "", fmt.Errorf("old_path and new_path are required")
	}
	// Expand ~ to home directory
	if strings.HasPrefix(oldPath, "~/") {
		home, _ := os.UserHomeDir()
		oldPath = filepath.Join(home, oldPath[2:])
	}
	if strings.HasPrefix(newPath, "~/") {
		home, _ := os.UserHomeDir()
		newPath = filepath.Join(home, newPath[2:])
	}

	// 允许覆盖，使用rename默认行为
	// // 检查new_path是否已存在，已存在报错
	// if _, err := os.Stat(newPath); err == nil {
	// 	return "", fmt.Errorf("new_path %s already exists", newPath)
	// }

	// Rename file or directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename: %w", err)
	}

	return fmt.Sprintf("Successfully renamed %s to %s", oldPath, newPath), nil
}
