package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	"github.com/yockii/wangshu/pkg/bus"
)

// handleCardAction 处理卡片交互事件
func (c *FeishuChannel) handleCardAction(event *callback.CardActionTriggerEvent) {
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

// SendKeyboard 发送键盘消息（飞书使用交互式卡片实现）
// 将通用的Keyboard格式转换为飞书的interactive card格式
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
				"tag":  "button",
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
