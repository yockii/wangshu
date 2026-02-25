package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/yockii/yoclaw/pkg/llm"
)

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

func (r *Registry) ExecuteExtended(ctx context.Context, name string, args map[string]interface{}, channel, chatID string) *ToolResult {
	tool, ok := r.Get(name)
	if !ok {
		return ErrorResult(fmt.Sprintf("tool %s not found", name))
	}

	if contextSetter, ok := tool.(ContextSetter); ok {
		contextSetter.SetContext(channel, chatID)
	}

	if extTool, ok := tool.(ExtendedTool); ok {
		return extTool.ExecuteExtended(ctx, args)
	}

	return ExtendedToolAdapter{Tool: tool}.ExecuteExtended(ctx, args)
}
