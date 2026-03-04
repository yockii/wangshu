package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
)

func TestNewSession(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session-id",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		SenderID:  "test-sender",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	if session.ID != "test-session-id" {
		t.Errorf("Expected ID 'test-session-id', got '%s'", session.ID)
	}

	if session.Channel != "test-channel" {
		t.Errorf("Expected Channel 'test-channel', got '%s'", session.Channel)
	}

	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(session.Messages))
	}
}

func TestSession_AddMessage(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加消息
	session.AddMessage(constant.RoleUser, "Hello, world!")

	if len(session.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(session.Messages))
	}

	if session.Messages[0].Role != constant.RoleUser {
		t.Errorf("Expected role '%s', got '%s'", constant.RoleUser, session.Messages[0].Role)
	}

	if session.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", session.Messages[0].Content)
	}

	// 验证文件被创建
	sessionFile := filepath.Join(tmpDir, constant.DirSessions, session.Channel, session.ChatID+constant.ExtJSONL)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("Session file should be created")
	}
}

func TestSession_AddMultipleMessages(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加多条消息
	session.AddMessage(constant.RoleUser, "First message")
	session.AddMessage(constant.RoleAssistant, "Second message")
	session.AddMessage(constant.RoleUser, "Third message")

	if len(session.Messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(session.Messages))
	}

	// 验证消息顺序
	if session.Messages[0].Content != "First message" {
		t.Errorf("First message incorrect, got '%s'", session.Messages[0].Content)
	}

	if session.Messages[1].Content != "Second message" {
		t.Errorf("Second message incorrect, got '%s'", session.Messages[1].Content)
	}

	if session.Messages[2].Content != "Third message" {
		t.Errorf("Third message incorrect, got '%s'", session.Messages[2].Content)
	}
}

func TestSession_AddMessageWithToolCalls(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加带工具调用的消息
	toolCalls := []types.ToolCall{
		{
			ID:        "tool-1",
			Name:      "test_tool",
			Arguments: `{"param1":"value1"}`,
		},
	}

	session.AddMessage(constant.RoleAssistant, "Using tool", toolCalls...)

	if len(session.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(session.Messages))
	}

	if len(session.Messages[0].ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(session.Messages[0].ToolCalls))
	}

	if session.Messages[0].ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", session.Messages[0].ToolCalls[0].Name)
	}
}

func TestSession_GetMessages(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 测试空消息
	messages := session.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(messages))
	}

	// 添加消息后获取
	session.AddMessage(constant.RoleUser, "Test message")
	messages = session.GetMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// 验证返回的是副本，不是引用
	messages[0].Content = "Modified"
	if session.Messages[0].Content == "Modified" {
		t.Error("GetMessages should return a copy, not reference")
	}
}

func TestSession_GetLastN(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加5条消息
	for i := 1; i <= 5; i++ {
		session.AddMessage(constant.RoleUser, fmt.Sprintf("Message %d", i))
	}

	// 获取最后3条
	last3 := session.GetLastN(3)
	if len(last3) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(last3))
	}

	// 获取超过总数的消息
	all := session.GetLastN(10)
	if len(all) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(all))
	}

	// 获取0条消息
	zero := session.GetLastN(0)
	if len(zero) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(zero))
	}

	// 获取负数消息
	negative := session.GetLastN(-1)
	if len(negative) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(negative))
	}
}

func TestSession_Clear(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加消息
	session.AddMessage(constant.RoleUser, "Test message")
	session.AddMessage(constant.RoleAssistant, "Response")

	if len(session.Messages) != 2 {
		t.Fatalf("Expected 2 messages before clear, got %d", len(session.Messages))
	}

	// 清空消息
	session.Clear()

	if len(session.Messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(session.Messages))
	}
}

func TestSession_Metadata(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 设置元数据
	session.SetMetadata("key1", "value1")
	session.SetMetadata("key2", "value2")

	// 获取元数据
	if session.GetMetadata("key1") != "value1" {
		t.Errorf("Expected 'value1', got '%s'", session.GetMetadata("key1"))
	}

	if session.GetMetadata("key2") != "value2" {
		t.Errorf("Expected 'value2', got '%s'", session.GetMetadata("key2"))
	}

	// 获取不存在的键
	if session.GetMetadata("nonexistent") != "" {
		t.Errorf("Expected empty string for nonexistent key, got '%s'", session.GetMetadata("nonexistent"))
	}

	// 删除元数据
	session.DeleteMetadata("key1")
	if session.GetMetadata("key1") != "" {
		t.Error("key1 should be deleted")
	}

	if session.GetMetadata("key2") != "value2" {
		t.Error("key2 should still exist")
	}
}

func TestSession_IsExpired(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 刚更新的会话不应该过期
	if session.IsExpired(1 * time.Hour) {
		t.Error("Fresh session should not be expired")
	}

	// 修改 UpdatedAt 为过去
	oldTime := time.Now().Add(-2 * time.Hour)
	session.UpdatedAt = oldTime

	// 现在应该过期
	if !session.IsExpired(1 * time.Hour) {
		t.Error("Old session should be expired")
	}

	// 但不应该过期于更长的TTL
	if session.IsExpired(3 * time.Hour) {
		t.Error("Session should not be expired with longer TTL")
	}
}

