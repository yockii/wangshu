package bundle

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/internal/agent"
	"github.com/yockii/wangshu/internal/app"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/store"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
	"github.com/yockii/wangshu/pkg/constant"
)

var DefaultChatBundle = &ChatBundle{
	channel: &BuiltinChannel{},
}

type ChatBundle struct {
	agent   *agent.Agent
	channel *BuiltinChannel
}

func (b *ChatBundle) SetAgent(agent *agent.Agent) {
	b.agent = agent
	channel.RegisterChannel(constant.BuiltinChannelName, b.channel)
	bus.Default().RegisterInboundHandler(constant.BuiltinChannelName, agent.SubscribeInbound)
	bus.Default().RegisterOutboundHandler(b.channel.SubscribeOutbound)

	bus.Default().RegisterEmotionHandler(func(emotion string) {
		if config.DefaultCfg.Live2D.Enabled {
			mapping, err := store.Get[*types.EmotionMapping](constant.StorePrefixEmotionMapping, config.DefaultCfg.Live2D.ModelName)
			if err != nil || mapping == nil {
				return
			}
			m, ok := mapping.Mappings[emotion]
			if !ok {
				return
			}

			if m.MotionGroup != "" {
				app.GetApp().Event.Emit(constant.EventLive2DDoMotion, map[string]any{
					"group": m.MotionGroup,
					"no":    m.MotionNo,
				})
			}
			if m.ExpressionId != "" {
				app.GetApp().Event.Emit(constant.EventLive2DDoExpression, m.ExpressionId)
			}
		}
	})
}

func (b *ChatBundle) HandleMessage(content string) {
	inboundMsg := &bus.InboundMessage{
		Message: bus.Message{
			Content: content,
			Type:    bus.MessageTypeText,
			Metadata: bus.MessageMetadata{
				Channel:    b.channel.GetName(),
				ChatID:     constant.Default,
				ChatName:   constant.Default,
				ChatType:   constant.ChatTypeP2P,
				SenderID:   constant.Default,
				SenderName: constant.Default,
			},
		},
	}
	bus.Default().PublishInbound(*inboundMsg)
}

func (b *ChatBundle) GetHistoryMessages(fetchedLength, length int) []types.Message {
	sess := b.agent.GetChannelSession(
		b.channel.GetName(),
		constant.Default,
		constant.ChatTypeP2P,
		constant.Default,
		constant.Default,
		constant.Default,
	)
	if sess == nil {
		return nil
	}
	msgs := sess.GetLastNBeforeLastL(fetchedLength, length)
	return msgs
}

// --------------------------------------------------------------------------------

type BuiltinChannel struct{}

func (c *BuiltinChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.GetName() {
		c.SendMessage(ctx, &msg)
	}
}

func (c *BuiltinChannel) Start() error    { return nil }
func (c *BuiltinChannel) Stop() error     { return nil }
func (c *BuiltinChannel) GetName() string { return constant.BuiltinChannelName }
func (c *BuiltinChannel) Capabilities() channel.ChannelCapabilities {
	return channel.ChannelCapabilities{
		CanSendText:     true,
		CanReceiveImage: true,
		CanSendVideo:    true,
		CanReceiveText:  true,
	}
}
func (c *BuiltinChannel) Supports(capability channel.ChannelCapability) bool {
	switch capability {
	case channel.CanSendText:
	case channel.CanReceiveImage:
	case channel.CanSendVideo:
	case channel.CanReceiveText:
		return true
	}
	return false
}
func (c *BuiltinChannel) SendMessage(ctx context.Context, msg *bus.Message) error {
	app.GetApp().Event.Emit(constant.EventMessage, *msg)
	return nil
}
func (c *BuiltinChannel) SendText(ctx context.Context, chatID, text string) error {
	msg := bus.NewOutboundMessage(chatID, text)
	return c.SendMessage(ctx, &msg)
}
func (c *BuiltinChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	msg := bus.NewOutboundMessage(chatID, caption)
	msg.Media = media
	return c.SendMessage(ctx, &msg)
}

// 高级操作（如果Channel不支持，返回错误）
func (c *BuiltinChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return fmt.Errorf("BuiltinChannel does not support editing messages")
}
func (c *BuiltinChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("BuiltinChannel does not support deleting messages")
}
func (c *BuiltinChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("BuiltinChannel does not support pinning messages")
}
func (c *BuiltinChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("BuiltinChannel does not support unpinning messages")
}

// 消息交互
func (c *BuiltinChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	return fmt.Errorf("BuiltinChannel does not support sending keyboard messages")
}
func (c *BuiltinChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return fmt.Errorf("BuiltinChannel does not support answering callbacks")
}

// 聊天操作
func (c *BuiltinChannel) GetChatInfo(ctx context.Context, chatID string) (*channel.ChatInfo, error) {
	chatInfo := &channel.ChatInfo{
		ID:    chatID,
		Title: chatID,
		Type:  channel.ChatTypePrivate,
	}
	return chatInfo, nil
}
