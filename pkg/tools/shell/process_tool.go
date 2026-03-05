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

	"github.com/yockii/wangshu/pkg/tools/basic"
)

// ProcessStatus represents the current status of a process
type ProcessStatus string

const (
	StatusRunning   ProcessStatus = "running"
	StatusCompleted ProcessStatus = "completed"
	StatusFailed    ProcessStatus = "failed"
	StatusStopped   ProcessStatus = "stopped"
)

// ProcessInfo stores information about a background process
type ProcessInfo struct {
	Cmd       *exec.Cmd
	Output    strings.Builder
	Error     strings.Builder
	StartTime time.Time
	EndTime   *time.Time
	Command   string
	Status    ProcessStatus
	ExitCode  *int
	mu        sync.RWMutex
	Timeout   time.Duration
}

// ProcessManager manages background processes
type ProcessManager struct {
	processes map[int]*ProcessInfo
	mu        sync.RWMutex
	nextID    int
}

var globalProcessManager = &ProcessManager{
	processes: make(map[int]*ProcessInfo),
	nextID:    1,
}

func GetProcessManager() *ProcessManager {
	return globalProcessManager
}

type ProcessTool struct {
	basic.SimpleTool
}

func NewProcessTool() *ProcessTool {
	tool := new(ProcessTool)
	tool.Name_ = "process"
	tool.Desc_ = "Execute commands with optional wait for completion. By default, waits for the command to finish and returns output (similar to exec but with better timeout control). Use background=true for long-running tasks like servers - in this case, you should create a task to monitor progress. Use this tool for: builds, downloads, tests, or any command where you need the output to make decisions."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Command to execute (required). Example: 'npm install', 'make build', 'pytest tests/'",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "Run in background without waiting for completion (default: false). If true, command runs asynchronously and you get a process ID. You should create a task to monitor it or check status periodically.",
			},
			"timeout": map[string]any{
				"type":        "string",
				"description": "Maximum time to wait for command completion (default: 5m). Examples: '30s', '5m', '1h'. Only applies when background=false.",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Working directory for the command (optional).",
			},
			"env": map[string]any{
				"type":        "string",
				"description": "Environment variables (optional). Format: 'KEY1=value1,KEY2=value2'",
			},
		},
		"required": []string{"command"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *ProcessTool) execute(ctx context.Context, params map[string]string) (string, error) {
	command := params["command"]
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	background := params["background"] == "true"

	if background {
		// Background mode - start and return immediately
		return t.startBackgroundProcess(params)
	}

	// Foreground mode - wait for completion and return output
	return t.executeAndWait(params)
}

// executeAndWait runs command synchronously and returns output
func (t *ProcessTool) executeAndWait(params map[string]string) (string, error) {
	command := params["command"]
	timeoutStr := params["timeout"]
	workingDir := params["working_dir"]
	envStr := params["env"]

	// Parse timeout (default 5 minutes)
	timeout := 5 * time.Minute
	if timeoutStr != "" {
		var err error
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return "", fmt.Errorf("invalid timeout format: %w. Use formats like '30s', '5m', '1h'", err)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	if envStr != "" {
		envPairs := strings.Split(envStr, ",")
		for _, pair := range envPairs {
			cmd.Env = append(cmd.Env, strings.TrimSpace(pair))
		}
	}

	// Execute and capture output
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %v. Try increasing the timeout or using background=true for long-running tasks.", timeout)
	}

	// Build result
	result := fmt.Sprintf("Command: %s\n\n", command)
	if len(output) > 0 {
		outputStr := string(output)
		// Truncate if too long
		if len(outputStr) > 20000 {
			outputStr = outputStr[:19700] + "\n\n... (output truncated, showing first 19700 of " + strconv.Itoa(len(outputStr)) + " bytes)"
		}
		result += outputStr
	} else {
		result += "(no output)"
	}

	if err != nil {
		exitCode := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
		return result, fmt.Errorf("command failed with exit code %d", exitCode)
	}

	return result, nil
}

