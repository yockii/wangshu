package system

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/yockii/yoclaw/internal/types"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type CronTool struct {
	basic.SimpleTool
}

func NewCronTool() *CronTool {
	tool := new(CronTool)
	tool.Name_ = constant.ToolNameCron
	tool.Desc_ = "Manage scheduled tasks that are stored in the agent workspace and persist across restarts. Supports add, list, pause, resume, disable, status, and update operations."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: add, list, pause, resume, disable, status, update",
				"enum": []string{
					constant.CronActionAdd,
					constant.CronActionList,
					constant.CronActionPause,
					constant.CronActionResume,
					constant.CronActionDisable,
					constant.CronActionStatus,
					constant.CronActionUpdate,
				},
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Job ID (required for pause, resume, disable, status, update)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Task description (required for add, optional for update)",
			},
			"schedule": map[string]any{
				"type":        "string",
				"description": "Cron schedule expression (e.g., '0 9 * * *' for daily at 9am, '*/5 * * * *' for every 5 minutes). Supports standard cron format with 6 fields (seconds, minutes, hours, day of month, month, day of week). Can be updated.",
			},
			"once": map[string]any{
				"type":        "boolean",
				"description": "Whether the task should only be executed once (optional, defaults to false). If true, the task will be deleted after success execution.",
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
	case constant.CronActionAdd:
		return t.addTask(params)
	case constant.CronActionList:
		return t.listTasks(params)
	case constant.CronActionPause:
		return t.pauseTask(params)
	case constant.CronActionResume:
		return t.resumeTask(params)
	case constant.CronActionDisable:
		return t.disableTask(params)
	case constant.CronActionStatus:
		return t.getTaskStatus(params)
	case constant.CronActionUpdate:
		return t.updateTask(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *CronTool) addTask(params map[string]string) (string, error) {
	schedule := params["schedule"]
	if schedule == "" {
		return "", fmt.Errorf("schedule is required for add action")
	}

	description := params["description"]
	workspace := params[constant.ToolCallParamWorkspace]
	channel := params[constant.ToolCallParamChannel]
	chatID := params[constant.ToolCallParamChatID]
	once := params["once"] == "true" || params["once"] == "1"

	jobInfo := &types.BasicJobInfo{
		ID:          uuid.NewString(),
		Schedule:    schedule,
		Description: description,
		Status:      constant.CronStatusEnabled,

		Channel: channel,
		ChatID:  chatID,
		Once:    once,
	}

	// 写入workspace/cron/{id}.json文件中
	cronDir := filepath.Join(workspace, constant.DirCron)
	os.MkdirAll(cronDir, 0755)
	jobJsonPath := filepath.Join(cronDir, jobInfo.ID+constant.ExtJSON)
	data, err := json.Marshal(jobInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write job file: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务已创建\n任务ID: %s",
		jobInfo.ID), nil
}

func (t *CronTool) listTasks(params map[string]string) (string, error) {
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)
	entries, err := os.ReadDir(cronDir)
	if err != nil {
		return "", fmt.Errorf("failed to read cron directory: %w", err)
	}
	var jobs []types.BasicJobInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != constant.ExtJSON {
			continue
		}
		jobJsonPath := filepath.Join(cronDir, entry.Name())
		data, err := os.ReadFile(jobJsonPath)
		if err != nil {
			slog.Warn("Failed to read job", "jobFile", jobJsonPath)
			continue
		}
		job := types.BasicJobInfo{}
		if err := json.Unmarshal(data, &job); err != nil {
			slog.Warn("Failed to unmarshal job", "jobFile", jobJsonPath)
			continue
		}
		if job.ID == "" {
			slog.Warn("Job ID is empty", "jobFile", jobJsonPath)
			continue
		}
		if job.Status != constant.CronStatusDisabled {
			jobs = append(jobs, job)
		}
	}

	if len(jobs) == 0 {
		return "暂无定时任务", nil
	}

	result := fmt.Sprintf("定时任务列表 (%d 个):\n\n", len(jobs))
	for _, job := range jobs {
		status := "❌ 已禁用"
		switch job.Status {
		case constant.CronStatusEnabled:
			status = "✅ 已启用"
		case constant.CronStatusPaused:
			status = "⏸ 已暂停"
		}

		result += fmt.Sprintf("- %s\n", job.ID)
		result += fmt.Sprintf("  状态: %s\n", status)
		result += fmt.Sprintf("  调度: %s\n", job.Schedule)
		if job.Description != "" {
			result += fmt.Sprintf("  描述: %s\n", job.Description)
		}
		if job.NextRun != nil {
			result += fmt.Sprintf("  下次执行: %s\n", job.NextRun.Format("2006-01-02 15:04:05"))
		}
		if job.LastRun != nil {
			result += fmt.Sprintf("  上次执行: %s\n", job.LastRun.Format("2006-01-02 15:04:05"))
		}
		result += "\n"
	}

	return result, nil
}

