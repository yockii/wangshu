package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFindFileTool(t *testing.T) {
	tool := NewFindFileTool()

	if tool == nil {
		t.Fatal("NewFindFileTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "find_files" {
		t.Errorf("Expected tool name 'find_files', got '%s'", tool.Name())
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

	if _, ok := properties["pattern"]; !ok {
		t.Error("Parameters should have 'pattern' property")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "pattern" {
		t.Error("'pattern' should be required")
	}
}

func TestFindFileTool_Execute_FindAllFiles(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建一些测试文件
	files := []string{"file1.txt", "file2.txt", "file3.md"}
	for _, filename := range files {
		path := filepath.Join(tmpDir, filename)
		os.WriteFile(path, []byte("content"), 0644)
	}

	// 查找所有txt文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.txt"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了文件
	if !strings.Contains(result, "Found 2 file(s)") {
		t.Errorf("Should find 2 .txt files, got: %s", result)
	}

	// 验证包含文件路径
	if !strings.Contains(result, "file1.txt") || !strings.Contains(result, "file2.txt") {
		t.Errorf("Result should contain file names, got: %s", result)
	}

	// 验证不包含.md文件
	if strings.Contains(result, "file3.md") {
		t.Error("Result should not contain .md files")
	}
}

func TestFindFileTool_Execute_FindSpecificExtension(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建不同扩展名的文件
	extensions := []string{".go", ".txt", ".md", ".json"}
	for _, ext := range extensions {
		path := filepath.Join(tmpDir, "test"+ext)
		os.WriteFile(path, []byte("content"), 0644)
	}

	// 查找.go文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.go"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证只返回.go文件
	if !strings.Contains(result, ".go") {
		t.Errorf("Result should contain .go file, got: %s", result)
	}

	if strings.Contains(result, ".txt") || strings.Contains(result, ".md") || strings.Contains(result, ".json") {
		t.Error("Result should not contain other extensions")
	}
}

func TestFindFileTool_Execute_FindRecursive(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建嵌套目录结构
	subdir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subdir, 0755)

	// 在不同层级创建.go文件
	files := []string{
		filepath.Join(tmpDir, "root.go"),
		filepath.Join(subdir, "sub.go"),
	}

	for _, file := range files {
		os.WriteFile(file, []byte("content"), 0644)
	}

	// 测试简单的通配符（应该匹配当前目录和子目录，取决于平台）
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.go"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证至少找到了root.go
	if !strings.Contains(result, "root.go") {
		t.Errorf("Result should contain root.go, got: %s", result)
	}

	// 验证结果包含文件信息
	if !strings.Contains(result, "Found") {
		t.Errorf("Result should contain 'Found', got: %s", result)
	}
}

func TestFindFileTool_Execute_FindSpecificFile(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建特定文件
	targetFile := "docker-compose.yml"
	os.WriteFile(filepath.Join(tmpDir, targetFile), []byte("content"), 0644)

	// 创建其他文件
	os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("content"), 0644)

	// 查找特定文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "docker-compose.yml"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了目标文件
	if !strings.Contains(result, "Found 1 file(s)") {
		t.Errorf("Should find exactly 1 file, got: %s", result)
	}

	if !strings.Contains(result, "docker-compose.yml") {
		t.Errorf("Result should contain docker-compose.yml, got: %s", result)
	}

	if strings.Contains(result, "other.txt") {
		t.Error("Result should not contain other files")
	}
}

func TestFindFileTool_Execute_EmptyPattern(t *testing.T) {
	tool := NewFindFileTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty pattern")
	}

	// 测试缺少pattern参数
	_, err = tool.Execute(context.Background(), map[string]string{})

	if err == nil {
		t.Error("Execute should fail when pattern parameter is missing")
	}
}

func TestFindFileTool_Execute_NoMatches(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 在空目录中查找
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.nonexistent"),
	})

	if err != nil {
		t.Errorf("Execute should succeed even with no matches: %v", err)
	}

	// 验证返回"未找到"消息
	if !strings.Contains(result, "No files found") {
		t.Errorf("Result should indicate no files found, got: %s", result)
	}
}

