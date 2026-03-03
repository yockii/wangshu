package agent

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yockii/yoclaw/internal/config"
	"github.com/yockii/yoclaw/internal/cron"
	"github.com/yockii/yoclaw/internal/session"
	"github.com/yockii/yoclaw/internal/task"
	"github.com/yockii/yoclaw/internal/types"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/skills"
	"github.com/yockii/yoclaw/pkg/tools"
)

type Agent struct {
	provider     llm.Provider
	model        string
	sessions     *session.Manager
	maxIter      int
	workspaceDir string
	cronManager  *cron.CronManager
	taskManager  *task.TaskManager
	agentName    string
}

func NewAgent(provider llm.Provider, name, model string, sessionTTL time.Duration, maxIter int, workspaceDir string) (*Agent, error) {
	agent := &Agent{
		agentName:    name,
		provider:     provider,
		model:        model,
		sessions:     session.NewManager(sessionTTL),
		maxIter:      maxIter,
		workspaceDir: workspaceDir,
	}

	err := config.EnsureWorkspace(workspaceDir)
	if err != nil {
		slog.Error("Failed to ensure workspace", "agent", agent.agentName, "error", err)
		return nil, err
	}

	// Initialize cron manager (without task creator - will be set via event handler)
	agent.cronManager = cron.NewManager(workspaceDir, agent.executionJob)

	agent.taskManager = task.NewTaskManager(workspaceDir, model, provider)

	return agent, nil
}

// GetName returns the agent name
func (a *Agent) GetName() string {
	return a.agentName
}

// GetCronManager returns the cron manager
func (a *Agent) GetCronManager() *cron.CronManager {
	return a.cronManager
}

func (a *Agent) GetWorkspace() string {
	return a.workspaceDir
}

func (a *Agent) Stop() {
	a.taskManager.Stop()
	a.cronManager.Stop()
}

func (a *Agent) RunWithChannel(ctx context.Context, sessionID, channel, ChatID, userInput, senderID string) (string, error) {
	sess := a.sessions.GetOrCreate(a.workspaceDir, sessionID, channel, ChatID, senderID)
	sess.AddMessage(constant.RoleUser, userInput)

	msgs, err := a.buildMessages(sess)
	if err != nil {
		return "", err
	}

	response, err := a.runLoop(ctx, sess, msgs)

	if err != nil {
		return "", fmt.Errorf("Agent loop failed: %w", err)
	}
	return response, nil

	// resp, err := a.provider.Chat(ctx, sessionID, msgs, tools, nil)
}

func (a *Agent) runLoop(ctx context.Context, sess *session.Session, msgs []llm.Message) (string, error) {
	var finalContent string

	availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
	for i := 0; i < a.maxIter; i++ {

		resp, err := a.provider.Chat(ctx, a.model, msgs, availableTools, nil)
		if err != nil {
			return "", fmt.Errorf("LLM call failed (iteration %d): %w", i+1, err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			// 不需要调用工具，则开始输出
			finalContent = resp.Message.Content
			break
		}

		assistantMsg := types.Message{
			Role:    constant.RoleAssistant,
			Content: resp.Message.Content,
		}

		for _, tc := range resp.Message.ToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, types.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
		// 加到session中
		sess.AddMessage(assistantMsg.Role, assistantMsg.Content, assistantMsg.ToolCalls...)

		// 加到发给大模型的对话列表中
		msgs = append(msgs, llm.Message{
			Role:      constant.RoleAssistant,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})

		if resp.Message.Content != "" && len(resp.Message.ToolCalls) > 0 {
			// 有内容，且调用工具，则说明还需要循环，但内容可以先直接发送给用户
			bus.Default().PublishOutbound(bus.OutboundMessage{
				Channel: sess.Channel,
				ChatID:  sess.ChatID,
				Content: resp.Message.Content,
			})
		}

		// 执行所有的工具调用
		for _, tc := range resp.Message.ToolCalls {
			// EmitToolStart(sess.ID, tc.Name, tc.ID, args)

			toolResult, err := a.executeToolCall(ctx, tc, sess.Channel, sess.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
				// EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, true)
			} else {
				// EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, false)
			}

			addToolResultMessage(sess, constant.RoleTool, toolResult, tc.ID)

			msgs = append(msgs, llm.Message{
				Role:      constant.RoleTool,
				Content:   toolResult,
				ToolCalls: []llm.ToolCall{tc},
			})
		}

	}

	if finalContent != "" {
		sess.AddMessage(constant.RoleAssistant, finalContent)
	}

	// EmitLifecycle(sess.ID, "end", "")
	return finalContent, nil
}

func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolCall, channel, chatID string) (string, error) {
	var args map[string]any
	if tc.Arguments != "" {
		err := json.Unmarshal([]byte(tc.Arguments), &args)
		if err != nil {
			return "", fmt.Errorf("Failed to parse tool arguments: %w", err)
		}
	}

	if args == nil {
		args = make(map[string]any)
	}

	args[constant.ToolCallParamWorkspace] = a.workspaceDir
	args[constant.ToolCallParamChannel] = channel
	args[constant.ToolCallParamChatID] = chatID

	// Create ToolContext with agent information
	toolCtx := tools.NewToolContext(
		a.agentName,
		"", // agent owner - can be added later
		a.workspaceDir,
		"", // sessionID - can be passed separately if needed
		channel,
		chatID,
		a.provider,
		a.model,
	)

	result := tools.GetDefaultToolRegistry().ExecuteWithContext(ctx, tc.Name, args, toolCtx, channel, chatID)
	if result.IsError {
		return result.ForLLM, fmt.Errorf("Tool execution failed")
	}
	return result.ForLLM, nil
}