// startBackgroundProcess starts a process in background
func (t *ProcessTool) startBackgroundProcess(params map[string]string) (string, error) {
	command := params["command"]
	workingDir := params["working_dir"]
	envStr := params["env"]

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	if envStr != "" {
		envPairs := strings.Split(envStr, ",")
		for _, pair := range envPairs {
			cmd.Env = append(cmd.Env, strings.TrimSpace(pair))
		}
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Generate process ID
	GetProcessManager().mu.Lock()
	processID := GetProcessManager().nextID
	GetProcessManager().nextID++

	// Create process info
	info := &ProcessInfo{
		Cmd:       cmd,
		StartTime: time.Now(),
		Command:   command,
		Status:    StatusRunning,
	}

	GetProcessManager().processes[processID] = info
	GetProcessManager().mu.Unlock()

	// Monitor the process in background
	go t.monitorProcess(processID, stdout, stderr)

	return fmt.Sprintf("✅ Background process started\nID: %d\nPID: %d\nCommand: %s\n\n💡 Important: This process is running in the background. To monitor it:\n1. Use process status/check_status to see progress\n2. Use process get_output to retrieve output\n3. Consider creating a task to automatically check and report results", processID, cmd.Process.Pid, command), nil
}

func (t *ProcessTool) monitorProcess(processID int, stdout, stderr interface{}) {
	// Read stdout
	if stdoutReader, ok := stdout.(interface{ Read([]byte) (int, error) }); ok {
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stdoutReader.Read(buf)
				// Get info pointer with manager lock, then release before locking info
				GetProcessManager().mu.RLock()
				info, exists := GetProcessManager().processes[processID]
				if exists {
					// Increment ref count to prevent info from being deleted
					info.mu.Lock()
					GetProcessManager().mu.RUnlock()
					info.Output.Write(buf[:n])
					info.mu.Unlock()
				} else {
					GetProcessManager().mu.RUnlock()
				}
				if err != nil {
					break
				}
			}
		}()
	}

	// Read stderr
	if stderrReader, ok := stderr.(interface{ Read([]byte) (int, error) }); ok {
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stderrReader.Read(buf)
				// Get info pointer with manager lock, then release before locking info
				GetProcessManager().mu.RLock()
				info, exists := GetProcessManager().processes[processID]
				if exists {
					info.mu.Lock()
					GetProcessManager().mu.RUnlock()
					info.Error.Write(buf[:n])
					info.mu.Unlock()
				} else {
					GetProcessManager().mu.RUnlock()
				}
				if err != nil {
					break
				}
			}
		}()
	}

	// Wait for command to finish
	GetProcessManager().mu.RLock()
	processInfo, exists := GetProcessManager().processes[processID]
	if !exists {
		GetProcessManager().mu.RUnlock()
		return
	}
	cmd := processInfo.Cmd
	GetProcessManager().mu.RUnlock()
	_ = cmd.Wait()

	// Mark process as completed when done (after cmd.Wait() returns)
	GetProcessManager().mu.RLock()
	info, exists := GetProcessManager().processes[processID]
	GetProcessManager().mu.RUnlock()

	if exists {
		info.mu.Lock()
		if info.Status == StatusRunning {
			now := time.Now()
			info.EndTime = &now
			// Try to get exit code
			if info.Cmd.ProcessState != nil {
				exitCode := info.Cmd.ProcessState.ExitCode()
				info.ExitCode = &exitCode
				if exitCode != 0 {
					info.Status = StatusFailed
				} else {
					info.Status = StatusCompleted
				}
			} else {
				info.Status = StatusCompleted
			}
		}
		info.mu.Unlock()
	}
}

// Helper methods for managing background processes

func GetProcessStatus(processID int) (string, error) {
	GetProcessManager().mu.RLock()
	info, exists := GetProcessManager().processes[processID]
	GetProcessManager().mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("process with ID %d not found", processID)
	}

	info.mu.RLock()
	defer info.mu.RUnlock()

	result := fmt.Sprintf("Process %d Status:\n", processID)
	result += fmt.Sprintf("Status: %s\n", info.getStatusText())
	result += fmt.Sprintf("Command: %s\n", info.Command)

	if info.Cmd.Process != nil {
		result += fmt.Sprintf("PID: %d\n", info.Cmd.Process.Pid)
	}

	duration := time.Since(info.StartTime)
	result += fmt.Sprintf("Runtime: %s\n", duration.Round(time.Second))

	if info.EndTime != nil {
		result += fmt.Sprintf("Ended: %s\n", info.EndTime.Format("2006-01-02 15:04:05"))
	}

	if info.ExitCode != nil {
		result += fmt.Sprintf("Exit Code: %d\n", *info.ExitCode)
	}

	outputLen := info.Output.Len()
	errorLen := info.Error.Len()
	result += fmt.Sprintf("Output: %d bytes", outputLen)
	if errorLen > 0 {
		result += fmt.Sprintf(" | Errors: %d bytes", errorLen)
	}

	return result, nil
}

func (i *ProcessInfo) getStatusText() string {
	switch i.Status {
	case StatusRunning:
		return "🟢 Running"
	case StatusCompleted:
		return "✅ Completed"
	case StatusFailed:
		return "❌ Failed"
	case StatusStopped:
		return "⏹️ Stopped"
	default:
		return string(i.Status)
	}
}

func KillAllProcesses() {
	GetProcessManager().mu.Lock()
	defer GetProcessManager().mu.Unlock()

	for id, info := range GetProcessManager().processes {
		info.mu.Lock()
		if info.Status == StatusRunning {
			info.Status = StatusStopped
			now := time.Now()
			info.EndTime = &now
		}
		info.mu.Unlock()

		if info.Cmd.Process != nil {
			info.Cmd.Process.Kill()
		}
		delete(GetProcessManager().processes, id)
	}
}

func GetProcessCount() int {
	GetProcessManager().mu.RLock()
	defer GetProcessManager().mu.RUnlock()
	return len(GetProcessManager().processes)
}

func CleanupOldProcesses(maxAge time.Duration) {
	GetProcessManager().mu.Lock()
	defer GetProcessManager().mu.Unlock()

	now := time.Now()
	for id, info := range GetProcessManager().processes {
		info.mu.RLock()
		finished := info.Status != StatusRunning
		var endTime time.Time
		if info.EndTime != nil {
			endTime = *info.EndTime
		} else {
			endTime = info.StartTime
		}
		info.mu.RUnlock()

		if finished && now.Sub(endTime) > maxAge {
			delete(GetProcessManager().processes, id)
		}
	}
}
