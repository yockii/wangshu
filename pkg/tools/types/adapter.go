package types

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

	result := a.Tool.Execute(ctx, params)

	return result
}