func addToolResultMessage(sess *session.Session, role, content, toolCallID string) {
	// If toolCallID is provided, find and update the tool call
	if toolCallID != "" {
		messages := sess.GetMessages()
		for i := len(messages) - 1; i >= 0; i-- {
			for _, tc := range messages[i].ToolCalls {
				if tc.ID == toolCallID {
					// Add the result to the tool call
					sess.AddMessage(role, content, types.ToolCall{
						ID:        toolCallID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
						Result:    content,
					})
					return
				}
			}
		}
	}

	// If no toolCallID or not found, add as regular message
	sess.AddMessage(role, content)
}

func formatMessages(messages []types.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role == constant.RoleUser || msg.Role == constant.RoleAssistant {
			sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}
	return sb.String()
}

func (a *Agent) compressHistory(sessionMsgs []types.Message) (string, error) {
	// 压缩历史消息
	toCompress := sessionMsgs[:len(sessionMsgs)-constant.KeptHistory]
	prompt := fmt.Sprintf(`请对以下混合了任务执行、情感交流和个性互动的历史对话进行“沉浸式压缩”。

**输入内容**：
"""
%s
"""

**执行要求**：
1. **人设优先**：务必捕捉智能体独特的性格色彩和当前的情感基调，不要让摘要变得像客服工单。
2. **情感无损**：重点保留用户的情绪变化轨迹和双方建立的“关系感”（如默契、玩笑、安慰过程）。
3. **任务清晰**：在保持情感连贯的前提下，清晰梳理多线任务的进度。
4. **直接输出**：不要输出任何前言后语，直接按照 System Prompt 定义的【角色状态】、【关系与情感脉络】、【核心任务板】、【关键事实库】格式输出 Markdown 内容。

`, formatMessages(toCompress))
	response, err := a.provider.Chat(context.Background(), a.model, []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: constant.CompressHistoryPrompt,
		},
		{
			Role:    constant.RoleUser,
			Content: prompt,
		},
	}, nil, nil)

	if err != nil {
		return "", err
	}
	return response.Message.Content, nil
}

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

	// 技能元数据加载
	skillList, err := skills.GetDefaultLoader().LoadSkills()
	if err != nil {
		return nil, err
	}
	// 将skills转为xml字符串
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

	msgs = append(msgs, llm.Message{
		Role: constant.RoleSystem,
		Content: fmt.Sprintf(
			constant.SystemPrompt,
			string(skillsXML),
			a.workspaceDir,
			agentContextInfo, // 各种个性化信息，包含路径
			runtimeInfo,
		),
	})
	for _, msg := range sessionMessages {
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
		msgs = append(msgs, m)
	}
	// }

	return msgs, nil
}

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

func (a *Agent) SubscribeInbound(ctx context.Context, msg bus.InboundMessage) {
	sessionID := msg.SessionKey
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
	}
	response, err := a.RunWithChannel(ctx, sessionID, msg.Channel, msg.ChatID, msg.Content, msg.SenderID)
	if err != nil {
		slog.Error("Failed to run with channel", "error", err)
		response = fmt.Sprintf("Agent dealing failed: %+v", err)
	}

	bus.Default().PublishOutbound(bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: response,
	})
}
