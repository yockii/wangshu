package openai

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
	selfConstant "github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

// convertMessages 将通用消息格式转换为OpenAI格式
func (p *Provider) convertMessages(messages []llm.Message) []openai.ChatCompletionMessageParamUnion {
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case selfConstant.RoleSystem:
			msgs = append(msgs, openai.SystemMessage(msg.Content))
		case selfConstant.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// 有工具调用的assistant消息
				toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					u := openai.ChatCompletionMessageToolCallUnionParam{}
					u.OfFunction = &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Arguments: tc.Arguments,
							Name:      tc.Name,
						},
					}
					toolCalls = append(toolCalls, u)
				}
				msgs = append(msgs, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Role: constant.Assistant(msg.Role),
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: openai.String(msg.Content),
						},
						ToolCalls: toolCalls,
					},
				})
			} else {
				// 普通assistant消息
				msgs = append(msgs, openai.AssistantMessage(msg.Content))
			}
		case selfConstant.RoleUser:
			msgs = append(msgs, openai.UserMessage(msg.Content))
		case selfConstant.RoleTool:
			if len(msg.ToolCalls) == 0 {
				continue
			}
			msgs = append(msgs, openai.ToolMessage(msg.Content, msg.ToolCalls[0].ID))
		}
	}
	return msgs
}

// convertTools 将通用工具定义转换为OpenAI格式
func (p *Provider) convertTools(tools []llm.ToolDefinition) []openai.ChatCompletionToolUnionParam {
	toolsUnion := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		toolsUnion = append(toolsUnion, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Type: constant.Function(tool.Type),
				Function: shared.FunctionDefinitionParam{
					Name:        tool.Function.Name,
					Description: openai.String(tool.Function.Description),
					Parameters:  tool.Function.Parameters,
				},
			},
		})
	}
	return toolsUnion
}

// convertResponse 将OpenAI响应转换为通用格式
func (p *Provider) convertResponse(resp openai.ChatCompletion) *llm.ChatResponse {
	respMsg := resp.Choices[0].Message

	// 转换tool_calls
	toolCalls := make([]llm.ToolCall, 0, len(respMsg.ToolCalls))
	for _, tc := range respMsg.ToolCalls {
		toolCalls = append(toolCalls, llm.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// 获取内容
	content := respMsg.Content
	if respMsg.Content == "" {
		content = respMsg.Refusal
	}

	return &llm.ChatResponse{
		Message: llm.Message{
			Role:      string(respMsg.Role),
			Content:   content,
			ToolCalls: toolCalls,
		},
		Usage: llm.Usage{
			PromptTokens:     int64(resp.Usage.PromptTokens),
			CompletionTokens: int64(resp.Usage.CompletionTokens),
			TotalTokens:      int64(resp.Usage.TotalTokens),
		},
		FinishReason: string(resp.Choices[0].FinishReason),
	}
}
