package channel

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yockii/wangshu/pkg/bus"
)

// BaseChannel 基础Channel，提供公共功能
type BaseChannel struct {
	name         string
	stopCh       chan struct{}
	reconnectCh  chan struct{}
	mu           sync.RWMutex
	running      bool
	capabilities ChannelCapabilities
}

// NewBaseChannel 创建基础Channel
func NewBaseChannel(name string, capabilities ChannelCapabilities) *BaseChannel {
	return &BaseChannel{
		name:         name,
		stopCh:       make(chan struct{}, 1),
		reconnectCh:  make(chan struct{}, 1),
		capabilities: capabilities,
	}
}

// GetName 返回Channel名称
func (b *BaseChannel) GetName() string {
	return b.name
}

// Capabilities 返回Channel能力
func (b *BaseChannel) Capabilities() ChannelCapabilities {
	return b.capabilities
}

// Supports 检查是否支持某个能力
func (b *BaseChannel) Supports(capability ChannelCapability) bool {
	switch capability {
	case CanSendText:
		return b.capabilities.CanSendText
	case CanSendImage:
		return b.capabilities.CanSendImage
	case CanSendVideo:
		return b.capabilities.CanSendVideo
	case CanSendAudio:
		return b.capabilities.CanSendAudio
	case CanSendFile:
		return b.capabilities.CanSendFile
	case CanSendLocation:
		return b.capabilities.CanSendLocation
	case CanSendSticker:
		return b.capabilities.CanSendSticker
	case CanSendRichMedia:
		return b.capabilities.CanSendRichMedia
	case CanSendKeyboard:
		return b.capabilities.CanSendKeyboard
	case CanReceiveText:
		return b.capabilities.CanReceiveText
	case CanReceiveImage:
		return b.capabilities.CanReceiveImage
	case CanReceiveVideo:
		return b.capabilities.CanReceiveVideo
	case CanReceiveAudio:
		return b.capabilities.CanReceiveAudio
	case CanReceiveFile:
		return b.capabilities.CanReceiveFile
	case CanReceiveLocation:
		return b.capabilities.CanReceiveLocation
	case CanReceiveSticker:
		return b.capabilities.CanReceiveSticker
	case CanEditMessage:
		return b.capabilities.CanEditMessage
	case CanDeleteMessage:
		return b.capabilities.CanDeleteMessage
	case CanPinMessage:
		return b.capabilities.CanPinMessage
	case CanReplyMessage:
		return b.capabilities.CanReplyMessage
	case CanForwardMessage:
		return b.capabilities.CanForwardMessage
	case CanMentionUsers:
		return b.capabilities.CanMentionUsers
	case CanMentionAll:
		return b.capabilities.CanMentionAll
	case CanGetChatInfo:
		return b.capabilities.CanGetChatInfo
	case CanGetMembers:
		return b.capabilities.CanGetMembers
	case CanKickMembers:
		return b.capabilities.CanKickMembers
	case CanInviteMembers:
		return b.capabilities.CanInviteMembers
	case SupportsWebhook:
		return b.capabilities.SupportsWebhook
	case SupportsPolling:
		return b.capabilities.SupportsPolling
	case SupportsStreaming:
		return b.capabilities.SupportsStreaming
	default:
		return false
	}
}

// SendText 发送文本消息（默认实现）
// 注意：这个方法需要在具体Channel中实现SendMessage
func (b *BaseChannel) SendText(ctx context.Context, chatID, text string) error {
	_ = chatID
	_ = text
	// 需要由具体Channel实现SendMessage
	return fmt.Errorf("SendMessage not implemented in BaseChannel")
}

// SendMedia 发送媒体消息（默认实现）
// 注意：这个方法需要在具体Channel中实现SendMessage
func (b *BaseChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	_ = chatID
	_ = media
	_ = caption
	// 需要由具体Channel实现SendMessage
	return fmt.Errorf("SendMessage not implemented in BaseChannel")
}

// SendKeyboard 发送带键盘的消息（默认实现）
// 注意：这个方法需要在具体Channel中实现SendMessage
func (b *BaseChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	if !b.capabilities.CanSendKeyboard {
		return fmt.Errorf("channel %s does not support keyboard", b.name)
	}
	_ = chatID
	_ = text
	_ = keyboard
	// 需要由具体Channel实现SendMessage
	return fmt.Errorf("SendMessage not implemented in BaseChannel")
}

