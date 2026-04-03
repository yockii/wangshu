package wechat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	wechatbot "github.com/corespeed-io/wechatbot/golang"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
)

type IlinkChannel struct {
	name       string
	credPath   string
	bot        *wechatbot.Bot
	stopCh     chan struct{}
	running    bool
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	onQRCode   func(url string)
	onScanned  func()
	onExpired  func()
	onError    func(err error)
	onLoggedIn func()

	pendingMessages sync.Map
}

type IlinkOption func(*IlinkChannel)

func WithCredPath(path string) IlinkOption {
	return func(c *IlinkChannel) {
		c.credPath = path
	}
}

func WithOnQRCode(fn func(url string)) IlinkOption {
	return func(c *IlinkChannel) {
		c.onQRCode = fn
	}
}

func WithOnScanned(fn func()) IlinkOption {
	return func(c *IlinkChannel) {
		c.onScanned = fn
	}
}

func WithOnExpired(fn func()) IlinkOption {
	return func(c *IlinkChannel) {
		c.onExpired = fn
	}
}

func WithOnError(fn func(err error)) IlinkOption {
	return func(c *IlinkChannel) {
		c.onError = fn
	}
}

func WithOnLoggedIn(fn func()) IlinkOption {
	return func(c *IlinkChannel) {
		c.onLoggedIn = fn
	}
}

func NewIlinkChannel(name string, opts ...IlinkOption) *IlinkChannel {
	c := &IlinkChannel{
		name:   name,
		stopCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.credPath == "" {
		c.credPath, _ = filepath.Abs(filepath.Join(".wechatbot", fmt.Sprintf("%s_credentials.json", name)))
	}

	return c
}

func (c *IlinkChannel) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.run()

	return nil
}

func (c *IlinkChannel) run() {
	options := wechatbot.Options{
		CredPath: c.credPath,
		LogLevel: "info",
	}

	if c.onQRCode != nil {
		options.OnQRURL = c.onQRCode
	} else {
		options.OnQRURL = func(url string) {
			slog.Info("WeChat iLink QR Code", "url", url)
		}
	}

	if c.onScanned != nil {
		options.OnScanned = c.onScanned
	} else {
		options.OnScanned = func() {
			slog.Info("WeChat iLink scanned, waiting for confirmation")
		}
	}

	if c.onExpired != nil {
		options.OnExpired = c.onExpired
	} else {
		options.OnExpired = func() {
			slog.Warn("WeChat iLink QR code expired")
		}
	}

	if c.onError != nil {
		options.OnError = c.onError
	} else {
		options.OnError = func(err error) {
			slog.Error("WeChat iLink error", "error", err)
		}
	}

	c.bot = wechatbot.New(options)

	creds, err := c.bot.Login(c.ctx, false)
	if err != nil {
		slog.Error("WeChat iLink login failed", "error", err)
		return
	}
	slog.Info("WeChat iLink logged in", "accountID", creds.AccountID)

	if c.onLoggedIn != nil {
		c.onLoggedIn()
	}

	c.bot.OnMessage(c.handleMessage)

	if err := c.bot.Run(c.ctx); err != nil {
		slog.Error("WeChat iLink run error", "error", err)
	}
}

func (c *IlinkChannel) handleMessage(msg *wechatbot.IncomingMessage) {
	inboundMsg := c.convertMessage(msg)
	bus.Default().PublishInbound(inboundMsg)
}

func (c *IlinkChannel) convertMessage(msg *wechatbot.IncomingMessage) bus.InboundMessage {
	inboundMsg := bus.InboundMessage{
		Message: bus.Message{
			Metadata: bus.MessageMetadata{
				Channel:   c.name,
				SenderID:  msg.UserID,
				ChatID:    msg.UserID,
				Timestamp: msg.Timestamp,
			},
		},
	}

	switch msg.Type {
	case "text":
		inboundMsg.Type = bus.MessageTypeText
		inboundMsg.Content = msg.Text
	case "image":
		inboundMsg.Type = bus.MessageTypeImage
		if len(msg.Images) > 0 {
			inboundMsg.Media = &bus.MediaContent{
				Type: bus.MediaTypeImage,
				URL:  msg.Images[0].URL,
			}
		}
	case "voice":
		inboundMsg.Type = bus.MessageTypeVoice
		if len(msg.Voices) > 0 {
			inboundMsg.Content = msg.Voices[0].Text
			inboundMsg.Media = &bus.MediaContent{
				Type:     bus.MediaTypeAudio,
				Duration: msg.Voices[0].DurationMs / 1000,
			}
		}
	case "video":
		inboundMsg.Type = bus.MessageTypeVideo
		if len(msg.Videos) > 0 && msg.Videos[0].Media != nil {
			inboundMsg.Media = &bus.MediaContent{
				Type: bus.MediaTypeVideo,
			}
		}
	case "file":
		inboundMsg.Type = bus.MessageTypeFile
		if len(msg.Files) > 0 {
			inboundMsg.Media = &bus.MediaContent{
				Type:     bus.MediaTypeFile,
				FileName: msg.Files[0].FileName,
				Size:     msg.Files[0].Size,
			}
		}
	default:
		inboundMsg.Type = bus.MessageTypeText
		inboundMsg.Content = msg.Text
	}

	if msg.QuotedMessage != nil {
		inboundMsg.Reference = &bus.MessageReference{
			Content:       msg.QuotedMessage.Title,
			ReferenceType: bus.ReferenceTypeQuote,
		}
	}

	return inboundMsg
}

