package tui

import (
	"context"
	"fmt"
	"sync"

	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
)

const TUIChannelName = "_builtin_"

type TUIChannel struct {
	name string
	// messageChan   chan *bus.Message
	outboundChan  chan *bus.Message
	stopCh        chan struct{}
	mu            sync.RWMutex
	onMessageRecv func(msg *bus.Message)
}

func NewTUIChannel() *TUIChannel {
	return &TUIChannel{
		name: TUIChannelName,
		// messageChan:  make(chan *bus.Message, 100),
		outboundChan: make(chan *bus.Message, 100),
		stopCh:       make(chan struct{}),
	}
}

func (c *TUIChannel) Start() error {
	return nil
}

func (c *TUIChannel) Stop() error {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
	return nil
}

func (c *TUIChannel) GetName() string {
	return c.name
}

func (c *TUIChannel) Capabilities() channel.ChannelCapabilities {
	return channel.ChannelCapabilities{
		CanSendText:       true,
		CanReceiveText:    true,
		SupportsStreaming: true,
	}
}

func (c *TUIChannel) Supports(capability channel.ChannelCapability) bool {
	switch capability {
	case channel.CanSendText, channel.CanReceiveText, channel.SupportsStreaming:
		return true
	default:
		return false
	}
}

func (c *TUIChannel) SendMessage(ctx context.Context, msg *bus.Message) error {
	select {
	case c.outboundChan <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *TUIChannel) SendText(ctx context.Context, chatID, text string) error {
	msg := bus.NewOutboundMessage(chatID, text)
	return c.SendMessage(ctx, &msg)
}

func (c *TUIChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	return fmt.Errorf("TUIChannel does not support sending media")
}

func (c *TUIChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return fmt.Errorf("TUIChannel does not support editing messages")
}

func (c *TUIChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("TUIChannel does not support deleting messages")
}

func (c *TUIChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("TUIChannel does not support pinning messages")
}

func (c *TUIChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("TUIChannel does not support unpinning messages")
}

func (c *TUIChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	return fmt.Errorf("TUIChannel does not support keyboard")
}

func (c *TUIChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return fmt.Errorf("TUIChannel does not support callback")
}

func (c *TUIChannel) GetChatInfo(ctx context.Context, chatID string) (*channel.ChatInfo, error) {
	return nil, fmt.Errorf("TUIChannel does not support getting chat info")
}

func (c *TUIChannel) GetChatMembers(ctx context.Context, chatID string) ([]channel.ChatMember, error) {
	return nil, fmt.Errorf("TUIChannel does not support getting chat members")
}

func (c *TUIChannel) ReceiveOutbound() <-chan *bus.Message {
	return c.outboundChan
}

func (c *TUIChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

func (c *TUIChannel) SetOnMessageRecv(handler func(msg *bus.Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessageRecv = handler
}

func (c *TUIChannel) PublishUserMessage(content string) {
	msg := bus.InboundMessage{
		Message: bus.Message{
			Type:    bus.MessageTypeText,
			Content: content,
			Metadata: bus.MessageMetadata{
				Channel:    c.name,
				ChatID:     "tui_chat",
				ChatType:   "p2p",
				ChatName:   "TUI Chat",
				SenderID:   "tui_user",
				SenderName: "User",
			},
		},
	}

	bus.Default().PublishInbound(msg)

	// c.messageChan <- &msg.Message
}

// func (c *TUIChannel) GetInboundChan() <-chan *bus.Message {
// 	return c.messageChan
// }
