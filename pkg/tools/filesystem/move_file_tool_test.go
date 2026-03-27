package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRenameFileTool(t *testing.T) {
	tool := NewMoveFileTool()

	if tool == nil {
		t.Fatal("NewRenameFileTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "rename_file" {
		t.Errorf("Expected tool name 'rename_file', got '%s'", tool.Name())
	}

	// 测试工具描述
	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	// 测试参数定义
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Tool should have parameters")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	// 验证必需的参数
	required, ok := params["required"].([]string)
	if !ok || len(required) != 2 {
		t.Error("Should have 2 required parameters")
	}

	// 验证old_path和new_path都是必需的
	hasOldPath := false
	hasNewPath := false
	for _, r := range required {
		if r == "old_path" {
			hasOldPath = true
		}
		if r == "new_path" {
			hasNewPath = true
		}
	}
	if !hasOldPath || !hasNewPath {
		t.Error("Both 'old_path' and 'new_path' should be required")
	}

	// 验证参数属性
	if _, ok := properties["old_path"]; !ok {
		t.Error("Parameters should have 'old_path' property")
	}
	if _, ok := properties["new_path"]; !ok {
		t.Error("Parameters should have 'new_path' property")
	}
}

func TestRenameFileTool_Execute_RenameFile(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old_name.txt")
	newPath := filepath.Join(tmpDir, "new_name.txt")

	// 创建源文件
	content := "test content"
	os.WriteFile(oldPath, []byte(content), 0644)

	// 重命名文件
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should contain success message, got: %s", result.Raw)
	}

	// 验证旧文件不存在
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file should not exist after rename")
	}

	// 验证新文件存在
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be preserved after rename")
	}
}

func TestRenameFileTool_Execute_RenameDirectory(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old_dir")
	newPath := filepath.Join(tmpDir, "new_dir")

	// 创建源目录
	os.Mkdir(oldPath, 0755)

	// 在目录中创建文件
	testFile := filepath.Join(oldPath, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	// 重命名目录
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证旧目录不存在
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old directory should not exist after rename")
	}

	// 验证新目录存在
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("New directory should exist after rename")
	}

	// 验证目录中的文件也被移动
	movedFile := filepath.Join(newPath, "test.txt")
	if _, err := os.Stat(movedFile); os.IsNotExist(err) {
		t.Error("Files inside directory should be moved with the directory")
	}

	// 验证返回消息包含路径
	if !strings.Contains(result.Raw, oldPath) || !strings.Contains(result.Raw, newPath) {
		t.Errorf("Result should contain both paths, got: %s", result.Raw)
	}
}

func TestRenameFileTool_Execute_MoveToDifferentDirectory(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	// 创建源目录
	sourceDir := filepath.Join(tmpDir, "source")
	os.Mkdir(sourceDir, 0755)

	oldPath := filepath.Join(sourceDir, "file.txt")
	content := "content to move"
	os.WriteFile(oldPath, []byte(content), 0644)

	// 创建目标目录
	targetDir := filepath.Join(tmpDir, "target")
	os.Mkdir(targetDir, 0755)

	newPath := filepath.Join(targetDir, "file.txt")

	// 移动文件
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证文件已移动
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("File should not exist in old location")
	}

	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be preserved after move")
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should indicate success, got: %s", result.Raw)
	}
}

func TestRenameFileTool_Execute_EmptyOldPath(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	result := tool.Execute(context.Background(), map[string]string{
		"old_path": "",
		"new_path": filepath.Join(tmpDir, "new.txt"),
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty old_path")
	}
}

func TestRenameFileTool_Execute_EmptyNewPath(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old.txt")
	os.WriteFile(oldPath, []byte("content"), 0644)

	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": "",
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty new_path")
	}
}

func TestRenameFileTool_Execute_BothPathsEmpty(t *testing.T) {
	tool := NewMoveFileTool()

	result := tool.Execute(context.Background(), map[string]string{
		"old_path": "",
		"new_path": "",
	})

	if result.Err == nil {
		t.Error("Execute should fail when both paths are empty")
	}

	// 测试缺少参数
	result = tool.Execute(context.Background(), map[string]string{})

	if result.Err == nil {
		t.Error("Execute should fail when both parameters are missing")
	}
}

func TestRenameFileTool_Execute_SourceNotExist(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	result := tool.Execute(context.Background(), map[string]string{
		"old_path": filepath.Join(tmpDir, "nonexistent.txt"),
		"new_path": filepath.Join(tmpDir, "new.txt"),
	})

	if result.Err == nil {
		t.Error("Execute should fail when source file does not exist")
	}
}

func TestRenameFileTool_Execute_TargetDirectoryNotExist(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(oldPath, []byte("content"), 0644)

	newPath := filepath.Join(tmpDir, "nonexistent", "file.txt")

	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err == nil {
		t.Error("Execute should fail when target directory does not exist")
	}
}

func TestRenameFileTool_Execute_OverwriteExisting(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")

	// 创建两个文件
	os.WriteFile(oldPath, []byte("old content"), 0644)
	os.WriteFile(newPath, []byte("new content"), 0644)

	// 重命名（应该覆盖）
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed and overwrite: %v", result.Err)
	}

	// 验证文件内容
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != "old content" {
		t.Errorf("File should be overwritten with old content")
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should indicate success, got: %s", result.Raw)
	}
}

func TestRenameFileTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewMoveFileTool()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	// 创建测试文件
	oldPath := filepath.Join(homeDir, ".wangshu_test_rename_old.txt")
	newPath := filepath.Join(homeDir, ".wangshu_test_rename_new.txt")
	content := "tilde test"

	defer os.Remove(oldPath)
	defer os.Remove(newPath)

	os.WriteFile(oldPath, []byte(content), 0644)

	// 使用波浪号路径重命名
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": "~/.wangshu_test_rename_old.txt",
		"new_path": "~/.wangshu_test_rename_new.txt",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with tilde paths: %v", result.Err)
	}

	// 验证重命名成功
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file should not exist")
	}

	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be preserved")
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should indicate success, got: %s", result.Raw)
	}
}

func TestRenameFileTool_Execute_RelativePath(t *testing.T) {
	tool := NewMoveFileTool()

	// 切换到临时目录
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 创建文件
	oldPath := "old.txt"
	newPath := "new.txt"
	content := "relative path test"

	os.WriteFile(oldPath, []byte(content), 0644)

	// 使用相对路径重命名
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": oldPath,
		"new_path": newPath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with relative paths: %v", result.Err)
	}

	// 验证重命名成功
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be preserved")
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should indicate success, got: %s", result.Raw)
	}
}

func TestRenameFileTool_Execute_RenameToSameName(t *testing.T) {
	tool := NewMoveFileTool()
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "file.txt")
	content := "test content"
	os.WriteFile(filePath, []byte(content), 0644)

	// 重命名到相同名称（应该成功但不做任何事）
	result := tool.Execute(context.Background(), map[string]string{
		"old_path": filePath,
		"new_path": filePath,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed when renaming to same name: %v", result.Err)
	}

	// 验证文件仍然存在且内容未变
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be unchanged")
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully renamed") {
		t.Errorf("Result should indicate success, got: %s", result.Raw)
	}
}
