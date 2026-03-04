package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEditFileTool(t *testing.T) {
	tool := NewEditFileTool()

	if tool == nil {
		t.Fatal("NewEditFileTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "edit_file" {
		t.Errorf("Expected tool name 'edit_file', got '%s'", tool.Name())
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
	if !ok || len(required) != 3 {
		t.Error("Should have 3 required parameters")
	}

	// 验证所有必需参数
	requiredParams := make(map[string]bool)
	for _, r := range required {
		requiredParams[r] = true
	}

	if !requiredParams["file_path"] || !requiredParams["old_str"] || !requiredParams["new_str"] {
		t.Error("All of 'file_path', 'old_str', 'new_str' should be required")
	}

	// 验证参数属性
	if _, ok := properties["file_path"]; !ok {
		t.Error("Parameters should have 'file_path' property")
	}
	if _, ok := properties["old_str"]; !ok {
		t.Error("Parameters should have 'old_str' property")
	}
	if _, ok := properties["new_str"]; !ok {
		t.Error("Parameters should have 'new_str' property")
	}
}

func TestEditFileTool_Execute_SimpleReplace(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	// 使用唯一的内容避免歧义
	originalContent := "Hello World\nGoodbye Universe"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	// 替换"World"为"Universe"
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Hello World",
		"new_str":   "Hi Universe",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证返回成功消息
	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证文件已修改
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "Hi Universe\nGoodbye Universe"
	if string(modifiedContent) != expectedContent {
		t.Errorf("File content not updated correctly, expected: %s, got: %s", expectedContent, string(modifiedContent))
	}
}

func TestEditFileTool_Execute_MissingFilePath(t *testing.T) {
	tool := NewEditFileTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": "",
		"old_str":   "old",
		"new_str":   "new",
	})

	if err == nil {
		t.Error("Execute should fail with missing file_path")
	}

	// 测试缺少file_path参数
	_, err = tool.Execute(context.Background(), map[string]string{
		"old_str": "old",
		"new_str": "new",
	})

	if err == nil {
		t.Error("Execute should fail when file_path parameter is missing")
	}
}

func TestEditFileTool_Execute_MissingOldStr(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "",
		"new_str":   "new",
	})

	if err == nil {
		t.Error("Execute should fail with missing old_str")
	}
}

func TestEditFileTool_Execute_OldStrNotFound(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello World\nGoodbye World"
	os.WriteFile(testFile, []byte(content), 0644)

	// 尝试替换不存在的内容
	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "NonExistent",
		"new_str":   "Replacement",
	})

	if err == nil {
		t.Error("Execute should fail when old_str not found")
	}

	// 验证错误消息包含预览
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}

	if !strings.Contains(err.Error(), "File preview") {
		t.Errorf("Error should contain file preview, got: %v", err)
	}
}

func TestEditFileTool_Execute_AmbiguousMatch(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello World\nHello World\nHello World"
	os.WriteFile(testFile, []byte(content), 0644)

	// 尝试替换出现多次的内容
	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Hello World",
		"new_str":   "Goodbye Universe",
	})

	if err == nil {
		t.Error("Execute should fail when old_str appears multiple times")
	}

	// 验证错误消息
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("Error should mention 'ambiguous', got: %v", err)
	}

	if !strings.Contains(err.Error(), "3 occurrences") {
		t.Errorf("Error should mention occurrence count, got: %v", err)
	}
}

func TestEditFileTool_Execute_PreserveWhitespace(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	// 注意缩进
	content := "Line 1\n  Indented Line\nLine 3"
	os.WriteFile(testFile, []byte(content), 0644)

	// 替换包括缩进的内容
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "  Indented Line",
		"new_str":   "  Modified Indented Line",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证缩进被保留
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "Line 1\n  Modified Indented Line\nLine 3"
	if string(modifiedContent) != expectedContent {
		t.Errorf("Whitespace not preserved, expected: %s, got: %s", expectedContent, string(modifiedContent))
	}
}

func TestEditFileTool_Execute_MultilineReplace(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4"
	os.WriteFile(testFile, []byte(content), 0644)

	// 替换多行内容
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Line 2\nLine 3",
		"new_str":   "Modified Line 2\nModified Line 3",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证多行替换成功
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "Line 1\nModified Line 2\nModified Line 3\nLine 4"
	if string(modifiedContent) != expectedContent {
		t.Errorf("Multiline replace failed, expected: %s, got: %s", expectedContent, string(modifiedContent))
	}
}

func TestEditFileTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewEditFileTool()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	// 创建测试文件
	testFile := filepath.Join(homeDir, ".wangshu_test_edit.txt")
	originalContent := "Hello World"
	os.WriteFile(testFile, []byte(originalContent), 0644)
	defer os.Remove(testFile)

	// 使用波浪号路径编辑
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": "~/.wangshu_test_edit.txt",
		"old_str":   "Hello World",
		"new_str":   "Goodbye Universe",
	})

	if err != nil {
		t.Errorf("Execute should succeed with tilde path: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证文件已修改
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != "Goodbye Universe" {
		t.Errorf("File content not updated correctly")
	}
}

func TestEditFileTool_Execute_FileNotExist(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": filepath.Join(tmpDir, "nonexistent.txt"),
		"old_str":   "old",
		"new_str":   "new",
	})

	if err == nil {
		t.Error("Execute should fail when file does not exist")
	}

	if !strings.Contains(err.Error(), "read failed") {
		t.Errorf("Error should mention 'read failed', got: %v", err)
	}
}

func TestEditFileTool_Execute_ReplaceWithEmpty(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Keep This\nRemove This\nKeep That"
	os.WriteFile(testFile, []byte(content), 0644)

	// 替换为空字符串（删除）
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Remove This",
		"new_str":   "",
	})

	if err != nil {
		t.Errorf("Execute should succeed with empty new_str: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证内容被删除
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "Keep This\n\nKeep That"
	if string(modifiedContent) != expectedContent {
		t.Errorf("Content not deleted correctly, expected: %s, got: %s", expectedContent, string(modifiedContent))
	}
}

func TestEditFileTool_Execute_SpecialCharacters(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := `Price: $100
Email: test@example.com
Phone: 123-456-7890`
	os.WriteFile(testFile, []byte(content), 0644)

	// 替换包含特殊字符的行
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Price: $100",
		"new_str":   "Price: $200",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证特殊字符被正确处理
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if !strings.Contains(string(modifiedContent), "Price: $200") {
		t.Error("Special characters not handled correctly")
	}

	if strings.Contains(string(modifiedContent), "Price: $100") {
		t.Error("Old content should be replaced")
	}
}

func TestEditFileTool_Execute_EmptyFile(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte(""), 0644)

	// 在空文件中搜索
	_, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "anything",
		"new_str":   "replacement",
	})

	if err == nil {
		t.Error("Execute should fail when old_str not found in empty file")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestEditFileTool_Execute_ReplaceExactMatch(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	// 创建有重复内容但有细微差异的情况
	content := `function test() {
    return "hello";
}
function another() {
    return "world";
}`
	os.WriteFile(testFile, []byte(content), 0644)

	// 替换第一个函数（包含足够的上下文使其唯一）
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str": `function test() {
    return "hello";
}`,
		"new_str": `function test() {
    return "goodbye";
}`,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证只替换了第一个函数，第二个保持不变
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if !strings.Contains(string(modifiedContent), `return "goodbye";`) {
		t.Error("First function should be modified")
	}

	if !strings.Contains(string(modifiedContent), `return "world";`) {
		t.Error("Second function should remain unchanged")
	}
}

func TestEditFileTool_Execute_LongContent(t *testing.T) {
	tool := NewEditFileTool()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	// 创建一个较长的文件，每行内容都不同
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("Line %d\n", i))
	}
	longContent := strings.Join(lines, "")
	os.WriteFile(testFile, []byte(longContent), 0644)

	// 替换中间的某一行（使用唯一的内容）
	result, err := tool.Execute(context.Background(), map[string]string{
		"file_path": testFile,
		"old_str":   "Line 50\n",
		"new_str":   "Modified Line 50\n",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证执行成功
	if result != "Success" {
		t.Errorf("Result should be 'Success', got: %s", result)
	}

	// 验证只替换了第50行
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if !strings.Contains(string(modifiedContent), "Modified Line 50") {
		t.Error("Line 50 should be modified")
	}

	if !strings.Contains(string(modifiedContent), "Line 49\n") {
		t.Error("Line 49 should remain unchanged")
	}

	if !strings.Contains(string(modifiedContent), "Line 51\n") {
		t.Error("Line 51 should remain unchanged")
	}
}
