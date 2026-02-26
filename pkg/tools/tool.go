package tools

import "context"

const (
	ToolCallParamWorkspace = "workspace"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, params map[string]string) (string, error)
}

type ContextSetter interface {
	SetContext(channel, chatID string)
}

type AsyncCallback func(ctx context.Context, result *ToolResult)

type AsyncTool interface {
	SetCallback(cb AsyncCallback)
}

type ExtendedTool interface {
	Tool
	ExecuteExtended(ctx context.Context, args map[string]any) *ToolResult
}
