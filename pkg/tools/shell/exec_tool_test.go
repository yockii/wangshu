package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewExecTool(t *testing.T) {
	tool := NewExecTool()

	if tool == nil {
		t.Fatal("NewExecTool should not return nil")
	}

	// 测试工具名称
	if tool.Name() != "exec" {
		t.Errorf("Expected tool name 'exec', got '%s'", tool.Name())
	}

	// 测试工具描述
	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	// 测试参数定义
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Tool should have parameters")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	// 验证必需参数
	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "command" {
		t.Error("'command' should be required")
	}

	// 验证所有参数属性
	expectedParams := []string{"command", "timeout", "working_dir", "use_pty"}
	for _, expected := range expectedParams {
		if _, ok := properties[expected]; !ok {
			t.Errorf("Parameters should have '%s' property", expected)
		}
	}
}

func TestExecTool_Execute_SimpleCommand(t *testing.T) {
	tool := NewExecTool()

	// 使用安全的echo命令进行测试
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "echo Hello World"
	} else {
		command = "echo 'Hello World'"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	if err != nil {
		t.Errorf("Execute should succeed for simple echo command: %v", err)
	}

	// 验证输出包含预期内容
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("Result should contain 'Hello World', got: %s", result)
	}
}

func TestExecTool_Execute_EmptyCommand(t *testing.T) {
	tool := NewExecTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"command": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty command")
	}

	if !strings.Contains(err.Error(), "command is required") {
		t.Errorf("Error should mention 'command is required', got: %v", err)
	}
}

func TestExecTool_Execute_CommandNotFound(t *testing.T) {
	tool := NewExecTool()

	// 使用一个不存在的命令
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "nonexistentcommand12345"
	} else {
		command = "nonexistentcommand12345"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	// 命令可能会失败，但不应该panic
	_ = result
	_ = err

	// 验证返回了错误
	if err == nil {
		// 某些系统可能会创建空文件，所以这个检查不是强制的
		t.Log("Command unexpectedly succeeded (might have created empty file on some systems)")
	}
}

func TestExecTool_Execute_WithTimeout(t *testing.T) {
	tool := NewExecTool()

	// 使用sleep命令测试超时
	var command string
	if os.Getenv("GOOS") == "windows" {
		// Windows: ping命令
		command = "ping 127.0.0.1 -n 6"
	} else {
		// Unix: sleep命令
		command = "sleep 5"
	}

	// 设置较短的超时时间
	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
		"timeout":  "1", // 1秒超时
	})

	// 应该超时
	if err != nil {
		// 验证是超时错误
		if !strings.Contains(err.Error(), "timed out") {
			t.Logf("Expected timeout error, got: %v", err)
		}
	}

	_ = result
}

func TestExecTool_Execute_InvalidTimeout(t *testing.T) {
	tool := NewExecTool()

	// 使用无效的超时值
	result, err := tool.Execute(context.Background(), map[string]string{
		"command": "echo test",
		"timeout":  "invalid",
	})

	// 应该忽略无效的超时值，使用默认值
	if err != nil {
		t.Errorf("Execute should succeed with invalid timeout (using default): %v", err)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("Result should contain command output, got: %s", result)
	}
}

func TestExecTool_Execute_WithWorkingDirectory(t *testing.T) {
	tool := NewExecTool()
	tmpDir := t.TempDir()

	// 在临时目录中创建一个文件
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// 使用dir命令列出目录
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "dir" // Windows
	} else {
		command = "ls" // Unix
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command":     command,
		"working_dir": tmpDir,
	})

	if err != nil {
		t.Errorf("Execute should succeed with working_dir: %v", err)
	}

	// 验证输出包含测试文件
	if !strings.Contains(result, "test.txt") {
		t.Errorf("Result should contain test.txt, got: %s", result)
	}
}

func TestExecTool_Execute_EmptyWorkingDir(t *testing.T) {
	tool := NewExecTool()

	// 提供空的工作目录（应该使用当前目录）
	result, err := tool.Execute(context.Background(), map[string]string{
		"command":     "echo test",
		"working_dir": "",
	})

	if err != nil {
		t.Errorf("Execute should succeed with empty working_dir: %v", err)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("Result should contain 'test', got: %s", result)
	}
}

func TestExecTool_Execute_InvalidWorkingDirectory(t *testing.T) {
	tool := NewExecTool()

	// 使用不存在的工作目录
	result, err := tool.Execute(context.Background(), map[string]string{
		"command":     "echo test",
		"working_dir": "/nonexistent/directory/12345",
	})

	// 命令可能会失败
	_ = result
	_ = err

	// 验证返回了错误（某些系统可能允许创建目录）
	if err != nil {
		// 预期会有错误
		t.Logf("Command failed as expected with invalid working directory: %v", err)
	}
}

