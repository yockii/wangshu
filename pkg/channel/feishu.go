package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/yockii/wangshu/pkg/bus"
)

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

type FeishuChannel struct {
	name        string
	appID       string
	appSecret   string
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
// msgType为"interactive"，content为卡片JSON
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

// convertMentionsToAtTags 将文本中的@用户名转换为<at user_id="open_id"></at>格式
// 支持格式: @用户名 后跟空格或行尾
func (c *FeishuChannel) convertMentionsToAtTags(chatID, text string) string {
	// 正则匹配 @用户名 格式（包括后面的空格）
	// 另外再单独处理行尾的情况
	re := regexp.MustCompile(`@[\p{Han}a-zA-Z0-9_]+ `)

	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	// 获取群聊用户映射
	var userMap map[string]string
	if val, ok := c.groupUsers.Load(chatID); ok {
		userMap = val.(map[string]string)
	}

	// 如果没有用户映射，尝试获取
	if userMap == nil {
		userMap = make(map[string]string)
		if err := c.getAllGroupMembers(chatID, "", userMap); err != nil {
			slog.Warn("Feishu Channel convertMentionsToAtTags: failed to get group members", "chatID", chatID, "error", err)
			// 如果获取失败，返回原始文本
			return text
		}
		c.groupUsers.Store(chatID, userMap)
	}

	// 创建反向映射：用户名 -> open_id
	nameToOpenID := make(map[string]string)
	for openID, name := range userMap {
		nameToOpenID[name] = openID
	}

	result := text
	// 从后向前替换，避免索引变化问题
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]

		start, end := match[0], match[1]
		mention := text[start:end]

		// 提取用户名（去掉@和尾部空格）
		userName := strings.TrimPrefix(mention, "@")
		userName = strings.TrimRight(userName, " ")

		// 查找对应的 open_id
		if openID, found := nameToOpenID[userName]; found {
			// 计算用户名的实际结束位置（@用户名 部分，不包括后面的空格）
			userNameEnd := start + 1 + len(userName)
			atTag := fmt.Sprintf("<at user_id=\"%s\"></at>", openID)
			result = result[:start] + atTag + result[userNameEnd:]
			slog.Debug("Feishu Channel converted mention", "userName", userName, "openID", openID, "mention", mention, "atTag", atTag)
		} else {
			slog.Debug("Feishu Channel mention not found in user list", "userName", userName, "mention", mention)
		}
	}

	return result
}

