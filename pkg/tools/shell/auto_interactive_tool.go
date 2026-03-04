package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
	"github.com/yockii/wangshu/pkg/tools/basic"
)

// InteractionResponse represents LLM analysis result
type InteractionResponse struct {
	SessionID    string       `json:"session_id"`
	Suggestion   string       `json:"suggestion"` // Suggested input (text or ANSI sequence)
	Reasoning    string       `json:"reasoning"`  // Analysis reasoning
	Confidence   float64      `json:"confidence"` // Confidence score 0-1
	InputType    string       `json:"input_type"` // "text", "arrow", "enter"
	MenuDetected bool         `json:"menu_detected"`
	MenuOptions  []MenuOption `json:"menu_options,omitempty"`
	Context      string       `json:"context"` // Current output content
}

// AutoInteractiveSession manages an auto-interactive shell session
type AutoInteractiveSession struct {
	ID            string
	Cmd           *exec.Cmd
	Pty           *os.File
	Output        strings.Builder
	LastOutputPos int // Position for incremental output
	StartTime     time.Time
	State         SessionState
	mu            sync.RWMutex

	// Auto-interactive specific fields
	maxIterations   int
	iteration       int
	autoMode        bool // true = fully automatic, false = requires confirmation
	lastLLMResponse *InteractionResponse
	lastAnalysis    *MenuAnalysis

	// Detection configuration
	inputWaitThreshold time.Duration     // Output silence threshold (default 2 seconds)
	preferences        map[string]string // User preferences
	lastOutput         time.Time

	// Captured context from args
	workspace string
	channel   string
	chatID    string
	agentName string

	// LLM context for analysis
	llmProvider llm.Provider
	llmModel    string
}

// AutoInteractiveTool provides intelligent interactive shell capabilities
type AutoInteractiveTool struct {
	basic.SimpleTool
	sessions     map[string]*AutoInteractiveSession
	sessionsMu   sync.RWMutex
	menuAnalyzer *MenuAnalyzer
	keySeq       *TerminalKeySequence
	nextID       int
	// LLM provider and model (set from first ExecuteWithContext call)
	provider llm.Provider
	model    string
}

// NewAutoInteractiveTool creates a new auto-interactive tool
func NewAutoInteractiveTool() *AutoInteractiveTool {
	tool := new(AutoInteractiveTool)
	tool.Name_ = "auto_interactive"
	tool.Desc_ = "Execute interactive commands with intelligent automation. Automatically detects prompts, analyzes output with LLM, and suggests responses. Supports menu navigation with arrow keys. The tool will return confirmation requests when LLM confidence is low, allowing you to approve or modify suggestions before execution."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action: start, confirm, continue, end",
				"enum":        []string{"start", "confirm", "continue", "end"},
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Command to execute (required for start action). Example: 'npm create vite@latest my-app'",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID (required for confirm, continue, end actions). Obtained from start action.",
			},
			"max_iterations": map[string]any{
				"type":        "integer",
				"description": "Maximum interaction iterations (default: 10)",
			},
			"auto_mode": map[string]any{
				"type":        "boolean",
				"description": "If true, automatically execute LLM suggestions without confirmation (default: false)",
			},
			"preferences": map[string]any{
				"type":        "object",
				"description": "User preferences to guide LLM. Example: {\"framework\": \"react\", \"language\": \"typescript\"}",
			},
			"confirm_action": map[string]any{
				"type":        "string",
				"description": "Action for confirm: 'confirm' (use LLM suggestion), 'modify' (custom input), 'auto_continue' (switch to auto mode)",
				"enum":        []string{"confirm", "modify", "auto_continue"},
			},
			"custom_input": map[string]any{
				"type":        "string",
				"description": "Custom input when confirm_action is 'modify'",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory (optional, defaults to workspace from context)",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute

	tool.sessions = make(map[string]*AutoInteractiveSession)
	tool.menuAnalyzer = NewMenuAnalyzer()
	tool.keySeq = NewTerminalKeySequence()
	tool.nextID = 1

	return tool
}

// Name returns the tool name
func (t *AutoInteractiveTool) Name() string {
	return t.Name_
}

// Description returns the tool description
func (t *AutoInteractiveTool) Description() string {
	return t.Desc_
}

