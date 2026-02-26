package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type ExecTool struct {
	basic.SimpleTool
}

func NewExecTool() *ExecTool {
	tool := new(ExecTool)
	tool.Name_ = "exec"
	tool.Desc_ = "Execute a shell command and return its output. Supports PTY for interactive commands that require TTY."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in seconds (default: 30)",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
			"use_pty": map[string]any{
				"type":        "boolean",
				"description": "Whether to use PTY (pseudo-terminal) for interactive commands (default: false)",
			},
		},
		"required": []string{"command"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *ExecTool) execute(ctx context.Context, params map[string]string) (string, error) {
	command := params["command"]
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := 30 * time.Second
	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	workingDir := params["working_dir"]
	usePTY := params["use_pty"] == "true"

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if usePTY {
		return t.executeWithPTY(ctx, command, workingDir, timeout)
	}

	return t.executeStandard(ctx, command, workingDir)
}

// executeWithPTY executes a command using PTY for interactive commands
func (t *ExecTool) executeWithPTY(ctx context.Context, command string, workingDir string, timeout time.Duration) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Windows: use conhost for PTY-like behavior
		// Note: Windows PTY support is limited, using winpty as fallback
		cmd = exec.Command("cmd")
	} else {
		// Unix: use sh -c with PTY
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

	// Start the command with PTY
	pseudoTerminal, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
	}
	defer pseudoTerminal.Close()

	// Set up a channel to receive the output
	outputChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	// Read output in background
	go func() {
		buf := make([]byte, 4096)
		var output []byte
		for {
			n, err := pseudoTerminal.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		outputChan <- output
	}()

	// Wait for command completion
	go func() {
		errorChan <- cmd.Wait()
	}()

	// Wait for output or timeout
	select {
	case output := <-outputChan:
		err := <-errorChan
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					return string(output), fmt.Errorf("command exited with status %d", status.ExitStatus())
				}
			}
			return string(output), fmt.Errorf("command failed: %w", err)
		}
		return string(output), nil
	case <-time.After(timeout):
		// Kill the process group
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("command timed out after %v", timeout)
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("command cancelled")
	}
}

// executeStandard executes a command without PTY
func (t *ExecTool) executeStandard(ctx context.Context, command string, workingDir string) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out")
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return string(output), fmt.Errorf("command exited with status %d: %w", status.ExitStatus(), err)
			}
		}
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// RunInteractive executes a command with PTY for interactive programs
func RunInteractive(ctx context.Context, command string, timeout time.Duration) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"FORCE_COLOR=1",
	)

	// Start with PTY
	pseudoTerminal, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
	}
	defer pseudoTerminal.Close()

	// Read output
	buf := make([]byte, 4096)
	var output []byte

	done := make(chan error, 1)
	go func() {
		for {
			n, err := pseudoTerminal.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
			}
			if err != nil {
				done <- err
				return
			}
		}
	}()

	select {
	case <-done:
		err := <-done
		if err != nil {
			return string(output), err
		}
		return string(output), nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return string(output), fmt.Errorf("command timed out")
	}
}

// ExecInBackground executes a command in the background
func ExecInBackground(ctx context.Context, command string, workingDir string) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Windows: use start /b to run in background
		cmd = exec.Command("cmd", "/c", "start", "/b", command)
	} else {
		// Unix: use nohup to run in background
		cmd = exec.Command("sh", "-c", "nohup "+command+" > /dev/null 2>&1 &")
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	cmd.Env = os.Environ()

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start background command: %w", err)
	}

	return fmt.Sprintf("Background process started with PID: %d", cmd.Process.Pid), nil
}

// ParseCommandString parses a command string into program and args
func ParseCommandString(command string) (string, []string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
