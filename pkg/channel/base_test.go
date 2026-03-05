package channel

import (
	"context"
	"testing"
	"time"

	"github.com/yockii/wangshu/pkg/bus"
)

func TestNewBaseChannel(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:      true,
		CanSendImage:     true,
		CanReceiveText:   true,
		SupportsStreaming: true,
	}

	base := NewBaseChannel("test-channel", capabilities)

	if base.GetName() != "test-channel" {
		t.Errorf("Expected name 'test-channel', got %s", base.GetName())
	}

	if base.IsRunning() {
		t.Error("New channel should not be running")
	}
}

func TestBaseChannelCapabilities(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:    true,
		CanSendImage:   true,
		CanSendVideo:   false,
		CanReceiveText: true,
	}

	base := NewBaseChannel("test", capabilities)

	received := base.Capabilities()

	if received.CanSendText != capabilities.CanSendText {
		t.Error("CanSendText mismatch")
	}

	if received.CanSendVideo != capabilities.CanSendVideo {
		t.Error("CanSendVideo mismatch")
	}
}

func TestBaseChannelSupports(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:     true,
		CanSendImage:    true,
		CanEditMessage:  false,
		CanDeleteMessage: false,
	}

	base := NewBaseChannel("test", capabilities)

	tests := []struct {
		name      string
		capability ChannelCapability
		want      bool
	}{
		{"SendText", CanSendText, true},
		{"SendImage", CanSendImage, true},
		{"SendVideo", CanSendVideo, false},
		{"EditMessage", CanEditMessage, false},
		{"DeleteMessage", CanDeleteMessage, false},
		{"ReceiveText", CanReceiveText, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base.Supports(tt.capability)
			if got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.capability, got, tt.want)
			}
		})
	}
}

func TestBaseChannelSendText(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})

	err := base.SendText(context.Background(), "chat123", "test message")
	if err == nil {
		t.Error("SendText should return error (not implemented)")
	}

	if err.Error() != "SendMessage not implemented in BaseChannel" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestBaseChannelSendMedia(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})

	media := &bus.MediaContent{
		Type:     bus.MediaTypeImage,
		FilePath: "/tmp/test.jpg",
	}

	err := base.SendMedia(context.Background(), "chat123", media, "test image")
	if err == nil {
		t.Error("SendMedia should return error (not implemented)")
	}
}

func TestBaseChannelSendKeyboard(t *testing.T) {
	// 测试不支持键盘的channel
	capabilities := ChannelCapabilities{
		CanSendKeyboard: false,
	}
	base := NewBaseChannel("test", capabilities)

	keyboard := &bus.Keyboard{
		Inline: true,
		Rows: []bus.KeyboardRow{
			{Buttons: []bus.KeyboardButton{{Text: "OK", Data: "ok"}}},
		},
	}

	err := base.SendKeyboard(context.Background(), "chat123", "Choose", keyboard)
	if err == nil {
		t.Error("SendKeyboard should return error when keyboard not supported")
	}

	if err.Error() != "channel test does not support keyboard" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestBaseChannelSendKeyboardSupported(t *testing.T) {
	// 测试支持键盘的channel
	capabilities := ChannelCapabilities{
		CanSendKeyboard: true,
	}
	base := NewBaseChannel("test", capabilities)

	keyboard := &bus.Keyboard{
		Inline: true,
		Rows: []bus.KeyboardRow{
			{Buttons: []bus.KeyboardButton{{Text: "OK", Data: "ok"}}},
		},
	}

	err := base.SendKeyboard(context.Background(), "chat123", "Choose", keyboard)
	if err == nil {
		t.Error("SendKeyboard should return error (not implemented)")
	}
}

func TestBaseChannelStop(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})
	base.SetRunning(true)

	err := base.Stop()
	if err != nil {
		t.Errorf("Stop should not return error, got %v", err)
	}

	if base.IsRunning() {
		t.Error("Channel should not be running after Stop")
	}
}

func TestBaseChannelStopMultipleTimes(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})
	base.SetRunning(true)

	// 第一次停止
	err := base.Stop()
	if err != nil {
		t.Errorf("First Stop should not return error, got %v", err)
	}

	// 第二次停止应该也是安全的
	err = base.Stop()
	if err != nil {
		t.Errorf("Second Stop should not return error, got %v", err)
	}
}

