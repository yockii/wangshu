package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type ReadFileTool struct {
	basic.SimpleTool
}

func NewReadFileTool() *ReadFileTool {
	tool := new(ReadFileTool)
	tool.Name_ = "read_file"
	tool.Desc_ = "Read the content of a file. Returns file content as string."
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

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}
