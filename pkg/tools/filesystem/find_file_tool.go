package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type FindFileTool struct {
	basic.SimpleTool
}

func NewFindFileTool() *FindFileTool {
	tool := new(FindFileTool)
	tool.Name_ = "find_files"
	tool.Desc_ = "Find files by name pattern (glob). Returns a list of absolute paths. Use this to locate specific files like '*.go', 'docker-compose.yml', or 'tests/**/*_test.go'."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match file names (e.g., '*.go', '**/main.go', 'src/**/*.ts'). Supports relative to current working directory.",
			},
		},
		"required": []string{"pattern"},
	}
	return tool
}
func (t *FindFileTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	pattern := params["pattern"]
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// 处理 ~ 路径
	if strings.HasPrefix(pattern, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		pattern = filepath.Join(home, pattern[2:])
	}

	// 执行 Glob 搜索
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return "No files found matching the pattern.", nil
	}

	// 格式化输出，每行一个路径
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d file(s):\n", len(matches)))
	for _, match := range matches {
		// 转为绝对路径，方便后续操作
		absPath, _ := filepath.Abs(match)
		result.WriteString("- " + absPath + "\n")
	}

	return result.String(), nil
}
