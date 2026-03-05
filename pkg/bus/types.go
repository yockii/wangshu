package bus

// InboundMessage 入站消息（从渠道到智能体）
// 使用 Message 结构体，保留 SessionKey 用于会话管理
type InboundMessage struct {
	Message
}

// NewOutboundMessage 创建出站消息的辅助函数
func NewOutboundMessage(chatID, content string) Message {
	return Message{
		Type:    MessageTypeText,
		Content: content,
		Metadata: MessageMetadata{
			ChatID: chatID,
		},
	}
}
