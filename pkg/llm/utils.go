package llm

import (
	"strings"
)

// CleanJSONMarkdown 清理 JSON 内容中的 markdown 代码块标记
// 处理以下情况：
// - ```json ... ```
// - ``` ... ```
// - 前后的空白字符
// - 代码块前有额外文本
func CleanJSONMarkdown(content string) string {
	// 去除首尾空白
	content = strings.TrimSpace(content)

	// 查找第一个代码块开始标记
	codeBlockStart := strings.Index(content, "```")
	if codeBlockStart == -1 {
		// 没有代码块标记，直接返回
		return content
	}

	// 从代码块开始的位置处理
	content = content[codeBlockStart:]

	// 找到第一个换行符，这通常是标记行的结束
	firstNewline := strings.Index(content, "\n")
	if firstNewline == -1 {
		// 如果没有换行符，可能是单行格式，直接返回
		return content
	}

	// 跳过第一行标记
	content = content[firstNewline+1:]

	// 去除尾部的代码块标记
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimRight(content, "\n\r")
	} else {
		// 检查是否有独立的 ``` 行
		lastBackticks := strings.LastIndex(content, "\n```")
		if lastBackticks != -1 && lastBackticks == strings.LastIndex(content, "```")-1 {
			content = content[:lastBackticks]
			content = strings.TrimRight(content, "\n\r")
		}
	}

	// 再次去除首尾空白
	return strings.TrimSpace(content)
}

// ExtractJSONFromContent 从响应内容中提取并清理 JSON
// 这是一个便捷方法，结合了 CleanJSONMarkdown 的功能
func ExtractJSONFromContent(content string) string {
	return CleanJSONMarkdown(content)
}
