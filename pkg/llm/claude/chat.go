package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/yockii/wangshu/pkg/llm"
)

// containsStreamingRequiredError 检查错误信息是否包含需要流式API的错误
func containsStreamingRequiredError(errMsg string) bool {
	return strings.Contains(errMsg, "streaming is required") ||
		strings.Contains(errMsg, "10 minutes")
}

// Chat 发送聊天请求
// 如果 options 中有 "streaming": true，则直接使用流式 API
func (p *Provider) Chat(ctx context.Context, model string, message []llm.Message, tools []llm.ToolDefinition, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	// 检查是否主动要求使用流式 API
	if useStreaming, ok := options["streaming"].(bool); ok && useStreaming {
		return p.ChatStreaming(ctx, model, message, tools, jsonSchema, options)
	}

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

	if jsonSchema != nil {
		// 设置JSON Schema
		params.OutputConfig = anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: jsonSchema.Schema,
			},
		}
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

	// 调用Claude API
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		// 检查是否是超时错误，如果是则使用流式API重试
		errMsg := err.Error()
		if containsStreamingRequiredError(errMsg) {
			// 自动回退到流式API
			return p.ChatStreaming(ctx, model, message, tools, jsonSchema, options)
		}
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// 转换响应
	return p.convertResponse(*resp), nil
}

// ChatWithJSONSchema 发送带有JSON Schema的聊天请求
// 使用 Claude 的 OutputConfig 功能实现结构化输出，响应的 JSON 直接在 content 中
// 如果 options 中有 "streaming": true，则直接使用流式 API
func (p *Provider) ChatWithJSONSchema(ctx context.Context, model string, message []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	// 检查是否主动要求使用流式 API
	if useStreaming, ok := options["streaming"].(bool); ok && useStreaming {
		return p.ChatWithJSONSchemaStreaming(ctx, model, message, jsonSchema, options)
	}

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

	// 设置输出配置为 JSON 模式，直接在 content 中返回 JSON
	params.OutputConfig = anthropic.OutputConfigParam{
		Format: anthropic.JSONOutputFormatParam{
			Schema: jsonSchema.Schema,
		},
	}

	// 调用Claude API
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		// 检查是否是超时错误，如果是则使用流式API重试
		errMsg := err.Error()
		if containsStreamingRequiredError(errMsg) {
			// 自动回退到流式API
			return p.ChatWithJSONSchemaStreaming(ctx, model, message, jsonSchema, options)
		}
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// 转换响应（JSON直接在content中，不是tool_calls）
	return p.convertResponse(*resp), nil
}
