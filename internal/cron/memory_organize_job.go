package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/netresearch/go-cron"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

const memoryOrganizeJobName = "memory-organize"

type MemoryOrganizeResult struct {
	DailyMemory     string `json:"dailyMemory"`
	ImportantMemory string `json:"importantMemory"`
}

func (mgr *CronManager) addMemoryOrganizeJob() {
	t := []int{0, 0}
	if mgr.memoryOrganizeTime != "" {
		ts := strings.Split(mgr.memoryOrganizeTime, ":")
		if len(ts) >= 2 {
			t[0], _ = strconv.Atoi(ts[0])
			t[1], _ = strconv.Atoi(ts[1])
		}
	}
	if t[0] < 0 || t[0] > 23 {
		t[0] = 0
	}
	if t[1] < 0 || t[1] > 59 {
		t[1] = 0
	}
	cronReg := fmt.Sprintf("%d %d * * *", t[1], t[0])
	mgr.c.AddFunc(cronReg, mgr.memoryOrganizeJob, cron.WithName(memoryOrganizeJobName))
}

func (mgr *CronManager) memoryOrganizeJob() {
	// 检查workspace下的sessions目录中的所有渠道
	sessionsDir := filepath.Join(mgr.workspace, constant.DirSessions)
	if fi, err := os.Stat(sessionsDir); err != nil {
		if os.IsNotExist(err) {
			return
		}
	} else if !fi.IsDir() {
		return
	}
	// 遍历sessionsDir下的所有文件
	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		slog.Error("Failed to read sessions directory", "error", err)
		return
	}
	var content strings.Builder
	for _, fi := range files {
		if !fi.IsDir() {
			continue
		}
		historyMessages, err := mgr.loadHistoryForMemoryOrganize(fi.Name())
		if err != nil {
			slog.Error("Failed to load history for memory organize", "error", err)
			continue
		}
		content.WriteString("==========")
		content.WriteString(fi.Name())
		content.WriteString("==========\n")
		content.WriteString(historyMessages)
		content.WriteString("\n")
	}

	// 内容整理完毕，准备messages发送给大模型

	profileContent := mgr.loadProfileContent()

	msgs := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: fmt.Sprintf(constant.MemoryOrganizePrompt, profileContent),
		},
		{
			Role:    constant.RoleUser,
			Content: content.String(),
		},
	}
	schema := &llm.JSONSchema{
		Name:        "MemoryOrganizeResult",
		Description: "记忆整理",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dailyMemory": map[string]any{
					"type":        "string",
					"description": "每日记忆，根据当日会话历史记录，整理出当日记忆",
				},
				"importantMemory": map[string]any{
					"type":        "string",
					"description": "重要记忆，根据当日会话历史记录及之前的重要记忆(MEMORY.md)，重新输出完整的重要记忆",
				},
			},
			"required": []string{"dailyMemory", "importantMemory"},
		},
		Strict: true,
	}

	options := make(map[string]any)
	if agentCfg, ok := config.DefaultCfg.Agents[mgr.agentName]; ok {
		// if agentCfg.Temperature > 0 {
		// 	options["temperature"] = agentCfg.Temperature
		// }
		if agentCfg.MaxTokens > 0 {
			options["max_tokens"] = agentCfg.MaxTokens
		}
	}
	// 需要精确响应json
	options["temperature"] = 0.1

	resp, err := mgr.provider.ChatWithJSONSchema(
		context.Background(),
		mgr.model,
		msgs,
		schema,
		options,
	)
	if err != nil {
		slog.Error("Memory organize failed", "error", err)
		return
	}

	var result MemoryOrganizeResult
	respContent := llm.ExtractJSONFromContent(resp.Message.Content)
	if err := json.Unmarshal([]byte(respContent), &result); err != nil {
		slog.Error("Failed to parse memory organize result", "error", err)
		return
	}

	// 写入到 MEMORY.md 文件中
	memoryFile := filepath.Join(mgr.workspace, constant.DirProfile, constant.ProfileFileMemory)
	if err := os.WriteFile(memoryFile, []byte(result.ImportantMemory), 0644); err != nil {
		slog.Error("Failed to write memory file", "error", err)
		return
	}
	// 写入到memory/yyyy-mm-dd.md中
	memoryDateFile := filepath.Join(mgr.workspace, constant.DirProfile, constant.DirMemory, fmt.Sprintf("%s.md", time.Now().AddDate(0, 0, -1).Format("2006-01-02")))
	os.MkdirAll(filepath.Dir(memoryDateFile), 0755)
	if err := os.WriteFile(memoryDateFile, []byte(result.DailyMemory), 0644); err != nil {
		slog.Error("Failed to write memory date file", "error", err)
		return
	}
}

