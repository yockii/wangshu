package filesystem

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
	"github.com/yockii/wangshu/pkg/utils"
)

type DeleteFileTool struct {
	basic.SimpleTool
}

func NewDeleteFileTool() *DeleteFileTool {
	tool := new(DeleteFileTool)
	tool.Name_ = constant.ToolNameFSDelete
	tool.Desc_ = "DELETE a file or directory. Returns success message."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file or directory to delete",
			},
		},
		"required": []string{"path"},
	}
	return tool
}

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	path := params["path"]
	if path == "" {
		return types.NewToolResult().WithError(fmt.Errorf("path is required"))
	}
	path = utils.ExpandPath(path)

	// 只允许workspace下的文件
	workspace := params[constant.ToolCallParamWorkspace]
	if workspace == "" {
		return types.NewToolResult().WithError(fmt.Errorf("workspace is required"))
	}
	if !strings.HasPrefix(path, workspace) {
		return types.NewToolResult().WithError(fmt.Errorf("path must be under workspace"))
	}

	// 删除文件或目录
	if err := os.RemoveAll(path); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to delete %s: %w", path, err))
	}
	return types.NewToolResult().WithRaw(fmt.Sprintf("✅ Successfully deleted %s", path)).WithStructured(
		actiontypes.NewDeleteFileData(path),
	)
}
