package filesystem

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yockii/wangshu/pkg/tools/basic"
)

type GrepTool struct {
	basic.SimpleTool
}

func NewGrepTool() *GrepTool {
	tool := new(GrepTool)
	tool.Name_ = "grep_search"
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

func (t *GrepTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	pattern := params["pattern"]
	searchPath := params["path"]
	includePattern := params["include"]

	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// 默认搜索当前目录
	if searchPath == "" {
		searchPath = "."
	}

	// 1. 编译正则表达式
	// 如果用户只是搜普通字符串，regexp.Compile 也能处理，除非有特殊字符
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var results []string
	matchCount := 0
	const maxMatches = 100 // 限制最大匹配行数，防止输出过长
	const maxFiles = 500   // 限制最大扫描文件数，防止耗时过长

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
				resultLine := fmt.Sprintf("%s:%d:%s", path, lineNum, line)
				results = append(results, resultLine)
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
		return "", fmt.Errorf("search error: %w", err)
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	output := strings.Join(results, "\n")
	if matchCount >= maxMatches {
		output += "\n... (Results truncated. Refine your pattern or path to see more.)"
	}

	return fmt.Sprintf("Found %d matches:\n%s", len(results), output), nil
}
