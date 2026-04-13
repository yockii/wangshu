package agent

import (
	"context"

	"github.com/yockii/wangshu/pkg/llm"
)

func (a *Agent) chatInLoop(ctx context.Context, msgs []llm.Message, tools []llm.ToolDefinition, jsonSchema *llm.JSONSchema, opts map[string]any) (isFinal bool, resp *llm.ChatResponse, err error) {
	resp, err = a.provider.Chat(ctx, a.model, msgs, tools, jsonSchema, opts)
	if err != nil {
		return
	}

	if len(resp.Message.ToolCalls) == 0 {
		if resp.Message.Content == "" && jsonSchema != nil {
			// 可能不支持结构化响应，重新调用，使用非结构化模式
			return a.chatInLoop(ctx, msgs, tools, nil, opts)
		}
		isFinal = true
		return
	}
	return
}