func (t *CronTool) updateTask(params map[string]string) (string, error) {
	id := params["id"]
	if id == "" {
		return "", fmt.Errorf("id is required for update action")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)

	jobJsonPath := filepath.Join(cronDir, id+constant.ExtJSON)
	data, err := os.ReadFile(jobJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read job file: %w", err)
	}
	job := types.BasicJobInfo{}
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("failed to unmarshal job: %w", err)
	}
	if job.ID == "" {
		return "", fmt.Errorf("job ID is empty")
	}

	if schedule, ok := params["schedule"]; ok && schedule != "" {
		job.Schedule = schedule
	}
	if description, ok := params["description"]; ok && description != "" {
		job.Description = description
	}
	if once, ok := params["once"]; ok {
		job.Once = once == "true" || once == "1"
	}

	data, err = json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write job file: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已更新", id), nil
}

func (t *CronTool) pauseTask(params map[string]string) (string, error) {
	id := params["id"]
	if id == "" {
		return "", fmt.Errorf("id is required for pause action")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)

	jobJsonPath := filepath.Join(cronDir, id+constant.ExtJSON)
	data, err := os.ReadFile(jobJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read job file: %w", err)
	}
	job := types.BasicJobInfo{}
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("failed to unmarshal job: %w", err)
	}
	if job.ID == "" {
		return "", fmt.Errorf("job ID is empty")
	}

	if job.Status != constant.CronStatusEnabled {
		return "", fmt.Errorf("task '%s' status is '%s'", id, job.Status)
	}

	job.Status = constant.CronStatusPaused
	data, err = json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write job file: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已暂停", id), nil
}

func (t *CronTool) resumeTask(params map[string]string) (string, error) {
	id := params["id"]
	if id == "" {
		return "", fmt.Errorf("id is required for pause action")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)

	jobJsonPath := filepath.Join(cronDir, id+constant.ExtJSON)
	data, err := os.ReadFile(jobJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read job file: %w", err)
	}
	job := types.BasicJobInfo{}
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("failed to unmarshal job: %w", err)
	}
	if job.ID == "" {
		return "", fmt.Errorf("job ID is empty")
	}

	if job.Status != constant.CronStatusPaused {
		return "", fmt.Errorf("task '%s' status is '%s'", id, job.Status)
	}

	job.Status = constant.CronStatusEnabled
	data, err = json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write job file: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已启用", id), nil
}

func (t *CronTool) disableTask(params map[string]string) (string, error) {
	id := params["id"]
	if id == "" {
		return "", fmt.Errorf("id is required for disable action")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)

	jobJsonPath := filepath.Join(cronDir, id+constant.ExtJSON)
	data, err := os.ReadFile(jobJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read job file: %w", err)
	}
	job := types.BasicJobInfo{}
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("failed to unmarshal job: %w", err)
	}
	if job.ID == "" {
		return "", fmt.Errorf("job ID is empty")
	}
	job.Status = constant.CronStatusDisabled

	data, err = json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write job file: %w", err)
	}

	return fmt.Sprintf("✅ 定时任务 '%s' 已禁用", id), nil
}

func (t *CronTool) getTaskStatus(params map[string]string) (string, error) {
	id := params["id"]
	if id == "" {
		return "", fmt.Errorf("id is required for status action")
	}
	workspace := params[constant.ToolCallParamWorkspace]
	cronDir := filepath.Join(workspace, constant.DirCron)

	jobJsonPath := filepath.Join(cronDir, id+constant.ExtJSON)
	data, err := os.ReadFile(jobJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read job file: %w", err)
	}
	job := types.BasicJobInfo{}
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("failed to unmarshal job: %w", err)
	}
	if job.ID == "" {
		return "", fmt.Errorf("job ID is empty")
	}

	result := fmt.Sprintf("定时任务状态: %s\n\n", job.ID)
	result += "状态: "
	switch job.Status {
	case constant.CronStatusEnabled:
		result += "✅ 已启用\n"
	case constant.CronStatusPaused:
		result += "⚠️ 已暂停\n"
	case constant.CronStatusDisabled:
		result += "⛔ 已禁用\n"
	default:
		result += fmt.Sprintf("未知状态: %s\n", job.Status)
	}

	return result, nil
}
