package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type TaskInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // low, normal, high, urgent
	Status      string `json:"status"`   // pending, running, completed, failed, cancelled, remove
	LastResult  string `json:"last_result"`

	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
}

type TaskTool struct {
	basic.SimpleTool
}

func NewTaskTool() *TaskTool {
	tool := new(TaskTool)
	tool.Name_ = "task"
	tool.Desc_ = "Create and manage asynchronous tasks that execute in the background without blocking the current conversation loop. Unlike synchronous tool calls that wait for completion, tasks created here run independently and do not occupy the current session context. This is ideal for long-running operations, scheduled tasks triggered by cron, or any work that should proceed without keeping the user waiting. Tasks persist in the agent's workspace and their status/results can be queried later."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform on the task",
				"enum":        []string{"create", "list", "status", "cancel", "clean", "restart"},
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for status, cancel, and clean actions). For clean action without id, all completed/failed/cancelled tasks will be cleaned.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Task name/identifier (required for create action)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed task description including objectives, instructions, or commands to execute (required for create action)",
			},
			"priority": map[string]any{
				"type":        "string",
				"description": "Task execution priority (optional, defaults to 'normal'). Higher priority tasks are processed first.",
				"enum":        []string{"low", "normal", "high", "urgent"},
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *TaskTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action, ok := params["action"]
	if !ok {
		return "", fmt.Errorf("action parameter is required")
	}
	switch action {
	case "create":
		return t.create(params)
	case "list":
		return t.list(params)
	case "status":
		return t.status(params)
	case "cancel":
		return t.cancel(params)
	case "clean":
		return t.clean(params)
	case "restart":
		return t.restart(params)
	default:
		return "", fmt.Errorf("invalid action: %s", action)
	}
}

func (t *TaskTool) create(params map[string]string) (string, error) {
	name, ok := params["name"]
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}
	description, ok := params["description"]
	if !ok || description == "" {
		return "", fmt.Errorf("description parameter is required")
	}
	priority, ok := params["priority"]
	if !ok {
		priority = "normal"
	}

	workspace := params[constant.ToolCallParamWorkspace]
	channel := params[constant.ToolCallParamChannel]
	chatID := params[constant.ToolCallParamChatID]

	at := &TaskInfo{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Priority:    priority,
		Status:      "pending",

		Channel: channel,
		ChatID:  chatID,
	}

	// 写入对应文件
	taskDir := filepath.Join(workspace, "tasks", at.ID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create task directory: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, "task.json")
	data, err := json.Marshal(at)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}
	if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %w", err)
	}

	return fmt.Sprintf("Task Created: %s", at.ID), nil
}

func (t *TaskTool) list(params map[string]string) (string, error) {
	workspace := params[constant.ToolCallParamWorkspace]
	taskDir := filepath.Join(workspace, "tasks")
	files, err := os.ReadDir(taskDir)
	if err != nil {
		return "", fmt.Errorf("failed to read tasks directory: %w", err)
	}
	result := "Task List:\n"
	for _, file := range files {
		if file.IsDir() {
			taskFilePath := filepath.Join(taskDir, file.Name(), "task.json")
			if _, err := os.Stat(taskFilePath); err != nil {
				continue
			}
			data, err := os.ReadFile(taskFilePath)
			if err != nil {
				continue
			}
			var at TaskInfo
			if err := json.Unmarshal(data, &at); err != nil {
				continue
			}
			result += fmt.Sprintf("- %s: %s [%s]\n", at.ID, at.Name, at.Status)
		}
	}

	return result, nil
}

func (t *TaskTool) status(params map[string]string) (string, error) {
	id, ok := params["id"]
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	taskDir := filepath.Join(workspace, "tasks", id)
	if _, err := os.Stat(taskDir); err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, "task.json")
	data, err := os.ReadFile(taskFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read task file: %w", err)
	}
	var at TaskInfo
	if err := json.Unmarshal(data, &at); err != nil {
		return "", fmt.Errorf("failed to unmarshal task: %w", err)
	}
	return fmt.Sprintf("Task Status: %s\nLast Result: %s", at.Status, at.LastResult), nil

}

func (t *TaskTool) cancel(params map[string]string) (string, error) {
	id, ok := params["id"]
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	taskDir := filepath.Join(workspace, "tasks", id)
	if _, err := os.Stat(taskDir); err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, "task.json")
	data, err := os.ReadFile(taskFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read task file: %w", err)
	}
	var at TaskInfo
	if err := json.Unmarshal(data, &at); err != nil {
		return "", fmt.Errorf("failed to unmarshal task: %w", err)
	}

	at.Status = "cancelled"
	data, err = json.Marshal(at)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}
	if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %w", err)
	}

	return fmt.Sprintf("Task Cancelled: %s\nLast Result: %s", at.ID, at.LastResult), nil

}

func (t *TaskTool) clean(params map[string]string) (string, error) {
	id, _ := params["id"]

	workspace := params[constant.ToolCallParamWorkspace]
	tasksDir := filepath.Join(workspace, "tasks")

	if id != "" {
		taskFilePath := filepath.Join(tasksDir, id, "task.json")
		if _, err := os.Stat(taskFilePath); err != nil {
			return "", fmt.Errorf("task not found: %w", err)
		}
		// 标记status为remove
		data, err := os.ReadFile(taskFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read task file: %w", err)
		}
		var at TaskInfo
		if err = json.Unmarshal(data, &at); err != nil {
			return "", fmt.Errorf("failed to unmarshal task: %w", err)
		}
		at.Status = "remove"
		data, err = json.Marshal(at)
		if err != nil {
			return "", fmt.Errorf("failed to marshal task: %w", err)
		}
		if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
			return "", fmt.Errorf("failed to write task file: %w", err)
		}
		return fmt.Sprintf("Task Cleaned: %s\nLast Result: %s", at.ID, at.LastResult), nil
	}

	// 没有传递ID，则清理所有完成、取消的任务
	files, err := os.ReadDir(tasksDir)
	if err != nil {
		return "", fmt.Errorf("failed to read tasks directory: %w", err)
	}
	for _, file := range files {
		if file.IsDir() {
			taskFilePath := filepath.Join(tasksDir, file.Name(), "task.json")
			if _, err := os.Stat(taskFilePath); err != nil {
				continue
			}
			data, err := os.ReadFile(taskFilePath)
			if err != nil {
				continue
			}
			var at TaskInfo
			if err = json.Unmarshal(data, &at); err != nil {
				continue
			}
			if at.Status == "completed" || at.Status == "cancelled" {
				at.Status = "remove"
				data, err = json.Marshal(at)
				if err != nil {
					continue
				}
				if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
					continue
				}
			}
		}
	}
	return "Completed & Cancelled Tasks Cleaned", nil
}

func (t *TaskTool) restart(params map[string]string) (string, error) {
	id, ok := params["id"]
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	taskDir := filepath.Join(workspace, "tasks", id)
	if _, err := os.Stat(taskDir); err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, "task.json")
	data, err := os.ReadFile(taskFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read task file: %w", err)
	}
	var at TaskInfo
	if err := json.Unmarshal(data, &at); err != nil {
		return "", fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// 只有已完成、失败或已取消的任务可以重启
	if at.Status != "completed" && at.Status != "failed" && at.Status != "cancelled" {
		return "", fmt.Errorf("cannot restart task with status '%s', only completed, failed or cancelled tasks can be restarted", at.Status)
	}

	at.Status = "pending"
	at.LastResult = ""
	data, err = json.Marshal(at)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}
	if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %w", err)
	}

	return fmt.Sprintf("Task Restarted: %s", at.ID), nil
}
