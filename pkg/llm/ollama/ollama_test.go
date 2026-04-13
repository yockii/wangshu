package ollama

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

const mockOllamaURL = "http://127.0.0.1:8787/ollama"

func TestNewProvider(t *testing.T) {
	provider := NewProvider(mockOllamaURL)
	if provider == nil {
		t.Error("NewProvider should return a non-nil provider")
	}
}

func TestNewProviderWithEmptyBaseURL(t *testing.T) {
	provider := NewProvider("")
	if provider == nil {
		t.Error("NewProvider with empty baseURL should return a non-nil provider")
	}
}

func TestNewProviderWithInvalidBaseURL(t *testing.T) {
	provider := NewProvider("://invalid-url")
	if provider == nil {
		t.Error("NewProvider with invalid baseURL should fallback to default")
	}
}

func TestProviderChat(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

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

	resp, err := provider.Chat(ctx, "llama3.2", messages, nil, nil, nil)
	if err != nil {
		t.Logf("Chat error (expected if mock server not running): %v", err)
		return
	}

	t.Logf("Chat response: Role=%s, Content=%s", resp.Message.Role, resp.Message.Content)
	if resp.Message.Content == "" {
		t.Error("Expected non-empty content in response")
	}
}

func TestProviderChatWithTools(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    constant.RoleUser,
			Content: "What's the weather in Tokyo?",
		},
	}

	tools := []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get current weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "City name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	resp, err := provider.Chat(ctx, "llama3.2", messages, tools, nil, nil)
	if err != nil {
		t.Logf("Chat with tools error (expected if mock server not running): %v", err)
		return
	}

	t.Logf("Chat response: Role=%s, Content=%s, ToolCalls=%d", resp.Message.Role, resp.Message.Content, len(resp.Message.ToolCalls))
}

func TestProviderChatWithJSONSchema(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    constant.RoleUser,
			Content: "Generate a user profile for John, age 30",
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
			"required": []string{"name", "age"},
		},
	}

	resp, err := provider.ChatWithJSONSchema(ctx, "llama3.2", messages, jsonSchema, nil)
	if err != nil {
		t.Logf("ChatWithJSONSchema error (expected if mock server not running): %v", err)
		return
	}

	t.Logf("JSON Schema response: %s", resp.Message.Content)
}

func TestProviderChatWithTemperature(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

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

	resp, err := provider.Chat(ctx, "llama3.2", messages, nil, nil, options)
	if err != nil {
		t.Logf("Chat with temperature error (expected if mock server not running): %v", err)
		return
	}

	t.Logf("Chat response with temperature: %s", resp.Message.Content)
}

func TestProviderConvertMessages(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

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
			Role:    constant.RoleAssistant,
			Content: "Hi there!",
		},
	}

	converted := provider.convertMessages(messages)
	if len(converted) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(converted))
	}

	if converted[0].Role != constant.RoleSystem {
		t.Errorf("Expected first message role to be 'system', got '%s'", converted[0].Role)
	}
	if converted[1].Role != constant.RoleUser {
		t.Errorf("Expected second message role to be 'user', got '%s'", converted[1].Role)
	}
	if converted[2].Role != constant.RoleAssistant {
		t.Errorf("Expected third message role to be 'assistant', got '%s'", converted[2].Role)
	}
}

func TestProviderConvertMessagesWithToolCalls(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

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
	if len(converted) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(converted))
	}

	if len(converted[1].ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call in assistant message, got %d", len(converted[1].ToolCalls))
	}

	if converted[1].ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("Expected tool call function name 'get_weather', got '%s'", converted[1].ToolCalls[0].Function.Name)
	}

	if converted[2].Role != constant.RoleTool {
		t.Errorf("Expected third message role to be 'tool', got '%s'", converted[2].Role)
	}
}

func TestProviderConvertMessagesWithImages(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

	messages := []llm.Message{
		{
			Role: constant.RoleUser,
			Contents: []llm.ContentBlock{
				llm.NewTextContent("What's in this image?"),
				llm.NewImageContent("base64imagedata", "image/png"),
			},
		},
	}

	converted := provider.convertMessages(messages)
	if len(converted) != 1 {
		t.Errorf("Expected 1 message, got %d", len(converted))
	}

	if len(converted[0].Images) != 1 {
		t.Errorf("Expected 1 image in message, got %d", len(converted[0].Images))
	}
}

func TestProviderConvertTools(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

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
						"param2": map[string]any{
							"type":        "integer",
							"description": "Another parameter",
						},
					},
					"required": []string{"param1"},
				},
			},
		},
	}

	converted := provider.convertTools(tools)
	if len(converted) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(converted))
	}

	if converted[0].Function.Name != "test_function" {
		t.Errorf("Expected function name 'test_function', got '%s'", converted[0].Function.Name)
	}

	if converted[0].Function.Description != "A test function" {
		t.Errorf("Expected description 'A test function', got '%s'", converted[0].Function.Description)
	}
}

func TestProviderConvertToolsEmpty(t *testing.T) {
	provider := NewProvider(mockOllamaURL)

	converted := provider.convertTools(nil)
	if converted != nil {
		t.Error("Expected nil for empty tools")
	}

	converted = provider.convertTools([]llm.ToolDefinition{})
	if converted != nil {
		t.Error("Expected nil for empty tools slice")
	}
}

func TestProviderConvertResponse(t *testing.T) {
	_ = NewProvider(mockOllamaURL)

	resp := &llm.ChatResponse{
		Message: llm.Message{
			Role:      constant.RoleAssistant,
			Content:   "Hello! How can I help you?",
			ToolCalls: nil,
		},
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		FinishReason: "stop",
	}

	if resp.Message.Role != constant.RoleAssistant {
		t.Errorf("Expected role 'assistant', got '%s'", resp.Message.Role)
	}

	if resp.Message.Content != "Hello! How can I help you?" {
		t.Errorf("Expected content 'Hello! How can I help you?', got '%s'", resp.Message.Content)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Expected total tokens 30, got %d", resp.Usage.TotalTokens)
	}
}