func (c *FeishuChannel) SendMessage(ctx context.Context, om *bus.Message) error {
	// 发送文本内容
	if om.Content != "" {
		type BodyText struct {
			Text string `json:"text"`
		}

		// 处理@用户
		// 1. 首先处理文本中的@用户名（如 @张三），转换为 <at> 标签
		// 2. 然后处理 Entities 中的 @用户（如果有的话）
		content := om.Content

		// 如果是群聊消息，尝试转换文本中的@用户名
		if _, ok := c.groupUsers.Load(om.Metadata.ChatID); ok {
			content = c.convertMentionsToAtTags(om.Metadata.ChatID, content)
		}

		// 处理 Entities 中的 @用户（如果有明确的 open_id）
		var atText strings.Builder
		for _, entity := range om.Entities {
			if entity.Type == bus.EntityTypeMention && entity.UserID != "" {
				atText.WriteString(fmt.Sprintf("<at user_id=\"%s\"></at>", entity.UserID))
			}
		}

		body := BodyText{
			Text: content,
		}

		// 将 Entities 中的 @标签插入到文本前
		if atText.Len() > 0 {
			atText.WriteString(body.Text)
			body.Text = atText.String()
		}

		bodyContent, err := json.Marshal(body)
		if err != nil {
			return err
		}

		bodyBuilder := larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(om.Metadata.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(bodyContent))

		// 注：飞书SDK当前版本不直接支持通过Builder设置mentions和parent_id
		// 这些功能需要：
		// 1. @用户：已在文本内容中插入<at>标签
		// 2. 回复消息：需要通过其他API或直接在content中引用
		// 这里我们保持基础实现，如果需要高级功能，可以考虑直接调用HTTP API

		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(larkim.ReceiveIdTypeChatId).
			Body(bodyBuilder.Build()).
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
	}

	// 发送媒体（如果有）
	if om.Media != nil {
		if om.Media.Type == bus.MediaTypeImage {
			key, err := c.uploadImage(om.Media.FilePath)
			if err != nil {
				slog.Error("Feishu Channel upload image error", "err", err)
			} else {
				body := struct {
					ImageKey string `json:"image_key"`
				}{
					ImageKey: key,
				}
				bodyContent, err := json.Marshal(body)
				if err != nil {
					slog.Error("Feishu Channel upload image error", "err", err)
				} else {
					err = c.sendMsg(ctx, om.Metadata.ChatID, string(bodyContent))
					if err != nil {
						slog.Error("Feishu Channel send image error", "err", err)
					}
				}
			}
		} else if om.Media.Type == bus.MediaTypeFile {
			key, err := c.uploadFile(om.Media.FilePath)
			if err != nil {
				slog.Error("Feishu Channel upload file error", "err", err)
			} else {
				body := struct {
					FileKey string `json:"file_key"`
				}{
					FileKey: key,
				}
				bodyContent, err := json.Marshal(body)
				if err != nil {
					slog.Error("Feishu Channel upload file error", "err", err)
				} else {
					err = c.sendMsg(ctx, om.Metadata.ChatID, string(bodyContent))
					if err != nil {
						slog.Error("Feishu Channel send file error", "err", err)
					}
				}
			}
		}
	}

	return nil
}

func (c *FeishuChannel) uploadImage(p string) (string, error) {
	file, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer file.Close()
	req := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType(`message`).
			Image(file).
			Build()).
		Build()
	resp, err := c.restClient.Im.V1.Image.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", resp.CodeError
	}
	return *resp.Data.ImageKey, nil
}