func TestSession_Touch(t *testing.T) {
	tmpDir := t.TempDir()

	oldTime := time.Now().Add(-1 * time.Hour)

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
		workspace: tmpDir,
	}

	// 会话应该过期
	if !session.IsExpired(30 * time.Minute) {
		t.Error("Old session should be expired before touch")
	}

	// Touch 会话
	session.Touch()

	// 现在不应该过期
	if session.IsExpired(30 * time.Minute) {
		t.Error("Touched session should not be expired")
	}
}

func TestSession_LoadMessage(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 先保存一些消息
	session.AddMessage(constant.RoleUser, "Message 1")
	session.AddMessage(constant.RoleAssistant, "Response 1")

	// 创建新会话并加载
	newSession := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	err := newSession.loadMessage()
	if err != nil {
		t.Fatalf("Failed to load messages: %v", err)
	}

	if len(newSession.Messages) != 2 {
		t.Errorf("Expected 2 loaded messages, got %d", len(newSession.Messages))
	}

	if newSession.Messages[0].Content != "Message 1" {
		t.Errorf("Expected 'Message 1', got '%s'", newSession.Messages[0].Content)
	}
}

func TestSession_LoadMessage_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "nonexistent-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 加载不存在的文件应该返回nil
	err := session.loadMessage()
	if err != nil {
		t.Errorf("Loading nonexistent file should succeed, got: %v", err)
	}
}

func TestSession_TrimMessages(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加10条消息
	for i := 1; i <= 10; i++ {
		session.AddMessage(constant.RoleUser, "Message "+fmt.Sprintf("%d", i))
	}

	// 修剪为保留最后3条，加上摘要
	trimmed := session.TrimMessages("Summary of conversation", 3)

	if len(trimmed) != 4 { // 1个摘要 + 3条历史
		t.Errorf("Expected 4 messages (1 summary + 3 history), got %d", len(trimmed))
	}

	if trimmed[0].Role != constant.RoleAssistant {
		t.Errorf("First message should be assistant (summary), got '%s'", trimmed[0].Role)
	}

	if trimmed[0].Content != "Summary of conversation" {
		t.Errorf("Summary content incorrect, got '%s'", trimmed[0].Content)
	}

	// 验证最后3条是原始消息的最后3条
	if trimmed[1].Content != "Message 8" {
		t.Errorf("Expected 'Message 8', got '%s'", trimmed[1].Content)
	}

	if trimmed[3].Content != "Message 10" {
		t.Errorf("Expected 'Message 10', got '%s'", trimmed[3].Content)
	}

	// 验证会话中的消息也被更新
	if len(session.Messages) != 4 {
		t.Errorf("Session should have 4 messages after trim, got %d", len(session.Messages))
	}
}

func TestSession_TrimMessages_NoTrim(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 添加3条消息
	for i := 1; i <= 3; i++ {
		session.AddMessage(constant.RoleUser, "Message "+fmt.Sprintf("%d", i))
	}

	// 尝试修剪为保留5条（比当前多）
	trimmed := session.TrimMessages("Summary", 5)

	if len(trimmed) != 3 {
		t.Errorf("Expected 3 messages (no trim), got %d", len(trimmed))
	}
}

func TestSession_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// 并发添加消息
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			session.AddMessage(constant.RoleUser, fmt.Sprintf("Concurrent message %d", idx))
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证消息数量
	if len(session.Messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(session.Messages))
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		go func() {
			_ = session.GetMessages()
			_ = session.GetLastN(5)
			_ = session.GetMetadata("test")
		}()
	}

	// 等待读取完成
	time.Sleep(100 * time.Millisecond)
}

func TestSession_MetadataInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: tmpDir,
	}

	// Metadata 初始为 nil
	if session.Metadata != nil {
		t.Error("Metadata should be nil initially")
	}

	// 设置元数据后应该初始化
	session.SetMetadata("key", "value")
	if session.Metadata == nil {
		t.Error("Metadata should be initialized after SetMetadata")
	}

	if session.Metadata["key"] != "value" {
		t.Errorf("Expected metadata 'key' to be 'value', got '%s'", session.Metadata["key"])
	}
}

func TestSession_AddMessageUpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	oldTime := time.Now().Add(-1 * time.Hour)

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
		workspace: tmpDir,
	}

	// 添加消息应该更新 UpdatedAt
	session.AddMessage(constant.RoleUser, "Test")

	if session.UpdatedAt.Before(oldTime) || session.UpdatedAt.Equal(oldTime) {
		t.Error("UpdatedAt should be updated after adding message")
	}
}

func TestSession_SetMetadataUpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	oldTime := time.Now().Add(-1 * time.Hour)

	session := &Session{
		ID:        "test-session",
		Channel:   "test-channel",
		ChatID:    "test-chat",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
		workspace: tmpDir,
	}

	// 设置元数据应该更新 UpdatedAt
	session.SetMetadata("key", "value")

	if session.UpdatedAt.Before(oldTime) || session.UpdatedAt.Equal(oldTime) {
		t.Error("UpdatedAt should be updated after setting metadata")
	}
}
