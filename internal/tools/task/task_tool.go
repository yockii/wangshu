package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

const (
	ToolNameTask = "task"
)

type ChangeLogEntry struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"` // 用户提出的变更内容
	Timestamp  time.Time `json:"timestamp"`
	Notified   bool      `json:"notified"`
	NotifiedAt time.Time `json:"notified_at,omitempty"`
}

type ChangeLog struct {
	Entries []*ChangeLogEntry `json:"entries"`
}
type TaskRelation struct {
	ParentID string `json:"parent_id,omitempty"`
	RootID   string `json:"root_id"`
}

type SubtaskInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary,omitempty"` // 子任务完成后的总结
	UpdatedAt time.Time `json:"updated_at"`
}

type SubtasksRecord struct {
	Subtasks  map[string]*SubtaskInfo `json:"subtasks"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type TaskRelations struct {
	Relations map[string]*TaskRelation `json:"relations"`
}

type TaskInfo struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id"`
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
	tool.Name_ = ToolNameTask
	tool.Desc_ = "Create and manage asynchronous tasks that execute in the background without blocking the current conversation loop. Unlike synchronous tool calls that wait for completion, tasks created here run independently and do not occupy the current session context. This is ideal for long-running operations, scheduled tasks triggered by cron, or any work that should proceed without keeping the user waiting. Tasks persist in the agent's workspace and their status/results can be queried later."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform on the task",
				"enum":        []string{"create", "list", "status", "cancel", "clean", "restart", "update", "add_change"},
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for status, cancel, clean, restart, update and add_change actions). For clean action without id, all completed/failed/cancelled tasks will be cleaned.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Task name/identifier (required for create action, optional for update action)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed task description including objectives, instructions, or commands to execute (required for create action, optional for update action)",
			},
			"parent_id": map[string]any{
				"type":        "string",
				"description": "Parent task ID (optional for create action). If provided, creates a subtask under the parent task.",
			},
			"change": map[string]any{
				"type":        "string",
				"description": "Change description (required for add_change action)",
			},
			"priority": map[string]any{
				"type":        "string",
				"description": "Task execution priority (optional, defaults to 'normal'). Higher priority tasks are processed first. Can be updated.",
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
	case "update":
		return t.update(params)
	case "add_change":
		return t.addChange(params)
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
	parentID, ok := params["parent_id"]
	if !ok {
		parentID = ""
	}

	workspace := params[constant.ToolCallParamWorkspace]
	channel := params[constant.ToolCallParamChannel]
	chatID := params[constant.ToolCallParamChatID]
	now := time.Now()
	at := &TaskInfo{
		ID:          uuid.NewString(),
		ParentID:    parentID,
		Name:        name,
		Description: description,
		Priority:    priority,
		Status:      "pending",

		Channel: channel,
		ChatID:  chatID,
	}

	taskRelationsFilePath := filepath.Join(workspace, "tasks", constant.TaskRelationsFileName)
	var trs TaskRelations = TaskRelations{
		Relations: map[string]*TaskRelation{},
	}
	data, err := os.ReadFile(taskRelationsFilePath)
	if err == nil {
		if err = json.Unmarshal(data, &trs); err != nil {
			return "", fmt.Errorf("failed to unmarshal task relations: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read task relations file: %w", err)
	}

	parentDir := t.getParentDir(at.ParentID, trs.Relations)
	taskDir := filepath.Join(workspace, "tasks", parentDir, at.ID)

	stRecord := &SubtasksRecord{
		Subtasks: map[string]*SubtaskInfo{},
	}
	subtasksFilePath := filepath.Join(workspace, "tasks", parentDir, constant.TaskSubtasksInfoFileName)
	if at.ParentID != "" {
		if _, ok := trs.Relations[at.ParentID]; !ok {
			return "", fmt.Errorf("parent task %s not found", at.ParentID)
		}
		if _, err = os.Stat(subtasksFilePath); err == nil {
			data, err = os.ReadFile(subtasksFilePath)
			if err != nil {
				return "", fmt.Errorf("failed to read subtasks file: %w", err)
			}
			if err = json.Unmarshal(data, &stRecord); err != nil {
				return "", fmt.Errorf("failed to unmarshal subtasks: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to read subtasks file: %w", err)
		}
	}

	// 写入对应文件
	if err = os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create task directory: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
	data, err = json.Marshal(at)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}
	if err = os.WriteFile(taskFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %w", err)
	}

	// 如果是子任务，还需要写入subtasks.json文件
	if at.ParentID != "" {
		stRecord.Subtasks[at.ID] = &SubtaskInfo{
			ID:        at.ID,
			Name:      at.Name,
			Status:    "pending",
			UpdatedAt: now,
		}
		stRecord.UpdatedAt = now
		data, err = json.Marshal(stRecord)
		if err != nil {
			return "", fmt.Errorf("failed to marshal subtasks: %w", err)
		}
		if err = os.WriteFile(subtasksFilePath, data, 0644); err != nil {
			return "", fmt.Errorf("failed to write subtasks file: %w", err)
		}
	}

	// 写入task关系文件
	rootTaskID := ""
	if at.ParentID != "" {
		rootTaskID = t.getRootTaskID(at.ParentID, trs.Relations)
	}
	trs.Relations[at.ID] = &TaskRelation{
		ParentID: at.ParentID,
		RootID:   rootTaskID,
	}
	data, err = json.Marshal(trs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task relations: %w", err)
	}
	if err := os.WriteFile(taskRelationsFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task relations file: %w", err)
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
			taskFilePath := filepath.Join(taskDir, file.Name(), constant.TaskInfoFileName)
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
	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
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
	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
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

	// 删除所有子任务文件夹
	subTasksDir, err := os.ReadDir(taskDir)
	if err != nil {
		return "", fmt.Errorf("failed to read sub tasks directory: %w", err)
	}
	for _, file := range subTasksDir {
		if file.IsDir() {
			subTaskDir := filepath.Join(taskDir, file.Name())
			if err := os.RemoveAll(subTaskDir); err != nil {
				return "", fmt.Errorf("failed to remove sub task directory: %w", err)
			}
		}
	}

	// 从task关系文件中删除该任务的所有子关系
	taskRelationsFilePath := filepath.Join(workspace, "tasks", constant.TaskRelationsFileName)
	data, err = os.ReadFile(taskRelationsFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read task relations file: %w", err)
	}
	var trs TaskRelations
	if err = json.Unmarshal(data, &trs); err != nil {
		return "", fmt.Errorf("failed to unmarshal task relations: %w", err)
	}

	for stID, st := range trs.Relations {
		if st.RootID == id || stID == id {
			delete(trs.Relations, stID)
		}
	}

	data, err = json.Marshal(trs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task relations: %w", err)
	}
	if err := os.WriteFile(taskRelationsFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task relations file: %w", err)
	}

	return fmt.Sprintf("Task Cancelled: %s\nLast Result: %s", at.ID, at.LastResult), nil

}

func (t *TaskTool) clean(params map[string]string) (string, error) {
	id, _ := params["id"]

	workspace := params[constant.ToolCallParamWorkspace]
	tasksDir := filepath.Join(workspace, "tasks")

	if id != "" {
		taskFilePath := filepath.Join(tasksDir, id, constant.TaskInfoFileName)
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
			taskFilePath := filepath.Join(tasksDir, file.Name(), constant.TaskInfoFileName)
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
	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
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

func (t *TaskTool) update(params map[string]string) (string, error) {
	id, ok := params["id"]
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	taskDir := filepath.Join(workspace, "tasks", id)
	if _, err := os.Stat(taskDir); err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}
	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
	data, err := os.ReadFile(taskFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read task file: %w", err)
	}
	var at TaskInfo
	if err := json.Unmarshal(data, &at); err != nil {
		return "", fmt.Errorf("failed to unmarshal task: %w", err)
	}

	if name, ok := params["name"]; ok && name != "" {
		at.Name = name
	}
	if description, ok := params["description"]; ok && description != "" {
		at.Description = description
	}
	if priority, ok := params["priority"]; ok && priority != "" {
		at.Priority = priority
	}

	data, err = json.Marshal(at)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}
	if err := os.WriteFile(taskFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %w", err)
	}

	return fmt.Sprintf("Task Updated: %s", at.ID), nil
}

func (t *TaskTool) addChange(params map[string]string) (string, error) {
	id, ok := params["id"]
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}
	change, ok := params["change"]
	if !ok || change == "" {
		return "", fmt.Errorf("change parameter is required")
	}

	workspace := params[constant.ToolCallParamWorkspace]

	tasksRelationFile := filepath.Join(workspace, "tasks", constant.TaskRelationsFileName)
	if _, err := os.Stat(tasksRelationFile); err != nil {
		return "", fmt.Errorf("task relations file not found: %w", err)
	}
	data, err := os.ReadFile(tasksRelationFile)
	if err != nil {
		return "", fmt.Errorf("failed to read task relations file: %w", err)
	}
	var trs TaskRelations
	if err := json.Unmarshal(data, &trs); err != nil {
		return "", fmt.Errorf("failed to unmarshal task relations: %w", err)
	}

	if trs.Relations[id].ParentID != "" {
		return "", fmt.Errorf("task %s is not a root task", id)
	}

	taskDir := filepath.Join(workspace, "tasks", id)
	if _, err := os.Stat(taskDir); err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}

	changeLogFilePath := filepath.Join(taskDir, constant.TaskChangeLogFileName)
	var cl ChangeLog = ChangeLog{
		Entries: []*ChangeLogEntry{},
	}
	data, err = os.ReadFile(changeLogFilePath)
	if err == nil {
		if err = json.Unmarshal(data, &cl); err != nil {
			return "", fmt.Errorf("failed to unmarshal change log: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read change log file: %w", err)
	}
	cl.Entries = append(cl.Entries, &ChangeLogEntry{
		ID:        uuid.New().String(),
		Content:   change,
		Timestamp: time.Now(),
	})
	data, err = json.Marshal(cl)
	if err != nil {
		return "", fmt.Errorf("failed to marshal change log: %w", err)
	}
	if err := os.WriteFile(changeLogFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write change log file: %w", err)
	}
	return fmt.Sprintf("Change Added: %s", change), nil
}

func (t *TaskTool) getParentDir(parentID string, relations map[string]*TaskRelation) string {
	// 反向拼装父目录
	if parentID == "" {
		return ""
	}
	if rel, ok := relations[parentID]; ok {
		if rel.ParentID == "" {
			return parentID
		}
		return filepath.Join(t.getParentDir(rel.ParentID, relations), parentID)
	}
	return ""
}

func (t *TaskTool) getRootTaskID(taskID string, relations map[string]*TaskRelation) string {
	if taskID == "" {
		return ""
	}
	if rel, ok := relations[taskID]; ok {
		if rel.ParentID == "" {
			return taskID
		}
		return t.getRootTaskID(rel.ParentID, relations)
	}
	return ""
}
