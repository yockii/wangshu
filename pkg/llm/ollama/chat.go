package ollama

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ollama/ollama/api"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

func (p *Provider) Chat(ctx context.Context, model string, message []llm.Message, tools []llm.ToolDefinition, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	temperature := float32(0.7)
	if t, ok := options["temperature"]; ok {
		switch tt := t.(type) {
		case float64:
			temperature = float32(tt)
		case float32:
			temperature = tt
		}
	}

	msgs := p.convertMessages(message)
	toolsParam := p.convertTools(tools)

	stream := false
	req := &api.ChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   &stream,
		Tools:    toolsParam,
		Options: map[string]any{
			"temperature": temperature,
		},
	}

	if jsonSchema != nil {
		raw, _ := json.Marshal(jsonSchema.Schema)
		req.Format = json.RawMessage(raw)
	}

	var resp *api.ChatResponse
	err := p.client.Chat(ctx, req, func(r api.ChatResponse) error {
		resp = &r
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}

	return p.convertResponse(resp), nil
}

func (p *Provider) ChatWithJSONSchema(ctx context.Context, model string, message []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	temperature := float32(0.7)
	if t, ok := options["temperature"]; ok {
		switch tt := t.(type) {
		case float64:
			temperature = float32(tt)
		case float32:
			temperature = tt
		}
	}

	msgs := p.convertMessages(message)

	stream := false
	req := &api.ChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   &stream,
		Format:   json.RawMessage(`"json"`),
		Options: map[string]any{
			"temperature": temperature,
		},
	}

	var resp *api.ChatResponse
	err := p.client.Chat(ctx, req, func(r api.ChatResponse) error {
		resp = &r
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}

	return p.convertResponse(resp), nil
}

func (p *Provider) convertMessages(messages []llm.Message) []api.Message {
	msgs := make([]api.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case constant.RoleSystem:
			msgs = append(msgs, api.Message{
				Role:    constant.RoleSystem,
				Content: msg.Content,
			})
		case constant.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]api.ToolCall, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					args := api.NewToolCallFunctionArguments()
					if tc.Arguments != "" {
						var m map[string]any
						if err := json.Unmarshal([]byte(tc.Arguments), &m); err == nil {
							for k, v := range m {
								args.Set(k, v)
							}
						}
					}
					toolCalls = append(toolCalls, api.ToolCall{
						ID: tc.ID,
						Function: api.ToolCallFunction{
							Name:      tc.Name,
							Arguments: args,
						},
					})
				}
				msgs = append(msgs, api.Message{
					Role:      constant.RoleAssistant,
					Content:   msg.Content,
					ToolCalls: toolCalls,
				})
			} else {
				msgs = append(msgs, api.Message{
					Role:    constant.RoleAssistant,
					Content: msg.Content,
				})
			}
		case constant.RoleUser:
			if len(msg.Contents) > 0 {
				images := make([]api.ImageData, 0)
				content := ""
				for _, c := range msg.Contents {
					switch c.Type {
					case "text":
						content += c.Text
					case "image":
						images = append(images, api.ImageData(c.ImageData))
					}
				}
				msgs = append(msgs, api.Message{
					Role:    constant.RoleUser,
					Content: content,
					Images:  images,
				})
			} else {
				msgs = append(msgs, api.Message{
					Role:    constant.RoleUser,
					Content: msg.Content,
				})
			}
		case constant.RoleTool:
			if len(msg.ToolCalls) == 0 {
				continue
			}
			msgs = append(msgs, api.Message{
				Role:       constant.RoleTool,
				Content:    msg.Content,
				ToolCallID: msg.ToolCalls[0].ID,
			})
		}
	}
	return msgs
}

func (p *Provider) convertTools(tools []llm.ToolDefinition) api.Tools {
	if len(tools) == 0 {
		return nil
	}
	var result api.Tools
	for _, tool := range tools {
		props := tool.Function.Parameters
		if props == nil {
			props = make(map[string]any)
		}

		params := api.ToolFunctionParameters{
			Type:       "object",
			Properties: api.NewToolPropertiesMap(),
		}

		if properties, ok := props["properties"].(map[string]any); ok {
			for k, v := range properties {
				if propMap, ok := v.(map[string]any); ok {
					tp := api.ToolProperty{}
					if t, ok := propMap["type"].(string); ok {
						tp.Type = api.PropertyType{t}
					}
					if desc, ok := propMap["description"].(string); ok {
						tp.Description = desc
					}
					params.Properties.Set(k, tp)
				}
			}
		}

		if required, ok := props["required"].([]string); ok {
			params.Required = required
		} else if requiredAny, ok := props["required"].([]any); ok {
			for _, r := range requiredAny {
				if rs, ok := r.(string); ok {
					params.Required = append(params.Required, rs)
				}
			}
		}

		result = append(result, api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  params,
			},
		})
	}
	return result
}

func (p *Provider) convertResponse(resp *api.ChatResponse) *llm.ChatResponse {
	toolCalls := make([]llm.ToolCall, 0, len(resp.Message.ToolCalls))
	for _, tc := range resp.Message.ToolCalls {
		argsStr := "{}"
		if m := tc.Function.Arguments.ToMap(); m != nil {
			if b, err := json.Marshal(m); err == nil {
				argsStr = string(b)
			}
		}
		toolCalls = append(toolCalls, llm.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: argsStr,
		})
	}

	return &llm.ChatResponse{
		Message: llm.Message{
			Role:      resp.Message.Role,
			Content:   resp.Message.Content,
			ToolCalls: toolCalls,
		},
		Usage: llm.Usage{
			PromptTokens:     int64(resp.PromptEvalCount),
			CompletionTokens: int64(resp.EvalCount),
			TotalTokens:      int64(resp.PromptEvalCount + resp.EvalCount),
		},
		FinishReason: resp.DoneReason,
	}
}
