package feishu

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// fileMutex 用于防止并发写入同一个文件
var fileMutex sync.Map

// GroupMemberFile 群成员文件结构
type GroupMemberFile struct {
	ChatID      string            `json:"chat_id"`
	LastUpdated string            `json:"last_updated"`
	MemberCount int               `json:"member_count"`
	Members     map[string]string `json:"members"` // open_id -> name
}

// loadGroupUsersFromFile 从文件加载群成员信息到内存
func (c *FeishuChannel) loadGroupUsersFromFile() error {
	memberDir := filepath.Join(c.workspace, "sessions", "feishu", "session_member")

	// 检查目录是否存在
	files, err := os.ReadDir(memberDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，创建并返回
			slog.Info("Feishu session member directory not found, creating", "dir", memberDir)
			return os.MkdirAll(memberDir, 0755)
		}
		slog.Error("Failed to read feishu session member directory", "dir", memberDir, "error", err)
		return err
	}

	loadCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 提取 chat_id
		chatID := strings.TrimSuffix(file.Name(), ".json")

		filePath := filepath.Join(memberDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			slog.Warn("Failed to read group member file", "file", file.Name(), "error", err)
			continue
		}

		var memberFile GroupMemberFile
		if err := json.Unmarshal(data, &memberFile); err != nil {
			slog.Warn("Failed to parse group member file", "file", file.Name(), "error", err)
			continue
		}

		// 加载到内存
		c.groupUsers.Store(chatID, memberFile.Members)
		loadCount++

		slog.Debug("Loaded group users from file",
			"chatID", chatID,
			"memberCount", memberFile.MemberCount,
			"lastUpdated", memberFile.LastUpdated,
		)
	}

	slog.Info("Loaded feishu group users from files", "count", loadCount, "dir", memberDir)
	return nil
}

// saveGroupUsersToFile 保存群成员信息到文件
func (c *FeishuChannel) saveGroupUsersToFile(chatID string, userMap map[string]string) error {
	memberDir := filepath.Join(c.workspace, "sessions", "feishu", "session_member")

	// 确保目录存在
	if err := os.MkdirAll(memberDir, 0755); err != nil {
		slog.Error("Failed to create feishu session member directory", "dir", memberDir, "error", err)
		return err
	}

	// 获取文件锁，避免并发写入
	mu, _ := fileMutex.LoadOrStore(chatID, &sync.Mutex{})
	mutex := mu.(*sync.Mutex)

	mutex.Lock()
	defer mutex.Unlock()

	// 构建文件内容
	memberFile := GroupMemberFile{
		ChatID:      chatID,
		LastUpdated: time.Now().Format(time.RFC3339),
		MemberCount: len(userMap),
		Members:     userMap,
	}

	data, err := json.MarshalIndent(memberFile, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal group member data", "chatID", chatID, "error", err)
		return err
	}

	// 写入文件
	filename := filepath.Join(memberDir, chatID+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		slog.Error("Failed to write group member file", "chatID", chatID, "file", filename, "error", err)
		return err
	}

	slog.Debug("Saved feishu group users to file",
		"chatID", chatID,
		"memberCount", len(userMap),
		"file", filename,
	)

	return nil
}

// updateGroupUsersCache 更新群成员缓存并保存到文件
func (c *FeishuChannel) updateGroupUsersCache(chatID string, userMap map[string]string) {
	// 存储到内存
	c.groupUsers.Store(chatID, userMap)

	// 异步保存到文件（避免阻塞）
	go func() {
		if err := c.saveGroupUsersToFile(chatID, userMap); err != nil {
			slog.Warn("Failed to save group users cache", "chatID", chatID, "error", err)
		}
	}()
}

// cleanExpiredMemberFiles 清理过期的成员文件（可选功能）
func (c *FeishuChannel) cleanExpiredMemberFiles(maxAge time.Duration) error {
	memberDir := filepath.Join(c.workspace, "sessions", "feishu", "session_member")

	files, err := os.ReadDir(memberDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	cleanedCount := 0

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(memberDir, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}

		// 检查文件是否过期
		if now.Sub(info.ModTime()) > maxAge {
			if err := os.Remove(filePath); err != nil {
				slog.Warn("Failed to remove expired feishu member file", "file", file.Name(), "error", err)
			} else {
				cleanedCount++
				slog.Debug("Removed expired feishu member file", "file", file.Name(), "age", now.Sub(info.ModTime()))
			}
		}
	}

	if cleanedCount > 0 {
		slog.Info("Cleaned expired feishu member files", "count", cleanedCount, "maxAge", maxAge)
	}

	return nil
}