// Parameters returns the tool parameters
func (t *AutoInteractiveTool) Parameters() map[string]any {
	return t.Params_
}

// ExecuteWithContext implements the extended tool interface with context support
func (t *AutoInteractiveTool) ExecuteWithContext(ctx context.Context, args map[string]interface{}, toolCtx *tools.ToolContext) *tools.ToolResult {
	// Save LLM provider and model from context
	if toolCtx != nil {
		t.provider = toolCtx.LLM
		t.model = toolCtx.Model
	}

	// Convert args to params
	params := make(map[string]string)
	for k, v := range args {
		if strVal, ok := v.(string); ok {
			params[k] = strVal
		} else if v != nil {
			params[k] = fmt.Sprintf("%v", v)
		}
	}

	// Call execute
	result, err := t.execute(ctx, params)
	if err != nil {
		return tools.ErrorResult(err.Error())
	}
	return tools.NewToolResult(result)
}

func (t *AutoInteractiveTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "start":
		return t.startSession(ctx, params)
	case "confirm":
		return t.executeConfirmation(ctx, params)
	case "continue":
		return t.continueSession(ctx, params)
	case "end":
		return t.endSession(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// extractContext extracts context information from params
func (t *AutoInteractiveTool) extractContext(params map[string]string) (workspace, channel, chatID string) {
	workspace = params[constant.ToolCallParamWorkspace]
	channel = params[constant.ToolCallParamChannel]
	chatID = params[constant.ToolCallParamChatID]
	return
}

// startSession starts a new auto-interactive session
func (t *AutoInteractiveTool) startSession(ctx context.Context, params map[string]string) (string, error) {
	command := params["command"]
	if command == "" {
		return "", fmt.Errorf("command is required for start action")
	}

	workingDir := params["working_dir"]
	workspace, channel, chatID := t.extractContext(params)

	// Use working_dir if specified, otherwise use workspace
	if workingDir == "" && workspace != "" {
		workingDir = workspace
	}

	// Parse max_iterations
	maxIterations := 10
	if maxIterStr := params["max_iterations"]; maxIterStr != "" {
		if val, err := strconv.Atoi(maxIterStr); err == nil {
			maxIterations = val
		}
	}

	// Parse auto_mode
	autoMode := params["auto_mode"] == "true"

	// Parse preferences
	var preferences map[string]string
	if prefsStr := params["preferences"]; prefsStr != "" {
		if err := json.Unmarshal([]byte(prefsStr), &preferences); err != nil {
			// If JSON parsing fails, treat as simple key=value format
			preferences = make(map[string]string)
			for _, pair := range strings.Split(prefsStr, ",") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					preferences[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}
	}

	// Create command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd")
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up environment for PTY
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"FORCE_COLOR=1",
	)

	// Start with PTY
	pseudoTerminal, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
	}

	// Generate session ID
	t.sessionsMu.Lock()
	sessionID := fmt.Sprintf("auto-%d", t.nextID)
	t.nextID++

	session := &AutoInteractiveSession{
		ID:                 sessionID,
		Cmd:                cmd,
		Pty:                pseudoTerminal,
		StartTime:          time.Now(),
		State:              SessionRunning,
		maxIterations:      maxIterations,
		iteration:          0,
		autoMode:           autoMode,
		inputWaitThreshold: 2 * time.Second,
		preferences:        preferences,
		lastOutput:         time.Now(),
		workspace:          workspace,
		channel:            channel,
		chatID:             chatID,
		llmProvider:        t.provider, // Capture from tool
		llmModel:           t.model,    // Capture from tool
	}

	t.sessions[sessionID] = session
	t.sessionsMu.Unlock()

	// Start output reader
	go t.readSessionOutput(session)

	// Start the auto-interactive loop
	resultChan := make(chan *autoInteractiveResult, 1)
	go t.runAutoInteractiveLoop(ctx, session, resultChan)

	// Wait a bit for initial output
	time.Sleep(500 * time.Millisecond)

	// Return initial result
	result := fmt.Sprintf("✅ Auto-interactive session started\nSession ID: %s\nCommand: %s\nAuto Mode: %v\n\n",
		sessionID, command, autoMode)

	// Try to get initial output
	initialOutput := t.getIncrementalOutput(session)
	if initialOutput != "" {
		result += "=== Initial Output ===\n" + initialOutput
		if len(initialOutput) > 500 {
			result += "\n... (output may be truncated, use get_output to see full)"
		}
	} else {
		result += "(Waiting for output...)"
	}

	return result, nil
}

// analyzeWithLLM analyzes output using LLM with structured JSON output
// It uses the LLM provider captured in the session
func (t *AutoInteractiveTool) analyzeWithLLM(ctx context.Context, session *AutoInteractiveSession, output, pattern string) (*InteractionResponse, error) {
	// Use the LLM provider and model from the session
	if session.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not available - session was created without LLM context")
	}

	// Build system prompt
	systemPrompt := `You are an expert at analyzing interactive command-line interfaces.
Your task is to analyze the command output and suggest an appropriate response.

Key sequences:
- "UP", "DOWN", "LEFT", "RIGHT" for arrow navigation
- "ENTER" for confirmation
- Numbers for numbered menus (e.g., "1" for first option)
- "y"/"n" for yes/no
- Plain text for other input

Guidelines:
- Use high confidence (>0.9) for obvious cases (clear yes/no, numbered menus)
- Use medium confidence (0.6-0.9) for reasonable but not certain cases
- Use low confidence (<0.6) for ambiguous cases requiring user input
- Extract menu options when a menu is detected
- Consider user preferences when provided`

	// Add user preferences to system prompt if available
	if len(session.preferences) > 0 {
		prefsJSON, _ := json.Marshal(session.preferences)
		systemPrompt += fmt.Sprintf("\n\nUser Preferences: %s", string(prefsJSON))
	}

	userPrompt := fmt.Sprintf("Analyze this command output and suggest a response:\n\n%s", output)

	// Build JSON schema for structured output
	jsonSchema := &llm.JSONSchema{
		Name:        "interaction_response",
		Description: "Analysis of interactive command-line interface output with suggested response",
		Strict:      true,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"suggestion": map[string]any{
					"type":        "string",
					"description": "The suggested input text or key sequence to send",
				},
				"reasoning": map[string]any{
					"type":        "string",
					"description": "Explanation of why this response is appropriate",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "Confidence score from 0.0 to 1.0",
				},
				"input_type": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "arrow", "enter"},
					"description": "Type of input being suggested",
				},
				"menu_detected": map[string]any{
					"type":        "boolean",
					"description": "Whether a menu was detected in the output",
				},
				"menu_options": map[string]any{
					"type":        "array",
					"description": "List of detected menu options (if menu_detected is true)",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"index": map[string]any{
								"type":        "integer",
								"description": "Option index (1-based)",
							},
							"text": map[string]any{
								"type":        "string",
								"description": "Option text description",
							},
							"input_value": map[string]any{
								"type":        "string",
								"description": "Value to input to select this option",
							},
						},
						"required": []string{"index", "text", "input_value"},
					},
				},
			},
			"required":             []string{"suggestion", "reasoning", "confidence", "input_type", "menu_detected"},
			"additionalProperties": false,
		},
	}

	// Call LLM using structured output
	msgs := []llm.Message{
		{Role: constant.RoleSystem, Content: systemPrompt},
		{Role: constant.RoleUser, Content: userPrompt},
	}

	resp, err := session.llmProvider.ChatWithJSONSchema(ctx, session.llmModel, msgs, jsonSchema, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response - with structured output, we should get clean JSON
	var llmResp InteractionResponse
	if err := json.Unmarshal([]byte(resp.Message.Content), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	llmResp.Context = output
	return &llmResp, nil
}

// executeSuggestion executes the LLM suggestion
func (t *AutoInteractiveTool) executeSuggestion(session *AutoInteractiveSession, llmResp *InteractionResponse) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	// Generate input from suggestion
	input := t.keySeq.GenerateFromSuggestion(llmResp.Suggestion, llmResp.InputType)

	// Send to PTY
	if _, err := session.Pty.WriteString(input); err != nil {
		return fmt.Errorf("failed to write to PTY: %w", err)
	}

	session.lastOutput = time.Now()
	return nil
}

