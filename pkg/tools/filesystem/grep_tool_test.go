package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewGrepTool(t *testing.T) {
	tool := NewGrepTool()

	if tool == nil {
		t.Fatal("NewGrepTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "grep_search" {
		t.Errorf("Expected tool name 'grep_search', got '%s'", tool.Name())
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

	// 验证必需参数
	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "pattern" {
		t.Error("'pattern' should be required")
	}

	// 验证可选参数
	if _, ok := properties["pattern"]; !ok {
		t.Error("Parameters should have 'pattern' property")
	}
	if _, ok := properties["path"]; !ok {
		t.Error("Parameters should have 'path' property")
	}
	if _, ok := properties["include"]; !ok {
		t.Error("Parameters should have 'include' property")
	}
}

func TestGrepTool_Execute_SimpleSearch(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello World\nThis is a test\nGoodbye World"
	os.WriteFile(testFile, []byte(content), 0644)

	// 搜索"World"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "World",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了匹配
	if !strings.Contains(result, "Found 2 matches") {
		t.Errorf("Should find 2 matches, got: %s", result)
	}

	// 验证包含行号
	if !strings.Contains(result, ":1:") || !strings.Contains(result, ":3:") {
		t.Errorf("Result should contain line numbers, got: %s", result)
	}

	// 验证包含文件内容
	if !strings.Contains(result, "Hello World") || !strings.Contains(result, "Goodbye World") {
		t.Error("Result should contain matching lines")
	}
}

func TestGrepTool_Execute_RegexPattern(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test123\n456test\nabc\n123456"
	os.WriteFile(testFile, []byte(content), 0644)

	// 使用正则表达式搜索数字
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "\\d+",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了包含数字的行
	if !strings.Contains(result, "Found") {
		t.Errorf("Should find matches, got: %s", result)
	}
}

func TestGrepTool_Execute_WithFileFilter(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建不同类型的文件
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main\nfunc main() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("# Test"), 0644)

	// 只在.go文件中搜索"package"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "package",
		"path":    tmpDir,
		"include": "*.go",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证只搜索了.go文件
	if !strings.Contains(result, "test.go") {
		t.Errorf("Result should contain test.go, got: %s", result)
	}

	if strings.Contains(result, "test.txt") || strings.Contains(result, "test.md") {
		t.Error("Result should not contain non-.go files")
	}
}

func TestGrepTool_Execute_EmptyPattern(t *testing.T) {
	tool := NewGrepTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty pattern")
	}
}

func TestGrepTool_Execute_InvalidPattern(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 使用无效的正则表达式
	_, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "[invalid",
		"path":    tmpDir,
	})

	if err == nil {
		t.Error("Execute should fail with invalid regex pattern")
	}

	if !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("Error should mention invalid regex, got: %v", err)
	}
}

func TestGrepTool_Execute_NoMatches(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	// 搜索不存在的内容
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "NonExistent",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed with no matches: %v", err)
	}

	// 验证返回"未找到"消息
	if !strings.Contains(result, "No matches found") {
		t.Errorf("Result should indicate no matches, got: %s", result)
	}
}

func TestGrepTool_Execute_DefaultPath(t *testing.T) {
	tool := NewGrepTool()

	// 切换到临时目录
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// 创建测试文件
	os.WriteFile("test.txt", []byte("Hello World"), 0644)

	// 不指定路径，应该搜索当前目录
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "Hello",
	})

	if err != nil {
		t.Errorf("Execute should succeed with default path: %v", err)
	}

	// 验证找到了匹配
	if !strings.Contains(result, "Found") {
		t.Errorf("Should find matches with default path, got: %s", result)
	}
}

func TestGrepTool_Execute_CaseSensitive(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello World\nHELLO WORLD\nhello world"
	os.WriteFile(testFile, []byte(content), 0644)

	// 搜索"Hello"（大小写敏感）
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "Hello",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证只匹配大小写完全相同的
	if !strings.Contains(result, "Found 1 matches") {
		t.Errorf("Should find exactly 1 match (case sensitive), got: %s", result)
	}

	if !strings.Contains(result, "Hello World") {
		t.Error("Result should contain 'Hello World'")
	}

	if strings.Contains(result, "HELLO WORLD") || strings.Contains(result, "hello world") {
		t.Error("Result should not contain case-insensitive matches")
	}
}

func TestGrepTool_Execute_MultipleFiles(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建多个测试文件
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("Hello World"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("Hello Universe"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("Goodbye World"), 0644)

	// 搜索"World"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "World",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了多个文件中的匹配
	if !strings.Contains(result, "Found 2 matches") {
		t.Errorf("Should find 2 matches in different files, got: %s", result)
	}

	// 验证结果包含不同的文件
	if !strings.Contains(result, "file1.txt") && !strings.Contains(result, "file3.txt") {
		t.Error("Result should contain matching files")
	}
}

func TestGrepTool_Execute_SpecialCharacters(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建包含特殊字符的文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Price: $100\nEmail: test@example.com\nPhone: 123-456-7890"
	os.WriteFile(testFile, []byte(content), 0644)

	// 搜索包含特殊字符的内容
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "\\$100",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证能够匹配特殊字符
	if !strings.Contains(result, "Price: $100") {
		t.Errorf("Should match special characters, got: %s", result)
	}
}

func TestGrepTool_Execute_EmptyFile(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(testFile, []byte(""), 0644)

	// 搜索任何内容
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "anything",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed with empty file: %v", err)
	}

	// 验证返回"未找到"消息
	if !strings.Contains(result, "No matches found") {
		t.Errorf("Result should indicate no matches in empty file, got: %s", result)
	}
}

func TestGrepTool_Execute_SkipsHiddenDirectories(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建.git目录
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("Hello World"), 0644)

	// 创建普通文件
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Hello World"), 0644)

	// 搜索"Hello"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "Hello",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证只搜索了普通文件，没有搜索.git目录
	if strings.Contains(result, ".git") {
		t.Error("Result should not contain files from .git directory")
	}

	if !strings.Contains(result, "test.txt") {
		t.Error("Result should contain test.txt")
	}
}

func TestGrepTool_Execute_LineNumbers(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建多行文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	os.WriteFile(testFile, []byte(content), 0644)

	// 搜索"line"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "line",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证每行都有行号
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "test.txt:") {
			// 提取行号
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 2 {
				// 验证行号存在且不为空
				if parts[2] == "" {
					t.Errorf("Line %d should have line number, got: %s", i, line)
				}
			}
		}
	}

	// 验证找到了5个匹配
	if !strings.Contains(result, "Found 5 matches") {
		t.Errorf("Should find 5 matches, got: %s", result)
	}
}

func TestGrepTool_Execute_MultilinePattern(t *testing.T) {
	tool := NewGrepTool()
	tmpDir := t.TempDir()

	// 创建多行内容的文件
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "start\nmiddle line\nend"
	os.WriteFile(testFile, []byte(content), 0644)

	// 搜索"start"
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "start",
		"path":    tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了匹配
	if !strings.Contains(result, "Found 1 matches") {
		t.Errorf("Should find 1 match, got: %s", result)
	}

	if !strings.Contains(result, "start") {
		t.Error("Result should contain 'start'")
	}
}
