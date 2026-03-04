package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewReadFileTool(t *testing.T) {
	tool := NewReadFileTool()

	if tool == nil {
		t.Fatal("NewReadFileTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "read_file" {
		t.Errorf("Expected tool name 'read_file', got '%s'", tool.Name())
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

	// 验证参数结构
	if params["type"] != "object" {
		t.Error("Parameters type should be 'object'")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	if _, ok := properties["path"]; !ok {
		t.Error("Parameters should have 'path' property")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "path" {
		t.Error("'path' should be required")
	}
}

func TestReadFileTool_Execute_Success(t *testing.T) {
	tool := NewReadFileTool()

	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	expectedContent := "Hello, World!\nThis is a test file."

	err := os.WriteFile(testFile, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 读取文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result)
	}
}

func TestReadFileTool_Execute_EmptyPath(t *testing.T) {
	tool := NewReadFileTool()

	// 测试空路径
	_, err := tool.Execute(context.Background(), map[string]string{
		"path": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty path")
	}

	// 测试缺少path参数
	_, err = tool.Execute(context.Background(), map[string]string{})

	if err == nil {
		t.Error("Execute should fail when path parameter is missing")
	}
}

func TestReadFileTool_Execute_FileNotExist(t *testing.T) {
	tool := NewReadFileTool()

	// 测试读取不存在的文件
	_, err := tool.Execute(context.Background(), map[string]string{
		"path": "/non/existent/file.txt",
	})

	if err == nil {
		t.Error("Execute should fail for non-existent file")
	}
}

func TestReadFileTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewReadFileTool()

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	// 创建临时文件
	testFile := filepath.Join(homeDir, ".wangshu_test_read_file.txt")
	expectedContent := "Tilde expansion test"

	defer os.Remove(testFile)

	err = os.WriteFile(testFile, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 使用波浪号路径读取
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": "~/.wangshu_test_read_file.txt",
	})

	if err != nil {
		t.Errorf("Execute should succeed with tilde path: %v", err)
	}

	if result != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result)
	}
}

func TestReadFileTool_Execute_RelativePath(t *testing.T) {
	tool := NewReadFileTool()

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// 创建临时文件
	testFile := filepath.Join(cwd, "test_relative.txt")
	expectedContent := "Relative path test"

	defer os.Remove(testFile)

	err = os.WriteFile(testFile, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 使用相对路径读取
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": "test_relative.txt",
	})

	if err != nil {
		t.Errorf("Execute should succeed with relative path: %v", err)
	}

	if result != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result)
	}
}

func TestReadFileTool_Execute_EmptyFile(t *testing.T) {
	tool := NewReadFileTool()

	// 创建空文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	// 读取空文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("Execute should succeed for empty file: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty content, got '%s'", result)
	}
}

func TestReadFileTool_Execute_BinaryFile(t *testing.T) {
	tool := NewReadFileTool()

	// 创建二进制文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "binary.bin")

	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	err := os.WriteFile(testFile, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}

	// 读取二进制文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("Execute should succeed for binary file: %v", err)
	}

	expectedResult := string(binaryContent)
	if result != expectedResult {
		t.Errorf("Binary content mismatch")
	}
}

func TestReadFileTool_Execute_LargeFile(t *testing.T) {
	tool := NewReadFileTool()

	// 创建较大的文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 创建1KB的文件
	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	err := os.WriteFile(testFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// 读取大文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("Execute should succeed for large file: %v", err)
	}

	if len(result) != len(largeContent) {
		t.Errorf("Large file content size mismatch, expected %d, got %d", len(largeContent), len(result))
	}
}