// executeConfirmation handles confirmation action
func (t *AutoInteractiveTool) executeConfirmation(ctx context.Context, params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	confirmAction := params["confirm_action"]
	if confirmAction == "" {
		return "", fmt.Errorf("confirm_action is required")
	}

	t.sessionsMu.RLock()
	session, exists := t.sessions[sessionID]
	t.sessionsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	switch confirmAction {
	case "confirm":
		// Use LLM suggestion
		if session.lastLLMResponse == nil {
			return "", fmt.Errorf("no LLM suggestion available")
		}
		if err := t.executeSuggestion(session, session.lastLLMResponse); err != nil {
			return "", err
		}
		session.iteration++

		// Continue the loop
		return t.continueSession(ctx, params)

	case "modify":
		// Use custom input
		customInput := params["custom_input"]
		if customInput == "" {
			return "", fmt.Errorf("custom_input is required for modify action")
		}
		session.mu.Lock()
		if _, err := session.Pty.WriteString(customInput + "\n"); err != nil {
			session.mu.Unlock()
			return "", fmt.Errorf("failed to write custom input: %w", err)
		}
		session.lastOutput = time.Now()
		session.mu.Unlock()
		session.iteration++

		// Continue the loop
		return t.continueSession(ctx, params)

	case "auto_continue":
		// Switch to auto mode and continue
		session.autoMode = true
		return t.continueSession(ctx, params)

	default:
		return "", fmt.Errorf("unknown confirm_action: %s", confirmAction)
	}
}

