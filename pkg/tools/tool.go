package tools

import "context"

// Tool is the basic tool interface
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, params map[string]string) (string, error)
}

// AsyncCallback is called when an async tool completes
type AsyncCallback func(ctx context.Context, result *ToolResult)

// AsyncTool is a tool that executes asynchronously
type AsyncTool interface {
	SetCallback(cb AsyncCallback)
}

// ExtendedTool extends Tool with richer execution
type ExtendedTool interface {
	Tool
	ExecuteExtended(ctx context.Context, args map[string]any) *ToolResult
}

// ContextualTool receives full execution context including agent info
// This is the recommended interface for new tools that need LLM access
type ContextualTool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	ExecuteWithContext(ctx context.Context, params map[string]string, toolCtx *ToolContext) *ToolResult
}
