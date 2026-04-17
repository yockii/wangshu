package message

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

type MessageTool struct {
	basic.SimpleTool
}

func NewMessageTool() *MessageTool {
	tool := new(MessageTool)
	tool.Name_ = constant.ToolNameMessage
	tool.Desc_ = "Send a message/notification to the user. Use this tool in the following scenarios: 1) In asynchronous tasks (like scheduled tasks) where you need to send messages to users without an ongoing conversation; 2) When you need to send files or images to users (regardless of whether it's in a regular conversation or not). Do NOT use this tool for regular text-only conversation responses, as the system will automatically send your regular response content. Using this tool for regular text-only responses will result in duplicate messages. This is the primary way to send messages to the user interface in asynchronous scenarios and to send files/images. Do NOT use write_file or other file operations to send messages to users."
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

func (t *MessageTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	content, ok := params["content"]
	if !ok || content == "" {
		return types.NewToolResult().WithError(fmt.Errorf("content parameter is required"))
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

	return types.NewToolResult().WithStructured(actiontypes.NewMessageSendData(channel, chatID, content))
}
