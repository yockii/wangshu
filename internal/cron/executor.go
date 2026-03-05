package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
)

// CronJobExecutionResult 定时任务执行结果（JSON Schema）
type CronJobExecutionResult struct {
	Type            string `json:"type" jsonschema:"type=string,enum=message,task,required"`
	MessageContent  string `json:"messageContent,omitempty" jsonschema:"type=string,description=要发送给用户的消息内容"`
	TaskName        string `json:"taskName,omitempty" jsonschema:"type=string,description=任务名称"`
	TaskDescription string `json:"taskDescription,omitempty" jsonschema:"type=string,description=任务描述"`
	TaskPriority    string `json:"taskPriority,omitempty" jsonschema:"type=string,enum=low,normal,high,description=任务优先级"`
}

// Execute 执行定时任务
func (mgr *CronManager) Execute(ctx context.Context, job *types.BasicJobInfo) error {
	slog.Debug("执行定时任务", "jobID", job.ID)
	// 1. 准备消息
	messages := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: constant.CronJobExecutionPrompt,
		},
		{
			Role:    constant.RoleUser,
			Content: fmt.Sprintf("任务ID: %s\n任务描述: %s", job.ID, job.Description),
		},
	}

	// 2. 使用JSON Schema调用LLM
	schema := &llm.JSONSchema{
		Name: "CronJobExecutionResult",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":     "string",
					"enum":     []string{"message", "task"},
					"required": true,
				},
				"messageContent": map[string]any{
					"type":        "string",
					"description": "要发送给用户的消息内容",
				},
				"taskName": map[string]any{
					"type":        "string",
					"description": "任务名称",
				},
				"taskDescription": map[string]any{
					"type":        "string",
					"description": "任务描述",
				},
				"taskPriority": map[string]any{
					"type":        "string",
					"enum":        []string{"low", "normal", "high"},
					"description": "任务优先级",
				},
			},
			"required": []string{"type"},
		},
		Strict: true,
	}

	resp, err := mgr.provider.ChatWithJSONSchema(ctx, mgr.model, messages, schema, nil)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 3. 解析JSON响应
	var result CronJobExecutionResult
	if err := json.Unmarshal([]byte(resp.Message.Content), &result); err != nil {
		return fmt.Errorf("解析LLM响应失败: %w", err)
	}

	slog.Info("定时任务执行结果", "jobId", job.ID, "type", result.Type)

	// 4. 根据type调用相应的工具
	switch result.Type {
	case "message":
		return mgr.executeTool(ctx, job.Channel, job.ChatID, "message", map[string]string{
			"content": result.MessageContent,
		})

	case "task":
		priority := result.TaskPriority
		if priority == "" {
			priority = "normal"
		}
		return mgr.executeTool(ctx, job.Channel, job.ChatID, "task", map[string]string{
			"name":        result.TaskName,
			"description": result.TaskDescription,
			"priority":    priority,
		})

	default:
		return fmt.Errorf("未知的type: %s", result.Type)
	}
}

// executeTool 执行工具调用
func (mgr *CronManager) executeTool(ctx context.Context, channel, chatID, toolName string, params map[string]string) error {
	// 构造工具参数（需要转换为map[string]any）
	args := make(map[string]any)
	for k, v := range params {
		args[k] = v
	}
	args[constant.ToolCallParamWorkspace] = mgr.workspace
	args[constant.ToolCallParamChannel] = channel
	args[constant.ToolCallParamChatID] = chatID

	// 创建ToolContext
	toolCtx := tools.NewToolContext(
		"cron", // agentName - 使用cron表示是定时任务触发
		"",     // agent owner
		mgr.workspace,
		"", // sessionID
		channel,
		chatID,
		mgr.provider,
		mgr.model,
	)

	// 执行工具
	result := tools.GetDefaultToolRegistry().ExecuteWithContext(ctx, toolName, args, toolCtx, channel, chatID)
	if result.IsError {
		return fmt.Errorf("工具执行失败: %v", result.Err)
	}

	slog.Info("定时任务工具执行完成", "jobChannel", channel, "jobChatID", chatID, "tool", toolName)
	return nil
}
