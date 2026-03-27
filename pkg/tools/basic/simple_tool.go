package basic

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/pkg/tools/types"
)

// SimpleTool is a helper for creating simple tools
type SimpleTool struct {
	Name_    string
	Desc_    string
	Params_  map[string]any
	ExecFunc func(ctx context.Context, params map[string]string) *types.ToolResult
}

// Name returns the tool name
func (t *SimpleTool) Name() string {
	return t.Name_
}

// Description returns the tool description
func (t *SimpleTool) Description() string {
	return t.Desc_
}

// Parameters returns the tool parameters schema
func (t *SimpleTool) Parameters() map[string]any {
	return t.Params_
}

// Execute runs the tool
func (t *SimpleTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	if t.ExecFunc == nil {
		return types.NewToolResult().WithError(fmt.Errorf("exec func is required"))
	}
	return t.ExecFunc(ctx, params)
}
