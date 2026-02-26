package system

import (
	"context"
	"fmt"

	"github.com/yockii/yoclaw/internal/agent"
	"github.com/yockii/yoclaw/internal/cron"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type CronTool struct {
	basic.SimpleTool
}

func NewCronTool() *CronTool {
	tool := new(CronTool)
	tool.Name_ = "cron"
	tool.Desc_ = "Manage scheduled tasks that are stored in the agent workspace and persist across restarts. Supports add, remove, list, enable, disable, and status operations."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: add, remove, list, enable, disable, status",
				"enum":        []string{"add", "remove", "list", "enable", "disable", "status"},
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Task name (required for add, remove, enable, disable, status)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Task description (required for add)",
			},
			"schedule": map[string]any{
				"type":        "string",
				"description": "Cron schedule expression (e.g., '0 9 * * *' for daily at 9am, '*/5 * * * *' for every 5 minutes). Supports standard cron format with 6 fields (seconds, minutes, hours, day of month, month, day of week).",
			},
			"owner": map[string]any{
				"type":        "string",
				"description": "Owner ID to notify when task executes (optional, defaults to current user)",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *CronTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "add":
		return t.addTask(params)
	case "remove":
		return t.removeTask(params)
	case "list":
		return t.listTasks()
	case "enable":
		return t.enableTask(params)
	case "disable":
		return t.disableTask(params)
	case "status":
		return t.getTaskStatus(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *CronTool) addTask(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for add action")
	}

	schedule := params["schedule"]
	if schedule == "" {
		return "", fmt.Errorf("schedule is required for add action")
	}

	description := params["description"]
	owner := params["owner"]

	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	// Add task
	task, err := cronMgr.AddTask(name, description, schedule, owner)
	if err != nil {
		return "", fmt.Errorf("failed to add task: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务已创建\n任务名: %s\n调度: %s\n下次执行: %s\n任务ID: %s",
		task.Name, task.Schedule,
		task.NextRun.Format("2006-01-02 15:04:05"),
		task.ID), nil
}

func (t *CronTool) removeTask(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for remove action")
	}

	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	// Find task by name
	tasks := cronMgr.ListTasks()
	var taskID string
	for _, task := range tasks {
		if task.Name == name {
			taskID = task.ID
			break
		}
	}

	if taskID == "" {
		return "", fmt.Errorf("task '%s' not found", name)
	}

	// Remove task
	if err := cronMgr.RemoveTask(taskID); err != nil {
		return "", fmt.Errorf("failed to remove task: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已删除", name), nil
}

func (t *CronTool) listTasks() (string, error) {
	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	tasks := cronMgr.ListTasks()

	if len(tasks) == 0 {
		return "暂无定时任务", nil
	}

	result := fmt.Sprintf("定时任务列表 (%d 个):\n\n", len(tasks))
	for _, task := range tasks {
		status := "❌ 已禁用"
		if task.Enabled {
			status = "✅ 已启用"
		}

		result += fmt.Sprintf("- %s\n", task.Name)
		result += fmt.Sprintf("  状态: %s\n", status)
		result += fmt.Sprintf("  调度: %s\n", task.Schedule)
		if task.Description != "" {
			result += fmt.Sprintf("  描述: %s\n", task.Description)
		}
		if task.NextRun != nil {
			result += fmt.Sprintf("  下次执行: %s\n", task.NextRun.Format("2006-01-02 15:04:05"))
		}
		if task.LastRun != nil {
			result += fmt.Sprintf("  上次执行: %s\n", task.LastRun.Format("2006-01-02 15:04:05"))
		}
		if task.OwnerID != "" {
			result += fmt.Sprintf("  所有者: %s\n", task.OwnerID)
		}
		result += "\n"
	}

	return result, nil
}

func (t *CronTool) enableTask(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for enable action")
	}

	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	// Find task by name
	tasks := cronMgr.ListTasks()
	var taskID string
	for _, task := range tasks {
		if task.Name == name {
			taskID = task.ID
			break
		}
	}

	if taskID == "" {
		return "", fmt.Errorf("task '%s' not found", name)
	}

	// Enable task
	if err := cronMgr.EnableTask(taskID); err != nil {
		return "", fmt.Errorf("failed to enable task: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已启用", name), nil
}

func (t *CronTool) disableTask(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for disable action")
	}

	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	// Find task by name
	tasks := cronMgr.ListTasks()
	var taskID string
	for _, task := range tasks {
		if task.Name == name {
			taskID = task.ID
			break
		}
	}

	if taskID == "" {
		return "", fmt.Errorf("task '%s' not found", name)
	}

	// Disable task
	if err := cronMgr.DisableTask(taskID); err != nil {
		return "", fmt.Errorf("failed to disable task: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已禁用", name), nil
}

func (t *CronTool) getTaskStatus(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for status action")
	}

	// Get current agent
	ag, err := t.getCurrentAgent()
	if err != nil {
		return "", err
	}

	cronMgr := ag.GetCronManager()
	if cronMgr == nil {
		return "", fmt.Errorf("cron manager not initialized")
	}

	// Find task by name
	tasks := cronMgr.ListTasks()
	var targetTask *cron.Task
	for _, task := range tasks {
		if task.Name == name {
			targetTask = task
			break
		}
	}

	if targetTask == nil {
		return "", fmt.Errorf("task '%s' not found", name)
	}

	result := fmt.Sprintf("定时任务状态: %s\n\n", targetTask.Name)
	result += fmt.Sprintf("状态: ")
	if targetTask.Enabled {
		result += "✅ 已启用\n"
	} else {
		result += "❌ 已禁用\n"
	}
	result += fmt.Sprintf("调度: %s\n", targetTask.Schedule)
	if targetTask.Description != "" {
		result += fmt.Sprintf("描述: %s\n", targetTask.Description)
	}
	result += fmt.Sprintf("创建时间: %s\n", targetTask.CreatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("更新时间: %s\n", targetTask.UpdatedAt.Format("2006-01-02 15:04:05"))
	if targetTask.NextRun != nil {
		result += fmt.Sprintf("下次执行: %s\n", targetTask.NextRun.Format("2006-01-02 15:04:05"))
	}
	if targetTask.LastRun != nil {
		result += fmt.Sprintf("上次执行: %s\n", targetTask.LastRun.Format("2006-01-02 15:04:05"))
	}
	if targetTask.OwnerID != "" {
		result += fmt.Sprintf("所有者: %s\n", targetTask.OwnerID)
	}

	return result, nil
}

// getCurrentAgent returns the current agent executing this tool
// In a real implementation, this would get the agent from context
func (t *CronTool) getCurrentAgent() (*agent.Agent, error) {
	// Get default agent
	ag, exists := agent.GetDefaultAgent()
	if !exists {
		// Try to get any agent
		ag, exists = agent.GetAnyAgent()
		if !exists {
			return nil, fmt.Errorf("no agents available")
		}
	}
	return ag, nil
}