// SendMessageWithCheck 发送消息并进行能力检查
// 由具体Channel实现调用，用于检查能力和自动降级
func (b *BaseChannel) SendMessageWithCheck(ctx context.Context, msg *bus.Message, sendFunc func(context.Context, *bus.Message) error) error {
	// 检查消息类型是否支持
	switch msg.Type {
	case bus.MessageTypeText:
		if !b.capabilities.CanSendText {
			return fmt.Errorf("channel %s does not support text messages", b.name)
		}
	case bus.MessageTypeImage:
		if !b.capabilities.CanSendImage {
			// 尝试降级为文件发送
			if b.capabilities.CanSendFile {
				slog.Warn("Channel does not support image, sending as file", "channel", b.name)
				msg.Type = bus.MessageTypeFile
			} else {
				return fmt.Errorf("channel %s does not support image messages", b.name)
			}
		}
	case bus.MessageTypeVideo:
		if !b.capabilities.CanSendVideo {
			// 尝试降级为文件发送
			if b.capabilities.CanSendFile {
				slog.Warn("Channel does not support video, sending as file", "channel", b.name)
				msg.Type = bus.MessageTypeFile
			} else {
				return fmt.Errorf("channel %s does not support video messages", b.name)
			}
		}
	case bus.MessageTypeAudio, bus.MessageTypeVoice:
		if !b.capabilities.CanSendAudio {
			return fmt.Errorf("channel %s does not support audio messages", b.name)
		}
	case bus.MessageTypeFile:
		if !b.capabilities.CanSendFile {
			return fmt.Errorf("channel %s does not support file messages", b.name)
		}
	case bus.MessageTypeLocation:
		if !b.capabilities.CanSendLocation {
			return fmt.Errorf("channel %s does not support location messages", b.name)
		}
	case bus.MessageTypeSticker:
		if !b.capabilities.CanSendSticker {
			// 尝试降级为图片发送
			if b.capabilities.CanSendImage {
				slog.Warn("Channel does not support sticker, sending as image", "channel", b.name)
				msg.Type = bus.MessageTypeImage
			} else {
				return fmt.Errorf("channel %s does not support sticker messages", b.name)
			}
		}
	case bus.MessageTypeRichMedia:
		if !b.capabilities.CanSendRichMedia {
			return fmt.Errorf("channel %s does not support rich media messages", b.name)
		}
	}

	// 检查消息引用（回复、转发）
	if msg.Reference != nil && !b.capabilities.CanReplyMessage {
		slog.Warn("Channel does not support message reference, removing it", "channel", b.name)
		msg.Reference = nil
	}

	// 检查键盘
	if msg.Keyboard != nil && !b.capabilities.CanSendKeyboard {
		slog.Warn("Channel does not support keyboard, removing it", "channel", b.name)
		msg.Keyboard = nil
	}

	// 检查@功能
	if len(msg.Entities) > 0 && !b.capabilities.CanMentionUsers {
		slog.Warn("Channel does not support mentions, removing entities", "channel", b.name)
		msg.Entities = nil
	}

	// 调用具体的发送函数
	return sendFunc(ctx, msg)
}

// Monitor 通用监控循环，处理重连和保活
// 由具体Channel在Start中启动
func (b *BaseChannel) Monitor(ctx context.Context, reconnectFunc func()) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			slog.Debug("Stopping channel monitor", "channel", b.name)
			return
		case <-b.reconnectCh:
			slog.Info("Reconnecting to channel", "channel", b.name)
			time.Sleep(5 * time.Second)
			if reconnectFunc != nil {
				reconnectFunc()
			}
		case <-ticker.C:
			slog.Debug("Channel keepalive", "channel", b.name)
		case <-ctx.Done():
			slog.Debug("Context done in monitor", "channel", b.name)
			return
		}
	}
}

// TriggerReconnect 触发重连
func (b *BaseChannel) TriggerReconnect() {
	select {
	case b.reconnectCh <- struct{}{}:
		slog.Debug("Reconnect triggered", "channel", b.name)
	default:
		slog.Debug("Reconnect already pending", "channel", b.name)
	}
}

// Stop 停止Channel（默认实现）
func (b *BaseChannel) Stop() error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.mu.Unlock()

	close(b.stopCh)
	slog.Info("Channel stopped", "channel", b.name)
	return nil
}

// IsRunning 检查Channel是否正在运行
func (b *BaseChannel) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SetRunning 设置运行状态
func (b *BaseChannel) SetRunning(running bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = running
}

// 以下方法提供默认实现，如果不支持则返回错误

// EditMessage 编辑消息（默认实现，不支持）
func (b *BaseChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	if !b.capabilities.CanEditMessage {
		return fmt.Errorf("channel %s does not support editing messages", b.name)
	}
	return fmt.Errorf("EditMessage not implemented")
}

// DeleteMessage 删除消息（默认实现，不支持）
func (b *BaseChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	if !b.capabilities.CanDeleteMessage {
		return fmt.Errorf("channel %s does not support deleting messages", b.name)
	}
	return fmt.Errorf("DeleteMessage not implemented")
}

// PinMessage 置顶消息（默认实现，不支持）
func (b *BaseChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	if !b.capabilities.CanPinMessage {
		return fmt.Errorf("channel %s does not support pinning messages", b.name)
	}
	return fmt.Errorf("PinMessage not implemented")
}

// UnpinMessage 取消置顶消息（默认实现，不支持）
func (b *BaseChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	if !b.capabilities.CanPinMessage {
		return fmt.Errorf("channel %s does not support unpinning messages", b.name)
	}
	return fmt.Errorf("UnpinMessage not implemented")
}

// AnswerCallback 回调查询（默认实现，不支持）
func (b *BaseChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return fmt.Errorf("AnswerCallback not implemented")
}

// GetChatInfo 获取聊天信息（默认实现，不支持）
func (b *BaseChannel) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	if !b.capabilities.CanGetChatInfo {
		return nil, fmt.Errorf("channel %s does not support getting chat info", b.name)
	}
	return nil, fmt.Errorf("GetChatInfo not implemented")
}

// GetChatMembers 获取聊天成员（默认实现，不支持）
func (b *BaseChannel) GetChatMembers(ctx context.Context, chatID string) ([]ChatMember, error) {
	if !b.capabilities.CanGetMembers {
		return nil, fmt.Errorf("channel %s does not support getting chat members", b.name)
	}
	return nil, fmt.Errorf("GetChatMembers not implemented")
}
