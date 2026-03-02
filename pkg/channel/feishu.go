package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/yockii/yoclaw/pkg/bus"
)

func NewFeishuChannel(name, appID, appSecret string) *FeishuChannel {
	c := &FeishuChannel{
		name:         name,
		appID:        appID,
		appSecret:    appSecret,
		stopCh:       make(chan struct{}, 1),
		reconnectCh:  make(chan struct{}, 1),
		groupHistory: make(map[string][]string),
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

	groupHistory map[string][]string // 群聊chat_id -> 最近10条消息列表

	openID        string
	channelStatus int
}

func (c *FeishuChannel) Start() error {
	// 获取机器人的openID
	if err := c.getBotOpenID(); err != nil {
		return err
	}

	go c.connectToFeishu()
	go c.monitor()
	return nil
}

func (c *FeishuChannel) getBotOpenID() error {
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(c.appID).
			AppSecret(c.appSecret).
			Build()).
		Build()
	resp, err := c.restClient.Auth.V3.TenantAccessToken.Internal(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}
	if !resp.Success() {
		return fmt.Errorf("failed to get token: %v", resp.CodeError)
	}
	tat := struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		Expire            int    `json:"expire"`
		TenantAccessToken string `json:"tenant_access_token"`
	}{}
	if err := json.Unmarshal(resp.RawBody, &tat); err != nil {
		return fmt.Errorf("failed to unmarshal token: %v", err)
	}
	token := tat.TenantAccessToken
	if token == "" {
		return fmt.Errorf("tenant_access_token is empty")
	}
	// 获取机器人信息
	httpReq, err := http.NewRequest(http.MethodGet, "https://open.feishu.cn/open-apis/bot/v3/info", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get bot info: %v", httpResp.Status)
	}
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	botInfo := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Bot  struct {
			ActivateStatus int    `json:"activate_status"`
			AppName        string `json:"app_name"`
			OpenID         string `json:"open_id"`
		} `json:"bot"`
	}{}
	if err := json.Unmarshal(bodyBytes, &botInfo); err != nil {
		return fmt.Errorf("failed to unmarshal bot info: %v", err)
	}
	c.openID = botInfo.Bot.OpenID
	if c.openID == "" {
		return fmt.Errorf("open_id is empty")
	}
	c.channelStatus = botInfo.Bot.ActivateStatus
	if c.channelStatus != 2 {
		return fmt.Errorf("bot is not activated")
	}
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

	msgTypePtr := event.Event.Message.MessageType
	msgType := ""
	if msgTypePtr != nil {
		msgType = *msgTypePtr
	}

	chatTypePtr := event.Event.Message.ChatType
	chatType := ""
	if chatTypePtr != nil {
		chatType = *chatTypePtr
	}
	if chatType == "p2p" {
		if msgType == larkim.MsgTypeText {
			body := struct {
				Text string `json:"text"`
			}{}
			if err := json.Unmarshal([]byte(content), &body); err != nil {
				slog.Error("Feishu Channel handleMessage error", "err", err)
				return
			}
			content = body.Text
		}
	} else {
		// group
		if msgType == larkim.MsgTypeText {
			body := struct {
				Text string `json:"text"`
			}{}
			if err := json.Unmarshal([]byte(content), &body); err != nil {
				slog.Error("Feishu Channel handleMessage error", "err", err)
				return
			}
			content = body.Text
			// 去掉所有 @_user_{0-100} 格式的字符串
			content = regexp.MustCompile(`@_user_[0-9]+`).ReplaceAllString(content, "")
		} else {
			return
		}
		// 看看是否@机器人
		methionMe := false
		if len(event.Event.Message.Mentions) > 0 {
			for _, mention := range event.Event.Message.Mentions {
				if mention != nil && mention.Id != nil && mention.Id.OpenId != nil && *mention.Id.OpenId == c.openID {
					methionMe = true
					break
				}
			}
		}
		if methionMe {
			// 构造消息内容，将历史十条消息拼接起来
			content = fmt.Sprintf("最近10条消息:\n%s\n当前消息:%s", strings.Join(c.groupHistory[chatID], "\n"), content)
		} else {
			// 将消息保留到最近10条
			if c.groupHistory == nil {
				c.groupHistory = make(map[string][]string)
			}
			c.groupHistory[chatID] = append(c.groupHistory[chatID], content)
			if len(c.groupHistory[chatID]) > 10 {
				c.groupHistory[chatID] = c.groupHistory[chatID][1:]
			}
			return
		}
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
