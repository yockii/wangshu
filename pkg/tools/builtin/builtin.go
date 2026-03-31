package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

// SleepTool pauses execution for a specified duration
type SleepTool struct{}

func (t *SleepTool) Name() string {
	return constant.ToolNameSleep
}

func (t *SleepTool) Description() string {
	return "Pauses execution for a specified number of seconds. Useful for testing delays."
}

func (t *SleepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"seconds": map[string]any{
				"type":        "number",
				"description": "Number of seconds to sleep",
			},
		},
		"required": []string{"seconds"},
	}
}

func (t *SleepTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	secondsStr, ok := params["seconds"]
	if !ok {
		return types.NewToolResult().WithError(fmt.Errorf("missing required parameter: seconds"))
	}

	var seconds float64
	n, err := fmt.Sscanf(secondsStr, "%f", &seconds)
	if err != nil || n != 1 {
		return types.NewToolResult().WithError(fmt.Errorf("invalid seconds value: %w", err))
	}

	select {
	case <-ctx.Done():
		return types.NewToolResult().WithError(ctx.Err())
	case <-time.After(time.Duration(seconds*1000) * time.Millisecond):
		return types.NewToolResult().WithRaw(fmt.Sprintf("Slept for %v seconds", seconds))
	}
}

// GetTimeTool returns the current time
type GetTimeTool struct{}

func (t *GetTimeTool) Name() string {
	return constant.ToolNameCurrentTime
}

func (t *GetTimeTool) Description() string {
	return "Returns the current time in ISO 8601 format. Useful for timestamping or time-aware operations."
}

func (t *GetTimeTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *GetTimeTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	now := time.Now()
	return types.NewToolResult().WithRaw(now.Format(time.DateTime)).WithStructured(actiontypes.NewTimeNowData(now.Format(time.DateTime)))
}