// continueSession continues the auto-interactive loop
func (t *AutoInteractiveTool) continueSession(ctx context.Context, params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	t.sessionsMu.RLock()
	session, exists := t.sessions[sessionID]
	t.sessionsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	// Restart the loop
	resultChan := make(chan *autoInteractiveResult, 1)
	go t.runAutoInteractiveLoop(ctx, session, resultChan)

	// Wait for result with timeout
	select {
	case r := <-resultChan:
		if r.needsConfirmation {
			return t.formatConfirmationRequest(sessionID, r), nil
		}
		return r.message, nil
	case <-time.After(5 * time.Second):
		return "⏳ Continuing execution... (use 'continue' action to check status)", nil
	}
}

// endSession ends an auto-interactive session
func (t *AutoInteractiveTool) endSession(params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	t.sessionsMu.Lock()
	session, exists := t.sessions[sessionID]
	if exists {
		delete(t.sessions, sessionID)
	}
	t.sessionsMu.Unlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	wasClosed := session.State == SessionClosed
	session.State = SessionClosed
	session.mu.Unlock()

	// Close PTY
	session.Pty.Close()

	// Kill process if still running
	if session.Cmd.Process != nil {
		session.Cmd.Process.Kill()
		session.Cmd.Wait()
	}

	result := fmt.Sprintf("✅ Session %s ended\n", sessionID)
	if !wasClosed {
		result += "(Session was still running, terminated)\n"
	}

	// Get final output
	finalOutput := t.getFullOutput(session)
	if finalOutput != "" {
		result += "\n=== Final Output ===\n" + finalOutput
		if len(finalOutput) > 10000 {
			result += "\n... (final output was very large)"
		}
	}

	return result, nil
}

// runAutoInteractiveLoop runs the main auto-interactive loop
func (t *AutoInteractiveTool) runAutoInteractiveLoop(ctx context.Context, session *AutoInteractiveSession, resultChan chan<- *autoInteractiveResult) {
	for session.iteration < session.maxIterations {
		// Wait for and detect input waiting
		waitDetected, _, pattern := t.waitForInput(ctx, session)

		if !waitDetected {
			if t.isSessionEnded(session) {
				resultChan <- &autoInteractiveResult{
					message: t.formatFinalResult(session, "Command completed"),
				}
				return
			}
			continue
		}

		// Get incremental output
		currentOutput := t.getIncrementalOutput(session)

		// LLM analysis (using session's LLM provider)
		llmResp, err := t.analyzeWithLLM(ctx, session, currentOutput, pattern)
		if err != nil {
			// If LLM fails, return confirmation request
			resultChan <- &autoInteractiveResult{
				message:           fmt.Sprintf("⚠️ LLM analysis failed: %v\n\nOutput:\n%s", err, currentOutput),
				needsConfirmation: true,
				output:            currentOutput,
			}
			return
		}

		session.lastLLMResponse = llmResp

		// Determine if confirmation is needed
		requiresConfirmation := !session.autoMode && llmResp.Confidence < 0.8

		if requiresConfirmation {
			// Return confirmation request to Agent
			resultChan <- &autoInteractiveResult{
				needsConfirmation: true,
				llmResponse:       llmResp,
				output:            currentOutput,
			}
			return
		}

		// Auto-execute suggestion
		t.executeSuggestion(session, llmResp)
		session.iteration++
	}

	resultChan <- &autoInteractiveResult{
		message: t.formatFinalResult(session, "Maximum iterations reached"),
	}
}

