package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/yockii/wangshu/pkg/llm"
)

// ChatStreaming 使用流式API来避免10分钟超时限制
func (p *Provider) ChatStreaming(ctx context.Context, model string, message []llm.Message, tools []llm.ToolDefinition, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	// 提取system消息
	systemPrompt := p.extractSystemMessage(message)

	// 转换消息格式（已过滤system消息）
	msgs := p.convertMessages(message)

	// 读取max_tokens配置，默认8192（Anthropic必需参数）
	var maxTokens int64 = 8192
	if mt, ok := options["max_tokens"].(int64); ok && mt > 0 {
		maxTokens = mt
	}

	// 构建请求参数
	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(model),
		Temperature: anthropic.Float(temperature),
		Messages:    msgs,
		MaxTokens:   maxTokens,
	}

	// 设置system参数
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemPrompt}}
	}

	// 转换工具定义
	if len(tools) > 0 {
		toolsParams := p.convertTools(tools)
		params.Tools = toolsParams
	}

	if jsonSchema != nil {
		// 设置JSON Schema
		params.OutputConfig = anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: jsonSchema.Schema,
			},
		}
	}

	// 使用流式API
	stream := p.client.Messages.NewStreaming(ctx, params)

	// 累积流式响应
	accumulatedMessage := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := accumulatedMessage.Accumulate(event)
		if err != nil {
			return nil, fmt.Errorf("failed to accumulate streaming response: %w", err)
		}
	}

	if stream.Err() != nil {
		return nil, fmt.Errorf("Claude API streaming error: %w", stream.Err())
	}

	// 转换响应
	return p.convertResponse(accumulatedMessage), nil
}

// ChatWithJSONSchemaStreaming 使用流式API进行JSON Schema结构化输出
func (p *Provider) ChatWithJSONSchemaStreaming(ctx context.Context, model string, message []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	// 提取system消息
	systemPrompt := p.extractSystemMessage(message)

	// 转换消息格式（已过滤system消息）
	msgs := p.convertMessages(message)

	// 读取max_tokens配置，默认8192
	var maxTokens int64 = 8192
	if mt, ok := options["max_tokens"].(int64); ok && mt > 0 {
		maxTokens = mt
	}

	// 构建请求参数
	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(model),
		Temperature: anthropic.Float(temperature),
		Messages:    msgs,
		MaxTokens:   maxTokens,
	}

	// 设置system参数
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemPrompt}}
	}

	// 设置输出配置为 JSON 模式
	params.OutputConfig = anthropic.OutputConfigParam{
		Format: anthropic.JSONOutputFormatParam{
			Schema: jsonSchema.Schema,
		},
	}

	// 使用流式API
	stream := p.client.Messages.NewStreaming(ctx, params)

	// 累积流式响应
	accumulated := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := accumulated.Accumulate(event)
		if err != nil {
			return nil, fmt.Errorf("failed to accumulate streaming response: %w", err)
		}
	}

	if stream.Err() != nil {
		return nil, fmt.Errorf("Claude API streaming error: %w", stream.Err())
	}

	// 转换响应（JSON直接在content中）
	return p.convertResponse(accumulated), nil
}
