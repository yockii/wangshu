package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type CopyFileTool struct {
	basic.SimpleTool
}

func NewCopyFileTool() *CopyFileTool {
	tool := new(CopyFileTool)
	tool.Name_ = "copy_file"
	tool.Desc_ = "Copy a file or directory. Returns success message. Supports recursive directory copying. By default, it will not overwrite existing files unless overwrite parameter is set to true."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source_path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the source file or directory",
			},
			"target_path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the target file or directory",
			},
			"overwrite": map[string]any{
				"type":        "boolean",
				"description": "If true, overwrite existing target files. Default is false.",
			},
		},
		"required": []string{"source_path", "target_path"},
	}
	return tool
}

func (t *CopyFileTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	sourcePath := params["source_path"]
	targetPath := params["target_path"]
	if sourcePath == "" || targetPath == "" {
		return types.NewToolResult().WithError(fmt.Errorf("source_path and target_path are required"))
	}

	overwrite := params["overwrite"] == "true" || params["overwrite"] == "1"

	sourcePath = t.expandPath(sourcePath)
	targetPath = t.expandPath(targetPath)

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to stat source: %w", err))
	}

	if sourceInfo.IsDir() {
		return t.copyDirectory(sourcePath, targetPath, overwrite)
	}

	return t.copyFile(sourcePath, targetPath, overwrite)
}

func (t *CopyFileTool) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	return path
}

func (t *CopyFileTool) copyFile(sourcePath, targetPath string, overwrite bool) *types.ToolResult {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to open source file: %w", err))
	}
	defer sourceFile.Close()

	targetInfo, err := os.Stat(targetPath)
	if err == nil && targetInfo.IsDir() {
		targetPath = filepath.Join(targetPath, filepath.Base(sourcePath))
		targetInfo, err = os.Stat(targetPath)
	}

	absSource, _ := filepath.Abs(sourcePath)
	absTarget, _ := filepath.Abs(targetPath)
	if absSource == absTarget {
		sourceInfo, _ := os.Stat(sourcePath)
		return types.NewToolResult().WithRaw(fmt.Sprintf("Successfully copied %s to %s (%d bytes)", sourcePath, targetPath, sourceInfo.Size()))
	}

	if err == nil && !overwrite {
		return types.NewToolResult().WithError(fmt.Errorf("target file %s already exists", targetPath))
	}

	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create target file: %w", err))
	}
	defer targetFile.Close()

	copied, err := io.Copy(targetFile, sourceFile)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to copy file: %w", err))
	}

	sourceInfo, _ := os.Stat(sourcePath)
	os.Chmod(targetPath, sourceInfo.Mode())

	return types.NewToolResult().WithRaw(fmt.Sprintf("Successfully copied %s to %s (%d bytes)", sourcePath, targetPath, copied))
}

func (t *CopyFileTool) copyDirectory(sourcePath, targetPath string, overwrite bool) *types.ToolResult {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to stat source directory: %w", err))
	}

	targetInfo, err := os.Stat(targetPath)
	if err == nil {
		if !targetInfo.IsDir() {
			return types.NewToolResult().WithError(fmt.Errorf("target exists but is not a directory"))
		}
		if !overwrite {
			return types.NewToolResult().WithError(fmt.Errorf("target directory %s already exists", targetPath))
		}
	}

	if err := os.MkdirAll(targetPath, sourceInfo.Mode()); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create target directory: %w", err))
	}

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read source directory: %w", err))
	}

	var copiedFiles int
	for _, entry := range entries {
		sourceEntry := filepath.Join(sourcePath, entry.Name())
		targetEntry := filepath.Join(targetPath, entry.Name())

		if entry.IsDir() {
			result := t.copyDirectory(sourceEntry, targetEntry, overwrite)
			if result.Err != nil {
				return result
			}
		} else {
			result := t.copyFile(sourceEntry, targetEntry, overwrite)
			if result.Err != nil {
				return result
			}
			copiedFiles++
		}
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("Successfully copied directory %s to %s (%d files)", sourcePath, targetPath, copiedFiles))
}
