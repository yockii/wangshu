package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/yockii/wangshu/pkg/bus"
)

// <think>或<thinking>标签，去掉标签之间的内容
var thinkRegex = regexp.MustCompile(`^(\s*)<think>[\s\S]*?</think>|<thinking>[\s\S]*?</thinking>`)

// handleMessage 处理接收到的消息
func (c *FeishuChannel) handleMessage(event *larkim.P2MessageReceiveV1) {
	inboundMsg := &bus.InboundMessage{
		Message: bus.Message{
			Content: "",
			Metadata: bus.MessageMetadata{
				Channel: c.name,
			},
		},
	}

	if event.Event.Sender.SenderId.OpenId != nil {
		inboundMsg.Metadata.SenderID = *event.Event.Sender.SenderId.OpenId
	}

	if event.Event.Message.ChatId != nil {
		inboundMsg.Metadata.ChatID = *event.Event.Message.ChatId
	}

	if event.Event.Message.MessageId != nil {
		msgID := *event.Event.Message.MessageId
		// 检查历史消息是否已经接收过
		c.groupMu.RLock()
		history := c.groupHistory[inboundMsg.Metadata.ChatID]
		c.groupMu.RUnlock()
		for _, msg := range history {
			if msgID == msg.Metadata.MessageID {
				slog.Debug("Feishu Channel handleMessage error", "err", "message already received")
				return
			}
		}
		inboundMsg.Metadata.MessageID = msgID
	}

	// Parse message content from Feishu JSON format
	if event.Event.Message.Content != nil {
		inboundMsg.Content = *event.Event.Message.Content
	}

	if event.Event.Message.MessageType != nil {
		msgType := *event.Event.Message.MessageType

		// 根据实际消息类型设置正确的消息类型
		var busMsgType bus.MessageType
		switch msgType {
		case "text":
			busMsgType = bus.MessageTypeText
		case "post":
			busMsgType = bus.MessageTypeRichMedia
		case "image":
			busMsgType = bus.MessageTypeImage
		case "file":
			busMsgType = bus.MessageTypeFile
		case "audio":
			busMsgType = bus.MessageTypeAudio
		case "video":
			busMsgType = bus.MessageTypeVideo
		default:
			busMsgType = bus.MessageTypeText
		}
		inboundMsg.Type = busMsgType
	}

	chatTypePtr := event.Event.Message.ChatType
	chatType := ""
	if chatTypePtr != nil {
		chatType = *chatTypePtr
	}
	if chatType == "p2p" {
		c.dealReceivedMessage(inboundMsg)
	} else { // group
		// 看看哪个用户发的，获取用户名
		senderName := c.getSenderName(inboundMsg.Metadata.ChatID, inboundMsg.Metadata.SenderID)
		if senderName == "" {
			slog.Error("Feishu Channel handleMessage error", "err", "sender name not found")
			return
		}

		inboundMsg.Metadata.SenderName = senderName

		c.dealReceivedMessage(inboundMsg)
		// 看看是否@机器人
		methionMe := false
		if len(event.Event.Message.Mentions) > 0 {
			for _, mention := range event.Event.Message.Mentions {
				if mention != nil && mention.Id != nil && mention.Id.OpenId != nil && *mention.Id.OpenId == c.openID {
					methionMe = true
					break
				}
			}
		}
		if methionMe {
			c.groupMu.RLock()
			historyLen := len(c.groupHistory[inboundMsg.Metadata.ChatID])
			b, has := c.groupChatInitilized[inboundMsg.Metadata.ChatID]
			needFetch := (!has || !b) && historyLen > 0
			c.groupMu.RUnlock()
			if needFetch {
				c.getGroupHistory(inboundMsg.Metadata.ChatID)
			}
			// 构造消息内容，将历史十条消息拼接起来
			c.groupMu.RLock()
			history := c.groupHistory[inboundMsg.Metadata.ChatID]
			c.groupMu.RUnlock()
			historyContent := ""
			for _, msg := range history {
				historyContent += fmt.Sprintf("%s: %s\n", msg.Metadata.SenderName, msg.Content)
			}
			inboundMsg.Content = fmt.Sprintf("你当前在群聊中，群聊最近消息:\n%s\n当前消息(提到了你):%s\n**有些信息可能与提到你时要你完成的任务无关，仅作为参考**", historyContent, fmt.Sprintf("%s: %s", senderName, inboundMsg.Content))
			// 由于这里已经提到了agent，所以历史消息可以清空，不再需要作为下一轮的参考
			c.groupMu.Lock()
			c.groupHistory[inboundMsg.Metadata.ChatID] = nil
			c.groupMu.Unlock()
		} else {
			// 将消息保留到最近10条
			c.groupMu.Lock()
			defer c.groupMu.Unlock()
			if c.groupHistory == nil {
				c.groupHistory = make(map[string][]*bus.InboundMessage)
			}
			c.groupHistory[inboundMsg.Metadata.ChatID] = append(c.groupHistory[inboundMsg.Metadata.ChatID], inboundMsg)
			if len(c.groupHistory[inboundMsg.Metadata.ChatID]) > 10 {
				c.groupHistory[inboundMsg.Metadata.ChatID] = c.groupHistory[inboundMsg.Metadata.ChatID][1:]
			}

			c.groupChatInitilized[inboundMsg.Metadata.ChatID] = true
			return
		}
	}

	bus.Default().PublishInbound(*inboundMsg)
}

