package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"github.com/yockii/wangshu/pkg/llm"
)

// Chat 发送聊天请求
func (p *Provider) Chat(ctx context.Context, model string, message []llm.Message, tools []llm.ToolDefinition, options map[string]any) (*llm.ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	// 转换消息格式
	msgs := p.convertMessages(message)

	// 转换工具定义
	toolsUnion := p.convertTools(tools)

	body := openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(model),
		Temperature: openai.Float(temperature),
		Messages:    msgs,
		Tools:       toolsUnion,
	}

	// 调用OpenAI API
	resp, err := p.client.Chat.Completions.New(ctx, body)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("error: %s", resp.RawJSON())
	}

	// 转换响应
	return p.convertResponse(*resp), nil
}

// ChatWithJSONSchema 发送带有JSON Schema的聊天请求
func (p *Provider) ChatWithJSONSchema(ctx context.Context, model string, message []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	// 转换消息格式
	msgs := p.convertMessages(message)

	// 构建JSON Schema参数
	jsonSchemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{
		Name: jsonSchema.Name,
	}
	if jsonSchema.Description != "" {
		jsonSchemaParam.Description = openai.String(jsonSchema.Description)
	}
	if jsonSchema.Schema != nil {
		jsonSchemaParam.Schema = jsonSchema.Schema
	}
	if jsonSchema.Strict {
		jsonSchemaParam.Strict = openai.Bool(true)
	}

	// 构建响应格式
	responseFormat := openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
			JSONSchema: jsonSchemaParam,
		},
	}

	// 调用OpenAI API
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:          openai.ChatModel(model),
		Temperature:    openai.Float(temperature),
		Messages:       msgs,
		ResponseFormat: responseFormat,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("error: %s", resp.RawJSON())
	}

	// 转换响应（JSON Schema模式没有tool_calls）
	respMsg := resp.Choices[0].Message
	content := respMsg.Content
	if respMsg.Content == "" {
		content = respMsg.Refusal
	}

	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    string(respMsg.Role),
			Content: content,
		},
		Usage: llm.Usage{
			PromptTokens:     int64(resp.Usage.PromptTokens),
			CompletionTokens: int64(resp.Usage.CompletionTokens),
			TotalTokens:      int64(resp.Usage.TotalTokens),
		},
		FinishReason: string(resp.Choices[0].FinishReason),
	}, nil
}
