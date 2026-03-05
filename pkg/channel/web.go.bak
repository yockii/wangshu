package channel

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yockii/wangshu/pkg/bus"
)

type WebChannel struct {
	name        string
	conn        *websocket.Conn
	connMu      sync.RWMutex
	hostAddress string
	token       string
	stopCh      chan struct{}
	reconnectCh chan struct{}
}

func NewWebChannel(name, hostAddress, token string) *WebChannel {
	return &WebChannel{
		name:        name,
		hostAddress: hostAddress,
		token:       token,
		stopCh:      make(chan struct{}, 1),
		reconnectCh: make(chan struct{}, 1),
	}
}

func (c *WebChannel) Start() error {
	slog.Info("Web channel starting", "name", c.name, "host", c.hostAddress)
	go c.connectToServer()
	go c.monitor()
	return nil
}

func (c *WebChannel) connectToServer() {
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	u := url.URL{Scheme: "ws", Host: c.hostAddress, Path: "/ws"}
	if c.token != "" {
		u.RawQuery = "token=" + c.token
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		slog.Error("Failed to connect to web server", "error", err)
		c.reconnectCh <- struct{}{}
		return
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	go c.readLoop()
}

func (c *WebChannel) Stop() error {
	slog.Info("Web channel stopping", "name", c.name)

	// 安全地关闭stopCh
	select {
	case <-c.stopCh:
		// 已经关闭
	default:
		close(c.stopCh)
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *WebChannel) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			slog.Debug("stop web channel")
			return
		case <-c.reconnectCh:
			slog.Debug("reconnect to web server")
			time.Sleep(5 * time.Second)
			c.connectToServer()
		case <-ticker.C:
			slog.Debug("keepalive web ws")
		}
	}
}

func (c *WebChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

func (c *WebChannel) readLoop() {
	defer func() {
		c.connMu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
		c.reconnectCh <- struct{}{}
	}()

	for {
		select {
		case <-c.stopCh:
			return
		default:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			var msg struct {
				Type    string `json:"type"`
				Content string `json:"content"`
				Session string `json:"session,omitempty"`
			}
			if err := conn.ReadJSON(&msg); err != nil {
				slog.Error("Failed to read message from web server", "error", err)
				return
			}

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
	}
}

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

// GetName 返回channel名称
func (c *WebChannel) GetName() string {
	return c.name
}

// Capabilities 返回WebChannel的能力
func (c *WebChannel) Capabilities() ChannelCapabilities {
	return ChannelCapabilities{
		CanSendText:       true,
		CanReceiveText:    true,
		SupportsStreaming: true,
	}
}

// Supports 检查是否支持某个能力
func (c *WebChannel) Supports(capability ChannelCapability) bool {
	switch capability {
	case CanSendText, CanReceiveText, SupportsStreaming:
		return true
	default:
		return false
	}
}

// SendText 发送文本消息
func (c *WebChannel) SendText(ctx context.Context, chatID, text string) error {
	msg := bus.NewOutboundMessage(chatID, text)
	return c.SendMessage(ctx, &msg)
}

// SendMedia 发送媒体消息（暂不支持）
func (c *WebChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	return fmt.Errorf("WebChannel does not support sending media")
}

// EditMessage 编辑消息（暂不支持）
func (c *WebChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return fmt.Errorf("WebChannel does not support editing messages")
}

// DeleteMessage 删除消息（暂不支持）
func (c *WebChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WebChannel does not support deleting messages")
}

// PinMessage 置顶消息（暂不支持）
func (c *WebChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WebChannel does not support pinning messages")
}

// UnpinMessage 取消置顶消息（暂不支持）
func (c *WebChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WebChannel does not support unpinning messages")
}

// SendKeyboard 发送键盘消息（暂不支持）
func (c *WebChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	return fmt.Errorf("WebChannel does not support keyboard")
}

// AnswerCallback 回调查询（暂不支持）
func (c *WebChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return fmt.Errorf("WebChannel does not support callback")
}

// GetChatInfo 获取聊天信息（暂不支持）
func (c *WebChannel) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	return nil, fmt.Errorf("WebChannel does not support getting chat info")
}

// GetChatMembers 获取聊天成员（暂不支持）
func (c *WebChannel) GetChatMembers(ctx context.Context, chatID string) ([]ChatMember, error) {
	return nil, fmt.Errorf("WebChannel does not support getting chat members")
}
