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
	"time"

	"github.com/yockii/yoclaw/internal/cron"
	"github.com/yockii/yoclaw/internal/notification"
	"github.com/yockii/yoclaw/internal/session"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/skills"
	"github.com/yockii/yoclaw/pkg/tools"
)

type Agent struct {
	provider     llm.Provider
	model        string
	tools        *tools.Registry
	sessions     *session.Manager
	maxIter      int
	workspaceDir string
	skillLoader  *skills.Loader
	cronManager  *cron.Manager
	agentName    string
}

func NewAgent(provider llm.Provider, model string, tools *tools.Registry, sessionTTL time.Duration, maxIter int, workspaceDir string, skillLoader *skills.Loader) *Agent {
	agent := &Agent{
		provider:     provider,
		model:        model,
		tools:        tools,
		sessions:     session.NewManager(sessionTTL),
		maxIter:      maxIter,
		workspaceDir: workspaceDir,
		skillLoader:  skillLoader,
	}

	// Initialize cron manager (without task creator - will be set via event handler)
	agent.cronManager = cron.NewManager(workspaceDir, nil, nil)

	return agent
}

// SetName sets the agent name
func (a *Agent) SetName(name string) {
	a.agentName = name
	// Update cron manager with agent name
	if a.cronManager != nil {
		a.cronManager.SetAgentName(name)
	}
}

// GetName returns the agent name
func (a *Agent) GetName() string {
	return a.agentName
}

// GetCronManager returns the cron manager
func (a *Agent) GetCronManager() *cron.Manager {
	return a.cronManager
}

func (a *Agent) GetWorkspace() string {
	return a.workspaceDir
}

func (a *Agent) Stop() {
	// Cron manager no longer needs to be stopped (it's now a lightweight recorder)
}

func (a *Agent) RunWithChannel(ctx context.Context, sessionID, channel, ChatID, userInput, senderID string) (string, error) {
	// Record user for proactive notifications
	if senderID != "" {
		notification.GetManager().RecordUser(channel, ChatID, senderID, a.GetWorkspace())
	}

	sess := a.sessions.GetOrCreate(a.workspaceDir, sessionID, channel, ChatID, senderID)
	sess.AddMessage("user", userInput)

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

	for i := 0; i < a.maxIter; i++ {

		resp, err := a.provider.Chat(ctx, a.model, msgs, a.tools.GetProviderDefs(), nil)
		if err != nil {
			return "", fmt.Errorf("LLM call failed (iteration %d): %w", i+1, err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			// 不需要调用工具，则开始输出
			finalContent = resp.Message.Content
			break
		}

		assistantMsg := session.Message{
			Role:    "assistant",
			Content: resp.Message.Content,
		}

		for _, tc := range resp.Message.ToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, session.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
		// 加到session中
		sess.AddMessage(assistantMsg.Role, assistantMsg.Content, assistantMsg.ToolCalls...)

		// 加到发给大模型的对话列表中
		msgs = append(msgs, llm.Message{
			Role:      "assistant",
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})

		// 执行所有的工具调用
		for _, tc := range resp.Message.ToolCalls {
			var args any
			if tc.Arguments != "" {
				var parsedArgs map[string]any
				if err := json.Unmarshal([]byte(tc.Arguments), &parsedArgs); err == nil {
					args = parsedArgs
				}
			}

			EmitToolStart(sess.ID, tc.Name, tc.ID, args)

			toolResult, err := a.executeToolCall(ctx, tc, sess.Channel, sess.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
				EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, true)
			} else {
				EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, false)
			}

			addToolResultMessage(sess, "tool", toolResult, tc.ID)

			msgs = append(msgs, llm.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
			})
		}

	}

	if finalContent != "" {
		sess.AddMessage("assistant", finalContent)
	}

	EmitLifecycle(sess.ID, "end", "")
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

	args[tools.ToolCallParamWorkspace] = a.workspaceDir

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

	result := a.tools.ExecuteWithContext(ctx, tc.Name, args, toolCtx, channel, chatID)
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
					sess.AddMessage(role, content, session.ToolCall{
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

func (a *Agent) buildMessages(sess *session.Session) ([]llm.Message, error) {
	sessionMessages := sess.GetMessages()

	msgs := make([]llm.Message, 0, len(sessionMessages)+1)

	// if len(sessionMessages) > 1 { // 之前已经加载过对应的tools和skills数据，不用单独加载
	// 	for _, msg := range sessionMessages {
	// 		tc := make([]llm.ToolCall, 0, len(msg.ToolCalls))
	// 		for _, toolCall := range msg.ToolCalls {
	// 			tc = append(tc, llm.ToolCall{
	// 				ID:        toolCall.ID,
	// 				Name:      toolCall.Name,
	// 				Arguments: toolCall.Arguments,
	// 			})
	// 		}
	// 		m := llm.Message{
	// 			Role:      msg.Role,
	// 			Content:   msg.Content,
	// 			ToolCalls: tc,
	// 		}
	// 		msgs = append(msgs, m)
	// 	}
	// } else {
	// 技能元数据加载
	skillList, err := a.skillLoader.LoadSkills()
	if err != nil {
		return nil, err
	}
	// 将skills转为xml字符串
	type SkillsParent struct {
		XMLName   xml.Name        `xml:"skills"`
		SkillList []*skills.Skill `xml:"skill"`
	}
	parent := SkillsParent{
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
		Role: "system",
		Content: fmt.Sprintf(
			SystemPrompt,
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
		"AGENTS.md",
		"BOOTSTRAP.md",
		"HEARTBEAT.md",
		"IDENTITY.md",
		"SOUL.md",
		"TOOLS.md",
		"USER.md",
		"MEMORY.md",
	}
	hasSoul := false
	for _, fileName := range mdFiles {
		fp := filepath.Join(a.workspaceDir, fileName)
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
		if fileName == "SOUL.md" {
			hasSoul = true
		}
	}

	if hasSoul {
		content += "\n因存在SOUL.md文件，需体现其人格特质与语气风格。避免生硬、千篇一律的回复；遵循其指导原则，除非有更高优先级指令覆盖。\n"
	}

	return content
}

// CallProvider calls the LLM provider directly (for task execution)
func (a *Agent) CallProvider(ctx context.Context, sessionID string, msgs []llm.Message) (*llm.ChatResponse, error) {
	return a.provider.Chat(ctx, a.model, msgs, a.tools.GetProviderDefs(), nil)
}

// GetTools returns the tool registry
func (a *Agent) GetTools() *tools.Registry {
	return a.tools
}

// GetLLMProvider returns the LLM provider and model
func (a *Agent) GetLLMProvider() (llm.Provider, string) {
	return a.provider, a.model
}

func (a *Agent) SubscribeInbound(ctx context.Context, msg bus.InboundMessage) {
	sessionID := msg.SessionKey
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
	}
	response, err := a.RunWithChannel(ctx, sessionID, msg.Channel, msg.ChatID, msg.Content, msg.SenderID)
	if err != nil {
		slog.Error("Failed to run with channel", "error", err)
		response = fmt.Sprintf("Agent dealing failed: %w", err)
	}
	bus.Default().PublishOutbound(bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: response,
	})
}
