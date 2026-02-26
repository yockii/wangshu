package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

// SessionState represents the state of an interactive session
type SessionState string

const (
	SessionRunning SessionState = "running"
	SessionClosed  SessionState = "closed"
)

// InteractiveSession manages an interactive shell session
type InteractiveSession struct {
	ID         string
	Cmd        *exec.Cmd
	Pty        *os.File
	Output     strings.Builder
	StartTime  time.Time
	State      SessionState
	mu         sync.RWMutex
	timeout    time.Duration
	lastOutput time.Time
}

// SessionManager manages interactive sessions
type SessionManager struct {
	sessions map[string]*InteractiveSession
	mu       sync.RWMutex
	nextID   int
}

var globalSessionManager = &SessionManager{
	sessions: make(map[string]*InteractiveSession),
	nextID:   1,
}

// InteractiveTool provides interactive shell capabilities
type InteractiveTool struct {
	basic.SimpleTool
}

func NewInteractiveTool() *InteractiveTool {
	tool := new(InteractiveTool)
	tool.Name_ = "interactive_shell"
	tool.Desc_ = "Execute commands in an interactive shell session. Use this when a command requires user input or prompts during execution (e.g., 'npm create vite', 'vue create project', installation wizards). The tool maintains a session where you can send input and receive output iteratively until the command completes. Example workflow: 1) start a session with a command, 2) read prompts, 3) send responses, 4) repeat until finished."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: start, send, get_output, end",
				"enum":        []string{"start", "send", "get_output", "end"},
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Command to start (required for 'start' action). Example: 'npm create vite@latest my-app'",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID (required for 'send', 'get_output', and 'end' actions). Obtained from 'start' action.",
			},
			"input": map[string]any{
				"type":        "string",
				"description": "Input to send to the command (required for 'send' action). Example: 'vue', 'yes', or option numbers like '1'.",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory (optional, for 'start' action).",
			},
			"wait_prompt": map[string]any{
				"type":        "boolean",
				"description": "Wait for initial prompt before returning (optional, for 'start' action, default: true).",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *InteractiveTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "start":
		return t.startSession(params)
	case "send":
		return t.sendInput(params)
	case "get_output":
		return t.getOutput(params)
	case "end":
		return t.endSession(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// startSession starts a new interactive session
func (t *InteractiveTool) startSession(params map[string]string) (string, error) {
	command := params["command"]
	if command == "" {
		return "", fmt.Errorf("command is required for start action")
	}

	workingDir := params["working_dir"]
	waitPrompt := params["wait_prompt"] != "false" // default true

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
	globalSessionManager.mu.Lock()
	sessionID := fmt.Sprintf("shell-%d", globalSessionManager.nextID)
	globalSessionManager.nextID++

	session := &InteractiveSession{
		ID:        sessionID,
		Cmd:       cmd,
		Pty:       pseudoTerminal,
		StartTime: time.Now(),
		State:     SessionRunning,
		timeout:   10 * time.Second,
	}

	globalSessionManager.sessions[sessionID] = session
	globalSessionManager.mu.Unlock()

	// Start output reader
	go t.readSessionOutput(session)

	// Wait for initial prompt if requested
	if waitPrompt {
		time.Sleep(500 * time.Millisecond) // Give command time to start
	}

	// Monitor process in background
	go t.monitorSession(session)

	result := fmt.Sprintf("✅ Interactive session started\nSession ID: %s\nCommand: %s\n\n", sessionID, command)
	result += "💡 How to use:\n"
	result += "1. Use 'get_output' action to read prompts\n"
	result += "2. Use 'send' action to respond to prompts\n"
	result += "3. Repeat until command completes\n"
	result += "4. Use 'end' action to close the session\n\n"

	// Try to get initial output
	initialOutput := t.getOutputInternal(session, false)
	if initialOutput != "" {
		result += "=== Initial Output ===\n" + initialOutput
		if len(initialOutput) > 500 {
			result += "\n... (output may be truncated, use get_output to see full)"
		}
	} else {
		result += "(Waiting for output... use get_output to check for prompts)"
	}

	return result, nil
}

// readSessionOutput continuously reads from PTY
func (t *InteractiveTool) readSessionOutput(session *InteractiveSession) {
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

// monitorSession monitors when the session ends
func (t *InteractiveTool) monitorSession(session *InteractiveSession) {
	err := session.Cmd.Wait()

	session.mu.Lock()
	session.State = SessionClosed
	session.mu.Unlock()

	if err != nil {
		fmt.Printf("[Session %s] Command exited with error: %v\n", session.ID, err)
	} else {
		fmt.Printf("[Session %s] Command completed successfully\n", session.ID)
	}
}

// sendInput sends input to the interactive session
func (t *InteractiveTool) sendInput(params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required for send action")
	}

	input := params["input"]
	if input == "" {
		return "", fmt.Errorf("input is required for send action")
	}

	globalSessionManager.mu.RLock()
	session, exists := globalSessionManager.sessions[sessionID]
	globalSessionManager.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	if session.State != SessionRunning {
		session.mu.Unlock()
		return "", fmt.Errorf("session %s is already closed", sessionID)
	}
	session.mu.Unlock()

	// Send input with newline
	data := []byte(input + "\n")
	if _, err := session.Pty.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to session: %w", err)
	}

	// Wait a bit for response
	time.Sleep(500 * time.Millisecond)

	// Get response
	response := t.getOutputInternal(session, false)

	result := fmt.Sprintf("✅ Input sent to session %s\nInput: %s\n\n", sessionID, input)
	if response != "" {
		result += "=== Response ===\n" + response
		if len(response) > 1000 {
			result += "\n... (response may be truncated, use get_output to see full)"
		}
	} else {
		result += "(No response yet. Use get_output to check for new output.)"
	}

	// Check if session is still running
	session.mu.Lock()
	state := session.State
	session.mu.Unlock()

	if state == SessionClosed {
		result += "\n\n⚠️ Session has closed (command completed)"
	}

	return result, nil
}

// getOutput retrieves output from the session
func (t *InteractiveTool) getOutput(params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required for get_output action")
	}

	globalSessionManager.mu.RLock()
	session, exists := globalSessionManager.sessions[sessionID]
	globalSessionManager.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	output := t.getOutputInternal(session, true)

	session.mu.RLock()
	state := session.State
	runtime := time.Since(session.StartTime).Round(time.Second)
	session.mu.RUnlock()

	result := fmt.Sprintf("📄 Session %s Output\n\n", sessionID)
	result += fmt.Sprintf("Status: %s\n", state)
	result += fmt.Sprintf("Runtime: %s\n\n", runtime)

	if output != "" {
		result += output
	} else {
		result += "(No output yet. Command may still be starting...)"
	}

	if state == SessionRunning {
		result += "\n\n💡 Session is still running. You can:"
		result += "\n- Send more input with 'send' action"
		result += "\n- Check for new output with 'get_output' again"
	} else {
		result += "\n\n✅ Session has closed. You can review the final output above."
	}

	return result, nil
}

