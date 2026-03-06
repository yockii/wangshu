package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
)

// NewFeishuChannel 创建一个新的飞书渠道
func NewFeishuChannel(name, appID, appSecret string) *FeishuChannel {
	c := &FeishuChannel{
		name:         name,
		appID:        appID,
		appSecret:    appSecret,
		stopCh:       make(chan struct{}, 1),
		reconnectCh:  make(chan struct{}, 1),
		groupHistory: make(map[string][]*bus.InboundMessage),
		groupUsers:   sync.Map{},
	}

	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			go c.handleMessage(event)
			return nil
		}).
		OnP2CardActionTrigger(func(ctx context.Context, event *larkcallback.CardActionTriggerEvent) (*larkcallback.CardActionTriggerResponse, error) {
			go c.handleCardAction(event)
			// 返回空响应表示已处理
			return &larkcallback.CardActionTriggerResponse{}, nil
		})
	c.wsClient = larkws.NewClient(
		c.appID,
		c.appSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	c.restClient = lark.NewClient(
		c.appID,
		c.appSecret,
		lark.WithLogLevel(larkcore.LogLevelDebug),
		lark.WithLogReqAtDebug(true),
	)

	return c
}

// SetWorkspace 设置工作空间目录
func (c *FeishuChannel) SetWorkspace(workspace string) {
	c.workspace = workspace
}

// FeishuChannel 飞书渠道实现
type FeishuChannel struct {
	name        string
	appID       string
	appSecret   string
	workspace   string // 工作空间目录
	wsClient    *larkws.Client
	restClient  *lark.Client
	stopCh      chan struct{}
	reconnectCh chan struct{}

	groupMu      sync.RWMutex
	groupHistory map[string][]*bus.InboundMessage // 群聊chat_id -> 最近10条消息列表
	groupUsers   sync.Map                         // map[string]map[string]string // 群聊chat_id -> 用户open_id -> 用户名

	cardCallbacks sync.Map // callback token -> chatID 映射

	openID        string
	channelStatus int
}

// Start 启动飞书渠道
func (c *FeishuChannel) Start() error {
	// 加载群成员缓存
	if err := c.loadGroupUsersFromFile(); err != nil {
		slog.Warn("Failed to load group users cache", "error", err)
		// 不阻塞启动，继续执行
	}

	// 获取机器人的openID
	if err := c.getBotOpenID(); err != nil {
		return err
	}

	go c.connectToFeishu()
	go c.monitor()
	return nil
}

// Stop 停止飞书渠道
func (c *FeishuChannel) Stop() error {
	// 安全地发送停止信号
	select {
	case c.stopCh <- struct{}{}:
		// 成功发送
	default:
		// 已经停止
	}

	// 注意：不等待goroutine结束，让它们自然退出
	// 在实际使用中，websocket客户端会在context取消时自动清理
	return nil
}

// connectToFeishu 连接到飞书WebSocket
func (c *FeishuChannel) connectToFeishu() {
	if err := c.wsClient.Start(context.Background()); err != nil {
		slog.Error("FeishuChannel connectToFeishu error", "err", err)
		c.reconnectCh <- struct{}{}
	}
}

// monitor 监控WebSocket连接状态
func (c *FeishuChannel) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			slog.Debug("stop feishu channel")
			return
		case <-c.reconnectCh:
			slog.Debug("reconnect to feishu")
			time.Sleep(5 * time.Second)
			c.connectToFeishu()
		case <-ticker.C:
			slog.Debug("keepalive feishu ws")
		}
	}
}

// SubscribeOutbound 订阅出站消息
func (c *FeishuChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

// sendMsg 发送消息
func (c *FeishuChannel) sendMsg(ctx context.Context, chatID, content string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(content).
			Build()).
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
	return nil
}

