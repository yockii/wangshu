package channel

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/bus"
)

func TestNewFeishuChannel(t *testing.T) {
	channel := NewFeishuChannel("test-feishu", "app_id_123", "app_secret_456")

	if channel.GetName() != "test-feishu" {
		t.Errorf("Expected name 'test-feishu', got %s", channel.GetName())
	}

	if channel.appID != "app_id_123" {
		t.Errorf("Expected appID 'app_id_123', got %s", channel.appID)
	}

	if channel.appSecret != "app_secret_456" {
		t.Errorf("Expected appSecret 'app_secret_456', got %s", channel.appSecret)
	}
}

func TestFeishuChannelCapabilities(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")
	capabilities := channel.Capabilities()

	// 发送能力
	if !capabilities.CanSendText {
		t.Error("FeishuChannel should support sending text")
	}
	if !capabilities.CanSendImage {
		t.Error("FeishuChannel should support sending image")
	}
	if !capabilities.CanSendFile {
		t.Error("FeishuChannel should support sending file")
	}
	if !capabilities.CanSendRichMedia {
		t.Error("FeishuChannel should support sending rich media")
	}

	// 接收能力
	if !capabilities.CanReceiveText {
		t.Error("FeishuChannel should support receiving text")
	}
	if !capabilities.CanReceiveImage {
		t.Error("FeishuChannel should support receiving image")
	}
	if !capabilities.CanReceiveFile {
		t.Error("FeishuChannel should support receiving file")
	}

	// 消息操作
	if !capabilities.CanDeleteMessage {
		t.Error("FeishuChannel should support deleting messages")
	}
	if !capabilities.CanReplyMessage {
		t.Error("FeishuChannel should support replying messages")
	}
	if !capabilities.CanMentionUsers {
		t.Error("FeishuChannel should support mentioning users")
	}

	// 聊天能力
	if !capabilities.CanGetChatInfo {
		t.Error("FeishuChannel should support getting chat info")
	}
	if !capabilities.CanGetMembers {
		t.Error("FeishuChannel should support getting members")
	}

	// 连接方式
	if !capabilities.SupportsStreaming {
		t.Error("FeishuChannel should support streaming")
	}

	// 不支持的能力
	if capabilities.CanSendVideo {
		t.Error("FeishuChannel should not support sending video")
	}
	if capabilities.CanSendAudio {
		t.Error("FeishuChannel should not support sending audio")
	}
	if capabilities.CanEditMessage {
		t.Error("FeishuChannel should not support editing messages")
	}
	if capabilities.CanPinMessage {
		t.Error("FeishuChannel should not support pinning messages")
	}
}

func TestFeishuChannelSupports(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	tests := []struct {
		name      string
		capability ChannelCapability
		want      bool
	}{
		// 支持的能力
		{"SendText", CanSendText, true},
		{"SendImage", CanSendImage, true},
		{"SendFile", CanSendFile, true},
		{"SendRichMedia", CanSendRichMedia, true},
		{"ReceiveText", CanReceiveText, true},
		{"ReceiveImage", CanReceiveImage, true},
		{"ReceiveFile", CanReceiveFile, true},
		{"DeleteMessage", CanDeleteMessage, true},
		{"ReplyMessage", CanReplyMessage, true},
		{"MentionUsers", CanMentionUsers, true},
		{"GetChatInfo", CanGetChatInfo, true},
		{"GetMembers", CanGetMembers, true},
		{"SupportsStreaming", SupportsStreaming, true},

		// 不支持的能力
		{"SendVideo", CanSendVideo, false},
		{"SendAudio", CanSendAudio, false},
		{"SendLocation", CanSendLocation, false},
		{"SendSticker", CanSendSticker, false},
		{"EditMessage", CanEditMessage, false},
		{"PinMessage", CanPinMessage, false},
		{"ReplyMessage", CanReplyMessage, true}, // 飞书支持回复
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.Supports(tt.capability)
			if got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.capability, got, tt.want)
			}
		})
	}
}