// Helper functions

func (t *AutoInteractiveTool) readSessionOutput(session *AutoInteractiveSession) {
	buf := make([]byte, 4096)
	for {
		n, err := session.Pty.Read(buf)
		if n > 0 {
			session.mu.Lock()
			session.Output.Write(buf[:n])
			session.lastOutput = time.Now()
			session.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
}

func (t *AutoInteractiveTool) isSessionEnded(session *AutoInteractiveSession) bool {
	session.mu.RLock()
	defer session.mu.RUnlock()

	// Check if process has exited
	if session.Cmd.ProcessState != nil && session.Cmd.ProcessState.Exited() {
		session.State = SessionClosed
		return true
	}
	return false
}

func (t *AutoInteractiveTool) waitForInput(ctx context.Context, session *AutoInteractiveSession) (bool, InputWaitType, string) {
	session.mu.RLock()
	initialLen := session.Output.Len()
	session.mu.RUnlock()

	// Wait for output or timeout
	timeout := time.NewTimer(session.inputWaitThreshold)

	for {
		select {
		case <-ctx.Done():
			return false, WaitTypeContentPattern, ""
		case <-timeout.C:
			// Check if output changed
			session.mu.RLock()
			currentLen := session.Output.Len()
			output := session.Output.String()
			session.mu.RUnlock()

			if currentLen > initialLen {
				// Got new output, check for patterns
				if waitDetected, waitType, pattern := t.menuAnalyzer.DetectInputWaiting(output); waitDetected {
					return true, waitType, pattern
				}
				// Reset timer and continue waiting
				timeout.Reset(session.inputWaitThreshold)
			} else if currentLen > 0 {
				// Output exists but no new content for threshold time - likely waiting
				return true, WaitTypeOutputSilence, ""
			}
		}
	}
}

func (t *AutoInteractiveTool) getIncrementalOutput(session *AutoInteractiveSession) string {
	session.mu.Lock()
	defer session.mu.Unlock()

	output := session.Output.String()
	if session.LastOutputPos >= len(output) {
		return ""
	}

	incremental := output[session.LastOutputPos:]
	session.LastOutputPos = len(output)
	return incremental
}

func (t *AutoInteractiveTool) getFullOutput(session *AutoInteractiveSession) string {
	session.mu.Lock()
	defer session.mu.Unlock()

	return session.Output.String()
}

func (t *AutoInteractiveTool) formatFinalResult(session *AutoInteractiveSession, reason string) string {
	session.mu.RLock()
	defer session.mu.RUnlock()

	output := session.Output.String()
	result := fmt.Sprintf("✅ Session %s ended\nReason: %s\n\n", session.ID, reason)

	if output != "" {
		result += "=== Final Output ===\n" + output
	}

	return result
}

func (t *AutoInteractiveTool) formatConfirmationRequest(sessionID string, r *autoInteractiveResult) string {
	result := fmt.Sprintf("🤔 Command requires input:\n\n%s\n\n", r.output)
	if r.llmResponse != nil {
		result += fmt.Sprintf("💡 LLM Suggestion: %s\n", r.llmResponse.Suggestion)
		result += fmt.Sprintf("📊 Confidence: %.0f%%\n", r.llmResponse.Confidence*100)
		if r.llmResponse.Reasoning != "" {
			result += fmt.Sprintf("💭 Reasoning: %s\n", r.llmResponse.Reasoning)
		}
	}
	result += "\nUse confirm_action='confirm' to accept, 'modify' for custom input, or 'auto_continue' to switch to auto mode"
	return result
}

// autoInteractiveResult holds the result from auto-interactive loop
type autoInteractiveResult struct {
	message           string
	needsConfirmation bool
	llmResponse       *InteractionResponse
	output            string
}