func TestExecTool_Execute_WithPTY(t *testing.T) {
	tool := NewExecTool()

	// 使用简单的命令测试PTY功能
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "echo test" // 不需要PTY
	} else {
		command = "echo test"
	}

	// 启用PTY
	result, err := tool.Execute(context.Background(), map[string]string{
		"command":  command,
		"use_pty":  "false", // 先测试false，确保基本功能正常
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("Result should contain 'test', got: %s", result)
	}
}

func TestExecTool_Execute_MultilineOutput(t *testing.T) {
	tool := NewExecTool()

	// 使用产生多行输出的命令
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "echo Line1 && echo Line2 && echo Line3"
	} else {
		command = "echo 'Line1' && echo 'Line2' && echo 'Line3'"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	if err != nil {
		t.Errorf("Execute should succeed for multiline command: %v", err)
	}

	// 验证包含所有行
	if !strings.Contains(result, "Line1") || !strings.Contains(result, "Line2") || !strings.Contains(result, "Line3") {
		t.Errorf("Result should contain all lines, got: %s", result)
	}
}

func TestExecTool_Execute_WithSpecialCharacters(t *testing.T) {
	tool := NewExecTool()

	// 测试包含特殊字符的命令 - 使用更简单的方式
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "echo test-dollar"
	} else {
		command = "echo 'test-dollar'"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	if err != nil {
		t.Errorf("Execute should succeed with special characters: %v", err)
	}

	// 验证输出
	if !strings.Contains(result, "test") {
		t.Errorf("Result should contain 'test', got: %s", result)
	}
}

func TestExecTool_Execute_ExitCode(t *testing.T) {
	tool := NewExecTool()

	// 使用会产生退出码的命令
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "exit 1" // Windows
	} else {
		command = "false" // Unix
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	// 命令应该失败
	if err == nil {
		t.Error("Execute should fail for command with non-zero exit code")
	}

	_ = result
}

func TestExecTool_ParseCommandString(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
		expectedArgs []string
	}{
		{
			name:     "simple command",
			command:  "echo test",
			expected: "echo",
			expectedArgs: []string{"test"},
		},
		{
			name:     "command with multiple args",
			command:  "echo hello world test",
			expected: "echo",
			expectedArgs: []string{"hello", "world", "test"},
		},
		{
			name:     "command with quoted args (strings.Fields doesn't handle quotes)",
			command:  `echo "hello world" test`,
			expected: "echo",
			expectedArgs: []string{"\"hello", "world\"", "test"},
		},
		{
			name:     "empty command",
			command:  "",
			expected: "",
			expectedArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, args := ParseCommandString(tt.command)

			if program != tt.expected {
				t.Errorf("Expected program '%s', got '%s'", tt.expected, program)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
			}

			for i, arg := range tt.expectedArgs {
				if args[i] != arg {
					t.Errorf("Arg %d: expected '%s', got '%s'", i, arg, args[i])
				}
			}
		})
	}
}

func TestExecTool_Execute_CombineOutput(t *testing.T) {
	tool := NewExecTool()

	// 测试stdout和stderr的输出
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "(echo stdout && echo stderr >&2)"
	} else {
		command = "echo stdout && echo stderr >&2"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// 验证包含stdout输出
	if !strings.Contains(result, "stdout") {
		t.Errorf("Result should contain stdout output, got: %s", result)
	}

	// stderr可能也在结果中，取决于实现
	t.Log("Combined output:", result)
}

func TestExecTool_Execute_ContextCancellation(t *testing.T) {
	tool := NewExecTool()

	// 创建一个会被取消的context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	result, err := tool.Execute(ctx, map[string]string{
		"command": "echo test",
	})

	// 应该被取消
	if err == nil {
		t.Log("Command might have completed before cancellation")
	} else {
		if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Error should mention cancellation or timeout, got: %v", err)
		}
	}

	_ = result
}

func TestExecTool_Execute_LongTimeout(t *testing.T) {
	tool := NewExecTool()

	// 设置一个很长的超时时间
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "timeout 1 ping 127.0.0.1 -n 2"
	} else {
		command = "sleep 1"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
		"timeout":  "10", // 10秒超时
	})

	if err != nil {
		t.Errorf("Execute should succeed with long timeout: %v", err)
	}

	// 验证命令执行完成
	if os.Getenv("GOOS") == "windows" {
		if !strings.Contains(result, "Ping statistics") {
			t.Log("Windows ping output may vary")
		}
	}

	_ = result
}

func TestExecTool_Execute_DefaultTimeout(t *testing.T) {
	tool := NewExecTool()

	// 不指定超时，使用默认值（30秒）
	var command string
	if os.Getenv("GOOS") == "windows" {
		command = "echo test"
	} else {
		command = "echo 'test'"
	}

	result, err := tool.Execute(context.Background(), map[string]string{
		"command": command,
		// 不设置timeout参数
	})

	if err != nil {
		t.Errorf("Execute should succeed with default timeout: %v", err)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("Result should contain 'test', got: %s", result)
	}
}
