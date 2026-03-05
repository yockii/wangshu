package message

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
)

type MessageTool struct {
	basic.SimpleTool
}

func NewMessageTool() *MessageTool {
	tool := new(MessageTool)
	tool.Name_ = constant.ToolNameMessage
	tool.Desc_ = "Send a message/notification to the user. Use this tool when you need to communicate with the user, inform them about progress, report results, or provide any information that the user should see. This is the primary way to send messages to the user interface. Do NOT use write_file or other file operations to send messages to users."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The message to send",
			},
			"fileType": map[string]any{
				"type":        "string",
				"description": "The type of file to send, e.g. image, file",
				"enum":        []string{constant.FileTypeImage, constant.FileTypeFile},
			},
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to send",
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

	om := bus.NewOutboundMessage(chatID, content)
	om.Metadata.Channel = channel

	fileType := params["fileType"]
	filePath := params["filePath"]
	if fileType != "" && filePath != "" {
		om.Media = &bus.MediaContent{
			Type:     bus.MediaType(fileType),
			FilePath: filePath,
		}
	}

	bus.Default().PublishOutbound(om)

	return "", nil
}
