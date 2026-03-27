package filesystem

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	actiontypes "github.com/yockii/wangshu/pkg/action/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type GrepTool struct {
	basic.SimpleTool
}

func NewGrepTool() *GrepTool {
	tool := new(GrepTool)
	tool.Name_ = constant.ToolNameGrepFile
	tool.Desc_ = "Search for a string or regex pattern in files. Returns matching lines with file names and line numbers. Faster and more powerful than reading files manually. Use this to find where a function is defined or used."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The string or regular expression to search for.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "(Optional) Root directory to start searching.",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "(Optional) Glob pattern to filter files (e.g., '*.go', '*.md', 'src/**/*.ts'). If omitted, searches all files.",
			},
		},
		"required": []string{"pattern"},
	}
	return tool
}

func (t *GrepTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	pattern := params["pattern"]
	searchPath := params["path"]
	includePattern := params["include"]

	if pattern == "" {
		return types.NewToolResult().WithError(fmt.Errorf("pattern is required"))
	}

	// 默认搜索当前目录
	if searchPath == "" {
		searchPath = "."
	}

	// 1. 编译正则表达式
	// 如果用户只是搜普通字符串，regexp.Compile 也能处理，除非有特殊字符
	re, err := regexp.Compile(pattern)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("invalid regex pattern: %w", err))
	}

	matchCount := 0
	const maxMatches = 100 // 限制最大匹配行数，防止输出过长
	const maxFiles = 500   // 限制最大扫描文件数，防止耗时过长

	var results []struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}

	// 2. 遍历目录
	err = filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 忽略无法访问的文件/目录
		}

		// 跳过隐藏目录 (如 .git, node_modules) 以加快速度
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != searchPath {
			// 允许搜索根目录即使是 . 开头，但子目录跳过
			// 特殊处理：不要跳过 .github 等可能有用的，但通常跳过 .git 是最重要的
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == "dist" || d.Name() == "build" {
				return filepath.SkipDir
			}
		}

		if d.IsDir() {
			return nil
		}

		// 3. 文件过滤 (include 参数)
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, d.Name())
			if err != nil || !matched {
				return nil
			}
		}

		// 限制扫描文件数量
		if matchCount >= maxMatches {
			return filepath.SkipAll // 停止遍历
		}

		// 简单的文件计数限制，防止在大项目中卡死
		// 实际生产中可以用 atomic 计数，这里简化处理
		// 注意：WalkDir 是单线程的，这里只是粗略限制

		// 4. 打开并读取文件
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 5. 匹配
			if re.MatchString(line) {
				// 格式化输出：文件路径:行号:内容
				// Windows 路径分隔符统一为 / 方便 LLM 阅读，或者保持原样
				// resultLine := fmt.Sprintf("%s:%d:%s", path, lineNum, line)
				result := struct {
					Path string `json:"path"`
					Line int    `json:"line"`
					Text string `json:"text"`
				}{
					Path: path,
					Line: lineNum,
					Text: line,
				}
				results = append(results, result)
				matchCount++

				if matchCount >= maxMatches {
					return filepath.SkipAll
				}
			}
		}

		// 忽略扫描错误（如二进制文件）
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return types.NewToolResult().WithError(fmt.Errorf("search error: %w", err))
	}

	if len(results) == 0 {
		return types.NewToolResult().WithRaw("No matches found.")
	}

	var raw strings.Builder

	for _, result := range results {
		raw.WriteString(fmt.Sprintf("%s:%d:%s\n", result.Path, result.Line, result.Text))
	}
	if matchCount >= maxMatches {
		raw.WriteString("\n... (Results truncated. Refine your pattern or path to see more.)")
	}

	return types.NewToolResult().WithRaw(fmt.Sprintf("Found %d matches:\n%s", len(results), raw.String())).
		WithStructured(actiontypes.NewFsGrepData(pattern, results))
}
