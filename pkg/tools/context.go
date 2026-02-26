package tools

import (
	"context"
	"github.com/yockii/yoclaw/pkg/llm"
)

// ToolContext provides the execution context for a tool call
// This replaces the need for tools to call agent.GetAnyAgent()
type ToolContext struct {
	// AgentInfo provides information about the agent executing this tool
	AgentName  string
	AgentOwner string

	// LLM is the LLM provider for the current agent
	LLM llm.Provider

	// Model is the model name to use
	Model string

	// Workspace is the working directory
	Workspace string

	// SessionID identifies the current session
	SessionID string

	// Channel identifies the communication channel (e.g., "feishu")
	Channel string

	// ChatID identifies the chat within the channel
	ChatID string
}

// NewToolContext creates a new ToolContext
func NewToolContext(agentName, agentOwner, workspace, sessionID, channel, chatID string, llmProvider llm.Provider, model string) *ToolContext {
	return &ToolContext{
		AgentName:  agentName,
		AgentOwner: agentOwner,
		Workspace:  workspace,
		SessionID:  sessionID,
		Channel:    channel,
		ChatID:     chatID,
		LLM:        llmProvider,
		Model:      model,
	}
}

// CallLLM calls the LLM with the given messages
func (tc *ToolContext) CallLLM(ctx context.Context, sessionID string, messages []llm.Message) (*llm.ChatResponse, error) {
	if tc.LLM == nil {
		return nil, &ToolError{
			Code:    "ERR_NO_LLM_PROVIDER",
			Message: "LLM provider not available in tool context",
		}
	}

	return tc.LLM.Chat(ctx, tc.Model, messages, nil, nil)
}

// ToolError represents an error from tool execution
type ToolError struct {
	Code    string
	Message string
	Err     error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ToolError) Unwrap() error {
	return e.Err
}
