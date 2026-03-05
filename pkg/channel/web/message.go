package web

import (
	"context"
	"log/slog"

	"github.com/yockii/wangshu/pkg/bus"
)

// SubscribeOutbound 订阅出站消息
func (c *WebChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

// SendMessage 发送消息
func (c *WebChannel) SendMessage(ctx context.Context, om *bus.Message) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()

	if c.conn == nil {
		slog.Warn("Web channel not connected", "name", c.name)
		return nil
	}

	msg := map[string]any{
		"type":    "message",
		"content": om.Content,
		"chat_id": om.Metadata.ChatID,
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		slog.Error("Failed to send message to web server", "error", err)
		return err
	}

	return nil
}

// handleIncomingMessage 处理接收到的消息
func (c *WebChannel) handleIncomingMessage(msg struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Session string `json:"session,omitempty"`
}) {
	bus.Default().PublishInbound(bus.InboundMessage{
		Message: bus.Message{
			Type:    bus.MessageTypeText,
			Content: msg.Content,
			Metadata: bus.MessageMetadata{
				SenderID: "web",
				ChatID:   msg.Session,
				Channel:  c.name,
			},
		},
	})
}
