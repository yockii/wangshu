package session

import (
	"sync"
	"time"
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
func (m *Manager) GetOrCreate(workspace, id, channel, chatID, senderID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if ok {
		return session
	}

	session = &Session{
		ID:        id,
		Channel:   channel,
		ChatID:    chatID,
		SenderID:  senderID,
		Messages:  []Message{},
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		workspace: workspace,
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
