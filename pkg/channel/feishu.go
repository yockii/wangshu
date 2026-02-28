package channel

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/yockii/yoclaw/pkg/bus"
)

func NewFeishuChannel(name, appID, appSecret string) *FeishuChannel {
	c := &FeishuChannel{
		name:        name,
		appID:       appID,
		appSecret:   appSecret,
		stopCh:      make(chan struct{}, 1),
		reconnectCh: make(chan struct{}, 1),
	}

	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			go c.handleMessage(event)
			return nil
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

type FeishuChannel struct {
	name        string
	appID       string
	appSecret   string
	wsClient    *larkws.Client
	restClient  *lark.Client
	stopCh      chan struct{}
	reconnectCh chan struct{}
}

func (c *FeishuChannel) Start() error {
	go c.connectToFeishu()
	go c.monitor()
	return nil
}

func (c *FeishuChannel) Stop() error {
	c.stopCh <- struct{}{}
	return nil
}

func (c *FeishuChannel) SendMessage(ctx context.Context, chatID, message string) error {

	type BodyText struct {
		Text string `json:"text"`
	}

	body := BodyText{
		Text: message,
	}

	bodyContent, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(bodyContent)).
			Build()).
		Build()

	resp, err := c.restClient.Im.V1.Message.Create(ctx, req)
	if err != nil {
		slog.Error("Feishu Channel SendMessage error", "err", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel SendMessage error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
	}

	return nil
}

func (c *FeishuChannel) connectToFeishu() {
	if err := c.wsClient.Start(context.Background()); err != nil {
		slog.Error("FeishuChannel connectToFeishu error", "err", err)
		c.reconnectCh <- struct{}{}
	}
}

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

func (c *FeishuChannel) handleMessage(event *larkim.P2MessageReceiveV1) {
	senderID := ""
	if event.Event.Sender.SenderId.OpenId != nil {
		senderID = *event.Event.Sender.SenderId.OpenId
	}

	chatID := ""
	if event.Event.Message.ChatId != nil {
		chatID = *event.Event.Message.ChatId
	}

	// Parse message content from Feishu JSON format
	contentPtr := event.Event.Message.Content
	content := ""
	if contentPtr != nil {
		content = *contentPtr
	}

	bus.Default().PublishInbound(bus.InboundMessage{
		Channel:  c.name,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  content,
	})
}

func (c *FeishuChannel) SubscribeOutbound(ctx context.Context, msg bus.OutboundMessage) {
	if msg.Channel == c.name {
		c.SendMessage(ctx, msg.ChatID, msg.Content)
	}
}