func (c *IlinkChannel) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	if c.bot != nil {
		c.bot.Stop()
	}

	c.running = false
	return nil
}

func (c *IlinkChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

func (c *IlinkChannel) GetName() string {
	return c.name
}

func (c *IlinkChannel) Capabilities() channel.ChannelCapabilities {
	return channel.ChannelCapabilities{
		CanSendText:      true,
		CanSendImage:     true,
		CanSendVideo:     true,
		CanSendFile:      true,
		CanSendAudio:     false,
		CanReceiveText:   true,
		CanReceiveImage:  true,
		CanReceiveVideo:  true,
		CanReceiveFile:   true,
		CanReceiveAudio:  true,
		CanReplyMessage:  true,
		SupportsPolling:  true,
		CanGetChatInfo:   false,
		CanMentionUsers:  false,
		CanMentionAll:    false,
		CanEditMessage:   false,
		CanDeleteMessage: false,
		CanPinMessage:    false,
	}
}

func (c *IlinkChannel) Supports(capability channel.ChannelCapability) bool {
	caps := c.Capabilities()
	switch capability {
	case channel.CanSendText:
		return caps.CanSendText
	case channel.CanSendImage:
		return caps.CanSendImage
	case channel.CanSendVideo:
		return caps.CanSendVideo
	case channel.CanSendFile:
		return caps.CanSendFile
	case channel.CanReceiveText:
		return caps.CanReceiveText
	case channel.CanReceiveImage:
		return caps.CanReceiveImage
	case channel.CanReceiveVideo:
		return caps.CanReceiveVideo
	case channel.CanReceiveFile:
		return caps.CanReceiveFile
	case channel.CanReceiveAudio:
		return caps.CanReceiveAudio
	case channel.CanReplyMessage:
		return caps.CanReplyMessage
	case channel.SupportsPolling:
		return caps.SupportsPolling
	default:
		return false
	}
}

func (c *IlinkChannel) SendMessage(ctx context.Context, msg *bus.Message) error {
	if c.bot == nil {
		return fmt.Errorf("wechat ilink bot not initialized")
	}

	chatID := msg.Metadata.ChatID
	if chatID == "" {
		return fmt.Errorf("chatID is required")
	}

	if msg.Media != nil {
		return c.sendMediaMessage(ctx, chatID, msg)
	}

	if msg.Content != "" {
		return c.bot.Send(ctx, chatID, msg.Content)
	}

	return fmt.Errorf("no content to send")
}

func (c *IlinkChannel) sendMediaMessage(ctx context.Context, chatID string, msg *bus.Message) error {
	if c.bot == nil {
		return fmt.Errorf("wechat ilink bot not initialized")
	}

	media := msg.Media
	var content wechatbot.SendContent
	hasContent := false

	switch media.Type {
	case bus.MediaTypeImage:
		if len(media.FilePath) > 0 {
			data, err := os.ReadFile(media.FilePath)
			if err != nil {
				return fmt.Errorf("failed to read image file: %w", err)
			}
			content = wechatbot.SendImage(data)
			hasContent = true
		} else if len(media.URL) > 0 {
			return fmt.Errorf("sending image by URL is not supported, please download first")
		}
	case bus.MediaTypeVideo:
		if len(media.FilePath) > 0 {
			data, err := os.ReadFile(media.FilePath)
			if err != nil {
				return fmt.Errorf("failed to read video file: %w", err)
			}
			content = wechatbot.SendVideo(data)
			hasContent = true
		}
	case bus.MediaTypeFile:
		if len(media.FilePath) > 0 {
			data, err := os.ReadFile(media.FilePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			fileName := media.FileName
			if fileName == "" {
				fileName = filepath.Base(media.FilePath)
			}
			content = wechatbot.SendFile(data, fileName)
			hasContent = true
		}
	default:
		return fmt.Errorf("unsupported media type: %s", media.Type)
	}

	if !hasContent {
		return fmt.Errorf("failed to create media content")
	}

	return c.bot.SendMedia(ctx, chatID, content)
}

func (c *IlinkChannel) SendText(ctx context.Context, chatID, text string) error {
	msg := bus.NewOutboundMessage(chatID, text)
	return c.SendMessage(ctx, &msg)
}

func (c *IlinkChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	msg := bus.NewOutboundMessage(chatID, caption)
	msg.Media = media
	return c.SendMessage(ctx, &msg)
}

func (c *IlinkChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return fmt.Errorf("WeChat iLink does not support editing messages")
}

func (c *IlinkChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WeChat iLink does not support deleting messages")
}

func (c *IlinkChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WeChat iLink does not support pinning messages")
}

func (c *IlinkChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("WeChat iLink does not support unpinning messages")
}

func (c *IlinkChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	return fmt.Errorf("WeChat iLink does not support keyboard messages")
}

func (c *IlinkChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return fmt.Errorf("WeChat iLink does not support callback answers")
}

func (c *IlinkChannel) GetChatInfo(ctx context.Context, chatID string) (*channel.ChatInfo, error) {
	return nil, fmt.Errorf("WeChat iLink does not support getting chat info")
}