func (c *FeishuChannel) uploadFile(p string) (string, error) {
	file, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer file.Close()
	req := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileName(file.Name()).
			File(file).
			Build()).
		Build()
	resp, err := c.restClient.Im.V1.File.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", resp.CodeError
	}
	return *resp.Data.FileKey, nil
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
	inboundMsg := &bus.InboundMessage{
		Message: bus.Message{
			Content: "",
			Metadata: bus.MessageMetadata{
				Channel: c.name,
			},
		},
	}

	if event.Event.Sender.SenderId.OpenId != nil {
		inboundMsg.Metadata.SenderID = *event.Event.Sender.SenderId.OpenId
	}

	if event.Event.Message.ChatId != nil {
		inboundMsg.Metadata.ChatID = *event.Event.Message.ChatId
	}

	if event.Event.Message.MessageId != nil {
		msgID := *event.Event.Message.MessageId
		// 检查历史消息是否已经接收过
		c.groupMu.RLock()
		history := c.groupHistory[inboundMsg.Metadata.ChatID]
		c.groupMu.RUnlock()
		for _, msg := range history {
			if msgID == msg.Metadata.MessageID {
				slog.Debug("Feishu Channel handleMessage error", "err", "message already received")
				return
			}
		}
		inboundMsg.Metadata.MessageID = msgID
	}

	// Parse message content from Feishu JSON format
	if event.Event.Message.Content != nil {
		inboundMsg.Content = *event.Event.Message.Content
	}

	if event.Event.Message.MessageType != nil {
		msgType := *event.Event.Message.MessageType

		// 根据实际消息类型设置正确的消息类型
		var busMsgType bus.MessageType
		switch msgType {
		case "text":
			busMsgType = bus.MessageTypeText
		case "post":
			busMsgType = bus.MessageTypeRichMedia
		case "image":
			busMsgType = bus.MessageTypeImage
		case "file":
			busMsgType = bus.MessageTypeFile
		case "audio":
			busMsgType = bus.MessageTypeAudio
		case "video":
			busMsgType = bus.MessageTypeVideo
		default:
			busMsgType = bus.MessageTypeText
		}
		inboundMsg.Type = busMsgType
	}

	chatTypePtr := event.Event.Message.ChatType
	chatType := ""
	if chatTypePtr != nil {
		chatType = *chatTypePtr
	}
	if chatType == "p2p" {
		inboundMsg.Content = c.dealReceivedMessage(inboundMsg.Type, inboundMsg.Content)
	} else { // group
		// 看看哪个用户发的，获取用户名
		senderName := c.getSenderName(inboundMsg.Metadata.ChatID, inboundMsg.Metadata.SenderID)
		if senderName == "" {
			slog.Error("Feishu Channel handleMessage error", "err", "sender name not found")
			return
		}

		inboundMsg.Metadata.SenderName = senderName

		inboundMsg.Content = c.dealReceivedMessage(inboundMsg.Type, inboundMsg.Content)
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
			c.groupMu.RLock()
			historyLen := len(c.groupHistory[inboundMsg.Metadata.ChatID])
			c.groupMu.RUnlock()
			if historyLen == 0 {
				c.getGroupHistory(inboundMsg.Metadata.ChatID)
			}
			// 构造消息内容，将历史十条消息拼接起来
			c.groupMu.RLock()
			history := c.groupHistory[inboundMsg.Metadata.ChatID]
			c.groupMu.RUnlock()
			historyContent := ""
			for _, msg := range history {
				historyContent += fmt.Sprintf("%s: %s\n", msg.Metadata.SenderName, msg.Content)
			}
			inboundMsg.Content = fmt.Sprintf("最近10条消息:\n%s\n当前消息(提到了你):%s\n**有些信息可能与提到你时要你完成的任务无关，仅作为参考**", historyContent, fmt.Sprintf("%s: %s", senderName, inboundMsg.Content))
		} else {
			// 将消息保留到最近10条
			c.groupMu.Lock()
			defer c.groupMu.Unlock()
			if c.groupHistory == nil {
				c.groupHistory = make(map[string][]*bus.InboundMessage)
			}
			c.groupHistory[inboundMsg.Metadata.ChatID] = append(c.groupHistory[inboundMsg.Metadata.ChatID], inboundMsg)
			if len(c.groupHistory[inboundMsg.Metadata.ChatID]) > 10 {
				c.groupHistory[inboundMsg.Metadata.ChatID] = c.groupHistory[inboundMsg.Metadata.ChatID][1:]
			}
			return
		}
	}

	bus.Default().PublishInbound(*inboundMsg)
}

// handleCardAction 处理卡片交互事件
func (c *FeishuChannel) handleCardAction(event *larkcallback.CardActionTriggerEvent) {
	token := ""
	if event.Event != nil && event.Event.Token != "" {
		token = event.Event.Token
	}

	// 解析回调数据
	var actionValue map[string]interface{}
	if event.Event != nil && event.Event.Action != nil {
		// 尝试将Action.Value转换为map
		// 注意：这里需要根据实际的CallBackAction结构来解析
		// 暂时使用空map
		actionValue = make(map[string]interface{})
	}

	callbackData, err := json.Marshal(actionValue)
	if err != nil {
		slog.Error("Feishu Channel handleCardAction error", "err", err)
		return
	}

	// 尝试获取chatID（从action value中）
	chatID := ""
	if actionValue != nil {
		if chatIDVal, ok := actionValue["chat_id"].(string); ok {
			chatID = chatIDVal
		}
	}

	// 保存token到chatID的映射
	if chatID != "" && token != "" {
		c.cardCallbacks.Store(token, chatID)
	}

	// 发布到总线，让上层处理
	// 使用MessageID字段存储callback token，因为这是我们可以自由使用的字段
	bus.Default().PublishInbound(bus.InboundMessage{
		Message: bus.Message{
			Type:    bus.MessageTypeText,
			Content: string(callbackData),
			Metadata: bus.MessageMetadata{
				MessageID: token, // 使用MessageID字段存储callback token
				ChatID:    chatID,
				Channel:   c.name,
			},
		},
	})

	slog.Info("Feishu card action received", "token", token, "data", string(callbackData))
}

