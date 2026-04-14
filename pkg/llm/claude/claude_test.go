package claude

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")
	if provider == nil {
		t.Error("NewProvider should return a non-nil provider")
	}
}

func TestNewProviderWithBaseURL(t *testing.T) {
	provider := NewProvider("test-key", "https://custom.api.com")
	if provider == nil {
		t.Error("NewProvider with baseURL should return a non-nil provider")
	}
}

func TestNewProviderWithEmptyBaseURL(t *testing.T) {
	provider := NewProvider("test-key", "")
	if provider == nil {
		t.Error("NewProvider with empty baseURL should return a non-nil provider")
	}
}

func TestProviderChat(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	// 由于没有真实的API key，预期会返回错误
	_, err := provider.Chat(ctx, "claude-3-5-sonnet-20241022", messages, nil, nil)
	if err == nil {
		t.Log("Chat succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderChatWithTools(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "What's the weather?",
		},
	}

	tools := []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get current weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "City name",
						},
					},
				},
			},
		},
	}

	_, err := provider.Chat(ctx, "claude-3-5-sonnet-20241022", messages, tools, nil)
	if err == nil {
		t.Log("Chat with tools succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderConvertMessages(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
		{
			Role:      constant.RoleAssistant,
			Content:   "Hi there!",
			ToolCalls: []llm.ToolCall{},
		},
	}

	converted := provider.convertMessages(messages)
	// System消息应该被过滤掉，所以只返回2条消息
	if len(converted) != 2 {
		t.Errorf("Expected 2 messages (system filtered out), got %d", len(converted))
	}
}

func TestProviderConvertMessagesWithToolCalls(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "What's the weather?",
		},
		{
			Role:    constant.RoleAssistant,
			Content: "",
			ToolCalls: []llm.ToolCall{
				{
					ID:        "call_123",
					Name:      "get_weather",
					Arguments: `{"location":"Tokyo"}`,
				},
			},
		},
		{
			Role:      constant.RoleTool,
			Content:   `{"temperature":"20°C","condition":"sunny"}`,
			ToolCalls: []llm.ToolCall{{ID: "call_123"}},
		},
	}

	converted := provider.convertMessages(messages)
	// User, Assistant (with tool_use), and User (with tool_result) = 3 messages
	if len(converted) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(converted))
	}
}

func TestProviderConvertTools(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

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

func TestProviderExtractSystemMessage(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	systemMsg := provider.extractSystemMessage(messages)
	if systemMsg != "You are a helpful assistant." {
		t.Errorf("Expected system message 'You are a helpful assistant.', got '%s'", systemMsg)
	}
}

func TestProviderExtractSystemMessage_NoSystem(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	systemMsg := provider.extractSystemMessage(messages)
	if systemMsg != "" {
		t.Errorf("Expected empty system message, got '%s'", systemMsg)
	}
}

func TestProviderChatWithJSONSchema(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "Generate a user profile",
		},
	}

	jsonSchema := &llm.JSONSchema{
		Name:        "user_profile",
		Description: "A user profile",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "User's name",
				},
				"age": map[string]any{
					"type":        "integer",
					"description": "User's age",
				},
			},
			"required":             []string{"name", "age"},
			"additionalProperties": false,
		},
	}

	_, err := provider.ChatWithJSONSchema(ctx, "claude-3-5-sonnet-20241022", messages, jsonSchema, nil)
	if err == nil {
		t.Log("ChatWithJSONSchema succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderChatWithMaxTokens(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	options := map[string]any{
		"max_tokens": 16384,
	}

	_, err := provider.Chat(ctx, "claude-3-5-sonnet-20241022", messages, nil, options)
	if err == nil {
		t.Log("Chat with max_tokens succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderChatWithTemperature(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	options := map[string]any{
		"temperature": 0.5,
	}

	_, err := provider.Chat(ctx, "claude-3-5-sonnet-20241022", messages, nil, options)
	if err == nil {
		t.Log("Chat with temperature succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderChatWithStreamingOption(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Hello!",
		},
	}

	// 主动使用流式 API
	options := map[string]any{
		"streaming": true,
	}

	_, err := provider.Chat(ctx, "claude-3-5-sonnet-20241022", messages, nil, options)
	if err == nil {
		t.Log("Chat with streaming option succeeded (unexpected, might have valid credentials)")
	}
}

func TestProviderChatWithJSONSchemaWithStreamingOption(t *testing.T) {
	provider := NewProvider("test-key", "https://api.anthropic.com")

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Generate a user profile",
		},
	}

	jsonSchema := &llm.JSONSchema{
		Name:        "user_profile",
		Description: "A user profile",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "User's name",
				},
				"age": map[string]any{
					"type":        "integer",
					"description": "User's age",
				},
			},
			"required":             []string{"name", "age"},
			"additionalProperties": false,
		},
	}

	// 主动使用流式 API
	options := map[string]any{
		"streaming": true,
	}

	_, err := provider.ChatWithJSONSchema(ctx, "claude-3-5-sonnet-20241022", messages, jsonSchema, options)
	if err == nil {
		t.Log("ChatWithJSONSchema with streaming option succeeded (unexpected, might have valid credentials)")
	}
}