// getOutputInternal gets output from a session
func (t *InteractiveTool) getOutputInternal(session *InteractiveSession, truncate bool) string {
	session.mu.Lock()
	defer session.mu.Unlock()

	outputStr := session.Output.String()

	// Skip empty output
	if strings.TrimSpace(outputStr) == "" {
		return ""
	}

	// Truncate if too long
	if truncate && len(outputStr) > 5000 {
		outputStr = outputStr[:4970] + "\n\n... (output truncated, showing first 4970 of " + strconv.Itoa(len(outputStr)) + " bytes)\nUse get_output again to see latest output"
	}

	return outputStr
}

// endSession closes an interactive session
func (t *InteractiveTool) endSession(params map[string]string) (string, error) {
	sessionID := params["session_id"]
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required for end action")
	}

	globalSessionManager.mu.Lock()
	session, exists := globalSessionManager.sessions[sessionID]
	if !exists {
		globalSessionManager.mu.Unlock()
		return "", fmt.Errorf("session %s not found", sessionID)
	}
	delete(globalSessionManager.sessions, sessionID)
	globalSessionManager.mu.Unlock()

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

	// Get final output
	finalOutput := t.getOutputInternal(session, false)

	result := fmt.Sprintf("✅ Session %s ended\n", sessionID)
	if !wasClosed {
		result += "(Session was still running, terminated)\n"
	}

	if finalOutput != "" {
		result += "\n=== Final Output ===\n" + finalOutput
		if len(finalOutput) > 10000 {
			result += "\n... (final output was very large)"
		}
	}

	return result, nil
}

// CleanupOldSessions removes sessions older than specified duration
func CleanupOldSessions(maxAge time.Duration) {
	globalSessionManager.mu.Lock()
	defer globalSessionManager.mu.Unlock()

	now := time.Now()
	for id, session := range globalSessionManager.sessions {
		session.mu.RLock()
		age := now.Sub(session.StartTime)
		isClosed := session.State == SessionClosed
		session.mu.RUnlock()

		if isClosed && age > maxAge {
			session.Pty.Close()
			if session.Cmd.Process != nil {
				session.Cmd.Process.Kill()
			}
			delete(globalSessionManager.sessions, id)
		}
	}
}

// GetSessionCount returns the number of active sessions
func GetSessionCount() int {
	globalSessionManager.mu.RLock()
	defer globalSessionManager.mu.RUnlock()

	count := 0
	for _, session := range globalSessionManager.sessions {
		session.mu.RLock()
		if session.State == SessionRunning {
			count++
		}
		session.mu.RUnlock()
	}
	return count
}
