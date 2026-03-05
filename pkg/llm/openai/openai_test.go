package openai

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/llm"
	selfConstant "github.com/yockii/wangshu/pkg/constant"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-key", "https://api.openai.com/v1")
	if provider == nil {
		t.Error("NewProvider should return a non-nil provider")
	}
}

func TestProviderChat(t *testing.T) {
	provider := NewProvider("test-key", "https://api.openai.com/v1")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    selfConstant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    selfConstant.RoleUser,
			Content: "Hello!",
		},
	}

	// 由于没有真实的API key，预期会返回错误
	_, err := provider.Chat(ctx, "gpt-4o-mini", messages, nil, nil)
	if err == nil {
		t.Log("Chat succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderConvertMessages(t *testing.T) {
	provider := NewProvider("test-key", "https://api.openai.com/v1")

	messages := []llm.Message{
		{
			Role:    selfConstant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    selfConstant.RoleUser,
			Content: "Hello!",
		},
		{
			Role:      selfConstant.RoleAssistant,
			Content:   "Hi there!",
			ToolCalls: []llm.ToolCall{},
		},
	}

	converted := provider.convertMessages(messages)
	if len(converted) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(converted))
	}
}

func TestProviderConvertTools(t *testing.T) {
	provider := NewProvider("test-key", "https://api.openai.com/v1")

	tools := []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"param1": map[string]any{
							"type":        "string",
							"description": "A parameter",
						},
					},
				},
			},
		},
	}

	converted := provider.convertTools(tools)
	if len(converted) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(converted))
	}
}
