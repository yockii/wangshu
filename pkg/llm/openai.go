package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
	selfConstant "github.com/yockii/yoclaw/pkg/constant"
)

type OpenAIProvider struct {
	client openai.Client
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &OpenAIProvider{client: client}
}

func (p *OpenAIProvider) Chat(ctx context.Context, model string, message []Message, tools []ToolDefinition, options map[string]any) (*ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(message))
	for _, msg := range message {
		switch msg.Role {
		case selfConstant.RoleSystem:
			msgs = append(msgs, openai.SystemMessage(msg.Content))
		case selfConstant.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
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
				msgs = append(msgs, openai.AssistantMessage(msg.Content))
			}
		case selfConstant.RoleUser:
			msgs = append(msgs, openai.UserMessage(msg.Content))
		case selfConstant.RoleTool:
			msgs = append(msgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}

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

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(model),
		Temperature: openai.Float(temperature),
		Messages:    msgs,
		Tools:       toolsUnion,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("error: %s", resp.RawJSON())
	}

	respMsg := resp.Choices[0].Message
	toolCalls := make([]ToolCall, 0, len(respMsg.ToolCalls))
	for _, tc := range respMsg.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	content := respMsg.Content
	if respMsg.Content == "" {
		content = respMsg.Refusal
	}

	cr := &ChatResponse{
		Message: Message{
			Role:      string(respMsg.Role),
			Content:   content,
			ToolCalls: toolCalls,
		},
		Usage: Usage{
			PromptTokens:     (resp.Usage.PromptTokens),
			CompletionTokens: (resp.Usage.CompletionTokens),
			TotalTokens:      (resp.Usage.TotalTokens),
		},
		FinishReason: string(resp.Choices[0].FinishReason),
	}
	return cr, nil
}

// ChatWithJSONSchema sends a chat request with JSON schema for structured output
func (p *OpenAIProvider) ChatWithJSONSchema(ctx context.Context, model string, message []Message, jsonSchema *JSONSchema, options map[string]any) (*ChatResponse, error) {
	temperature := 0.7
	if t, ok := options["temperature"]; ok {
		if tt, ok := t.(float64); ok {
			temperature = tt
		}
	}

	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(message))
	for _, msg := range message {
		switch msg.Role {
		case selfConstant.RoleSystem:
			msgs = append(msgs, openai.SystemMessage(msg.Content))
		case selfConstant.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
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
				msgs = append(msgs, openai.AssistantMessage(msg.Content))
			}
		case selfConstant.RoleUser:
			msgs = append(msgs, openai.UserMessage(msg.Content))
		case selfConstant.RoleTool:
			msgs = append(msgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}

	// Build JSON schema param
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

	// Build response format with JSON schema
	responseFormat := openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
			JSONSchema: jsonSchemaParam,
		},
	}

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:          openai.ChatModel(model),
		Temperature:    openai.Float(temperature),
		Messages:       msgs,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONSchema: responseFormat.OfJSONSchema},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("error: %s", resp.RawJSON())
	}

	respMsg := resp.Choices[0].Message

	content := respMsg.Content
	if respMsg.Content == "" {
		content = respMsg.Refusal
	}

	cr := &ChatResponse{
		Message: Message{
			Role:    string(respMsg.Role),
			Content: content,
		},
		Usage: Usage{
			PromptTokens:     (resp.Usage.PromptTokens),
			CompletionTokens: (resp.Usage.CompletionTokens),
			TotalTokens:      (resp.Usage.TotalTokens),
		},
		FinishReason: string(resp.Choices[0].FinishReason),
	}
	return cr, nil
}
