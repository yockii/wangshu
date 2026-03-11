package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/yockii/wangshu/internal/types"
)

type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	ttl      time.Duration
}

func NewManager(ttl time.Duration) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
}
func (m *Manager) GetOrCreate(workspace, channel, chatID, chatType, chatName, senderID, senderName string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("%s:%s", channel, chatID)
	session, ok := m.sessions[id]
	if ok {
		return session
	}

	session = &Session{
		ID:         id,
		ChatType:   chatType,
		Channel:    channel,
		ChatID:     chatID,
		ChatName:   chatName,
		SenderID:   senderID,
		SenderName: senderName,
		Messages:   []types.Message{},
		Metadata:   make(map[string]string),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		workspace:  workspace,
	}

	m.sessions[id] = session

	session.loadMessage()

	return session
}
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	return session, ok
}
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, id)
}
func (m *Manager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, session := range m.sessions {
		if session.IsExpired(m.ttl) {
			delete(m.sessions, id)
			count++
		}
	}

	return count
}
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}
func (m *Manager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}