func TestFeishuChannelSendText(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 没有真实的app_id和app_secret，发送会失败，但我们可以测试代码路径
	err := channel.SendText(context.Background(), "chat123", "Hello, Feishu!")
	// 预期会返回错误（因为没有真实凭证），但不应该panic
	if err == nil {
		t.Log("SendText succeeded (unexpected, might have valid credentials)")
	}
}

func TestFeishuChannelSendMedia(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	media := &bus.MediaContent{
		Type:     bus.MediaTypeImage,
		FilePath: "/tmp/test.jpg",
	}

	err := channel.SendMedia(context.Background(), "chat123", media, "test image")
	// 预期会返回错误（文件不存在或没有凭证）
	if err == nil {
		t.Log("SendMedia succeeded (unexpected)")
	}
}

func TestFeishuChannelEditMessage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.EditMessage(context.Background(), "chat123", "msg456", "new content")
	if err == nil {
		t.Error("EditMessage should return error (not supported)")
	}

	expected := "FeishuChannel does not support editing messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestFeishuChannelDeleteMessage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.DeleteMessage(context.Background(), "chat123", "msg456")
	// 会返回错误（没有真实凭证），但不应该是"not implemented"
	if err == nil {
		t.Log("DeleteMessage succeeded (unexpected)")
	}

	// 确保错误消息不是"not implemented yet"
	if err != nil && err.Error() == "FeishuChannel delete message not implemented yet" {
		t.Error("DeleteMessage should be implemented")
	}
}

func TestFeishuChannelPinMessage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.PinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("PinMessage should return error (not supported)")
	}

	expected := "FeishuChannel does not support pinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestFeishuChannelUnpinMessage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.UnpinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("UnpinMessage should return error (not supported)")
	}

	expected := "FeishuChannel does not support unpinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestFeishuChannelSendKeyboard(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	keyboard := &bus.Keyboard{
		Inline: true,
		Rows: []bus.KeyboardRow{
			{
				Buttons: []bus.KeyboardButton{
					{Text: "是", Data: "yes"},
					{Text: "否", Data: "no"},
				},
			},
		},
	}

	err := channel.SendKeyboard(context.Background(), "chat123", "请选择", keyboard)
	// 会返回错误（没有真实凭证），但不应该是"not implemented yet"
	if err == nil {
		t.Log("SendKeyboard succeeded (unexpected)")
	}

	if err != nil && err.Error() == "FeishuChannel keyboard not implemented yet" {
		t.Error("SendKeyboard should be implemented")
	}
}

func TestFeishuChannelAnswerCallback(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.AnswerCallback(context.Background(), "callback123", "response")
	// AnswerCallback现在返回错误当callbackID未找到
	if err == nil {
		t.Error("AnswerCallback should return error when callbackID not found")
	}
}

func TestFeishuChannelGetChatInfo(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	_, err := channel.GetChatInfo(context.Background(), "chat123")
	// 会返回错误（没有真实凭证），但不应该是"not implemented yet"
	if err == nil {
		t.Log("GetChatInfo succeeded (unexpected)")
	}

	if err != nil && err.Error() == "FeishuChannel get chat info not implemented yet" {
		t.Error("GetChatInfo should be implemented")
	}
}

func TestFeishuChannelGetChatMembers(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	_, err := channel.GetChatMembers(context.Background(), "chat123")
	// 会返回错误（没有真实凭证），但不应该是"not implemented yet"
	if err == nil {
		t.Log("GetChatMembers succeeded (unexpected)")
	}

	if err != nil && err.Error() == "FeishuChannel get chat members not implemented yet" {
		t.Error("GetChatMembers should be implemented")
	}
}

func TestFeishuChannelSendMessageWithText(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证）
	if err == nil {
		t.Log("SendMessage succeeded (unexpected)")
	}
}

func TestFeishuChannelSendMessageWithImage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeImage,
		Content: "test image",
		Media: &bus.MediaContent{
			Type:     bus.MediaTypeImage,
			FilePath: "/tmp/test.jpg",
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（文件不存在或没有凭证）
	if err == nil {
		t.Log("SendMessage with image succeeded (unexpected)")
	}
}

