package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
)

const (
	repoOwner = "yockii"
	repoName  = "wangshu"
)

type VersionTool struct {
	basic.SimpleTool
}

func NewVersionTool() *VersionTool {
	tool := new(VersionTool)
	tool.Name_ = constant.ToolNameVersion
	tool.Desc_ = "Query and manage application version. Supports checking current version, fetching latest version from GitHub, performing automatic updates, and restarting the application. Use this tool when you need to check version information, update application, or restart the application."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: 'current' to get current version, 'latest' to get latest version, 'check' to compare versions, 'update' to perform automatic update, 'restart' to restart the application",
				"enum":        []string{"current", "latest", "check", "update", "restart"},
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *VersionTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]

	switch action {
	case "current":
		return t.getCurrentVersion()
	case "latest":
		return t.getLatestVersion(ctx)
	case "check":
		return t.checkVersion(ctx)
	case "update":
		return t.update(ctx)
	case "restart":
		return t.restart(params)
	default:
		return "", fmt.Errorf("invalid action: %s", action)
	}
}

func (t *VersionTool) getCurrentVersion() (string, error) {
	return fmt.Sprintf("Current version: %s", constant.Version), nil
}

func (t *VersionTool) getLatestVersion(ctx context.Context) (string, error) {
	repository := selfupdate.NewRepositorySlug(repoOwner, repoName)
	latest, found, err := selfupdate.DetectLatest(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return "", fmt.Errorf("no release found for %s/%s", repoOwner, repoName)
	}

	return fmt.Sprintf("Latest version: %s", latest.Version()), nil
}

func (t *VersionTool) checkVersion(ctx context.Context) (string, error) {
	current := constant.Version
	if current == "dev" {
		return "Development version detected. Cannot compare with latest release.", nil
	}

	repository := selfupdate.NewRepositorySlug(repoOwner, repoName)
	latest, found, err := selfupdate.DetectLatest(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return "", fmt.Errorf("no release found for %s/%s", repoOwner, repoName)
	}

	latestVersion := latest.Version()

	if latest.LessOrEqual(current) {
		return fmt.Sprintf("You are running the latest version: %s", current), nil
	}
	return fmt.Sprintf("Update available: %s -> %s", current, latestVersion), nil

}

func (t *VersionTool) update(ctx context.Context) (string, error) {
	current := constant.Version
	if current == "dev" {
		return "Cannot update development version. Please use a release build.", nil
	}

	repository := selfupdate.NewRepositorySlug(repoOwner, repoName)
	latest, found, err := selfupdate.DetectLatest(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return "", fmt.Errorf("no release found for %s/%s", repoOwner, repoName)
	}

	latestVersion := latest.Version()

	if latest.LessOrEqual(current) {
		return fmt.Sprintf("Already running the latest version: %s", current), nil
	}

	updatedRelease, err := selfupdate.UpdateSelf(ctx, current, repository)
	if err != nil {
		return "", fmt.Errorf("failed to update to version %s: %w", latestVersion, err)
	}

	return fmt.Sprintf("Successfully updated to version %s. Use 'restart' action to restart the application and load the new version.", updatedRelease.Version()), nil
}

func (t *VersionTool) restart(params map[string]string) (string, error) {
	agentName := params[constant.ToolCallParamAgentName]
	if agentName == "" {
		return "", fmt.Errorf("agent_name parameter is required")
	}
	channel := params[constant.ToolCallParamChannel]
	if channel == "" {
		return "", fmt.Errorf("channel parameter is required")
	}
	chatID := params[constant.ToolCallParamChatID]
	if chatID == "" {
		return "", fmt.Errorf("chat_id parameter is required")
	}
	senderID := params[constant.ToolCallParamSenderID]
	if senderID == "" {
		return "", fmt.Errorf("sender_id parameter is required")
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// 重启标记文件
	restartFlagPath := filepath.Join(filepath.Dir(exePath), ".restart_flag")
	flagData := fmt.Sprintf("%s|%s|%s|%s", agentName, channel, chatID, senderID)
	if err := os.WriteFile(restartFlagPath, []byte(flagData), 0644); err != nil {
		return "", fmt.Errorf("failed to create restart flag: %w", err)
	}

	if err := t.restartSelf(exePath); err != nil {
		return "", fmt.Errorf("failed to restart: %w", err)
	}

	return "Restarting application...", nil
}

// restartSelf 处理跨平台重启
func (t *VersionTool) restartSelf(exePath string) error {
	// 获取当前命令行参数
	args := os.Args[1:]

	// --- 平台差异化处理 ---
	if runtime.GOOS == "windows" {
		// Windows 特殊处理：
		// 1. 启动新进程
		cmd := exec.Command(exePath, args...)

		// 继承标准输入输出，让用户感觉不到进程切换
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start new process: %w", err)
		}

		// 2. 等待一小会儿，确保新进程已经启动
		// 注意：不能 wait()，否则旧进程会卡住直到新进程结束
		time.Sleep(500 * time.Millisecond)

		// 3. 退出当前旧进程
		os.Exit(0)

	} else {
		// Unix/Linux/macOS 使用 syscall.Exec 替换当前进程
		// 这样可以保持进程 ID 和文件描述符不变
		err := syscall.Exec(exePath, append([]string{os.Args[0]}, args...), os.Environ())
		if err != nil {
			return fmt.Errorf("failed to exec: %w", err)
		}

		// syscall.Exec 成功后不会执行到这里
		// 如果执行到这里说明 exec 失败了
		return fmt.Errorf("exec returned unexpectedly")
	}

	return nil
}
