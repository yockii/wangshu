package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yockii/wangshu/internal/session"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
)

// executeToolCall 执行工具调用
func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolCall, channel, chatID string) (string, error) {
	var args map[string]any
	if tc.Arguments != "" {
		err := json.Unmarshal([]byte(tc.Arguments), &args)
		if err != nil {
			return "", fmt.Errorf("Failed to parse tool arguments: %w", err)
		}
	}

	if args == nil {
		args = make(map[string]any)
	}

	args[constant.ToolCallParamWorkspace] = a.workspaceDir
	args[constant.ToolCallParamChannel] = channel
	args[constant.ToolCallParamChatID] = chatID

	// Create ToolContext with agent information
	toolCtx := tools.NewToolContext(
		a.agentName,
		"", // agent owner - can be added later
		a.workspaceDir,
		"", // sessionID - can be passed separately if needed
		channel,
		chatID,
		a.provider,
		a.model,
	)

	result := tools.GetDefaultToolRegistry().ExecuteWithContext(ctx, tc.Name, args, toolCtx, channel, chatID)
	if result.IsError {
		return result.ForLLM, fmt.Errorf("Tool execution failed: %q", result.Err)
	}
	return result.ForLLM, nil
}

// addToolResultMessage 添加工具结果消息到会话
func addToolResultMessage(sess *session.Session, role, content, toolCallID string) {
	// If toolCallID is provided, find and update the tool call
	if toolCallID != "" {
		messages := sess.GetMessages()
		for i := len(messages) - 1; i >= 0; i-- {
			for _, tc := range messages[i].ToolCalls {
				if tc.ID == toolCallID {
					// Add the result to the tool call
					sess.AddMessage(role, content, types.ToolCall{
						ID:        toolCallID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
						Result:    content,
					})
					return
				}
			}
		}
	}

	// If no toolCallID or not found, add as regular message
	sess.AddMessage(role, content)
}

// formatMessages 格式化消息用于显示
func formatMessages(messages []types.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role == constant.RoleUser || msg.Role == constant.RoleAssistant {
			sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}
	return sb.String()
}
