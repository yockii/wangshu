package web

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
)

// WebChannel WebSocket渠道实现
type WebChannel struct {
	name        string
	conn        *websocket.Conn
	connMu      sync.RWMutex
	hostAddress string
	token       string
	stopCh      chan struct{}
	reconnectCh chan struct{}
}

// NewWebChannel 创建一个新的Web渠道
func NewWebChannel(name, hostAddress, token string) *WebChannel {
	return &WebChannel{
		name:        name,
		hostAddress: hostAddress,
		token:       token,
		stopCh:      make(chan struct{}, 1),
		reconnectCh: make(chan struct{}, 1),
	}
}

// Start 启动Web渠道
func (c *WebChannel) Start() error {
	slog.Info("Web channel starting", "name", c.name, "host", c.hostAddress)
	go c.connectToServer()
	go c.monitor()
	return nil
}

// Stop 停止Web渠道
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

// GetName 返回channel名称
func (c *WebChannel) GetName() string {
	return c.name
}

// Capabilities 返回WebChannel的能力
func (c *WebChannel) Capabilities() channel.ChannelCapabilities {
	return channel.ChannelCapabilities{
		CanSendText:       true,
		CanReceiveText:    true,
		SupportsStreaming: true,
	}
}

// Supports 检查是否支持某个能力
func (c *WebChannel) Supports(capability channel.ChannelCapability) bool {
	switch capability {
	case channel.CanSendText, channel.CanReceiveText, channel.SupportsStreaming:
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
func (c *WebChannel) GetChatInfo(ctx context.Context, chatID string) (*channel.ChatInfo, error) {
	return nil, fmt.Errorf("WebChannel does not support getting chat info")
}

// GetChatMembers 获取聊天成员（暂不支持）
func (c *WebChannel) GetChatMembers(ctx context.Context, chatID string) ([]channel.ChatMember, error) {
	return nil, fmt.Errorf("WebChannel does not support getting chat members")
}

// SendMedia 发送媒体消息（暂不支持）
func (c *WebChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	return fmt.Errorf("WebChannel does not support sending media")
}