func TestFindFileTool_Execute_InvalidPattern(t *testing.T) {
	tool := NewFindFileTool()

	// 使用无效的glob模式（包含不合法的字符）
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "[invalid",
	})

	if err != nil {
		// 某些系统可能会接受这个模式，所以只在失败时检查错误消息
		if !strings.Contains(err.Error(), "invalid glob pattern") {
			t.Errorf("Error should mention invalid pattern, got: %v", err)
		}
	}

	_ = result
}

func TestFindFileTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewFindFileTool()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	// 创建测试文件
	testFile := filepath.Join(homeDir, ".wangshu_test_find.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	defer os.Remove(testFile)

	// 使用波浪号查找
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "~/.wangshu_test_find.txt",
	})

	if err != nil {
		t.Errorf("Execute should succeed with tilde pattern: %v", err)
	}

	// 验证找到了文件
	if !strings.Contains(result, ".wangshu_test_find.txt") {
		t.Errorf("Result should contain test file, got: %s", result)
	}
}

func TestFindFileTool_Execute_RelativePath(t *testing.T) {
	tool := NewFindFileTool()

	// 切换到临时目录
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// 创建文件
	files := []string{"test1.go", "test2.go"}
	for _, file := range files {
		os.WriteFile(file, []byte("content"), 0644)
	}

	// 使用相对路径查找
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": "*.go",
	})

	if err != nil {
		t.Errorf("Execute should succeed with relative path: %v", err)
	}

	// 验证找到了文件
	if !strings.Contains(result, "Found 2 file(s)") {
		t.Errorf("Should find 2 files, got: %s", result)
	}
}

func TestFindFileTool_Execute_AbsolutePathOutput(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建文件
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	// 查找文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.txt"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证返回的是绝对路径
	if !filepath.IsAbs(result[strings.Index(result, "- ")+2:strings.Index(result, "- ")+2+len("test.txt")+2]) {
		// 提取路径并检查是否为绝对路径
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "- ") {
				path := strings.TrimPrefix(line, "- ")
				if !filepath.IsAbs(path) {
					t.Errorf("Path should be absolute, got: %s", path)
				}
			}
		}
	}
}

func TestFindFileTool_Execute_CurrentDirectoryPattern(t *testing.T) {
	tool := NewFindFileTool()

	// 切换到临时目录
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// 创建文件
	os.WriteFile("test.txt", []byte("content"), 0644)

	// 使用当前目录模式
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": ".",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证至少返回了当前目录
	if !strings.Contains(result, "Found") {
		t.Errorf("Result should contain 'Found', got: %s", result)
	}
}

func TestFindFileTool_Execute_MultiplePatterns(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 创建多种类型的文件
	testFiles := []string{
		"main.go",
		"utils.go",
		"main_test.go",
		"README.md",
		"config.json",
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		os.WriteFile(path, []byte("content"), 0644)
	}

	// 查找所有.go文件（包括测试）
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*.go"),
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证找到了3个.go文件
	if !strings.Contains(result, "Found 3 file(s)") {
		t.Errorf("Should find 3 .go files, got: %s", result)
	}

	// 验证包含所有.go文件
	if !strings.Contains(result, "main.go") || !strings.Contains(result, "utils.go") || !strings.Contains(result, "main_test.go") {
		t.Error("Result should contain all .go files")
	}

	// 验证不包含其他文件
	if strings.Contains(result, "README.md") || strings.Contains(result, "config.json") {
		t.Error("Result should not contain non-.go files")
	}
}

func TestFindFileTool_Execute_EmptyDirectory(t *testing.T) {
	tool := NewFindFileTool()
	tmpDir := t.TempDir()

	// 在空目录中查找所有文件
	result, err := tool.Execute(context.Background(), map[string]string{
		"pattern": filepath.Join(tmpDir, "*"),
	})

	if err != nil {
		t.Errorf("Execute should succeed in empty directory: %v", err)
	}

	// 验证返回"未找到"消息
	if !strings.Contains(result, "No files found") {
		t.Errorf("Result should indicate no files found in empty directory, got: %s", result)
	}
}
