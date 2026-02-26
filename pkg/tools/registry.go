package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/yockii/yoclaw/pkg/llm"
)

var defaultToolRegistry = NewRegistry()

func GetDefaultToolRegistry() *Registry {
	return defaultToolRegistry
}

type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Execute(ctx context.Context, name string, params map[string]string) (string, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool %s not found", name)
	}
	return tool.Execute(ctx, params)
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]string, 0, len(r.tools))
	for name := range r.tools {
		tools = append(tools, name)
	}
	return tools
}

func (r *Registry) GetSummaries() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	summaries := make(map[string]string, len(r.tools))
	for name, tool := range r.tools {
		summaries[name] = tool.Description()
	}
	return summaries
}

func (r *Registry) GetProviderDefs() []llm.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return defs
}

// ExecuteExtended executes a tool with extended arguments
// Deprecated: Use ExecuteWithContext instead
func (r *Registry) ExecuteExtended(ctx context.Context, name string, args map[string]interface{}, channel, chatID string) *ToolResult {
	return r.ExecuteWithContext(ctx, name, args, nil, channel, chatID)
}

// ExecuteWithContext executes a tool with full tool context
func (r *Registry) ExecuteWithContext(ctx context.Context, name string, args map[string]interface{}, toolCtx *ToolContext, channel, chatID string) *ToolResult {
	tool, ok := r.Get(name)
	if !ok {
		return ErrorResult(fmt.Sprintf("tool %s not found", name))
	}

	// Try ContextualTool first (new interface)
	if ctxTool, ok := tool.(ContextualTool); ok {
		return ctxTool.ExecuteWithContext(ctx, argsToStringMap(args), toolCtx)
	}

	// Fall back to ExtendedTool (legacy interface)
	if extTool, ok := tool.(ExtendedTool); ok {
		return extTool.ExecuteExtended(ctx, args)
	}

	// Fall back to basic Tool
	return ExtendedToolAdapter{Tool: tool}.ExecuteExtended(ctx, args)
}

// argsToStringMap converts args to string map for legacy tools
func argsToStringMap(args map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range args {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