func (mgr *CronManager) loadHistoryForMemoryOrganize(channelID string) (string, error) {
	var result strings.Builder
	historyDir := filepath.Join(mgr.workspace, constant.DirSessions, channelID)
	if fi, err := os.Stat(historyDir); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	} else if !fi.IsDir() {
		return "", nil
	}

	// 读取chats.json和members.json放入对应 map中
	chatsMap := make(map[string]string)
	membersMap := make(map[string]string)
	chatsFile := filepath.Join(historyDir, constant.FileCachedChats)
	if data, err := os.ReadFile(chatsFile); err == nil {
		if err = json.Unmarshal(data, &chatsMap); err != nil {
			return "", err
		}
	}
	membersFile := filepath.Join(historyDir, constant.FileCachedMembers)
	if data, err := os.ReadFile(membersFile); err == nil {
		if err = json.Unmarshal(data, &membersMap); err != nil {
			return "", err
		}
	}

	// 遍历所有的jsonl文件
	files, err := os.ReadDir(historyDir)
	if err != nil {
		return "", err
	}
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		// 必须是.jsonl结尾
		if !strings.HasSuffix(fi.Name(), constant.ExtJSONL) {
			continue
		}

		// 读取jsonl文件
		filePath := filepath.Join(historyDir, fi.Name())
		msg, err := mgr.readMessageFromJSONLForMemoryOrganize(filePath)
		if err != nil {
			slog.Error("Some Error occurred when resolve history msgs", "filePath", filePath, "error", err)
		}
		if msg != "" {
			chatName := chatsMap[fi.Name()]
			if chatName == "" {
				chatName = fi.Name()
			}
			result.WriteString("\n-------------")
			result.WriteString(chatName)
			result.WriteString("-------------\n")
			result.WriteString(msg)
		}
	}

	return result.String(), nil
}

func (mgr *CronManager) readMessageFromJSONLForMemoryOrganize(fp string) (string, error) {
	var result strings.Builder
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	yesterdayZero := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	todayZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	file, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	for {
		var msg types.Message
		if err := decoder.Decode(&msg); err != nil {
			if err.Error() == "EOF" {
				break
			}
			slog.Warn("message decode failed", "error", err)
			continue
		}

		if msg.Timestamp.Before(yesterdayZero) || msg.Timestamp.After(todayZero) {
			continue
		}

		result.WriteString(msg.Role)
		result.WriteString("(")
		result.WriteString(msg.Timestamp.Format(time.DateTime))
		result.WriteString("): ")
		if msg.Role == constant.RoleTool {
			data, err := json.Marshal(msg.ToolCalls)
			if err != nil {
				result.WriteString(msg.Content)
			} else {
				result.WriteString(string(data))
			}
		} else {
			result.WriteString(msg.Content)
		}
		result.WriteString("\n")
	}
	return result.String(), nil
}

// loadAgentContextInfo 加载Agent的上下文信息（从profile目录）
func (mgr *CronManager) loadProfileContent() string {
	content := ""
	mdFiles := []string{
		// constant.ProfileFileAgents,
		// constant.ProfileFileBootstrap,
		// constant.ProfileFileHeartbeat,
		constant.ProfileFileIdentity,
		constant.ProfileFileSoul,
		// constant.ProfileFileTools,
		constant.ProfileFileUser,
		constant.ProfileFileMemory,
	}
	for _, fileName := range mdFiles {
		fp := filepath.Join(mgr.workspace, constant.DirProfile, fileName)
		mdFile, err := filepath.Abs(fp)
		if err != nil {
			continue
		}

		if fi, err := os.Stat(mdFile); err != nil {
			continue
		} else if fi.IsDir() {
			continue
		}
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		content += fmt.Sprintf("\n## %s\n%s\n", mdFile, string(data))
	}

	return content
}
