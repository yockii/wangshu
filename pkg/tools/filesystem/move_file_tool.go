package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type MoveFileTool struct {
	basic.SimpleTool
}

func NewMoveFileTool() *MoveFileTool {
	tool := new(MoveFileTool)
	tool.Name_ = constant.ToolNameFSMove
	tool.Desc_ = "MOVE a file or directory. Returns success message. It can also used for RENAME file or directory. Note: If the new_path already exists, it will be overwritten."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"old_path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file or directory to move",
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
func (t *MoveFileTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	oldPath := params["old_path"]
	newPath := params["new_path"]
	if oldPath == "" || newPath == "" {
		return types.NewToolResult().WithError(fmt.Errorf("old_path and new_path are required"))
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

	// Move file or directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to move: %w", err))
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("Successfully moved %s to %s", oldPath, newPath))
}
