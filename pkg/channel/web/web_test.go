package web

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/channel"
)

func TestNewWebChannel(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "token123")

	if c.GetName() != "test-web" {
		t.Errorf("Expected name 'test-web', got %s", c.GetName())
	}

	if c.name != "test-web" {
		t.Errorf("Expected internal name 'test-web', got %s", c.name)
	}

	if c.hostAddress != "localhost:8080" {
		t.Errorf("Expected hostAddress 'localhost:8080', got %s", c.hostAddress)
	}

	if c.token != "token123" {
		t.Errorf("Expected token 'token123', got %s", c.token)
	}
}

func TestWebChannelCapabilities(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")
	capabilities := c.Capabilities()

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
	c := NewWebChannel("test-web", "localhost:8080", "")

	tests := []struct {
		name      string
		capability channel.ChannelCapability
		want      bool
	}{
		{"SendText", channel.CanSendText, true},
		{"ReceiveText", channel.CanReceiveText, true},
		{"SupportsStreaming", channel.SupportsStreaming, true},
		{"SendImage", channel.CanSendImage, false},
		{"SendVideo", channel.CanSendVideo, false},
		{"SendAudio", channel.CanSendAudio, false},
		{"SendFile", channel.CanSendFile, false},
		{"SendLocation", channel.CanSendLocation, false},
		{"SendSticker", channel.CanSendSticker, false},
		{"SendRichMedia", channel.CanSendRichMedia, false},
		{"SendKeyboard", channel.CanSendKeyboard, false},
		{"EditMessage", channel.CanEditMessage, false},
		{"DeleteMessage", channel.CanDeleteMessage, false},
		{"PinMessage", channel.CanPinMessage, false},
		{"ReplyMessage", channel.CanReplyMessage, false},
		{"MentionUsers", channel.CanMentionUsers, false},
		{"MentionAll", channel.CanMentionAll, false},
		{"GetChatInfo", channel.CanGetChatInfo, false},
		{"GetMembers", channel.CanGetMembers, false},
		{"SupportsWebhook", channel.SupportsWebhook, false},
		{"SupportsPolling", channel.SupportsPolling, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Supports(tt.capability)
			if got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.capability, got, tt.want)
			}
		})
	}
}

func TestWebChannelSendText(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.SendText(context.Background(), "chat123", "Hello, World!")
	if err != nil {
		t.Errorf("SendText should succeed (not connected), got %v", err)
	}
}

func TestWebChannelSendMedia(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	media := &bus.MediaContent{
		Type:     bus.MediaTypeImage,
		FilePath: "/tmp/test.jpg",
	}

	err := c.SendMedia(context.Background(), "chat123", media, "test image")
	if err == nil {
		t.Error("SendMedia should return error (not supported)")
	}

	expected := "WebChannel does not support sending media"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelEditMessage(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.EditMessage(context.Background(), "chat123", "msg456", "new content")
	if err == nil {
		t.Error("EditMessage should return error (not supported)")
	}

	expected := "WebChannel does not support editing messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelDeleteMessage(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.DeleteMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("DeleteMessage should return error (not supported)")
	}

	expected := "WebChannel does not support deleting messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelPinMessage(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.PinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("PinMessage should return error (not supported)")
	}

	expected := "WebChannel does not support pinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelUnpinMessage(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.UnpinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("UnpinMessage should return error (not supported)")
	}

	expected := "WebChannel does not support unpinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelSendKeyboard(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	keyboard := &bus.Keyboard{
		Inline: true,
		Rows: []bus.KeyboardRow{
			{Buttons: []bus.KeyboardButton{{Text: "OK", Data: "ok"}}},
		},
	}

	err := c.SendKeyboard(context.Background(), "chat123", "Choose", keyboard)
	if err == nil {
		t.Error("SendKeyboard should return error (not supported)")
	}

	expected := "WebChannel does not support keyboard"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelAnswerCallback(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	err := c.AnswerCallback(context.Background(), "callback123", "response")
	if err == nil {
		t.Error("AnswerCallback should return error (not supported)")
	}

	expected := "WebChannel does not support callback"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelGetChatInfo(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	_, err := c.GetChatInfo(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatInfo should return error (not supported)")
	}

	expected := "WebChannel does not support getting chat info"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelGetChatMembers(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	_, err := c.GetChatMembers(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatMembers should return error (not supported)")
	}

	expected := "WebChannel does not support getting chat members"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestWebChannelStop(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	// 多次停止应该是安全的
	err := c.Stop()
	if err != nil {
		t.Errorf("First Stop should not return error, got %v", err)
	}

	err = c.Stop()
	if err != nil {
		t.Errorf("Second Stop should not return error, got %v", err)
	}
}

func TestWebChannelSendMessageNotConnected(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	// 未连接时发送应该只记录警告，不返回错误
	err := c.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage should not return error when not connected, got %v", err)
	}
}

func TestWebChannelSubscribeOutbound(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "test-web",
			ChatID:  "chat123",
		},
	}

	// 应该调用SendMessage
	c.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestWebChannelSubscribeOutboundDifferentChannel(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	msg := bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			Channel: "other-channel",  // 不同的channel
			ChatID:  "chat123",
		},
	}

	// 应该不处理其他channel的消息
	c.SubscribeOutbound(context.Background(), msg)
	// 只是验证不会panic
}

func TestWebChannelEmptyToken(t *testing.T) {
	c := NewWebChannel("test-web", "localhost:8080", "")

	if c.token != "" {
		t.Errorf("Expected empty token, got '%s'", c.token)
	}

	// 空token应该也是有效的
	if c.GetName() != "test-web" {
		t.Error("Channel name should be correct")
	}
}

func TestWebChannelLongToken(t *testing.T) {
	longToken := "this_is_a_very_long_token_that_contains_many_characters_and_numbers_123456789"
	c := NewWebChannel("test-web", "localhost:8080", longToken)

	if c.token != longToken {
		t.Error("Token should be stored correctly")
	}
}

func TestWebChannelEmptyHostAddress(t *testing.T) {
	c := NewWebChannel("test-web", "", "")

	if c.hostAddress != "" {
		t.Errorf("Expected empty hostAddress, got '%s'", c.hostAddress)
	}
}

func TestWebChannelDifferentNames(t *testing.T) {
	names := []string{"web-1", "web_prod", "webTest", "123web"}

	for _, name := range names {
		c := NewWebChannel(name, "localhost:8080", "")
		if c.GetName() != name {
			t.Errorf("Expected name '%s', got '%s'", name, c.GetName())
		}
	}
}
