package feishu

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
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
	memberData := GroupMemberFile{
		ChatID:      "oc_test123",
		LastUpdated: time.Now().Format(time.RFC3339),
		MemberCount: 2,
		Members: map[string]string{
			"ou_111": "张三",
			"ou_222": "李四",
		},
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
		workspace: tempDir,
		groupUsers: sync.Map{},
	}

	if err := c.loadGroupUsersFromFile(); err != nil {
		t.Fatalf("Failed to load group users: %v", err)
	}

	// 验证加载的数据
	val, ok := c.groupUsers.Load("oc_test123")
	if !ok {
		t.Fatal("Group users not loaded")
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
		workspace: tempDir,
		groupUsers: sync.Map{},
	}

	userMap := map[string]string{
		"ou_111": "张三",
		"ou_222": "李四",
	}

	// 保存到文件
	if err := c.saveGroupUsersToFile("oc_test456", userMap); err != nil {
		t.Fatalf("Failed to save group users: %v", err)
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

	var memberFile GroupMemberFile
	if err := json.Unmarshal(data, &memberFile); err != nil {
		t.Fatalf("Failed to unmarshal file: %v", err)
	}

	if memberFile.ChatID != "oc_test456" {
		t.Errorf("Expected chat_id oc_test456, got %s", memberFile.ChatID)
	}

	if memberFile.MemberCount != 2 {
		t.Errorf("Expected 2 members, got %d", memberFile.MemberCount)
	}

	if len(memberFile.Members) != 2 {
		t.Errorf("Expected 2 members in map, got %d", len(memberFile.Members))
	}
}

func TestSaveGroupUsersToFileCreatesDir(t *testing.T) {
	// 创建临时目录（但不创建 session_member 子目录）
	tempDir := t.TempDir()

	c := &FeishuChannel{
		workspace: tempDir,
		groupUsers: sync.Map{},
	}

	userMap := map[string]string{
		"ou_111": "测试用户",
	}

	// 保存到文件（应该自动创建目录）
	if err := c.saveGroupUsersToFile("oc_test789", userMap); err != nil {
		t.Fatalf("Failed to save group users: %v", err)
	}

	// 验证目录和文件都存在
	memberDir := filepath.Join(tempDir, "sessions", "feishu", "session_member")
	if _, err := os.Stat(memberDir); os.IsNotExist(err) {
		t.Fatal("Directory was not created")
	}

	testFile := filepath.Join(memberDir, "oc_test789.json")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}
}

func TestCleanExpiredMemberFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	memberDir := filepath.Join(tempDir, "sessions", "feishu", "session_member")
	if err := os.MkdirAll(memberDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 创建一个旧文件
	oldFile := filepath.Join(memberDir, "oc_old.json")
	memberData := GroupMemberFile{
		ChatID:      "oc_old",
		LastUpdated: time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		MemberCount: 1,
		Members:     map[string]string{"ou_111": "测试"},
	}

	data, _ := json.MarshalIndent(memberData, "", "  ")
	os.WriteFile(oldFile, data, 0644)

	// 设置文件的修改时间为过去
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// 等待一下确保文件时间不同
	time.Sleep(10 * time.Millisecond)

	// 创建一个新文件
	newFile := filepath.Join(memberDir, "oc_new.json")
	memberData.ChatID = "oc_new"
	memberData.LastUpdated = time.Now().Format(time.RFC3339)
	data, _ = json.MarshalIndent(memberData, "", "  ")
	os.WriteFile(newFile, data, 0644)

	c := &FeishuChannel{
		workspace: tempDir,
		groupUsers: sync.Map{},
	}

	// 清理超过 24 小时的文件
	if err := c.cleanExpiredMemberFiles(24 * time.Hour); err != nil {
		t.Fatalf("Failed to clean expired files: %v", err)
	}

	// 验证旧文件被删除，新文件保留
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file should have been deleted")
	}

	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("New file should still exist")
	}
}
