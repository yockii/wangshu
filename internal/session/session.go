package session

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yockii/yoclaw/pkg/constant"
)

type Session struct {
	ID        string
	Channel   string
	ChatID    string
	SenderID  string
	Messages  []Message
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
	workspace string
	mu        sync.RWMutex
}

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
	ToolCalls []ToolCall
}
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
	Result    string
}

// func (s *Session) AddMessage(role, content string, toolCallID string, toolCalls ...ToolCall) {
func (s *Session) AddMessage(role, content string, toolCalls ...ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
		// ToolCallID: toolCallID,
	}

	s.Messages = append(s.Messages, msg)

	s.UpdatedAt = time.Now()

	s.saveMessage(msg)
}
func (s *Session) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Messages) == 0 {
		return []Message{}
	}

	messages := make([]Message, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}
func (s *Session) GetLastN(n int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || len(s.Messages) == 0 {
		return []Message{}
	}

	if n >= len(s.Messages) {
		messages := make([]Message, len(s.Messages))
		copy(messages, s.Messages)
		return messages
	}

	start := len(s.Messages) - n
	messages := make([]Message, n)
	copy(messages, s.Messages[start:])
	return messages
}
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = []Message{}
	s.UpdatedAt = time.Now()
}
func (s *Session) SetMetadata(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}

	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}
func (s *Session) GetMetadata(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Metadata == nil {
		return ""
	}

	return s.Metadata[key]
}
func (s *Session) DeleteMetadata(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Metadata == nil {
		return
	}

	delete(s.Metadata, key)
	s.UpdatedAt = time.Now()
}
func (s *Session) IsExpired(ttl time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return time.Since(s.UpdatedAt) > ttl
}
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UpdatedAt = time.Now()
}

func (s *Session) saveMessage(msg Message) {
	if s.ID == "" {
		return
	}

	sessionFile := filepath.Join(s.workspace, "sessions", s.Channel, s.ChatID+".jsonl")
	// 如果目录不存在则创建
	if err := os.MkdirAll(filepath.Dir(sessionFile), 0755); err != nil {
		slog.Error("Failed to create session directory", "error", err)
		return
	}
	// 以追加模式打开文件
	file, err := os.OpenFile(sessionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open session file", "error", err)
		return
	}
	defer file.Close()
	// 使用 Encoder 写入 JSON，确保内容中的换行符被正确处理
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(msg); err != nil {
		slog.Error("Failed to encode message", "error", err)
		return
	}
}
func (s *Session) loadMessage() error {
	sessionFile := filepath.Join(s.workspace, "sessions", s.Channel, s.ChatID+".jsonl")
	// 如果文件不存在则直接返回
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return nil
	}
	// 如果存在，读取并转换为session中的message列表
	file, err := os.Open(sessionFile)
	if err != nil {
		slog.Error("Failed to open session file", "error", err)
		return err
	}
	defer file.Close()
	// 使用 Decoder 逐行读取 JSON，正确处理内容中的换行符
	decoder := json.NewDecoder(file)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			if err.Error() == "EOF" {
				break
			}
			slog.Warn("message decode failed", "error", err)
			continue
		}

		s.Messages = append(s.Messages, msg)
	}

	return nil
}

func (s *Session) TrimMessages(summary string, keptHistory int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Messages) <= keptHistory {
		return s.Messages
	}

	trimmedMessages := make([]Message, 0, keptHistory+1)
	trimmedMessages = append(trimmedMessages, Message{
		Role:    constant.RoleAssistant,
		Content: summary,
	})
	trimmedMessages = append(trimmedMessages, s.Messages[len(s.Messages)-keptHistory:]...)
	// 保存到文件
	s.Messages = trimmedMessages
	s.UpdatedAt = time.Now()

	sessionFile := filepath.Join(s.workspace, "sessions", s.Channel, s.ChatID+".jsonl")
	// 如果目录不存在则创建
	if err := os.MkdirAll(filepath.Dir(sessionFile), 0755); err != nil {
		slog.Error("Failed to create session directory", "error", err)
		return trimmedMessages
	}
	// 清空文件内容
	if err := os.Truncate(sessionFile, 0); err != nil {
		slog.Error("Failed to truncate session file", "error", err)
		return trimmedMessages
	}
	// 写入新内容
	file, err := os.OpenFile(sessionFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("Failed to open session file", "error", err)
		return trimmedMessages
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	for _, msg := range trimmedMessages {
		if err := encoder.Encode(msg); err != nil {
			slog.Error("Failed to encode message", "error", err)
			return trimmedMessages
		}
	}
	return trimmedMessages
}
