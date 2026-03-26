package filesystem

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	if result.Raw != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result.Raw)
	}
}

func TestReadFileTool_Execute_EmptyPath(t *testing.T) {
	tool := NewReadFileTool()

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

func TestReadFileTool_Execute_FileNotExist(t *testing.T) {
	tool := NewReadFileTool()

	// 测试读取不存在的文件
	result := tool.Execute(context.Background(), map[string]string{
		"path": "/non/existent/file.txt",
	})

	if result.Err == nil {
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": "~/.wangshu_test_read_file.txt",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with tilde path: %v", result.Err)
	}

	if result.Raw != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result.Raw)
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": "test_relative.txt",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed with relative path: %v", result.Err)
	}

	if result.Raw != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result.Raw)
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for empty file: %v", result.Err)
	}

	if result.Raw != "" {
		t.Errorf("Expected empty content, got '%s'", result.Raw)
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for binary file: %v", result.Err)
	}

	expectedResult := string(binaryContent)
	if result.Raw != expectedResult {
		t.Errorf("Expected content '%s', got '%s'", expectedResult, result.Raw)
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
	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for large file: %v", result.Err)
	}

	if len(result.Raw) != len(largeContent) {
		t.Errorf("Large file content size mismatch, expected %d, got %d", len(largeContent), len(result.Raw))
	}
}

func TestReadFileTool_Execute_XLSX(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Name")
	f.SetCellValue("Sheet1", "B1", "Age")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", "30")

	if err := f.SaveAs(testFile); err != nil {
		t.Fatalf("Failed to create test XLSX file: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for XLSX file: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Sheet1") {
		t.Error("Result should contain sheet name")
	}
	if !strings.Contains(result.Raw, "Name") {
		t.Error("Result should contain 'Name'")
	}
	if !strings.Contains(result.Raw, "Alice") {
		t.Error("Result should contain 'Alice'")
	}
}

func TestReadFileTool_Execute_XLSX_MultiSheet(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multi_sheet.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Sheet1Data")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Sheet2Data")

	if err := f.SaveAs(testFile); err != nil {
		t.Fatalf("Failed to create test XLSX file: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for multi-sheet XLSX file: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Sheet1") || !strings.Contains(result.Raw, "Sheet2") {
		t.Error("Result should contain both sheet names")
	}
	if !strings.Contains(result.Raw, "Sheet1Data") || !strings.Contains(result.Raw, "Sheet2Data") {
		t.Error("Result should contain data from both sheets")
	}
}

func TestReadFileTool_Execute_DOCX(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.docx")

	docContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>
<w:p><w:r><w:t>This is a test document.</w:t></w:r></w:p>
</w:body>
</w:document>`

	zipFile, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create docx file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	docWriter, err := zipWriter.Create("word/document.xml")
	if err != nil {
		t.Fatalf("Failed to create document.xml in zip: %v", err)
	}
	docWriter.Write([]byte(docContent))

	contentTypesWriter, _ := zipWriter.Create("[Content_Types].xml")
	contentTypesWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`))

	relsWriter, _ := zipWriter.Create("_rels/.rels")
	relsWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`))

	docRelsWriter, _ := zipWriter.Create("word/_rels/document.xml.rels")
	docRelsWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`))

	zipWriter.Close()

	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for DOCX file: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Hello World") {
		t.Errorf("Result should contain 'Hello World', got: %s", result.Raw)
	}
}

func TestReadFileTool_Execute_UnsupportedFormat(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()

	docFile := filepath.Join(tmpDir, "test.doc")
	os.WriteFile(docFile, []byte("test"), 0644)

	result := tool.Execute(context.Background(), map[string]string{
		"path": docFile,
	})

	if result.Err == nil {
		t.Error("Execute should fail for .doc format")
	}
	if !strings.Contains(result.Err.Error(), "not supported") {
		t.Errorf("Error should mention format not supported, got: %v", result.Err)
	}

	xlsFile := filepath.Join(tmpDir, "test.xls")
	os.WriteFile(xlsFile, []byte("test"), 0644)

	result = tool.Execute(context.Background(), map[string]string{
		"path": xlsFile,
	})

	if result.Err == nil {
		t.Error("Execute should fail for .xls format")
	}
	if !strings.Contains(result.Err.Error(), "not supported") {
		t.Errorf("Error should mention format not supported, got: %v", result.Err)
	}
}

func TestReadFileTool_Execute_XLSX_EmptySheet(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SaveAs(testFile); err != nil {
		t.Fatalf("Failed to create empty XLSX file: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]string{
		"path": testFile,
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed for empty XLSX file: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Sheet1") {
		t.Error("Result should contain sheet name even for empty sheet")
	}
}