// SendMessage 发送消息
func (c *FeishuChannel) SendMessage(ctx context.Context, om *bus.Message) error {
	// 发送文本内容
	if om.Content != "" {
		type BodyText struct {
			Text string `json:"text"`
		}

		// 处理@用户
		// 1. 首先处理文本中的@用户名（如 @张三），转换为 <at> 标签
		// 2. 然后处理 Entities 中的 @用户（如果有的话）
		content := om.Content

		// 如果有<think><thinking>的标签，去掉标签之间的内容
		content = thinkRegex.ReplaceAllString(content, "")

		// 如果是群聊消息，尝试转换文本中的@用户名
		if _, ok := c.groupUsers.Load(om.Metadata.ChatID); ok {
			content = c.convertMentionsToAtTags(om.Metadata.ChatID, content)
		}

		// 处理 Entities 中的 @用户（如果有明确的 open_id）
		var atText strings.Builder
		for _, entity := range om.Entities {
			if entity.Type == bus.EntityTypeMention && entity.UserID != "" {
				atText.WriteString(fmt.Sprintf("<at user_id=\"%s\"></at>", entity.UserID))
			}
		}

		body := BodyText{
			Text: content,
		}

		// 将 Entities 中的 @标签插入到文本前
		if atText.Len() > 0 {
			atText.WriteString(body.Text)
			body.Text = atText.String()
		}

		bodyContent, err := json.Marshal(body)
		if err != nil {
			return err
		}

		bodyBuilder := larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(om.Metadata.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(bodyContent))

		// 注：飞书SDK当前版本不直接支持通过Builder设置mentions和parent_id
		// 这些功能需要：
		// 1. @用户：已在文本内容中插入<at>标签
		// 2. 回复消息：需要通过其他API或直接在content中引用
		// 这里我们保持基础实现，如果需要高级功能，可以考虑直接调用HTTP API

		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(larkim.ReceiveIdTypeChatId).
			Body(bodyBuilder.Build()).
			Build()

		resp, err := c.restClient.Im.V1.Message.Create(ctx, req)
		if err != nil {
			slog.Error("Feishu Channel SendMessage error", "err", err)
			return err
		}

		if !resp.Success() {
			slog.Error("Feishu Channel SendMessage error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
			return resp.CodeError
		}
	}

	// 发送媒体（如果有）
	if om.Media != nil {
		switch om.Media.Type {
		case bus.MediaTypeImage:
			key, err := c.uploadImage(om.Media.FilePath)
			if err != nil {
				slog.Error("Feishu Channel upload image error", "err", err)
			} else {
				body := struct {
					ImageKey string `json:"image_key"`
				}{
					ImageKey: key,
				}
				bodyContent, err := json.Marshal(body)
				if err != nil {
					slog.Error("Feishu Channel upload image error", "err", err)
				} else {
					err = c.sendMsg(ctx, om.Metadata.ChatID, string(bodyContent), larkim.MsgTypeImage)
					if err != nil {
						slog.Error("Feishu Channel send image error", "err", err)
					}
				}
			}
		case bus.MediaTypeFile:
			key, err := c.uploadFile(om.Media.FilePath)
			if err != nil {
				slog.Error("Feishu Channel upload file error", "err", err)
			} else {
				body := struct {
					FileKey string `json:"file_key"`
				}{
					FileKey: key,
				}
				bodyContent, err := json.Marshal(body)
				if err != nil {
					slog.Error("Feishu Channel upload file error", "err", err)
				} else {
					err = c.sendMsg(ctx, om.Metadata.ChatID, string(bodyContent), larkim.MsgTypeFile)
					if err != nil {
						slog.Error("Feishu Channel send file error", "err", err)
					}
				}
			}
		}
	}

	return nil
}

// dealReceivedMessage 解析接收到的消息内容
func (c *FeishuChannel) dealReceivedMessage(msg *bus.InboundMessage) {
	content := msg.Content
	switch msg.Type {
	case bus.MessageTypeText:
		body := struct {
			Text string `json:"text"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return
		}
		msg.Content = body.Text
	case bus.MessageTypeRichMedia:
		body := struct {
			Title   string `json:"title"`
			Content [][]struct {
				Tag      string `json:"tag"`
				Text     string `json:"text"`
				Href     string `json:"href"`
				Language string `json:"language"`
			} `json:"content"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return
		}

		var content strings.Builder
		content.WriteString(body.Title)
		for _, item := range body.Content {
			for _, item := range item {
				content.WriteString("\n")
				switch item.Tag {
				case "text", "md":
					content.WriteString(item.Text)
				case "a":
					content.WriteString("[")
					content.WriteString(item.Text)
					content.WriteString("](")
					content.WriteString(item.Href)
					content.WriteString(")")
				case "code_block":
					content.WriteString("```")
					content.WriteString(item.Language)
					content.WriteString("\n")
					content.WriteString(item.Text)
					content.WriteString("```")
				}
			}
		}

		msg.Content = content.String()
	case bus.MessageTypeImage:
		p, err := c.downloadImage(msg.Metadata.MessageID, msg.Content)
		if err != nil {
			slog.Error("飞书渠道下载收到的图片失败", "error", err)
			msg.Content = "[用户发送了一张图片，但下载失败]"
		} else if p != "" {
			msg.Content = ""
			msg.Media = &bus.MediaContent{
				Type:     bus.MediaTypeImage,
				FilePath: p,
				FileName: filepath.Base(p),
			}
		}
	case bus.MessageTypeFile:
		// body := struct {
		// 	FileKey  string `json:"file_key"`
		// 	FileName string `json:"file_name"`
		// }{}
		// if err := json.Unmarshal([]byte(content), &body); err != nil {
		// 	slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
		// 	return ""
		// }
		// if body.FileName != "" {
		// 	return fmt.Sprintf("[文件: %s]", body.FileName)
		// }
		// return fmt.Sprintf("[文件: %s]", body.FileKey)
		p, err := c.donwloadFile(msg.Metadata.MessageID, msg.Content)
		if err != nil {
			slog.Error("飞书渠道下载收到的文件失败", "error", err)
			msg.Content = "[用户发送了一个文件，但下载失败]"
		} else if p != "" {
			fileName := filepath.Base(p)
			msg.Content = fmt.Sprintf("[用户发送了一个文件]\n文件名: %s\n文件路径: %s\n你可以读取这个文件来查看内容。", fileName, p)
		}
	case bus.MessageTypeAudio:
		body := struct {
			FileKey  string `json:"file_key"`
			Duration int    `json:"duration"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return
		}
		if body.Duration > 0 {
			msg.Content = fmt.Sprintf("[音频: %ds]", body.Duration)
			return
		}
		msg.Content = "[音频]"
	case bus.MessageTypeVideo:
		body := struct {
			FileKey  string `json:"file_key"`
			Duration int    `json:"duration"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return
		}
		if body.Duration > 0 {
			msg.Content = fmt.Sprintf("[视频: %ds]", body.Duration)
			return
		}
		msg.Content = "[视频]"
	default:
		return
	}
}
