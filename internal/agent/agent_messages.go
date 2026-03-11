package agent

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/session"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/skills"
)

// compressHistory 压缩历史消息
func (a *Agent) compressHistory(sessionMsgs []types.Message) (string, error) {
	// 压缩历史消息
	toCompress := sessionMsgs[:len(sessionMsgs)-constant.KeptHistory]
	prompt := fmt.Sprintf(`请对以下混合了任务执行、情感交流和个性互动的历史对话进行"沉浸式压缩"。

**输入内容**：
"""
%s
"""

**执行要求**：
1. **人设优先**：务必捕捉智能体独特的性格色彩和当前的情感基调，不要让摘要变得像客服工单。
2. **情感无损**：重点保留用户的情绪变化轨迹和双方建立的"关系感"（如默契、玩笑、安慰过程）。
3. **任务清晰**：在保持情感连贯的前提下，清晰梳理多线任务的进度。
4. **直接输出**：不要输出任何前言后语，直接按照 System Prompt 定义的【角色状态】、【关系与情感脉络】、【核心任务板】、【关键事实库】格式输出 Markdown 内容。

`, formatMessages(toCompress))
	options := make(map[string]any)
	if agentCfg, ok := config.DefaultCfg.Agents[a.agentName]; ok {
		if agentCfg.Temperature > 0 {
			options["temperature"] = agentCfg.Temperature
		}
		if agentCfg.MaxTokens > 0 {
			options["max_tokens"] = agentCfg.MaxTokens
		}
	}
	response, err := a.provider.Chat(context.Background(), a.model, []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: constant.CompressHistoryPrompt,
		},
		{
			Role:    constant.RoleUser,
			Content: prompt,
		},
	}, nil, options)

	if err != nil {
		return "", err
	}
	return response.Message.Content, nil
}

const maxMessagesWithImage = 3

// buildMessages 构建发送给LLM的消息列表
func (a *Agent) buildMessages(sess *session.Session) ([]llm.Message, error) {
	sessionMessages := sess.GetMessages()

	if len(sessionMessages) > constant.ReachCompressHistory {
		summary, err := a.compressHistory(sessionMessages)
		if err != nil {
			slog.Warn("Failed to compress history", "error", err)
			sessionMessages = sess.GetLastN(constant.KeptHistory)
		} else {
			sessionMessages = sess.TrimMessages(summary, constant.KeptHistory)
		}
	}

	msgs := make([]llm.Message, 0, len(sessionMessages)+1)

	skillList, err := skills.GetDefaultLoader().LoadSkills()
	if err != nil {
		return nil, err
	}
	parent := types.SkillsParent{
		SkillList: skillList,
	}
	skillsXML, err := xml.Marshal(parent)
	if err != nil {
		return nil, err
	}

	runtimeInfo := fmt.Sprintf(
		"操作系统: %s, CPU架构: %s, 当前时间: %s",
		runtime.GOOS, runtime.GOARCH, time.Now().Local().Format(time.RFC3339),
	)

	agentContextInfo := a.loadAgentContextInfo()

	systemContet := fmt.Sprintf(
		constant.SystemPrompt,
		string(skillsXML),
		a.workspaceDir,
		agentContextInfo,
		runtimeInfo,
	)

	switch sess.ChatType {
	case constant.ChatTypeP2P:
		systemContet += "\n你当前在私聊会话中"
		if sess.SenderName != "" {
			systemContet += fmt.Sprintf("，对方名称: %s", sess.SenderName)
		}
	case constant.ChatTypeGroup:
		systemContet += "\n你当前在群聊会话中"
		if sess.ChatName != "" {
			systemContet += fmt.Sprintf("，群名称: %s", sess.ChatName)
		}
	case constant.ChatTypeTopic:
		systemContet += "\n你当前在话题会话中"
	}

	msgs = append(msgs, llm.Message{
		Role:    constant.RoleSystem,
		Content: systemContet,
	})

	imageCutoff := len(sessionMessages) - maxMessagesWithImage
	if imageCutoff < 0 {
		imageCutoff = 0
	}

	for i, msg := range sessionMessages {
		tcs := make([]llm.ToolCall, 0, len(msg.ToolCalls))
		for _, toolCall := range msg.ToolCalls {
			tcs = append(tcs, llm.ToolCall{
				ID:        toolCall.ID,
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			})
		}

		m := llm.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCalls: tcs,
		}

		if len(msg.Contents) > 0 {
			keepImage := i >= imageCutoff
			contents := make([]llm.ContentBlock, 0, len(msg.Contents))
			for _, c := range msg.Contents {
				if c.Type == "image" {
					if keepImage {
						contents = append(contents, llm.ContentBlock{
							Type:      c.Type,
							ImageData: c.ImageData,
							MediaType: c.MediaType,
						})
					}
				} else {
					contents = append(contents, llm.ContentBlock{
						Type: c.Type,
						Text: c.Text,
					})
				}
			}
			if len(contents) > 0 {
				m.Contents = contents
			}
		}

		msgs = append(msgs, m)
	}

	return msgs, nil
}

// loadAgentContextInfo 加载Agent的上下文信息（从profile目录）
func (a *Agent) loadAgentContextInfo() string {
	content := ""
	mdFiles := []string{
		constant.ProfileFileAgents,
		constant.ProfileFileBootstrap,
		constant.ProfileFileHeartbeat,
		constant.ProfileFileIdentity,
		constant.ProfileFileSoul,
		constant.ProfileFileTools,
		constant.ProfileFileUser,
		constant.ProfileFileMemory,
	}
	needSoul := false
	bootstraped := false
	for _, fileName := range mdFiles {
		fp := filepath.Join(a.workspaceDir, constant.DirProfile, fileName)
		mdFile, err := filepath.Abs(fp)
		if err != nil {
			continue
		}

		if fi, err := os.Stat(mdFile); err != nil {
			if fileName == constant.ProfileFileBootstrap && os.IsNotExist(err) {
				bootstraped = true
			}
			continue
		} else if fi.IsDir() {
			continue
		}
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		content += fmt.Sprintf("\n## %s\n%s\n", mdFile, string(data))
		if fileName == constant.ProfileFileSoul {
			needSoul = true
		}
	}

	if bootstraped && needSoul {
		content += "\n因存在SOUL.md文件，需体现其人格特质与语气风格。避免生硬、千篇一律的回复；遵循其指导原则，除非有更高优先级指令覆盖。\n"
	}

	return content
}