func (c *FeishuChannel) SubscribeOutbound(ctx context.Context, msg bus.Message) {
	if msg.Metadata.Channel == c.name {
		c.SendMessage(ctx, &msg)
	}
}

func (c *FeishuChannel) getSenderName(chatID, senderID string) string {
	if val, ok := c.groupUsers.Load(chatID); ok {
		userMap := val.(map[string]string)
		if name, has := userMap[senderID]; has {
			return name
		}
	}

	userMap := make(map[string]string)
	// 如果没有，调用sdk查询
	allMembers := make(map[string]string)
	if err := c.getAllGroupMembers(chatID, "", allMembers); err != nil {
		slog.Error("Feishu Channel getSenderName error", "err", err)
		return ""
	}
	// 遍历成员列表，找到匹配的用户
	name := ""
	for openID, memberName := range allMembers {
		userMap[openID] = memberName
		if openID == senderID {
			name = memberName
		}
	}
	c.groupUsers.Store(chatID, userMap)

	return name
}

func (c *FeishuChannel) getAllGroupMembers(chatID string, pageToken string, result map[string]string) error {
	req := larkim.NewGetChatMembersReqBuilder().
		ChatId(chatID).
		MemberIdType("open_id").
		PageSize(100).
		PageToken(pageToken).
		Build()
	resp, err := c.restClient.Im.V1.ChatMembers.Get(context.Background(), req)
	if err != nil {
		slog.Error("Fetch Feishu Group Member Failed", "error", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel getSenderName error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return resp.CodeError
	}

	// 遍历成员列表
	for _, member := range resp.Data.Items {
		if member.MemberId != nil && member.Name != nil {
			openID := *member.MemberId
			result[openID] = *member.Name
		}
	}

	if resp.Data.HasMore != nil && *resp.Data.HasMore && resp.Data.PageToken != nil && *resp.Data.PageToken != "" {
		return c.getAllGroupMembers(chatID, *resp.Data.PageToken, result)
	}
	return nil
}

func (c *FeishuChannel) getGroupHistory(chatID string) error {
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("chat").
		ContainerId(chatID).
		SortType("ByCreateTimeDesc").
		PageSize(20).
		Build()
	resp, err := c.restClient.Im.V1.Message.List(context.Background(), req)
	if err != nil {
		slog.Error("Fetch Feishu Group Message Failed", "error", err)
		return err
	}

	if !resp.Success() {
		slog.Error("Feishu Channel getGroupHistory error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
		return resp.CodeError
	}

	// 遍历消息列表
	var msgs []*bus.InboundMessage
	for i, message := range resp.Data.Items {
		if i == 0 {
			continue
		}
		if len(msgs) > 9 {
			break
		}

		if message.Body != nil && message.Sender != nil && message.Sender.Id != nil && message.Sender.SenderType != nil {
			if message.MsgType == nil {
				continue
			}

			busMsg := &bus.InboundMessage{
				Message: bus.Message{
					Content: "",
					Metadata: bus.MessageMetadata{
						ChatID:   chatID,
						Channel:  c.name,
						SenderID: *message.Sender.Id,
					},
				},
			}

			switch *message.MsgType {
			case "text":
				busMsg.Type = bus.MessageTypeText
			case "post":
				busMsg.Type = bus.MessageTypeRichMedia
			case "image":
				busMsg.Type = bus.MessageTypeImage
			case "file":
				busMsg.Type = bus.MessageTypeFile
			case "audio":
				busMsg.Type = bus.MessageTypeAudio
			case "video":
				busMsg.Type = bus.MessageTypeVideo
			default:
				busMsg.Type = bus.MessageTypeText
			}

			busMsg.Content = c.dealReceivedMessage(busMsg.Type, *message.Body.Content)
			if busMsg.Content == "" {
				continue
			}

			if busMsg.Metadata.SenderID != "" {
				busMsg.Metadata.SenderName = c.getSenderName(busMsg.Metadata.ChatID, busMsg.Metadata.SenderID)
			}

			msgs = append(msgs, busMsg)
		}
	}
	// 将msgs倒一下
	slices.Reverse(msgs)
	c.groupMu.Lock()
	c.groupHistory[chatID] = msgs
	c.groupMu.Unlock()
	return nil
}

func (c *FeishuChannel) dealReceivedMessage(msgType bus.MessageType, content string) string {
	switch msgType {
	case bus.MessageTypeText:
		body := struct {
			Text string `json:"text"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}
		return body.Text
	case bus.MessageTypeRichMedia:
		body := struct {
			Title   string `json:"title"`
			Content [][]struct {
				Tag      string `json:"tag"`
				Text     string `json:"text"`
				Href     string `json:"href"`
				Language string `json:"language"`
			} `json:"content"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}

		var content strings.Builder
		content.WriteString(body.Title)
		for _, item := range body.Content {
			for _, item := range item {
				content.WriteString("\n")
				switch item.Tag {
				case "text", "md":
					content.WriteString(item.Text)
				case "a":
					content.WriteString("[")
					content.WriteString(item.Text)
					content.WriteString("](")
					content.WriteString(item.Href)
					content.WriteString(")")
				case "code_block":
					content.WriteString("```")
					content.WriteString(item.Language)
					content.WriteString("\n")
					content.WriteString(item.Text)
					content.WriteString("```")
				}
			}
		}

		return content.String()
	case bus.MessageTypeImage:
		body := struct {
			ImageKey string `json:"image_key"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}
		return fmt.Sprintf("[图片: %s]", body.ImageKey)
	case bus.MessageTypeFile:
		body := struct {
			FileKey  string `json:"file_key"`
			FileName string `json:"file_name"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}
		if body.FileName != "" {
			return fmt.Sprintf("[文件: %s]", body.FileName)
		}
		return fmt.Sprintf("[文件: %s]", body.FileKey)
	case bus.MessageTypeAudio:
		body := struct {
			FileKey  string `json:"file_key"`
			Duration int    `json:"duration"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}
		if body.Duration > 0 {
			return fmt.Sprintf("[音频: %ds]", body.Duration)
		}
		return "[音频]"
	case bus.MessageTypeVideo:
		body := struct {
			FileKey  string `json:"file_key"`
			Duration int    `json:"duration"`
		}{}
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			slog.Error("Feishu Channel dealReceivedMessage error", "err", err)
			return ""
		}
		if body.Duration > 0 {
			return fmt.Sprintf("[视频: %ds]", body.Duration)
		}
		return "[视频]"
	default:
		return ""
	}
}

// GetName 返回channel名称
func (c *FeishuChannel) GetName() string {
	return c.name
}

// Capabilities 返回FeishuChannel的能力
func (c *FeishuChannel) Capabilities() ChannelCapabilities {
	return ChannelCapabilities{
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
func (c *FeishuChannel) Supports(capability ChannelCapability) bool {
	switch capability {
	case CanSendText, CanSendImage, CanSendFile, CanSendRichMedia:
		return true
	case CanReceiveText, CanReceiveImage, CanReceiveFile:
		return true
	case CanDeleteMessage, CanReplyMessage, CanMentionUsers:
		return true
	case CanGetChatInfo, CanGetMembers, SupportsStreaming:
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
// 飞书支持删除/撤回消息，使用DELETE /im/v1/messages/:message_id
// 文档: https://open.feishu.cn/document/server-docs/im-message/delete_message
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

// SendKeyboard 发送键盘消息（飞书使用交互式卡片实现）
// 将通用的Keyboard格式转换为飞书的interactive card格式
// 文档: https://open.feishu.cn/document/feishu-cards/feishu-card-overview
func (c *FeishuChannel) SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error {
	// 构建飞书卡片
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": text,
				"tag":     "plain_text",
			},
		},
	}

	// 将键盘按钮转换为卡片元素
	elements := make([]map[string]interface{}, 0, len(keyboard.Rows))

	for _, row := range keyboard.Rows {
		if len(row.Buttons) == 0 {
			continue
		}

		// 如果一行有多个按钮，使用action元素
		actions := make([]map[string]interface{}, 0, len(row.Buttons))
		for _, btn := range row.Buttons {
			action := map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"content": btn.Text,
					"tag":     "plain_text",
				},
				"type": "default",
			}

			// 设置按钮类型
			if btn.URL != "" {
				// URL按钮
				action["type"] = "default"
				action["url"] = btn.URL
			} else if btn.Data != "" {
				// 回调按钮
				action["type"] = "primary"
				action["value"] = map[string]interface{}{
					"data": btn.Data,
				}
			}

			actions = append(actions, action)
		}

		// 将按钮添加到元素中
		if len(actions) == 1 {
			elements = append(elements, map[string]interface{}{
				"tag":     "action",
				"actions": actions,
			})
		} else if len(actions) > 1 {
			// 多个按钮放在一个action中
			elements = append(elements, map[string]interface{}{
				"tag":     "action",
				"actions": actions,
			})
		}
	}

	card["elements"] = elements

	// 转换为JSON
	cardJSON, err := json.Marshal(card)
	if err != nil {
		slog.Error("Feishu Channel SendKeyboard marshal card error", "err", err)
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	// 发送卡片
	return c.sendInteractive(ctx, chatID, string(cardJSON))
}

// AnswerCallback 回调查询
// 飞书支持卡片按钮回调，可以通过发送消息来响应用户操作
// callbackID是飞书返回的事件中的token
func (c *FeishuChannel) AnswerCallback(ctx context.Context, callbackID, text string) error {
	slog.Info("Feishu AnswerCallback", "callbackID", callbackID, "response", text)

	// 尝试从存储的映射中获取chatID
	chatIDValue, ok := c.cardCallbacks.Load(callbackID)
	if !ok {
		slog.Warn("Feishu AnswerCallback: callbackID not found", "callbackID", callbackID)
		return fmt.Errorf("callbackID not found: %s", callbackID)
	}

	chatID, ok := chatIDValue.(string)
	if !ok || chatID == "" {
		slog.Warn("Feishu AnswerCallback: invalid chatID", "callbackID", callbackID)
		c.cardCallbacks.Delete(callbackID)
		return fmt.Errorf("invalid chatID for callbackID: %s", callbackID)
	}

	// 确保清理已处理的回调映射
	defer c.cardCallbacks.Delete(callbackID)

	// 发送文本消息作为响应
	if err := c.SendText(ctx, chatID, text); err != nil {
		slog.Error("Feishu AnswerCallback: failed to send response", "err", err)
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// GetChatInfo 获取聊天信息
// 使用GET /im/v1/chats/:chat_id
// 文档: https://open.feishu.cn/document/server-docs/im-group/chat/get_info
func (c *FeishuChannel) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
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

	chatInfo := &ChatInfo{
		ID:    chatID,
		Title: chatData.Chat.Name,
	}

	// 解析聊天类型
	switch chatData.Chat.ChatType {
	case "p2p", "ptype":
		chatInfo.Type = ChatTypePrivate
	case "group", "public":
		chatInfo.Type = ChatTypeGroup
	default:
		chatInfo.Type = ChatTypeGroup
	}

	chatInfo.Description = chatData.Chat.Description

	return chatInfo, nil
}

// GetChatMembers 获取聊天成员
// 使用已有的getAllGroupMembers方法
func (c *FeishuChannel) GetChatMembers(ctx context.Context, chatID string) ([]ChatMember, error) {
	// 使用已有的方法获取成员
	memberMap := make(map[string]string)
	if err := c.getAllGroupMembers(chatID, "", memberMap); err != nil {
		return nil, err
	}

	// 转换为ChatMember数组
	members := make([]ChatMember, 0, len(memberMap))
	for openID, name := range memberMap {
		member := ChatMember{
			ID:   openID,
			Name: name,
			Role: MemberRoleMember, // 飞书API不直接返回角色信息
		}
		members = append(members, member)
	}

	return members, nil
}
