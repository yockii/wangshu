package claude

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

// convertMessages 将通用消息格式转换为Claude格式
// 注意：system消息会被单独处理，不在此函数中转换
func (p *Provider) convertMessages(messages []llm.Message) []anthropic.MessageParam {
	msgs := make([]anthropic.MessageParam, 0, len(messages))
	for _, msg := range messages {
		// 跳过system消息，需要单独处理
		if msg.Role == constant.RoleSystem {
			continue
		}

		switch msg.Role {
		case constant.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// 有工具调用的assistant消息
				contentBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)

				// 添加文本内容（如果有）
				if msg.Content != "" {
					contentBlocks = append(contentBlocks, anthropic.NewTextBlock(msg.Content))
				}

				// 添加工具调用
				for _, tc := range msg.ToolCalls {
					// tc.Arguments 是 JSON 字符串，需要转换为 map[string]any
					var inputMap map[string]any
					if tc.Arguments != "" && tc.Arguments != "{}" {
						if err := json.Unmarshal([]byte(tc.Arguments), &inputMap); err != nil {
							// 如果解析失败，创建空 map
							inputMap = make(map[string]any)
						}
					} else {
						inputMap = make(map[string]any)
					}
					contentBlocks = append(contentBlocks, anthropic.NewToolUseBlock(tc.ID, inputMap, tc.Name))
				}

				msgs = append(msgs, anthropic.NewAssistantMessage(contentBlocks...))
			} else {
				// 普通assistant消息
				msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case constant.RoleUser:
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case constant.RoleTool:
			if len(msg.ToolCalls) == 0 {
				continue
			}
			// 工具响应消息
			contentBlocks := []anthropic.ContentBlockParamUnion{
				anthropic.NewToolResultBlock(msg.ToolCalls[0].ID, msg.Content, false),
			}
			msgs = append(msgs, anthropic.NewUserMessage(contentBlocks...))
		}
	}
	return msgs
}

// extractSystemMessage 从消息数组中提取system消息
func (p *Provider) extractSystemMessage(messages []llm.Message) string {
	for _, msg := range messages {
		if msg.Role == constant.RoleSystem {
			return msg.Content
		}
	}
	return ""
}

// convertTools 将通用工具定义转换为Claude格式
func (p *Provider) convertTools(tools []llm.ToolDefinition) []anthropic.ToolUnionParam {
	toolsParams := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		// 创建ToolParam
		toolParam := anthropic.ToolParam{
			Name: tool.Function.Name,
		}

		// 设置描述（可选）
		if tool.Function.Description != "" {
			toolParam.Description = anthropic.String(tool.Function.Description)
		}

		// 将Parameters（map[string]any）转换为ToolInputSchemaParam
		parameters := tool.Function.Parameters
		// 提取properties和required
		var properties any
		var required []string
		if props, ok := parameters["properties"].(map[string]any); ok {
			properties = props
		} else {
			// 如果没有 properties 或格式不对，设置为空 map
			properties = make(map[string]any)
		}
		if req, ok := parameters["required"].([]string); ok {
			required = req
		}

		toolParam.InputSchema = anthropic.ToolInputSchemaParam{
			Properties: properties,
			Required:   required,
		}

		toolsParams = append(toolsParams, anthropic.ToolUnionParam{OfTool: &toolParam})
	}
	return toolsParams
}

// convertResponse 将Claude响应转换为通用格式
func (p *Provider) convertResponse(resp anthropic.Message) *llm.ChatResponse {
	// 转换tool_calls
	toolCalls := make([]llm.ToolCall, 0)

	// 遍历content blocks提取工具调用和文本
	for _, block := range resp.Content {
		switch block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			// 处理工具调用
			var inputMap map[string]any
			if len(block.Input) > 0 {
				json.Unmarshal(block.Input, &inputMap)
			}
			inputJSON := string(block.Input)
			if inputJSON == "null" || inputJSON == "" {
				inputJSON = "{}"
			}

			toolCalls = append(toolCalls, llm.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: inputJSON,
			})
		}
	}

	// 提取文本内容
	content := ""
	for _, block := range resp.Content {
		switch blk := block.AsAny().(type) {
		case anthropic.TextBlock:
			content += blk.Text
		}
	}

	return &llm.ChatResponse{
		Message: llm.Message{
			Role:      constant.RoleAssistant,
			Content:   content,
			ToolCalls: toolCalls,
		},
		Usage: llm.Usage{
			PromptTokens:     int64(resp.Usage.InputTokens),
			CompletionTokens: int64(resp.Usage.OutputTokens),
			TotalTokens:      int64(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
		FinishReason: string(resp.StopReason),
	}
}
