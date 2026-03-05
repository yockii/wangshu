package channel

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/bus"
)

func TestNewWebChannel(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "token123")

	if channel.GetName() != "test-web" {
		t.Errorf("Expected name 'test-web', got %s", channel.GetName())
	}

	if channel.name != "test-web" {
		t.Errorf("Expected internal name 'test-web', got %s", channel.name)
	}

	if channel.hostAddress != "localhost:8080" {
		t.Errorf("Expected hostAddress 'localhost:8080', got %s", channel.hostAddress)
	}

	if channel.token != "token123" {
		t.Errorf("Expected token 'token123', got %s", channel.token)
	}
}

func TestWebChannelCapabilities(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")
	capabilities := channel.Capabilities()

	if !capabilities.CanSendText {
		t.Error("WebChannel should support sending text")
	}

	if !capabilities.CanReceiveText {
		t.Error("WebChannel should support receiving text")
	}

	if !capabilities.SupportsStreaming {
		t.Error("WebChannel should support streaming")
	}

	// WebChannel不应该支持的功能
	if capabilities.CanSendImage {
		t.Error("WebChannel should not support sending image")
	}

	if capabilities.CanEditMessage {
		t.Error("WebChannel should not support editing messages")
	}

	if capabilities.CanSendKeyboard {
		t.Error("WebChannel should not support keyboard")
	}
}

func TestWebChannelSupports(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	tests := []struct {
		name      string
		capability ChannelCapability
		want      bool
	}{
		{"SendText", CanSendText, true},
		{"ReceiveText", CanReceiveText, true},
		{"SupportsStreaming", SupportsStreaming, true},
		{"SendImage", CanSendImage, false},
		{"SendVideo", CanSendVideo, false},
		{"SendAudio", CanSendAudio, false},
		{"SendFile", CanSendFile, false},
		{"SendLocation", CanSendLocation, false},
		{"SendSticker", CanSendSticker, false},
		{"SendRichMedia", CanSendRichMedia, false},
		{"SendKeyboard", CanSendKeyboard, false},
		{"EditMessage", CanEditMessage, false},
		{"DeleteMessage", CanDeleteMessage, false},
		{"PinMessage", CanPinMessage, false},
		{"ReplyMessage", CanReplyMessage, false},
		{"MentionUsers", CanMentionUsers, false},
		{"MentionAll", CanMentionAll, false},
		{"GetChatInfo", CanGetChatInfo, false},
		{"GetMembers", CanGetMembers, false},
		{"SupportsWebhook", SupportsWebhook, false},
		{"SupportsPolling", SupportsPolling, false},
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

func TestWebChannelSendText(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.SendText(context.Background(), "chat123", "Hello, World!")
	if err != nil {
		t.Errorf("SendText should succeed (not connected), got %v", err)
	}
}

func TestWebChannelSendMedia(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	media := &bus.MediaContent{
		Type:     bus.MediaTypeImage,
		FilePath: "/tmp/test.jpg",
	}

	err := channel.SendMedia(context.Background(), "chat123", media, "test image")
	if err == nil {
		t.Error("SendMedia should return error (not supported)")
	}

	expected := "WebChannel does not support sending media"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelEditMessage(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.EditMessage(context.Background(), "chat123", "msg456", "new content")
	if err == nil {
		t.Error("EditMessage should return error (not supported)")
	}

	expected := "WebChannel does not support editing messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelDeleteMessage(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.DeleteMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("DeleteMessage should return error (not supported)")
	}

	expected := "WebChannel does not support deleting messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelPinMessage(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.PinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("PinMessage should return error (not supported)")
	}

	expected := "WebChannel does not support pinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelUnpinMessage(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.UnpinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("UnpinMessage should return error (not supported)")
	}

	expected := "WebChannel does not support unpinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelSendKeyboard(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	keyboard := &bus.Keyboard{
		Inline: true,
		Rows: []bus.KeyboardRow{
			{Buttons: []bus.KeyboardButton{{Text: "OK", Data: "ok"}}},
		},
	}

	err := channel.SendKeyboard(context.Background(), "chat123", "Choose", keyboard)
	if err == nil {
		t.Error("SendKeyboard should return error (not supported)")
	}

	expected := "WebChannel does not support keyboard"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelAnswerCallback(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	err := channel.AnswerCallback(context.Background(), "callback123", "response")
	if err == nil {
		t.Error("AnswerCallback should return error (not supported)")
	}

	expected := "WebChannel does not support callback"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelGetChatInfo(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	_, err := channel.GetChatInfo(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatInfo should return error (not supported)")
	}

	expected := "WebChannel does not support getting chat info"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelGetChatMembers(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	_, err := channel.GetChatMembers(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatMembers should return error (not supported)")
	}

	expected := "WebChannel does not support getting chat members"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelStop(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

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

func TestWebChannelSendMessageNotConnected(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	// 未连接时发送应该只记录警告，不返回错误
	err := channel.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage should not return error when not connected, got %v", err)
	}
}

func TestWebChannelSubscribeOutbound(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "test-web",
			ChatID:  "chat123",
		},
	}

	// 应该调用SendMessage
	channel.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestWebChannelSubscribeOutboundDifferentChannel(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "other-channel",  // 不同的channel
			ChatID:  "chat123",
		},
	}

	// 应该不处理其他channel的消息
	channel.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestWebChannelEmptyToken(t *testing.T) {
	channel := NewWebChannel("test-web", "localhost:8080", "")

	if channel.token != "" {
		t.Errorf("Expected empty token, got '%s'", channel.token)
	}

	// 空token应该也是有效的
	if channel.GetName() != "test-web" {
		t.Error("Channel name should be correct")
	}
}

func TestWebChannelLongToken(t *testing.T) {
	longToken := "this_is_a_very_long_token_that_contains_many_characters_and_numbers_123456789"
	channel := NewWebChannel("test-web", "localhost:8080", longToken)

	if channel.token != longToken {
		t.Error("Token should be stored correctly")
	}
}

func TestWebChannelEmptyHostAddress(t *testing.T) {
	channel := NewWebChannel("test-web", "", "")

	if channel.hostAddress != "" {
		t.Errorf("Expected empty hostAddress, got '%s'", channel.hostAddress)
	}
}

func TestWebChannelDifferentNames(t *testing.T) {
	names := []string{"web-1", "web_prod", "webTest", "123web"}

	for _, name := range names {
		channel := NewWebChannel(name, "localhost:8080", "")
		if channel.GetName() != name {
			t.Errorf("Expected name '%s', got '%s'", name, channel.GetName())
		}
	}
}
