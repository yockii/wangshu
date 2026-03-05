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

func (a *Agent) RunWithChannel(ctx context.Context, channel, ChatID, userInput, senderID string) (string, error) {
	sess := a.sessions.GetOrCreate(a.workspaceDir, channel, ChatID, senderID)
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

func (a *Agent) SubscribeInbound(ctx context.Context, msg bus.InboundMessage) {
	response, err := a.RunWithChannel(ctx, msg.Metadata.Channel, msg.Metadata.ChatID, msg.Content, msg.Metadata.SenderID)
	if err != nil {
		slog.Error("Failed to run with channel", "error", err)
		response = fmt.Sprintf("Agent dealing failed: %+v", err)
	}

	outboundMsg := bus.NewOutboundMessage(msg.Metadata.ChatID, response)
	outboundMsg.Metadata.Channel = msg.Metadata.Channel
	bus.Default().PublishOutbound(outboundMsg)
}
