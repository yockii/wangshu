package llm

import (
	"context"
	"sync"
)

type Provider interface {
	Chat(ctx context.Context, model string, message []Message, tools []ToolDefinition, options map[string]any) (*ChatResponse, error)
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
