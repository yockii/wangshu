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

	// Initialize task manager
	agent.taskManager = task.NewTaskManager(agent.agentName, workspaceDir, model, provider)

	// Initialize cron manager with executor
	agent.cronManager = cron.NewCronManager(agent.agentName, workspaceDir, model, provider)

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

func (a *Agent) RunWithChannel(ctx context.Context, msg bus.InboundMessage) (string, error) {
	sess := a.sessions.GetOrCreate(a.workspaceDir, msg.Metadata.Channel, msg.Metadata.ChatID, msg.Metadata.SenderID)
	sess.AddMessage(constant.RoleUser, msg.Content)

	if msg.Type == bus.MessageTypeFile || msg.Type == bus.MessageTypeImage {
		return "", nil
	}

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

func (a *Agent) SubscribeInbound(ctx context.Context, msg bus.InboundMessage) {
	response, err := a.RunWithChannel(ctx, msg)
	if err != nil {
		slog.Error("Failed to run with channel", "error", err)
		response = fmt.Sprintf("Agent dealing failed: %+v", err)
	}

	outboundMsg := bus.NewOutboundMessage(msg.Metadata.ChatID, response)
	outboundMsg.Metadata.Channel = msg.Metadata.Channel
	bus.Default().PublishOutbound(outboundMsg)
}

func (a *Agent) RestartMessage(ctx context.Context, msg bus.InboundMessage) error {
	sess := a.sessions.GetOrCreate(a.workspaceDir, msg.Metadata.Channel, msg.Metadata.ChatID, msg.Metadata.SenderID)
	lastMsgs := sess.GetLastN(1)
	if len(lastMsgs) > 0 && lastMsgs[0].Role == constant.RoleAssistant {
		// 找到toolcall
		lastMsg := lastMsgs[0]
		toolMsg := types.Message{
			Role:    constant.RoleTool,
			Content: fmt.Sprintf("✅ Application restarted successfully. Current version: %s", constant.Version),
		}
		for _, tc := range lastMsg.ToolCalls {
			if tc.Name == constant.ToolNameVersion {
				toolMsg.ToolCalls = []types.ToolCall{tc}
				break
			}
		}
		if len(toolMsg.ToolCalls) > 0 {
			sess.AddMessage(toolMsg.Role, toolMsg.Content, toolMsg.ToolCalls...)
			msgs, err := a.buildMessages(sess)
			if err != nil {
				return err
			}

			response, err := a.runLoop(ctx, sess, msgs)

			if err != nil {
				return fmt.Errorf("Agent loop failed: %w", err)
			}

			bus.Default().PublishOutbound(bus.Message{
				Type:     bus.MessageTypeText,
				Content:  response,
				Metadata: msg.Metadata,
			})
		}
	}
	return nil
}