func TestFeishuChannelStop(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 多次停止应该是安全的
	err := channel.Stop()
	if err != nil {
		t.Errorf("First Stop should not return error, got %v", err)
	}

	err = channel.Stop()
	if err != nil {
		t.Errorf("Second Stop should not return error, got %v", err)
	}
}

func TestFeishuChannelEmptyCredentials(t *testing.T) {
	// 测试空凭证
	channel := NewFeishuChannel("test", "", "")
	if channel.appID != "" {
		t.Error("appID should be empty")
	}
	if channel.appSecret != "" {
		t.Error("appSecret should be empty")
	}
}

func TestFeishuChannelSubscribeOutbound(t *testing.T) {
	channel := NewFeishuChannel("test-feishu", "app", "secret")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "test-feishu",
			ChatID:  "chat123",
		},
	}

	// 应该调用SendMessage
	channel.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestFeishuChannelSubscribeOutboundDifferentChannel(t *testing.T) {
	channel := NewFeishuChannel("test-feishu", "app", "secret")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "other-channel", // 不同的channel
			ChatID:  "chat123",
		},
	}

	// 应该不处理其他channel的消息
	channel.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestFeishuChannelSendMessageWithReply(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "reply message",
		Reference: &bus.MessageReference{
			MessageID:     "parent_msg_123",
			ReferenceType: bus.ReferenceTypeReply,
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证），但代码路径应该是正确的
	if err == nil {
		t.Log("SendMessage with reply succeeded (unexpected)")
	}
}

func TestFeishuChannelSendMessageWithMention(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "hello user",
		Entities: []bus.MessageEntity{
			{
				Type:   bus.EntityTypeMention,
				UserID: "user_123",
			},
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证）
	if err == nil {
		t.Log("SendMessage with mention succeeded (unexpected)")
	}
}

func TestFeishuChannelDealReceivedMessageText(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"text":"hello world"}`
	result := channel.dealReceivedMessage("text", content)

	if result != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result)
	}
}

func TestFeishuChannelDealReceivedMessageImage(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"image_key":"img_12345"}`
	result := channel.dealReceivedMessage("image", content)

	expected := "[图片: img_12345]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageFile(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"file_12345","file_name":"test.pdf"}`
	result := channel.dealReceivedMessage("file", content)

	expected := "[文件: test.pdf]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageFileWithoutName(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"file_12345"}`
	result := channel.dealReceivedMessage("file", content)

	expected := "[文件: file_12345]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageAudio(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"audio_123","duration":30}`
	result := channel.dealReceivedMessage("audio", content)

	expected := "[音频: 30s]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageAudioWithoutDuration(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"audio_123"}`
	result := channel.dealReceivedMessage("audio", content)

	expected := "[音频]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageVideo(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"video_123","duration":60}`
	result := channel.dealReceivedMessage("video", content)

	expected := "[视频: 60s]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageVideoWithoutDuration(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"file_key":"video_123"}`
	result := channel.dealReceivedMessage("video", content)

	expected := "[视频]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFeishuChannelDealReceivedMessageUnknownType(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	content := `{"some":"data"}`
	result := channel.dealReceivedMessage("unknown", content)

	if result != "" {
		t.Errorf("Expected empty string for unknown type, got '%s'", result)
	}
}

func TestFeishuChannelAnswerCallbackWithStoredMapping(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 存储一个callback映射
	channel.cardCallbacks.Store("callback_token_123", "chat123")

	err := channel.AnswerCallback(context.Background(), "callback_token_123", "response text")
	// 会返回错误（没有真实凭证），但映射应该被找到
	if err == nil {
		t.Log("AnswerCallback with stored mapping succeeded (unexpected)")
	}

	// 验证映射被清理了
	_, exists := channel.cardCallbacks.Load("callback_token_123")
	if exists {
		t.Error("Callback mapping should be deleted after processing")
	}
}

func TestFeishuChannelAnswerCallbackWithoutStoredMapping(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	err := channel.AnswerCallback(context.Background(), "nonexistent_callback", "response")
	// 应该返回错误（callbackID未找到）
	if err == nil {
		t.Error("AnswerCallback should return error when callbackID not found")
	}

	if err != nil && err.Error() != "callbackID not found: nonexistent_callback" {
		t.Errorf("Expected 'callbackID not found' error, got '%v'", err)
	}
}

func TestFeishuChannelSendMessageWithMultipleMentions(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "hello team",
		Entities: []bus.MessageEntity{
			{
				Type:   bus.EntityTypeMention,
				UserID: "user_123",
			},
			{
				Type:   bus.EntityTypeMention,
				UserID: "user_456",
			},
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证）
	if err == nil {
		t.Log("SendMessage with multiple mentions succeeded (unexpected)")
	}
}

func TestFeishuChannelSendMessageWithReplyAndMention(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "replying with mention",
		Reference: &bus.MessageReference{
			MessageID:     "parent_msg_123",
			ReferenceType: bus.ReferenceTypeReply,
		},
		Entities: []bus.MessageEntity{
			{
				Type:   bus.EntityTypeMention,
				UserID: "user_789",
			},
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证）
	if err == nil {
		t.Log("SendMessage with reply and mention succeeded (unexpected)")
	}
}

func TestFeishuChannelConvertMentionsToAtTags(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 设置群聊用户映射
	userMap := map[string]string{
		"ou_123": "张三",
		"ou_456": "李四",
		"ou_789": "Alice",
	}
	channel.groupUsers.Store("chat123", userMap)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single mention with space",
			input:    "你好 @张三 ",
			expected: "你好 <at user_id=\"ou_123\"></at> ",
		},
		{
			name:     "mention at end without space",
			input:    "你好@张三",
			expected: "你好@张三", // 没有空格，不转换
		},
		{
			name:     "multiple mentions",
			input:    "@张三 和 @李四 ",
			expected: "<at user_id=\"ou_123\"></at> 和 <at user_id=\"ou_456\"></at> ",
		},
		{
			name:     "English name",
			input:    "Hello @Alice ",
			expected: "Hello <at user_id=\"ou_789\"></at> ",
		},
		{
			name:     "no mention",
			input:    "你好世界",
			expected: "你好世界",
		},
		{
			name:     "unknown user",
			input:    "你好 @未知用户 ",
			expected: "你好 @未知用户 ", // 未知用户不会被转换
		},
		{
			name:     "email not matched",
			input:    "我的邮箱是 test@example.com",
			expected: "我的邮箱是 test@example.com", // 邮箱中的@不应该被匹配
		},
		{
			name:     "at symbol in text",
			input:    "价格是 @100 元",
			expected: "价格是 @100 元", // 这里的@后面跟着数字和空格，但用户名是"100"不在列表中
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := channel.convertMentionsToAtTags("chat123", tt.input)
			if result != tt.expected {
				t.Errorf("convertMentionsToAtTags() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestFeishuChannelConvertMentionsToAtTagsNoUserMap(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 没有用户映射的情况
	input := "你好@张三"
	result := channel.convertMentionsToAtTags("chat456", input)

	// 应该返回原始文本（因为无法获取用户列表）
	if result != input {
		t.Errorf("convertMentionsToAtTags() without user map should return original, got %s", result)
	}
}

func TestFeishuChannelSendMessageWithTextMention(t *testing.T) {
	channel := NewFeishuChannel("test", "app", "secret")

	// 设置群聊用户映射
	userMap := map[string]string{
		"ou_123": "张三",
	}
	channel.groupUsers.Store("chat123", userMap)

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "@张三你好",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := channel.SendMessage(context.Background(), msg)
	// 会返回错误（没有真实凭证），但代码路径应该是正确的
	if err == nil {
		t.Log("SendMessage with text mention succeeded (unexpected)")
	}
}