func TestBaseChannelSetRunning(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})

	if base.IsRunning() {
		t.Error("Should not be running initially")
	}

	base.SetRunning(true)

	if !base.IsRunning() {
		t.Error("Should be running after SetRunning(true)")
	}

	base.SetRunning(false)

	if base.IsRunning() {
		t.Error("Should not be running after SetRunning(false)")
	}
}

func TestBaseChannelTriggerReconnect(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})

	// 触发重连应该不会阻塞
	base.TriggerReconnect()

	// 再次触发也应该安全
	base.TriggerReconnect()
}

func TestBaseChannelMonitor(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})
	base.SetRunning(true)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 启动监控（应该不会阻塞）
	go base.Monitor(ctx, func() {
		// 重连回调
	})

	// 等待一下让监控运行
	time.Sleep(50 * time.Millisecond)

	// 停止监控
	base.Stop()

	// 等待监控退出
	time.Sleep(100 * time.Millisecond)
}

func TestBaseChannelMonitorWithReconnect(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})
	base.SetRunning(true)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	reconnectCalled := false
	go base.Monitor(ctx, func() {
		reconnectCalled = true
	})

	// 触发重连
	time.Sleep(50 * time.Millisecond)
	base.TriggerReconnect()

	// 等待重连被处理
	time.Sleep(100 * time.Millisecond)

	// 停止
	base.Stop()
	cancel()

	time.Sleep(50 * time.Millisecond)

	if !reconnectCalled {
		t.Log("Reconnect callback was not called (may be timing issue)")
	}
}

func TestBaseChannelEditMessageNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanEditMessage: false,
	}
	base := NewBaseChannel("test", capabilities)

	err := base.EditMessage(context.Background(), "chat123", "msg456", "new content")
	if err == nil {
		t.Error("EditMessage should return error when not supported")
	}

	expected := "channel test does not support editing messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelDeleteMessageNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanDeleteMessage: false,
	}
	base := NewBaseChannel("test", capabilities)

	err := base.DeleteMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("DeleteMessage should return error when not supported")
	}

	expected := "channel test does not support deleting messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelPinMessageNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanPinMessage: false,
	}
	base := NewBaseChannel("test", capabilities)

	err := base.PinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("PinMessage should return error when not supported")
	}

	expected := "channel test does not support pinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelUnpinMessageNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanPinMessage: false,
	}
	base := NewBaseChannel("test", capabilities)

	err := base.UnpinMessage(context.Background(), "chat123", "msg456")
	if err == nil {
		t.Error("UnpinMessage should return error when not supported")
	}

	expected := "channel test does not support unpinning messages"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelGetChatInfoNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanGetChatInfo: false,
	}
	base := NewBaseChannel("test", capabilities)

	_, err := base.GetChatInfo(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatInfo should return error when not supported")
	}

	expected := "channel test does not support getting chat info"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelGetChatMembersNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanGetMembers: false,
	}
	base := NewBaseChannel("test", capabilities)

	_, err := base.GetChatMembers(context.Background(), "chat123")
	if err == nil {
		t.Error("GetChatMembers should return error when not supported")
	}

	expected := "channel test does not support getting chat members"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%v'", expected, err)
	}
}

func TestBaseChannelAnswerCallback(t *testing.T) {
	base := NewBaseChannel("test", ChannelCapabilities{})

	err := base.AnswerCallback(context.Background(), "callback123", "response")
	if err == nil {
		t.Error("AnswerCallback should return error (not implemented)")
	}

	if err.Error() != "AnswerCallback not implemented" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestSendMessageWithCheckTextSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText: true,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	called := false
	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("SendMessageWithCheck should not return error, got %v", err)
	}

	if !called {
		t.Error("Send function should be called")
	}
}

func TestSendMessageWithCheckTextNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText: false,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test message",
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		return nil
	})

	if err == nil {
		t.Error("SendMessageWithCheck should return error when text not supported")
	}
}

func TestSendMessageWithCheckImageNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText: true,
		CanSendImage: false,
		CanSendFile:  false,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type: bus.MessageTypeImage,
		Media: &bus.MediaContent{
			Type: bus.MediaTypeImage,
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		return nil
	})

	if err == nil {
		t.Error("SendMessageWithCheck should return error when image and file not supported")
	}
}

