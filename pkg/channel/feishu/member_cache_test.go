package feishu

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/yockii/wangshu/pkg/constant"
)

func TestLoadGroupUsersFromFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	memberDir := filepath.Join(tempDir, "sessions", "feishu", "session_member")
	if err := os.MkdirAll(memberDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(memberDir, "oc_test123.json")
	memberData := map[string]string{
		"ou_111": "张三",
		"ou_222": "李四",
	}

	data, err := json.MarshalIndent(memberData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal member data: %v", err)
	}

	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 创建 FeishuChannel 并加载
	c := &FeishuChannel{
		workspace:   tempDir,
		cachedUsers: sync.Map{},
	}

	if err := c.loadCachedUsersFromFile(); err != nil {
		t.Fatalf("Failed to load cached users: %v", err)
	}

	// 验证加载的数据
	val, ok := c.cachedUsers.Load("oc_test123")
	if !ok {
		t.Fatal("Cached users not loaded")
	}

	userMap := val.(map[string]string)
	if len(userMap) != 2 {
		t.Errorf("Expected 2 users, got %d", len(userMap))
	}

	if userMap["ou_111"] != "张三" {
		t.Errorf("Expected 张三, got %s", userMap["ou_111"])
	}
}

func TestSaveGroupUsersToFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	memberDir := filepath.Join(tempDir, "sessions", "feishu", "session_member")

	c := &FeishuChannel{
		workspace:   tempDir,
		cachedUsers: sync.Map{},
	}

	c.cachedUsers.Store("ou_111", "张三")
	c.cachedUsers.Store("ou_222", "李四")

	// 保存到文件
	if err := c.saveUsersInfoToCacheFile(); err != nil {
		t.Fatalf("Failed to save cached users: %v", err)
	}

	// 验证文件存在
	testFile := filepath.Join(memberDir, "oc_test456.json")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// 读取并验证内容
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var memberFile map[string]string
	if err := json.Unmarshal(data, &memberFile); err != nil {
		t.Fatalf("Failed to unmarshal file: %v", err)
	}

	if len(memberFile) != 2 {
		t.Errorf("Expected 2 members in map, got %d", len(memberFile))
	}

	if memberFile["ou_111"] != "张三" {
		t.Errorf("Expected 张三, got %s", memberFile["ou_111"])
	}
}

func TestSaveGroupUsersToFileCreatesDir(t *testing.T) {
	// 创建临时目录（但不创建 session_member 子目录）
	tempDir := t.TempDir()

	c := &FeishuChannel{
		name:        "feishu",
		workspace:   tempDir,
		cachedUsers: sync.Map{},
	}

	c.cachedUsers.Store("ou_111", "测试用户")

	// 保存到文件（应该自动创建目录）
	if err := c.saveUsersInfoToCacheFile(); err != nil {
		t.Fatalf("Failed to save group users: %v", err)
	}

	// 验证目录和文件都存在
	memberDir := filepath.Join(tempDir, constant.DirSessions, c.name)
	if _, err := os.Stat(memberDir); os.IsNotExist(err) {
		t.Fatal("Directory was not created")
	}

	testFile := filepath.Join(memberDir, constant.FileCachedMembers)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}
}
