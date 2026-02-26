package notification

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yockii/yoclaw/pkg/bus"
)

// Manager handles active user notifications and proactive messaging
type Manager struct {
	users map[string]*UserInfo // key: senderID
	mu    sync.RWMutex
}

// UserInfo stores information about a user for proactive notifications
type UserInfo struct {
	SenderID    string
	Channel     string
	ChatID      string
	LastSeen    time.Time
	DisplayName string
	Metadata    map[string]string
	// Store the workspace path for this user's data
	Workspace string
}

var globalManager = &Manager{
	users: make(map[string]*UserInfo),
}

// GetManager returns the global notification manager
func GetManager() *Manager {
	return globalManager
}

// RecordUser records a user's information for proactive notifications
// Called when a user sends a message
// workspace is the agent's workspace where user data should be stored
func (m *Manager) RecordUser(channel, chatID, senderID, workspace string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[senderID]; !exists {
		m.users[senderID] = &UserInfo{
			SenderID:  senderID,
			Channel:   channel,
			ChatID:    chatID,
			Workspace: workspace,
			Metadata:  make(map[string]string),
		}
	} else {
		// Update existing user info
		m.users[senderID].ChatID = chatID
		m.users[senderID].Channel = channel
		m.users[senderID].Workspace = workspace
	}

	m.users[senderID].LastSeen = time.Now()

	// Persist to disk in agent's workspace
	m.saveUser(senderID)
}

// Notify sends a proactive message to a user
func (m *Manager) Notify(senderID, message string) error {
	m.mu.RLock()
	user, exists := m.users[senderID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user %s not found", senderID)
	}

	// Check if user is still active (seen within last 24 hours)
	if time.Since(user.LastSeen) > 24*time.Hour {
		return fmt.Errorf("user %s is inactive (last seen: %s)", senderID, user.LastSeen.Format(time.RFC3339))
	}

	// Send message via bus
	bus.Default().PublishOutbound(bus.OutboundMessage{
		Channel: user.Channel,
		ChatID:  user.ChatID,
		Content: message,
	})

	return nil
}

// Broadcast sends a message to all active users
func (m *Manager) Broadcast(message string, activeOnly bool) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0

	for _, user := range m.users {
		// Skip inactive users if activeOnly is true
		if activeOnly && time.Since(user.LastSeen) > 24*time.Hour {
			continue
		}

		bus.Default().PublishOutbound(bus.OutboundMessage{
			Channel: user.Channel,
			ChatID:  user.ChatID,
			Content: message,
		})
		count++
	}

	return count
}

// NotifyChannel sends a message to all users in a specific channel
func (m *Manager) NotifyChannel(channel, message string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, user := range m.users {
		if user.Channel == channel {
			bus.Default().PublishOutbound(bus.OutboundMessage{
				Channel: user.Channel,
				ChatID:  user.ChatID,
				Content: message,
			})
			count++
		}
	}

	return count
}

// GetActiveUsers returns a list of active users (seen within last 24 hours)
func (m *Manager) GetActiveUsers() []*UserInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := []*UserInfo{}
	for _, user := range m.users {
		if time.Since(user.LastSeen) <= 24*time.Hour {
			active = append(active, user)
		}
	}

	return active
}

// GetUser retrieves information about a specific user
func (m *Manager) GetUser(senderID string) (*UserInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[senderID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", senderID)
	}

	return user, nil
}

// UpdateUserMetadata updates metadata for a user
func (m *Manager) UpdateUserMetadata(senderID, key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[senderID]
	if !exists {
		return
	}

	if user.Metadata == nil {
		user.Metadata = make(map[string]string)
	}

	user.Metadata[key] = value
	m.saveUser(senderID)
}

// SetDisplayName sets the display name for a user
func (m *Manager) SetDisplayName(senderID, displayName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user, exists := m.users[senderID]; exists {
		user.DisplayName = displayName
		m.saveUser(senderID)
	}
}

// RemoveUser removes a user from the notification list
func (m *Manager) RemoveUser(senderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[senderID]; !exists {
		return fmt.Errorf("user %s not found", senderID)
	}

	user := m.users[senderID]

	delete(m.users, senderID)

	// Remove from disk using user's workspace
	if user.Workspace != "" {
		userFile := filepath.Join(user.Workspace, "users", senderID+".json")
		os.Remove(userFile)
	}

	return nil
}

// CleanupInactive removes users who haven't been seen in a specified duration
func (m *Manager) CleanupInactive(inactiveDuration time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for senderID, user := range m.users {
		if time.Since(user.LastSeen) > inactiveDuration {
			delete(m.users, senderID)

			// Remove from disk using user's workspace
			if user.Workspace != "" {
				userFile := filepath.Join(user.Workspace, "users", senderID+".json")
				os.Remove(userFile)
			}
			count++
		}
	}

	return count
}

// GetStats returns statistics about the notification system
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCount := 0
	inactiveCount := 0
	now := time.Now()

	channelCounts := make(map[string]int)

	for _, user := range m.users {
		if now.Sub(user.LastSeen) <= 24*time.Hour {
			activeCount++
		} else {
			inactiveCount++
		}

		channelCounts[user.Channel]++
	}

	return map[string]interface{}{
		"total_users":    len(m.users),
		"active_users":   activeCount,
		"inactive_users": inactiveCount,
		"channels":       channelCounts,
	}
}

// saveUser saves a user's information to disk in the agent's workspace
func (m *Manager) saveUser(senderID string) error {
	user := m.users[senderID]
	if user.Workspace == "" {
		return nil // No workspace assigned yet, skip saving
	}

	userDir := filepath.Join(user.Workspace, "users")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	// In production, use proper JSON encoding
	// For now, just a placeholder
	userFile := filepath.Join(userDir, senderID+".json")

	// Create/modify user file
	f, err := os.Create(userFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write basic user info
	fmt.Fprintf(f, "senderID: %s\n", user.SenderID)
	fmt.Fprintf(f, "channel: %s\n", user.Channel)
	fmt.Fprintf(f, "chatID: %s\n", user.ChatID)
	fmt.Fprintf(f, "workspace: %s\n", user.Workspace)
	fmt.Fprintf(f, "lastSeen: %s\n", user.LastSeen.Format(time.RFC3339))
	if user.DisplayName != "" {
		fmt.Fprintf(f, "displayName: %s\n", user.DisplayName)
	}

	return nil
}