// sendInteractive 发送交互式卡片消息
func (c *FeishuChannel) sendInteractive(ctx context.Context, chatID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType("interactive").
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.restClient.Im.V1.Message.Create(ctx, req)
	if err != nil {
		slog.Error("Feishu Channel sendInteractive error", "err", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel sendInteractive error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return resp.CodeError
	}
	return nil
}

// GetName 返回channel名称
func (c *FeishuChannel) GetName() string {
	return c.name
}

// Capabilities 返回FeishuChannel的能力
func (c *FeishuChannel) Capabilities() channel.ChannelCapabilities {
	return channel.ChannelCapabilities{
		// 发送能力
		CanSendText:      true,
		CanSendImage:     true,
		CanSendFile:      true,
		CanSendRichMedia: true, // 飞书卡片

		// 接收能力
		CanReceiveText:  true,
		CanReceiveImage: true,
		CanReceiveFile:  true,

		// 消息操作
		CanDeleteMessage: true,
		CanReplyMessage:  true, // 通过引用实现
		CanMentionUsers:  true, // @人

		// 聊天能力
		CanGetChatInfo: true,
		CanGetMembers:  true,

		// 连接方式
		SupportsStreaming: true, // WebSocket
	}
}

// Supports 检查是否支持某个能力
func (c *FeishuChannel) Supports(capability channel.ChannelCapability) bool {
	switch capability {
	case channel.CanSendText, channel.CanSendImage, channel.CanSendFile, channel.CanSendRichMedia:
		return true
	case channel.CanReceiveText, channel.CanReceiveImage, channel.CanReceiveFile:
		return true
	case channel.CanDeleteMessage, channel.CanReplyMessage, channel.CanMentionUsers:
		return true
	case channel.CanGetChatInfo, channel.CanGetMembers, channel.SupportsStreaming:
		return true
	default:
		return false
	}
}

// SendText 发送文本消息
func (c *FeishuChannel) SendText(ctx context.Context, chatID, text string) error {
	msg := bus.NewOutboundMessage(chatID, text)
	return c.SendMessage(ctx, &msg)
}

// SendMedia 发送媒体消息
func (c *FeishuChannel) SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error {
	msg := bus.NewOutboundMessage(chatID, caption)
	msg.Media = media
	return c.SendMessage(ctx, &msg)
}

// EditMessage 编辑消息（飞书不支持）
func (c *FeishuChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return fmt.Errorf("FeishuChannel does not support editing messages")
}

// DeleteMessage 删除消息（撤回消息）
func (c *FeishuChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	req := larkim.NewDeleteMessageReqBuilder().
		MessageId(messageID).
		Build()

	resp, err := c.restClient.Im.V1.Message.Delete(ctx, req)
	if err != nil {
		slog.Error("Feishu Channel DeleteMessage error", "messageId", messageID, "error", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel DeleteMessage error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return resp.CodeError
	}

	slog.Info("Feishu message deleted", "messageId", messageID)
	return nil
}

// PinMessage 置顶消息（飞书不支持）
func (c *FeishuChannel) PinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("FeishuChannel does not support pinning messages")
}

// UnpinMessage 取消置顶消息（飞书不支持）
func (c *FeishuChannel) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	return fmt.Errorf("FeishuChannel does not support unpinning messages")
}

// GetChatInfo 获取聊天信息
func (c *FeishuChannel) GetChatInfo(ctx context.Context, chatID string) (*channel.ChatInfo, error) {
	req := larkim.NewGetChatReqBuilder().
		ChatId(chatID).
		Build()

	resp, err := c.restClient.Im.V1.Chat.Get(ctx, req)
	if err != nil {
		slog.Error("Feishu Channel GetChatInfo error", "chatId", chatID, "error", err)
		return nil, err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel GetChatInfo error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return nil, resp.CodeError
	}

	// 尝试从RawBody解析
	type ChatData struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Chat struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ChatType    string `json:"chat_type"`
		} `json:"chat"`
	}

	var chatData ChatData
	if err := json.Unmarshal(resp.RawBody, &chatData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat info: %w", err)
	}

	chatInfo := &channel.ChatInfo{
		ID:    chatID,
		Title: chatData.Chat.Name,
	}

	// 解析聊天类型
	switch chatData.Chat.ChatType {
	case "p2p", "ptype":
		chatInfo.Type = channel.ChatTypePrivate
	case "group", "public":
		chatInfo.Type = channel.ChatTypeGroup
	default:
		chatInfo.Type = channel.ChatTypeGroup
	}

	chatInfo.Description = chatData.Chat.Description

	return chatInfo, nil
}

// GetChatMembers 获取聊天成员
func (c *FeishuChannel) GetChatMembers(ctx context.Context, chatID string) ([]channel.ChatMember, error) {
	// 使用已有的方法获取成员
	memberMap := make(map[string]string)
	if err := c.getAllGroupMembers(chatID, "", memberMap); err != nil {
		return nil, err
	}

	// 转换为ChatMember数组
	members := make([]channel.ChatMember, 0, len(memberMap))
	for openID, name := range memberMap {
		member := channel.ChatMember{
			ID:   openID,
			Name: name,
			Role: channel.MemberRoleMember, // 飞书API不直接返回角色信息
		}
		members = append(members, member)
	}

	return members, nil
}
