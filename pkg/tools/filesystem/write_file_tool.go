package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type WriteFileTool struct {
	basic.SimpleTool
}

func NewWriteFileTool() *WriteFileTool {
	tool := new(WriteFileTool)
	tool.Name_ = "write_file"
	tool.Desc_ = "Write content to a file. Creates file if it doesn't exist, overwrites if it does."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
	return tool
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	path := params["path"]
	content := params["content"]

	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
