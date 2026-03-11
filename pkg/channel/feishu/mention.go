package feishu

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// convertMentionsToAtTags 将文本中的@用户名转换为<at user_id="open_id"></at>格式
// 支持格式: @用户名 (后面通常跟着空格或标点)
func (c *FeishuChannel) convertMentionsToAtTags(chatID, text string) string {
	// 正则匹配 @用户名 格式（包括后面的空格）
	// 另外再单独处理行尾的情况
	re := regexp.MustCompile(`@[\p{Han}a-zA-Z0-9_]+ `)

	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	// 获取群聊用户映射

	// 创建反向映射：用户名 -> open_id
	nameToOpenID := make(map[string]string)
	c.cachedUsers.Range(func(key, value interface{}) bool {
		nameToOpenID[value.(string)] = key.(string)
		return true
	})

	result := text
	// 从后向前替换，避免索引变化问题
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]

		start, end := match[0], match[1]
		mention := text[start:end]

		// 提取用户名（去掉@和尾部空格）
		userName := strings.TrimPrefix(mention, "@")
		userName = strings.TrimRight(userName, " ")

		// 查找对应的 open_id
		if openID, found := nameToOpenID[userName]; found {
			// 计算用户名的实际结束位置（@用户名 部分，不包括后面的空格）
			userNameEnd := start + 1 + len(userName)
			atTag := fmt.Sprintf("<at user_id=\"%s\"></at>", openID)
			result = result[:start] + atTag + result[userNameEnd:]
			slog.Debug("Feishu Channel converted mention", "userName", userName, "openID", openID, "mention", mention, "atTag", atTag)
		} else {
			slog.Debug("Feishu Channel mention not found in user list", "userName", userName, "mention", mention)
		}
	}

	return result
}
