package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWriteFileTool(t *testing.T) {
	tool := NewWriteFileTool()

	if tool == nil {
		t.Fatal("NewWriteFileTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "write_file" {
		t.Errorf("Expected tool name 'write_file', got '%s'", tool.Name())
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

	// 验证path和content都是必需的
	hasPath := false
	hasContent := false
	for _, r := range required {
		if r == "path" {
			hasPath = true
		}
		if r == "content" {
			hasContent = true
		}
	}
	if !hasPath || !hasContent {
		t.Error("Both 'path' and 'content' should be required")
	}

	// 验证参数属性
	if _, ok := properties["path"]; !ok {
		t.Error("Parameters should have 'path' property")
	}
	if _, ok := properties["content"]; !ok {
		t.Error("Parameters should have 'content' property")
	}
	if _, ok := properties["append"]; !ok {
		t.Error("Parameters should have 'append' property")
	}
}

func TestWriteFileTool_Execute_CreateNewFile(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new_file.txt")
	content := "Hello, World!"

	// 写入新文件
	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": content,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully wrote") {
		t.Errorf("Result should contain success message, got: %s", result.Raw)
	}

	// 验证文件已创建
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should be created")
	}

	// 验证文件内容
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch, expected '%s', got '%s'", content, string(data))
	}
}

func TestWriteFileTool_Execute_OverwriteExistingFile(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")

	// 创建已存在的文件
	originalContent := "Original content"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	// 覆盖文件
	newContent := "New content"
	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": newContent,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证文件内容被覆盖
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != newContent {
		t.Errorf("File should be overwritten, expected '%s', got '%s'", newContent, string(data))
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully wrote") {
		t.Errorf("Result should indicate overwrite, got: %s", result.Raw)
	}
}

func TestWriteFileTool_Execute_AppendMode(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append.txt")

	// 创建初始文件
	initialContent := "Line 1\n"
	os.WriteFile(testFile, []byte(initialContent), 0644)

	// 追加内容
	appendContent := "Line 2\n"
	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": appendContent,
		"append":  "true",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "appended") {
		t.Errorf("Result should indicate append mode, got: %s", result.Raw)
	}

	// 验证内容被追加
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := initialContent + appendContent
	if string(data) != expectedContent {
		t.Errorf("Content should be appended, expected '%s', got '%s'", expectedContent, string(data))
	}
}

func TestWriteFileTool_Execute_AppendModeNumeric(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append_numeric.txt")

	// 创建初始文件
	initialContent := "Start\n"
	os.WriteFile(testFile, []byte(initialContent), 0644)

	// 使用数字1表示追加模式
	appendContent := "End\n"
	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": appendContent,
		"append":  "1",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "appended") {
		t.Errorf("Result should indicate append mode, got: %s", result.Raw)
	}

	// 验证内容被追加
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := initialContent + appendContent
	if string(data) != expectedContent {
		t.Errorf("Content should be appended, expected '%s', got '%s'", expectedContent, string(data))
	}
}

func TestWriteFileTool_Execute_CreateDirectory(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()

	// 写入到不存在的子目录
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
	content := "Content in nested directory"

	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": content,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed and create directories: %v", result.Err)
	}

	// 验证文件已创建
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch")
	}
}

func TestWriteFileTool_Execute_EmptyPath(t *testing.T) {
	tool := NewWriteFileTool()

	// 测试空路径
	result := tool.Execute(context.Background(), map[string]string{
		"path":    "",
		"content": "content",
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty path")
	}

	// 测试缺少path参数
	result = tool.Execute(context.Background(), map[string]string{
		"content": "content",
	})

	if result.Err == nil {
		t.Error("Execute should fail when path parameter is missing")
	}
}

func TestWriteFileTool_Execute_EmptyContent(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	// 写入空内容（应该创建空文件）
	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": "",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with empty content: %v", result.Err)
	}

	// 验证文件已创建
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should be created even with empty content")
	}

	// 验证文件内容为空
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("File should be empty, got %d bytes", len(data))
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully wrote 0 bytes") {
		t.Errorf("Result should indicate 0 bytes written, got: %s", result.Raw)
	}
}

func TestWriteFileTool_Execute_MissingContent(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// 测试缺少content参数（会被当作空字符串处理）
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with missing content (creates empty file): %v", result.Err)
	}

	// 验证创建了空文件
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("File should be empty, got %d bytes", len(data))
	}
}

func TestWriteFileTool_Execute_LargeContent(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 创建大内容（1MB）
	largeContent := strings.Repeat("A", 1024*1024)

	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": largeContent,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with large content: %v", result.Err)
	}

	// 验证文件大小
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if info.Size() != int64(len(largeContent)) {
		t.Errorf("File size mismatch, expected %d, got %d", len(largeContent), info.Size())
	}

	// 验证返回消息
	if !strings.Contains(result.Raw, "Successfully wrote 1048576 bytes") {
		t.Errorf("Result should indicate correct byte count, got: %s", result.Raw)
	}
}

func TestWriteFileTool_Execute_MultilineContent(t *testing.T) {
	tool := NewWriteFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.txt")

	// 多行内容
	content := "Line 1\nLine 2\nLine 3\n"

	result := tool.Execute(context.Background(), map[string]string{
		"path":    testFile,
		"content": content,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证多行内容正确写入
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Multiline content mismatch")
	}
}
