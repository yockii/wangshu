package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type EditFileTool struct {
	basic.SimpleTool
}

func NewEditFileTool() *EditFileTool {
	tool := new(EditFileTool)
	tool.Name_ = "edit_file"
	tool.Desc_ = "Replace a unique block of text in a file. CRITICAL: 'old_str' must match exactly (including whitespace) and appear ONLY ONCE in the file. Include 2-3 lines of surrounding context to ensure uniqueness."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Absolute path to the file."},
			"old_str":   map[string]any{"type": "string", "description": "The exact text to replace. MUST be unique in the file. Include surrounding lines if necessary."},
			"new_str":   map[string]any{"type": "string", "description": "The replacement text."},
		},
		"required": []string{"file_path", "old_str", "new_str"},
	}
	return tool
}
func (t *EditFileTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	fPath := params["file_path"]
	oldStr := params["old_str"]
	newStr := params["new_str"]

	if fPath == "" || oldStr == "" {
		return "", fmt.Errorf("missing required parameters")
	}

	// 路径处理
	if strings.HasPrefix(fPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		fPath = filepath.Join(home, fPath[2:])
	}

	// 读取文件
	content, err := os.ReadFile(fPath)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	contentStr := string(content)

	// 【核心逻辑】检查出现次数
	count := strings.Count(contentStr, oldStr)

	if count == 0 {
		// 没找到，返回前几行帮它调试
		lines := strings.Split(contentStr, "\n")
		preview := strings.Join(lines[:min(len(lines), 30)], "\n")
		return "", fmt.Errorf("not found. Ensure exact match (indentation/spaces). File preview:\n%s", preview)
	}

	if count > 1 {
		return "", fmt.Errorf("ambiguous: found %d occurrences. Please add more surrounding context to 'old_str' to make it unique.", count)
	}

	// 唯一，执行替换
	newContent := strings.Replace(contentStr, oldStr, newStr, 1)

	if err := os.WriteFile(fPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	return "Success", nil
}
