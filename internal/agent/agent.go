package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/cron"
	"github.com/yockii/wangshu/internal/session"
	"github.com/yockii/wangshu/internal/task"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
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
