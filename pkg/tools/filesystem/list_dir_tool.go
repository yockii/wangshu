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
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

type ListDirectoryTool struct {
	basic.SimpleTool
}

func NewListDirectoryTool() *ListDirectoryTool {
	tool := new(ListDirectoryTool)
	tool.Name_ = constant.ToolNameFSList
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

func (t *ListDirectoryTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	path := params["path"]
	if path == "" {
		return types.NewToolResult().WithError(fmt.Errorf("path is required"))
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read directory: %w", err))
	}

	var result []struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}

	for _, entry := range entries {
		info, _ := entry.Info()
		if info.IsDir() {
			result = append(result, struct {
				Name  string `json:"name"`
				IsDir bool   `json:"is_dir"`
				Size  int64  `json:"size"`
			}{
				Name:  entry.Name(),
				IsDir: true,
				Size:  0,
			})
		} else {
			result = append(result, struct {
				Name  string `json:"name"`
				IsDir bool   `json:"is_dir"`
				Size  int64  `json:"size"`
			}{
				Name:  entry.Name(),
				IsDir: false,
				Size:  info.Size(),
			})
		}
	}

	var raw strings.Builder
	fmt.Fprintf(&raw, "Contents of %s:\n", path)
	for _, item := range result {
		fmt.Fprintf(&raw, "[%s] %s", item.Name, item.Name)
		if item.IsDir {
			fmt.Fprintf(&raw, " (DIR)")
		}
		fmt.Fprintf(&raw, "\n")
	}

	return types.NewToolResult().WithRaw(raw.String()).WithStructured(actiontypes.NewFsListData(path, result))
}
