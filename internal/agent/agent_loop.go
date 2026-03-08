package agent

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/session"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
)

// runLoop 运行LLM对话循环，处理工具调用
func (a *Agent) runLoop(ctx context.Context, sess *session.Session, msgs []llm.Message) (string, error) {
	var finalContent string

	options := map[string]any{}
	if agentCfg, ok := config.DefaultCfg.Agents[a.agentName]; ok {
		if agentCfg.Temperature != 0 {
			options["temperature"] = agentCfg.Temperature
		}
		if agentCfg.MaxTokens != 0 {
			options["max_tokens"] = agentCfg.MaxTokens
		}
	}

	availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
	for i := 0; i < a.maxIter; i++ {
		resp, err := a.provider.Chat(ctx, a.model, msgs, availableTools, options)
		if err != nil {
			return "", fmt.Errorf("LLM call failed (iteration %d): %w", i+1, err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			// 不需要调用工具，则开始输出
			finalContent = resp.Message.Content
			break
		}

		assistantMsg := types.Message{
			Role:    constant.RoleAssistant,
			Content: resp.Message.Content,
		}

		for _, tc := range resp.Message.ToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, types.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
		// 加到session中
		sess.AddMessage(assistantMsg.Role, assistantMsg.Content, assistantMsg.ToolCalls...)

		// 加到发给大模型的对话列表中
		msgs = append(msgs, llm.Message{
			Role:      constant.RoleAssistant,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})

		if resp.Message.Content != "" && len(resp.Message.ToolCalls) > 0 {
			// 有内容，且调用工具，则说明还需要循环，但内容可以先直接发送给用户
			msg := bus.NewOutboundMessage(sess.ChatID, resp.Message.Content)
			msg.Metadata.Channel = sess.Channel
			bus.Default().PublishOutbound(msg)
		}

		// 执行所有的工具调用
		for _, tc := range resp.Message.ToolCalls {
			toolResult, err := a.executeToolCall(ctx, tc, sess.Channel, sess.ChatID, sess.SenderID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
			}

			addToolResultMessage(sess, constant.RoleTool, toolResult, tc.ID)

			msgs = append(msgs, llm.Message{
				Role:      constant.RoleTool,
				Content:   toolResult,
				ToolCalls: []llm.ToolCall{tc},
			})
		}

	}

	if finalContent != "" {
		sess.AddMessage(constant.RoleAssistant, finalContent)
	}

	return finalContent, nil
}
