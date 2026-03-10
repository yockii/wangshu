package session

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
)

type Session struct {
	ID        string
	Channel   string
	ChatID    string
	SenderID  string
	Messages  []types.Message
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
	workspace string
	mu        sync.RWMutex

	PendingImage   *types.ContentBlock
	PendingImageAt time.Time
}

// func (s *Session) AddMessage(role, content string, toolCallID string, toolCalls ...ToolCall) {
func (s *Session) AddMessage(role, content string, toolCalls ...types.ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := types.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}

	s.Messages = append(s.Messages, msg)

	s.UpdatedAt = time.Now()

	s.saveMessage(msg)
}

func (s *Session) AddMessageWithContents(role string, contents []types.ContentBlock, toolCalls ...types.ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := types.Message{
		Role:      role,
		Contents:  contents,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}

	for _, c := range contents {
		if c.Type == "text" {
			msg.Content = c.Text
			break
		}
	}

	s.Messages = append(s.Messages, msg)

	s.UpdatedAt = time.Now()

	s.saveMessage(msg)
}

func (s *Session) SetPendingImage(image *types.ContentBlock) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingImage = image
	s.PendingImageAt = time.Now()
}

func (s *Session) GetAndClearPendingImage() *types.ContentBlock {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.PendingImage == nil {
		return nil
	}
	if time.Since(s.PendingImageAt) > 5*time.Minute {
		s.PendingImage = nil
		return nil
	}
	img := s.PendingImage
	s.PendingImage = nil
	return img
}
func (s *Session) GetMessages() []types.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Messages) == 0 {
		return []types.Message{}
	}

	messages := make([]types.Message, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}
func (s *Session) GetLastN(n int) []types.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || len(s.Messages) == 0 {
		return []types.Message{}
	}

	if n >= len(s.Messages) {
		messages := make([]types.Message, len(s.Messages))
		copy(messages, s.Messages)
		return messages
	}

	start := len(s.Messages) - n
	messages := make([]types.Message, n)
	copy(messages, s.Messages[start:])
	return messages
}
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = []types.Message{}
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

func (s *Session) saveMessage(msg types.Message) {
	if s.ID == "" {
		return
	}

	sessionFile := filepath.Join(s.workspace, constant.DirSessions, s.Channel, s.ChatID+constant.ExtJSONL)
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
	sessionFile := filepath.Join(s.workspace, constant.DirSessions, s.Channel, s.ChatID+constant.ExtJSONL)
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
		var msg types.Message
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

func (s *Session) TrimMessages(summary string, keptHistory int) []types.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Messages) <= keptHistory {
		return s.Messages
	}

	trimmedMessages := make([]types.Message, 0, keptHistory+1)
	trimmedMessages = append(trimmedMessages, types.Message{
		Role:    constant.RoleAssistant,
		Content: summary,
	})
	trimmedMessages = append(trimmedMessages, s.Messages[len(s.Messages)-keptHistory:]...)
	// 保存到文件
	s.Messages = trimmedMessages
	s.UpdatedAt = time.Now()

	sessionFile := filepath.Join(s.workspace, constant.DirSessions, s.Channel, s.ChatID+constant.ExtJSONL)
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
