package location

import (
	"context"

	"github.com/yockii/wangshu/internal/variable"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type LocationTool struct {
	basic.SimpleTool
}

func NewLocationTool() *LocationTool {
	tool := new(LocationTool)
	tool.Name_ = constant.ToolNameLocation
	tool.Desc_ = "获取当前地理位置信息, 可能为空"
	tool.Params_ = map[string]any{}
	tool.ExecFunc = func(ctx context.Context, params map[string]string) *types.ToolResult {
		if variable.Geolocation != "" {
			return types.NewToolResult().WithRaw(variable.Geolocation)
		}
		return types.NewToolResult().WithRaw("")
	}
	return tool
}
