package llm

import (
	"context"
	"sync"
)

type Provider interface {
	Chat(ctx context.Context, model string, message []Message, tools []ToolDefinition, jsonSchema *JSONSchema, options map[string]any) (*ChatResponse, error)

	// ChatWithJSONSchema sends a chat request with JSON schema for structured output.
	// The LLM will respond with JSON that conforms to the provided schema.
	// This is useful for getting structured responses like analysis results.
	ChatWithJSONSchema(ctx context.Context, model string, message []Message, jsonSchema *JSONSchema, options map[string]any) (*ChatResponse, error)
}

var providerMu sync.RWMutex
var Providers = make(map[string]Provider)

func RegisterProvider(name string, provider Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()

	Providers[name] = provider
}

func GetProvider(name string) Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return Providers[name]
}

func ClearProviders() {
	providerMu.Lock()
	defer providerMu.Unlock()
	Providers = make(map[string]Provider)
}
