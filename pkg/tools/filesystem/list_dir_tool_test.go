package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewListDirectoryTool(t *testing.T) {
	tool := NewListDirectoryTool()

	if tool == nil {
		t.Fatal("NewListDirectoryTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "list_directory" {
		t.Errorf("Expected tool name 'list_directory', got '%s'", tool.Name())
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

	if _, ok := properties["path"]; !ok {
		t.Error("Parameters should have 'path' property")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "path" {
		t.Error("'path' should be required")
	}
}

func TestListDirectoryTool_Execute_EmptyDirectory(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 列出空目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证结果包含目录路径
	if !strings.Contains(result.Raw, tmpDir) {
		t.Errorf("Result should contain directory path, got: %s", result.Raw)
	}

	// 验证结果包含标题
	if !strings.Contains(result.Raw, "Contents of") {
		t.Errorf("Result should contain 'Contents of', got: %s", result.Raw)
	}
}

func TestListDirectoryTool_Execute_WithFiles(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建测试文件
	testFiles := []string{"file1.txt", "file2.md", "file3.log"}
	for _, filename := range testFiles {
		path := filepath.Join(tmpDir, filename)
		err := os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// 列出目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证所有文件都被列出
	for _, filename := range testFiles {
		if !strings.Contains(result.Raw, filename) {
			t.Errorf("Result should contain file '%s', got: %s", filename, result.Raw)
		}
	}

	// 验证文件标记
	if !strings.Contains(result.Raw, "[FILE]") {
		t.Errorf("Result should contain [FILE] marker, got: %s", result.Raw)
	}

	// 验证大小信息
	if !strings.Contains(result.Raw, "bytes") {
		t.Errorf("Result should contain file size, got: %s", result.Raw)
	}
}

func TestListDirectoryTool_Execute_WithSubdirectories(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建子目录
	subdirs := []string{"subdir1", "subdir2", "subdir3"}
	for _, dirname := range subdirs {
		path := filepath.Join(tmpDir, dirname)
		err := os.Mkdir(path, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}
	}

	// 列出目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证所有子目录都被列出
	for _, dirname := range subdirs {
		if !strings.Contains(result.Raw, dirname) {
			t.Errorf("Result should contain directory '%s', got: %s", dirname, result.Raw)
		}
	}

	// 验证目录标记
	if !strings.Contains(result.Raw, "[DIR]") {
		t.Errorf("Result should contain [DIR] marker, got: %s", result.Raw)
	}
}

func TestListDirectoryTool_Execute_MixedContents(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建文件
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)

	// 创建子目录
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	// 列出目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证同时包含文件和目录标记
	if !strings.Contains(result.Raw, "[FILE]") {
		t.Error("Result should contain [FILE] marker")
	}

	if !strings.Contains(result.Raw, "[DIR]") {
		t.Error("Result should contain [DIR] marker")
	}

	if !strings.Contains(result.Raw, "file.txt") {
		t.Error("Result should contain file.txt")
	}

	if !strings.Contains(result.Raw, "subdir") {
		t.Error("Result should contain subdir")
	}
}

func TestListDirectoryTool_Execute_NotExist(t *testing.T) {
	tool := NewListDirectoryTool()

	// 列出不存在的目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": "/non/existent/directory",
	})

	if result.Err == nil {
		t.Error("Execute should fail for non-existent directory")
	}
}

func TestListDirectoryTool_Execute_EmptyPath(t *testing.T) {
	tool := NewListDirectoryTool()

	// 测试空路径
	result := tool.Execute(context.Background(), map[string]string{
		"path": "",
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty path")
	}

	// 测试缺少path参数
	result = tool.Execute(context.Background(), map[string]string{})

	if result.Err == nil {
		t.Error("Execute should fail when path parameter is missing")
	}
}

func TestListDirectoryTool_Execute_FileInsteadOfDirectory(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建文件而不是目录
	testFile := filepath.Join(tmpDir, "notadir.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// 尝试列出文件
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err == nil {
		t.Error("Execute should fail when path is a file, not a directory")
	}
}

func TestListDirectoryTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewListDirectoryTool()

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	// 创建测试子目录
	testDir := filepath.Join(homeDir, ".wangshu_test_list_dir")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// 使用波浪号路径列出子目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": "~/.wangshu_test_list_dir",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with tilde path: %v", result.Err)
	}

	// 验证结果包含目录列表
	if !strings.Contains(result.Raw, "Contents of") {
		t.Errorf("Result should contain directory listing, got: %s", result.Raw)
	}

	// 验证路径被正确扩展
	if strings.Contains(result.Raw, "~") {
		t.Error("Tilde should be expanded in result path")
	}
}

func TestListDirectoryTool_Execute_RelativePath(t *testing.T) {
	tool := NewListDirectoryTool()

	// 使用当前目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": ".",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with relative path: %v", result.Err)
	}

	// 验证结果包含内容
	if !strings.Contains(result.Raw, "Contents of") {
		t.Errorf("Result should contain directory listing, got: %s", result.Raw)
	}
}

func TestListDirectoryTool_Execute_NestedDirectory(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建深层嵌套目录结构
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// 在深层目录中创建文件
	testFile := filepath.Join(nestedDir, "deepfile.txt")
	os.WriteFile(testFile, []byte("deep content"), 0644)

	// 列出深层目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": nestedDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for nested directory: %v", result.Err)
	}

	// 验证包含文件
	if !strings.Contains(result.Raw, "deepfile.txt") {
		t.Errorf("Result should contain deepfile.txt, got: %s", result.Raw)
	}
}

func TestListDirectoryTool_Execute_HiddenFiles(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建隐藏文件（以.开头）
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	os.WriteFile(hiddenFile, []byte("hidden content"), 0644)

	// 创建普通文件
	normalFile := filepath.Join(tmpDir, "normal.txt")
	os.WriteFile(normalFile, []byte("normal content"), 0644)

	// 列出目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	// 验证两个文件都被列出
	if !strings.Contains(result.Raw, ".hidden") {
		t.Error("Result should contain hidden file .hidden")
	}

	if !strings.Contains(result.Raw, "normal.txt") {
		t.Error("Result should contain normal.txt")
	}
}

func TestListDirectoryTool_Execute_LargeDirectory(t *testing.T) {
	tool := NewListDirectoryTool()
	tmpDir := t.TempDir()

	// 创建大量文件
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%03d.txt", i))
		err := os.WriteFile(filename, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// 列出目录
	result := tool.Execute(context.Background(), map[string]string{
		"path": tmpDir,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for large directory: %v", result.Err)
	}

	// 验证至少包含一些文件
	if !strings.Contains(result.Raw, "file000.txt") {
		t.Error("Result should contain file000.txt")
	}

	if !strings.Contains(result.Raw, "file099.txt") {
		t.Error("Result should contain file099.txt")
	}
}