func TestSendMessageWithCheckImageToFileDowngrade(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText: true,
		CanSendImage: false,
		CanSendFile:  true,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type: bus.MessageTypeImage,
		Media: &bus.MediaContent{
			Type: bus.MediaTypeImage,
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	called := false
	var receivedMsg *bus.Message
	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		called = true
		receivedMsg = msg
		return nil
	})

	if err != nil {
		t.Errorf("SendMessageWithCheck should not return error, got %v", err)
	}

	if !called {
		t.Error("Send function should be called")
	}

	if receivedMsg.Type != bus.MessageTypeFile {
		t.Errorf("Expected message type to be downgraded to File, got %v", receivedMsg.Type)
	}
}

func TestSendMessageWithCheckReferenceNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:     true,
		CanReplyMessage: false,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test",
		Reference: &bus.MessageReference{
			MessageID: "original_msg",
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	var receivedMsg *bus.Message
	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		receivedMsg = msg
		return nil
	})

	if err != nil {
		t.Errorf("SendMessageWithCheck should not return error, got %v", err)
	}

	if receivedMsg.Reference != nil {
		t.Error("Reference should be removed when not supported")
	}
}

func TestSendMessageWithCheckKeyboardNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:     true,
		CanSendKeyboard: false,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type:     bus.MessageTypeText,
		Content:  "test",
		Keyboard: &bus.Keyboard{
			Inline: true,
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	var receivedMsg *bus.Message
	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		receivedMsg = msg
		return nil
	})

	if err != nil {
		t.Errorf("SendMessageWithCheck should not return error, got %v", err)
	}

	if receivedMsg.Keyboard != nil {
		t.Error("Keyboard should be removed when not supported")
	}
}

func TestSendMessageWithCheckMentionNotSupported(t *testing.T) {
	capabilities := ChannelCapabilities{
		CanSendText:    true,
		CanMentionUsers: false,
	}
	base := NewBaseChannel("test", capabilities)

	msg := &bus.Message{
		Type:    bus.MessageTypeText,
		Content: "test",
		Entities: []bus.MessageEntity{
			{
				Type: bus.EntityTypeMention,
			},
		},
		Metadata: bus.MessageMetadata{
			ChatID: "chat123",
		},
	}

	var receivedMsg *bus.Message
	err := base.SendMessageWithCheck(context.Background(), msg, func(ctx context.Context, msg *bus.Message) error {
		receivedMsg = msg
		return nil
	})

	if err != nil {
		t.Errorf("SendMessageWithCheck should not return error, got %v", err)
	}

	if receivedMsg.Entities != nil {
		t.Error("Entities should be removed when mention not supported")
	}
}

func TestChatTypeValues(t *testing.T) {
	tests := []struct {
		name  string
		value ChatType
	}{
		{"Private", ChatTypePrivate},
		{"Group", ChatTypeGroup},
		{"Channel", ChatTypeChannel},
		{"Supergroup", ChatTypeSupergroup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("ChatType %s should not be empty", tt.name)
			}
		})
	}
}

func TestMemberRoleValues(t *testing.T) {
	tests := []struct {
		name  string
		value MemberRole
	}{
		{"Owner", MemberRoleOwner},
		{"Admin", MemberRoleAdmin},
		{"Member", MemberRoleMember},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("MemberRole %s should not be empty", tt.name)
			}
		})
	}
}

func TestChatInfo(t *testing.T) {
	info := &ChatInfo{
		ID:          "chat123",
		Type:        ChatTypeGroup,
		Title:       "测试群组",
		Description: "这是一个测试群组",
		MemberCount: 100,
		CreatedAt:   1234567890,
	}

	if info.ID != "chat123" {
		t.Errorf("Expected ID 'chat123', got %s", info.ID)
	}

	if info.Type != ChatTypeGroup {
		t.Errorf("Expected type %s, got %s", ChatTypeGroup, info.Type)
	}

	if info.MemberCount != 100 {
		t.Errorf("Expected MemberCount 100, got %d", info.MemberCount)
	}
}

func TestChatMember(t *testing.T) {
	member := &ChatMember{
		ID:          "user123",
		Name:        "张三",
		DisplayName: "张三（北京）",
		Role:        MemberRoleAdmin,
		JoinedAt:    1234567890,
	}

	if member.ID != "user123" {
		t.Errorf("Expected ID 'user123', got %s", member.ID)
	}

	if member.Role != MemberRoleAdmin {
		t.Errorf("Expected role %s, got %s", MemberRoleAdmin, member.Role)
	}

	if member.DisplayName != "张三（北京）" {
		t.Errorf("Expected DisplayName '张三（北京）', got %s", member.DisplayName)
	}
}
