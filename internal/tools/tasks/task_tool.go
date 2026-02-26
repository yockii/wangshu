package tasks

import (
	"context"
	"fmt"

	"github.com/yockii/yoclaw/internal/tasks"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type TaskTool struct {
	basic.SimpleTool
}

func NewTaskTool() *TaskTool {
	tool := new(TaskTool)
	tool.Name_ = "task"
	tool.Desc_ = "Manage long-running tasks with agent interaction. Supports create, list, status, pause, resume, cancel, progress operations."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: create, list, status, pause, resume, cancel, progress",
				"enum":        []string{"create", "list", "status", "pause", "resume", "cancel", "progress"},
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Task name (required for create)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Task description (required for create)",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for status, pause, resume, cancel, progress)",
			},
			"agent": map[string]any{
				"type":        "string",
				"description": "Agent name to assign task to (required for create, defaults to 'default')",
			},
			"schedule": map[string]any{
				"type":        "string",
				"description": "Optional cron schedule for recurring tasks (e.g., '0 9 * * *' for daily at 9am)",
			},
		},
		"required": []string{"action"},
	}
	return tool
}

func (t *TaskTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	mgr := tasks.GetManager()
	if mgr == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	switch action {
	case "create":
		return t.createTask(params)
	case "list":
		return t.listTasks(params)
	case "status":
		return t.getTaskStatus(params)
	case "pause":
		return t.pauseTask(params)
	case "resume":
		return t.resumeTask(params)
	case "cancel":
		return t.cancelTask(params)
	case "progress":
		return t.getTaskProgress(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *TaskTool) createTask(params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required for create action")
	}

	description := params["description"]
	if description == "" {
		return "", fmt.Errorf("description is required for create action")
	}

	schedule := params["schedule"]
	agentName := params["agent"]
	if agentName == "" {
		agentName = "default"
	}
	ownerID := "system"

	task, err := tasks.GetManager().CreateTask(name, description, ownerID, agentName, schedule)
	if err != nil {
		return "", err
	}

	// Automatically start the task
	if err := tasks.GetManager().ExecuteTask(task.ID); err != nil {
		return "", fmt.Errorf("created task but failed to start: %w", err)
	}

	return fmt.Sprintf("Task created and started:\nID: %s\nName: %s\nDescription: %s\nAgent: %s", task.ID, task.Name, task.Description, task.AgentName), nil
}

func (t *TaskTool) listTasks(params map[string]string) (string, error) {
	allTasks, err := tasks.GetManager().ListTasks(false)
	if err != nil {
		return "", err
	}

	if len(allTasks) == 0 {
		return "No tasks found", nil
	}

	result := fmt.Sprintf("Found %d tasks:\n\n", len(allTasks))
	for _, task := range allTasks {
		result += fmt.Sprintf("- %s (ID: %s)\n", task.Name, task.ID)
		result += fmt.Sprintf("  Status: %s\n", task.Status)
		result += fmt.Sprintf("  Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		if task.StartedAt != nil {
			result += fmt.Sprintf("  Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if task.CompletedAt != nil {
			result += fmt.Sprintf("  Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
		}
		if task.Schedule != "" {
			result += fmt.Sprintf("  Schedule: %s\n", task.Schedule)
		}
		result += "\n"
	}

	return result, nil
}

func (t *TaskTool) getTaskStatus(params map[string]string) (string, error) {
	taskID := params["task_id"]
	if taskID == "" {
		return "", fmt.Errorf("task_id is required for status action")
	}

	task, err := tasks.GetManager().GetTask(taskID)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Task: %s\n", task.Name)
	result += fmt.Sprintf("ID: %s\n", task.ID)
	result += fmt.Sprintf("Status: %s\n", task.Status)
	result += fmt.Sprintf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

	if task.StartedAt != nil {
		result += fmt.Sprintf("Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
	}

	if task.CompletedAt != nil {
		result += fmt.Sprintf("Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if task.Schedule != "" {
		result += fmt.Sprintf("Schedule: %s\n", task.Schedule)
		if task.LastRun != nil {
			result += fmt.Sprintf("Last run: %s\n", task.LastRun.Format("2006-01-02 15:04:05"))
		}
		if task.NextRun != nil {
			result += fmt.Sprintf("Next run: %s\n", task.NextRun.Format("2006-01-02 15:04:05"))
		}
	}

	return result, nil
}

func (t *TaskTool) pauseTask(params map[string]string) (string, error) {
	taskID := params["task_id"]
	if taskID == "" {
		return "", fmt.Errorf("task_id is required for pause action")
	}

	err := tasks.GetManager().PauseTask(taskID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Task %s paused", taskID), nil
}

func (t *TaskTool) resumeTask(params map[string]string) (string, error) {
	taskID := params["task_id"]
	if taskID == "" {
		return "", fmt.Errorf("task_id is required for resume action")
	}

	err := tasks.GetManager().ResumeTask(taskID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Task %s resumed", taskID), nil
}

func (t *TaskTool) cancelTask(params map[string]string) (string, error) {
	taskID := params["task_id"]
	if taskID == "" {
		return "", fmt.Errorf("task_id is required for cancel action")
	}

	err := tasks.GetManager().CancelTask(taskID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Task %s cancelled", taskID), nil
}

func (t *TaskTool) getTaskProgress(params map[string]string) (string, error) {
	taskID := params["task_id"]
	if taskID == "" {
		return "", fmt.Errorf("task_id is required for progress action")
	}

	progress, err := tasks.GetManager().GetTaskProgress(taskID)
	if err != nil {
		return "", err
	}

	return progress, nil
}
