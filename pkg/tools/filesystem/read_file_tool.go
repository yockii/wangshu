package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/xuri/excelize/v2"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
)

type ReadFileTool struct {
	basic.SimpleTool
}

func NewReadFileTool() *ReadFileTool {
	tool := new(ReadFileTool)
	tool.Name_ = constant.ToolNameFSRead
	tool.Desc_ = "Read the content of a file. Supports plain text, PDF, DOCX, and XLSX formats. Returns file content as string."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file to read",
			},
		},
		"required": []string{"path"},
	}
	return tool
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	path := params["path"]
	if path == "" {
		return types.NewToolResult().WithError(fmt.Errorf("path is required"))
	}

	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".pdf":
		return t.readPDF(path)
	case ".docx":
		return t.readDOCX(path)
	case ".xlsx":
		return t.readXLSX(path)
	case ".doc":
		return types.NewToolResult().WithError(fmt.Errorf(".doc format is not supported, please convert to .docx"))
	case ".xls":
		return types.NewToolResult().WithError(fmt.Errorf(".xls format is not supported, please convert to .xlsx"))
	default:
		content, err := os.ReadFile(path)
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("failed to read file: %w", err))
		}
		r := string(content)
		return types.NewToolResult().WithRaw(r).WithStructured(actiontypes.NewFsReadData(path, r, "text"))
	}
}

func (t *ReadFileTool) readPDF(path string) *types.ToolResult {
	file, r, err := pdf.Open(path)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to open PDF: %w", err))
	}
	defer file.Close()

	var builder strings.Builder
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		builder.WriteString(fmt.Sprintf("--- Page %d ---\n", pageNum))
		builder.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			builder.WriteString("\n")
		}
	}

	result := builder.String()
	if result == "" {
		return types.NewToolResult().WithError(fmt.Errorf("no text content found in PDF"))
	}
	return types.NewToolResult().WithRaw(result).WithStructured(actiontypes.NewFsReadData(path, result, "pdf"))
}

func (t *ReadFileTool) readDOCX(path string) *types.ToolResult {
	doc, err := docx.ReadDocxFile(path)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to open DOCX: %w", err))
	}
	defer doc.Close()

	docContent := doc.Editable()
	content := docContent.GetContent()

	if content == "" {
		return types.NewToolResult().WithError(fmt.Errorf("no text content found in DOCX"))
	}
	return types.NewToolResult().WithRaw(content).WithStructured(actiontypes.NewFsReadData(path, content, "docx"))
}

func (t *ReadFileTool) readXLSX(path string) *types.ToolResult {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to open XLSX: %w", err))
	}
	defer f.Close()

	sheets := f.GetSheetList()

	var sheetStruct map[string]any = make(map[string]any)

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			sheetStruct[sheet] = fmt.Sprintf("Error reading sheet: %v", err)
			continue
		}

		if len(rows) == 0 {
			sheetStruct[sheet] = "(empty sheet)"
			continue
		}

		rowsStructData := make([]map[string]string, len(rows))

		for _, row := range rows {
			colStructData := make(map[string]string)
			for _, cell := range row {
				colStructData[cell] = cell
			}
			rowsStructData = append(rowsStructData, colStructData)
		}
		sheetStruct[sheet] = rowsStructData
	}

	var builder strings.Builder
	for name, data := range sheetStruct {
		builder.WriteString(fmt.Sprintf("=== Sheet: %s ===\n", name))
		switch data.(type) {
		case string:
			builder.WriteString(data.(string))
			builder.WriteString("\n")
		case []map[string]string:
			for _, row := range data.([]map[string]string) {
				for cell := range row {
					builder.WriteString(fmt.Sprintf("%s: %s\n", cell, row[cell]))
				}
				builder.WriteString("\n")
			}
		}
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return types.NewToolResult().WithError(fmt.Errorf("no content found in XLSX"))
	}
	return types.NewToolResult().WithRaw(result).WithStructured(actiontypes.NewFsReadData(path, result, "xlsx"))
}
