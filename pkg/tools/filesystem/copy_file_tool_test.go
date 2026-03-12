package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCopyFileTool(t *testing.T) {
	tool := NewCopyFileTool()

	if tool == nil {
		t.Fatal("NewCopyFileTool should not return nil")
	}

	if tool.Name() != "copy_file" {
		t.Errorf("Expected tool name 'copy_file', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	params := tool.Parameters()
	if params == nil {
		t.Fatal("Tool should have parameters")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) != 2 {
		t.Error("Should have 2 required parameters")
	}

	hasSourcePath := false
	hasTargetPath := false
	for _, r := range required {
		if r == "source_path" {
			hasSourcePath = true
		}
		if r == "target_path" {
			hasTargetPath = true
		}
	}
	if !hasSourcePath || !hasTargetPath {
		t.Error("Both 'source_path' and 'target_path' should be required")
	}

	if _, ok := properties["source_path"]; !ok {
		t.Error("Parameters should have 'source_path' property")
	}
	if _, ok := properties["target_path"]; !ok {
		t.Error("Parameters should have 'target_path' property")
	}
	if _, ok := properties["overwrite"]; !ok {
		t.Error("Parameters should have 'overwrite' property")
	}
}

func TestCopyFileTool_Execute_CopyFile(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "source.txt")
	targetPath := filepath.Join(tmpDir, "target.txt")

	content := "test content for copying"
	os.WriteFile(sourcePath, []byte(content), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should contain success message, got: %s", result)
	}

	sourceData, _ := os.ReadFile(sourcePath)
	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(sourceData) != string(targetData) {
		t.Errorf("Source and target content should match")
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		t.Error("Source file should still exist after copy")
	}
}

func TestCopyFileTool_Execute_CopyToDirectory(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "source.txt")
	targetDir := filepath.Join(tmpDir, "target")
	os.Mkdir(targetDir, 0755)

	content := "content to copy to directory"
	os.WriteFile(sourcePath, []byte(content), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	expectedTarget := filepath.Join(targetDir, "source.txt")
	data, err := os.ReadFile(expectedTarget)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content should be preserved")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_CopyDirectory(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	os.Mkdir(sourceDir, 0755)

	testFile1 := filepath.Join(sourceDir, "file1.txt")
	testFile2 := filepath.Join(sourceDir, "file2.txt")
	os.WriteFile(testFile1, []byte("content 1"), 0644)
	os.WriteFile(testFile2, []byte("content 2"), 0644)

	subDir := filepath.Join(sourceDir, "subdir")
	os.Mkdir(subDir, 0755)
	subFile := filepath.Join(subDir, "subfile.txt")
	os.WriteFile(subFile, []byte("sub content"), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourceDir,
		"target_path": targetDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("Target directory should exist")
	}

	copiedFile1 := filepath.Join(targetDir, "file1.txt")
	copiedFile2 := filepath.Join(targetDir, "file2.txt")
	copiedSubDir := filepath.Join(targetDir, "subdir")
	copiedSubFile := filepath.Join(copiedSubDir, "subfile.txt")

	if _, err := os.Stat(copiedFile1); os.IsNotExist(err) {
		t.Error("File1 should be copied")
	}
	if _, err := os.Stat(copiedFile2); os.IsNotExist(err) {
		t.Error("File2 should be copied")
	}
	if _, err := os.Stat(copiedSubDir); os.IsNotExist(err) {
		t.Error("Subdirectory should be copied")
	}
	if _, err := os.Stat(copiedSubFile); os.IsNotExist(err) {
		t.Error("File in subdirectory should be copied")
	}

	if !strings.Contains(result, "Successfully copied directory") {
		t.Errorf("Result should indicate directory copy, got: %s", result)
	}
}

func TestCopyFileTool_Execute_EmptySourcePath(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	_, err := tool.Execute(context.Background(), map[string]string{
		"source_path": "",
		"target_path": filepath.Join(tmpDir, "target.txt"),
	})

	if err == nil {
		t.Error("Execute should fail with empty source_path")
	}
}

func TestCopyFileTool_Execute_EmptyTargetPath(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "source.txt")
	os.WriteFile(sourcePath, []byte("content"), 0644)

	_, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty target_path")
	}
}

func TestCopyFileTool_Execute_BothPathsEmpty(t *testing.T) {
	tool := NewCopyFileTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"source_path": "",
		"target_path": "",
	})

	if err == nil {
		t.Error("Execute should fail when both paths are empty")
	}

	_, err = tool.Execute(context.Background(), map[string]string{})

	if err == nil {
		t.Error("Execute should fail when both parameters are missing")
	}
}

