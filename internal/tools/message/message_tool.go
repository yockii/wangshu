package message

import (
	"context"
	"fmt"

	"github.com/yockii/yoclaw/internal/constant"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type MessageTool struct {
	basic.SimpleTool
}

func NewMessageTool() *MessageTool {
	tool := new(MessageTool)
	tool.Name_ = "message"
	tool.Desc_ = "Send a message to the user"
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The message to send",
			},
		},
		"required": []string{"content"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *MessageTool) execute(ctx context.Context, params map[string]string) (string, error) {
	content, ok := params["content"]
	if !ok || content == "" {
		return "", fmt.Errorf("content parameter is required")
	}
	channel := params[constant.ToolCallParamChannel]
	chatID := params[constant.ToolCallParamChatID]

	bus.Default().PublishOutbound(bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: content,
	})

	return "", nil
}
