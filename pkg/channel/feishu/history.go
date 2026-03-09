package feishu

import (
	"context"
	"log/slog"
	"slices"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/yockii/wangshu/pkg/bus"
)

// getGroupHistory 获取群聊历史消息
func (c *FeishuChannel) getGroupHistory(chatID string) error {
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("chat").
		ContainerId(chatID).
		SortType("ByCreateTimeDesc").
		PageSize(20).
		Build()
	resp, err := c.restClient.Im.V1.Message.List(context.Background(), req)
	if err != nil {
		slog.Error("Fetch Feishu Group Message Failed", "error", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel getGroupHistory error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return resp.CodeError
	}

	// 遍历消息列表
	var msgs []*bus.InboundMessage
	for i, message := range resp.Data.Items {
		if i == 0 {
			continue
		}
		if len(msgs) > 9 {
			break
		}

		if message.Body != nil && message.Sender != nil && message.Sender.Id != nil && message.Sender.SenderType != nil {
			if message.MsgType == nil {
				continue
			}

			// 看看是否有提到机器人
			// 看看是否@机器人
			methionMe := false
			if len(message.Mentions) > 0 {
				for _, mention := range message.Mentions {
					if mention != nil && mention.Id != nil && *mention.Id == c.openID {
						methionMe = true
						break
					}
				}
			}
			if methionMe {
				break
			}

			busMsg := &bus.InboundMessage{
				Message: bus.Message{
					Content: "",
					Metadata: bus.MessageMetadata{
						ChatID:   chatID,
						Channel:  c.name,
						SenderID: *message.Sender.Id,
					},
				},
			}

			switch *message.MsgType {
			case "text":
				busMsg.Type = bus.MessageTypeText
			case "post":
				busMsg.Type = bus.MessageTypeRichMedia
			case "image":
				busMsg.Type = bus.MessageTypeImage
			case "file":
				busMsg.Type = bus.MessageTypeFile
			case "audio":
				busMsg.Type = bus.MessageTypeAudio
			case "video":
				busMsg.Type = bus.MessageTypeVideo
			default:
				busMsg.Type = bus.MessageTypeText
			}
			busMsg.Content = *message.Body.Content
			c.dealReceivedMessage(busMsg)
			if busMsg.Content == "" {
				continue
			}

			if busMsg.Metadata.SenderID != "" {
				busMsg.Metadata.SenderName = c.getSenderName(busMsg.Metadata.ChatID, busMsg.Metadata.SenderID)
			}

			msgs = append(msgs, busMsg)
		}
	}
	// 将msgs倒一下
	slices.Reverse(msgs)

	c.groupMu.Lock()
	c.groupHistory[chatID] = msgs
	c.groupChatInitilized[chatID] = true
	c.groupMu.Unlock()
	return nil
}