func TestCopyFileTool_Execute_SourceNotExist(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	_, err := tool.Execute(context.Background(), map[string]string{
		"source_path": filepath.Join(tmpDir, "nonexistent.txt"),
		"target_path": filepath.Join(tmpDir, "target.txt"),
	})

	if err == nil {
		t.Error("Execute should fail when source file does not exist")
	}
}

func TestCopyFileTool_Execute_TargetExistsNoOverwrite(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "source.txt")
	targetPath := filepath.Join(tmpDir, "target.txt")

	os.WriteFile(sourcePath, []byte("source content"), 0644)
	os.WriteFile(targetPath, []byte("target content"), 0644)

	_, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
	})

	if err == nil {
		t.Error("Execute should fail when target exists and overwrite is false")
	}

	data, _ := os.ReadFile(targetPath)
	if string(data) != "target content" {
		t.Error("Target file should not be modified")
	}
}

func TestCopyFileTool_Execute_TargetExistsWithOverwrite(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "source.txt")
	targetPath := filepath.Join(tmpDir, "target.txt")

	os.WriteFile(sourcePath, []byte("source content"), 0644)
	os.WriteFile(targetPath, []byte("target content"), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
		"overwrite":   "true",
	})

	if err != nil {
		t.Errorf("Execute should succeed with overwrite=true: %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(data) != "source content" {
		t.Error("Target file should be overwritten with source content")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_TildeExpansion(t *testing.T) {
	tool := NewCopyFileTool()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}

	sourcePath := filepath.Join(homeDir, ".wangshu_test_copy_source.txt")
	targetPath := filepath.Join(homeDir, ".wangshu_test_copy_target.txt")
	content := "tilde test"

	defer os.Remove(sourcePath)
	defer os.Remove(targetPath)

	os.WriteFile(sourcePath, []byte(content), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": "~/.wangshu_test_copy_source.txt",
		"target_path": "~/.wangshu_test_copy_target.txt",
	})

	if err != nil {
		t.Errorf("Execute should succeed with tilde paths: %v", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		t.Error("Source file should still exist")
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(data) != content {
		t.Error("File content should be preserved")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_RelativePath(t *testing.T) {
	tool := NewCopyFileTool()

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

	sourcePath := "source.txt"
	targetPath := "target.txt"
	content := "relative path test"

	os.WriteFile(sourcePath, []byte(content), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
	})

	if err != nil {
		t.Errorf("Execute should succeed with relative paths: %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(data) != content {
		t.Error("File content should be preserved")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_CopyToSameLocation(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "file.txt")
	content := "test content"
	os.WriteFile(filePath, []byte(content), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": filePath,
		"target_path": filePath,
		"overwrite":   "true",
	})

	if err != nil {
		t.Errorf("Execute should succeed when copying to same location with overwrite: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Error("File content should be unchanged")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_CopyBinaryFile(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "binary.bin")
	targetPath := filepath.Join(tmpDir, "binary_copy.bin")

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x80, 0x7F}
	os.WriteFile(sourcePath, binaryData, 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
	})

	if err != nil {
		t.Errorf("Execute should succeed for binary file: %v", err)
	}

	copiedData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if len(copiedData) != len(binaryData) {
		t.Errorf("Binary data length mismatch: got %d, want %d", len(copiedData), len(binaryData))
	}

	for i := range binaryData {
		if copiedData[i] != binaryData[i] {
			t.Errorf("Binary data mismatch at byte %d: got 0x%02X, want 0x%02X", i, copiedData[i], binaryData[i])
		}
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}

func TestCopyFileTool_Execute_CopyLargeFile(t *testing.T) {
	tool := NewCopyFileTool()
	tmpDir := t.TempDir()

	sourcePath := filepath.Join(tmpDir, "large.txt")
	targetPath := filepath.Join(tmpDir, "large_copy.txt")

	largeContent := strings.Repeat("This is a test line.\n", 10000)
	os.WriteFile(sourcePath, []byte(largeContent), 0644)

	result, err := tool.Execute(context.Background(), map[string]string{
		"source_path": sourcePath,
		"target_path": targetPath,
	})

	if err != nil {
		t.Errorf("Execute should succeed for large file: %v", err)
	}

	copiedData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedData) != largeContent {
		t.Error("Large file content should be preserved")
	}

	if !strings.Contains(result, "Successfully copied") {
		t.Errorf("Result should indicate success, got: %s", result)
	}
}
