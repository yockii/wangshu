package tools

import (
	"context"
	"fmt"
)

type ExtendedToolAdapter struct {
	Tool
}

func (a ExtendedToolAdapter) ExecuteExtended(ctx context.Context, args map[string]any) *ToolResult {
	params := make(map[string]string)
	for k, v := range args {
		params[k] = fmt.Sprintf("%v", v)
	}

	result, err := a.Tool.Execute(ctx, params)
	if err != nil {
		return ErrorResult(result).WithError(err)
	}

	return &ToolResult{ForLLM: result, ForUser: result}
}
