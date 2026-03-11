package feishu

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/yockii/wangshu/pkg/constant"
)

// loadCachedUsersFromFile 从文件加载缓存的成员信息到内存
func (c *FeishuChannel) loadCachedUsersFromFile() error {
	cachedMemberFile := filepath.Join(c.workspace, constant.DirSessions, c.name, constant.FileCachedMembers)

	// 检查是否存在
	if _, err := os.Stat(cachedMemberFile); os.IsNotExist(err) {
		slog.Info("Feishu cache member file not found, creating", "file", cachedMemberFile)
		if err = os.MkdirAll(filepath.Dir(cachedMemberFile), 0755); err != nil {
			slog.Error("Failed to create feishu cache member directory", "dir", filepath.Dir(cachedMemberFile), "error", err)
			return err
		}
		if err = os.WriteFile(cachedMemberFile, []byte("{}"), 0644); err != nil {
			slog.Error("Failed to create feishu cache member file", "file", cachedMemberFile, "error", err)
			return err
		}
		return nil
	}
	loadCount := 0
	data, err := os.ReadFile(cachedMemberFile)
	if err != nil {
		slog.Error("Failed to read feishu cache member file", "file", cachedMemberFile, "error", err)
		return err
	}

	var memberFile map[string]string
	if err := json.Unmarshal(data, &memberFile); err != nil {
		slog.Error("Failed to parse feishu cache member file", "file", cachedMemberFile, "error", err)
		return err
	}

	// 加载到内存
	for openID, userName := range memberFile {
		c.cachedUsers.Store(openID, userName)
		loadCount++
	}

	slog.Info("Loaded feishu group users from files", "count", loadCount, "file", cachedMemberFile)
	return nil
}

// saveUsersInfoToCacheFile 保存成员信息到缓存文件
func (c *FeishuChannel) saveUsersInfoToCacheFile() error {
	cachedMemberFile := filepath.Join(c.workspace, constant.DirSessions, c.name, constant.FileCachedMembers)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(cachedMemberFile), 0755); err != nil {
		slog.Error("Failed to create feishu cache member directory", "dir", filepath.Dir(cachedMemberFile), "error", err)
		return err
	}

	// 获取文件锁，避免并发写入
	c.cacheFileMu.Lock()
	defer c.cacheFileMu.Unlock()

	// 构建文件内容
	cachedUsers := make(map[string]string)
	c.cachedUsers.Range(func(key, value interface{}) bool {
		cachedUsers[key.(string)] = value.(string)
		return true
	})

	data, err := json.MarshalIndent(cachedUsers, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal feishu cache member data", "error", err)
		return err
	}

	// 写入文件
	if err := os.WriteFile(cachedMemberFile, data, 0644); err != nil {
		slog.Error("Failed to write feishu cache member file", "file", cachedMemberFile, "error", err)
		return err
	}

	slog.Debug("Saved feishu cache users to file",
		"userCount", len(cachedUsers),
		"file", cachedMemberFile,
	)

	return nil
}

// updateGroupUsersCache 更新群成员缓存并保存到文件
func (c *FeishuChannel) updateCachedUsersCache(openID, name string) {
	// 存储到内存
	c.cachedUsers.Store(openID, name)

	// 异步保存到文件（避免阻塞）
	go func() {
		if err := c.saveUsersInfoToCacheFile(); err != nil {
			slog.Warn("Failed to save feishu cache users cache", "openID", openID, "error", err)
		}
	}()
}
