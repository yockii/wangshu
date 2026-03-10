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
	"github.com/yockii/wangshu/pkg/tools/basic"
)

type ReadFileTool struct {
	basic.SimpleTool
}

func NewReadFileTool() *ReadFileTool {
	tool := new(ReadFileTool)
	tool.Name_ = "read_file"
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

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	path := params["path"]
	if path == "" {
		return "", fmt.Errorf("path is required")
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
		return "", fmt.Errorf(".doc format is not supported, please convert to .docx")
	case ".xls":
		return "", fmt.Errorf(".xls format is not supported, please convert to .xlsx")
	default:
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(content), nil
	}
}

func (t *ReadFileTool) readPDF(path string) (string, error) {
	file, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
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
		return "", fmt.Errorf("no text content found in PDF")
	}
	return result, nil
}

func (t *ReadFileTool) readDOCX(path string) (string, error) {
	doc, err := docx.ReadDocxFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX: %w", err)
	}
	defer doc.Close()

	docContent := doc.Editable()
	content := docContent.GetContent()

	if content == "" {
		return "", fmt.Errorf("no text content found in DOCX")
	}
	return content, nil
}

func (t *ReadFileTool) readXLSX(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to open XLSX: %w", err)
	}
	defer f.Close()

	var builder strings.Builder
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		builder.WriteString(fmt.Sprintf("=== Sheet: %s ===\n", sheet))

		rows, err := f.GetRows(sheet)
		if err != nil {
			builder.WriteString(fmt.Sprintf("Error reading sheet: %v\n", err))
			continue
		}

		if len(rows) == 0 {
			builder.WriteString("(empty sheet)\n\n")
			continue
		}

		colWidths := t.calculateColumnWidths(rows)

		for _, row := range rows {
			for colIdx, cell := range row {
				if colIdx > 0 {
					builder.WriteString(" | ")
				}
				width := 10
				if colIdx < len(colWidths) {
					width = colWidths[colIdx]
				}
				if len(cell) > width {
					cell = cell[:width]
				}
				builder.WriteString(cell)
				if padding := width - len(cell); padding > 0 {
					builder.WriteString(strings.Repeat(" ", padding))
				}
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("no content found in XLSX")
	}
	return result, nil
}

func (t *ReadFileTool) calculateColumnWidths(rows [][]string) []int {
	if len(rows) == 0 {
		return nil
	}

	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	widths := make([]int, maxCols)
	for i := range widths {
		widths[i] = 10
	}

	for _, row := range rows {
		for colIdx, cell := range row {
			cellLen := len(cell)
			if cellLen > widths[colIdx] && cellLen <= 50 {
				widths[colIdx] = cellLen
			} else if cellLen > 50 {
				widths[colIdx] = 50
			}
		}
	}

	return widths
}
