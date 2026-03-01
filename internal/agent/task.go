package agent

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yockii/yoclaw/internal/session"
	"github.com/yockii/yoclaw/internal/tools/task"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/skills"
	"github.com/yockii/yoclaw/pkg/tools"
)

func (a *Agent) startTaskLoop() {
	for range a.taskTicker.C {
		a.processTask()
	}
}

func (a *Agent) processTask() {
	tasksDir := filepath.Join(a.workspaceDir, "tasks")
	os.MkdirAll(tasksDir, 0755)
	taskFiles, err := os.ReadDir(tasksDir)
	if err != nil {
		slog.Error("Failed to read tasks directory", "agent", a.agentName, "error", err)
		return
	}
	var dealTask *task.TaskInfo
	for _, taskFile := range taskFiles {
		if !taskFile.IsDir() {
			continue
		}
		taskFilePath := filepath.Join(tasksDir, taskFile.Name(), "task.json")
		if _, err := os.Stat(taskFilePath); err != nil {
			slog.Error("Failed to stat task file", "task", taskFile.Name(), "error", err)
			continue
		}

		data, err := os.ReadFile(taskFilePath)
		if err != nil {
			slog.Error("Failed to read task file", "task", taskFile.Name(), "error", err)
			continue
		}
		var taskInfo task.TaskInfo
		err = json.Unmarshal(data, &taskInfo)
		if err != nil {
			slog.Error("Failed to unmarshal task file", "task", taskFile.Name(), "error", err)
			continue
		}

		if taskInfo.Status == "remove" {
			// 删除该任务目录
			err := os.RemoveAll(filepath.Join(tasksDir, taskFile.Name()))
			if err != nil {
				slog.Error("Failed to remove task directory", "task", taskFile.Name(), "error", err)
			}
			continue
		} else if taskInfo.Status == "completed" || taskInfo.Status == "failed" || taskInfo.Status == "cancelled" {
			continue
		}

		if dealTask == nil {
			dealTask = &taskInfo
			if dealTask.Priority == "urgent" {
				break
			}
		} else {
			if taskInfo.Priority == "urgent" {
				dealTask = &taskInfo
				break
			}
			if taskInfo.Priority == "high" && (dealTask.Priority == "normal" || dealTask.Priority == "low") {
				dealTask = &taskInfo
			} else if taskInfo.Priority == "normal" && dealTask.Priority == "low" {
				dealTask = &taskInfo
			}
		}
	}

	if dealTask != nil {
		lastMsg, finished, err := a.doTask(dealTask)
		dealTask.Status = "running"
		if err != nil {
			dealTask.Status = "failed"
			dealTask.LastResult = err.Error()
		} else {
			dealTask.LastResult = lastMsg
		}
		if finished {
			dealTask.Status = "completed"
		}
		taskFilePath := filepath.Join(a.workspaceDir, "tasks", dealTask.ID, "task.json")
		data, err := json.Marshal(dealTask)
		if err != nil {
			slog.Error("Failed to marshal task file", "task", dealTask.ID, "error", err)
			return
		}
		err = os.WriteFile(taskFilePath, data, 0644)
		if err != nil {
			slog.Error("Failed to write task file", "task", dealTask.ID, "error", err)
			return
		}
		switch dealTask.Status {
		case "completed":
			if lastMsg != "TASK_COMPLETED" {
				// 发送任务完成消息
				bus.Default().PublishOutbound(bus.OutboundMessage{
					Channel: dealTask.Channel,
					ChatID:  dealTask.ChatID,
					Content: lastMsg,
				})
			}
			// 任务完成，让大模型总结任务信息并记入profile/memory/YYYY-MM-DD-{slug}.md
			a.summaryTask(dealTask)
		case "failed":
			// 任务失败，通知用户
			sess := a.sessions.GetOrCreate(a.workspaceDir, fmt.Sprintf("%s:%s", dealTask.Channel, dealTask.ChatID), dealTask.Channel, dealTask.ChatID, "")
			sess.AddMessage(constant.RoleAssistant, fmt.Sprintf("任务[%s] %s 执行失败: %s", dealTask.ID, dealTask.Name, dealTask.LastResult))

			bus.Default().PublishOutbound(bus.OutboundMessage{
				Channel: dealTask.Channel,
				ChatID:  dealTask.ChatID,
				Content: fmt.Sprintf("任务[%s] %s 执行失败: %s", dealTask.ID, dealTask.Name, dealTask.LastResult),
			})
		}
	}
}

func (a *Agent) doTask(taskInfo *task.TaskInfo) (string, bool, error) {
	result := ""

	// 技能元数据加载
	skillList, err := skills.GetDefaultLoader().LoadSkills()
	if err != nil {
		slog.Error("Failed to load skills in running task", "agent", a.agentName, "error", err)
		return "", false, err
	}
	// 将skills转为xml字符串
	parent := SkillsParent{
		SkillList: skillList,
	}
	skillsXML, err := xml.Marshal(parent)
	if err != nil {
		slog.Error("Failed to marshal skills to xml in running task", "agent", a.agentName, "error", err)
		return "", false, err
	}

	runtimeInfo := fmt.Sprintf(
		"操作系统: %s, CPU架构: %s, 当前时间: %s",
		runtime.GOOS, runtime.GOARCH, time.Now().Local().Format(time.RFC3339),
	)

	msgs := []llm.Message{
		{
			Role: constant.RoleSystem,
			Content: fmt.Sprintf(TaskExecutionPrompt,
				skillsXML,
				a.workspaceDir,
				runtimeInfo,
			),
		},
		{
			Role: constant.RoleUser,
			Content: fmt.Sprintf(`## 任务信息
任务ID: %s
任务名称: %s
任务描述: %s
优先级: %s
`,
				taskInfo.ID,
				taskInfo.Name,
				taskInfo.Description,
				taskInfo.Priority,
			),
		},
	}

	// 读取之前的消息信息
	history, err := a.readTaskHistory(taskInfo)
	if err != nil {
		return "", false, err
	}
	msgs = append(msgs, history...)

	ctx := context.Background()
	availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
	resp, err := a.provider.Chat(ctx, a.model, msgs, availableTools, nil)
	if err != nil {
		slog.Error("Failed to chat in running task", "agent", a.agentName, "error", err)
		return "", false, err
	}

	respMsg := llm.Message{
		Role:      constant.RoleAssistant,
		Content:   resp.Message.Content,
		ToolCalls: resp.Message.ToolCalls,
	}
	result = resp.Message.Content
	a.appendTaskHistory(taskInfo, respMsg)

	msgs = append(msgs, respMsg)

	isFinished := true

	for _, tc := range resp.Message.ToolCalls {
		isFinished = false
		toolResult, err := a.executeToolCall(ctx, tc, taskInfo.Channel, taskInfo.ChatID)
		if err != nil {
			toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
		}
		result += "\n" + toolResult
		toolMsg := llm.Message{
			Role:       constant.RoleTool,
			Content:    toolResult,
			ToolCallID: tc.ID,
		}
		a.appendTaskHistory(taskInfo, toolMsg)
		msgs = append(msgs, toolMsg)
	}

	if isFinished {
		// 再次检查完成标记，确保不会提前完成任务
		if !strings.Contains(result, "TASK_COMPLETED") {
			isFinished = false
			userMsg := llm.Message{
				Role:    constant.RoleUser,
				Content: "任务尚未完成，你必须调用工具继续执行任务。如果认为任务已完成，请输出`TASK_COMPLETED`。",
			}
			a.appendTaskHistory(taskInfo, userMsg)
			msgs = append(msgs, userMsg)
		}
	}

	return result, isFinished, nil
}

func (a *Agent) readTaskHistory(taskInfo *task.TaskInfo) ([]llm.Message, error) {
	result := make([]llm.Message, 0)
	taskHistoryFile := filepath.Join(a.workspaceDir, "tasks", taskInfo.ID, "history.jsonl")
	if _, err := os.Stat(taskHistoryFile); err == nil {
		f, err := os.Open(taskHistoryFile)
		if err != nil {
			slog.Error("Failed to open task history file", "error", err)
			return result, err
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		for {
			var msg session.Message
			if err := decoder.Decode(&msg); err != nil {
				if err.Error() == "EOF" {
					break
				}
				slog.Warn("task history decode failed", "error", err)
				continue
			}
			var tcList []llm.ToolCall
			for _, tc := range msg.ToolCalls {
				tcList = append(tcList, llm.ToolCall{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
			result = append(result, llm.Message{
				Role:      msg.Role,
				Content:   msg.Content,
				ToolCalls: tcList,
			})
		}
	}
	return result, nil
}

func (a *Agent) appendTaskHistory(taskInfo *task.TaskInfo, msg llm.Message) {
	taskHistoryFile := filepath.Join(a.workspaceDir, "tasks", taskInfo.ID, "history.jsonl")

	os.MkdirAll(filepath.Dir(taskHistoryFile), 0755)

	f, err := os.OpenFile(taskHistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open task history file", "error", err)
		return
	}
	defer f.Close()

	var tcs []session.ToolCall
	for _, tc := range msg.ToolCalls {
		tcs = append(tcs, session.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		})
	}

	sMsg := session.Message{
		Role:      msg.Role,
		Content:   msg.Content,
		ToolCalls: tcs,
		Timestamp: time.Now(),
	}

	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(sMsg); err != nil {
		slog.Error("Failed to write task history file", "error", err)
		return
	}
}

func (a *Agent) summaryTask(taskInfo *task.TaskInfo) {
	// 任务完成，让大模型总结任务信息并记入profile/memory/YYYY-MM-DD-{slug}.md
	history, err := a.readTaskHistory(taskInfo)
	if err != nil {
		slog.Error("Failed to read task history", "task", taskInfo.ID, "error", err)
		return
	}

	historyContent := ""
	for _, msg := range history {
		if msg.Role == constant.RoleTool {
			continue
		}
		historyContent += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	msgs := []llm.Message{
		{
			Role: constant.RoleSystem,
			Content: fmt.Sprintf(TaskSummaryPrompt,
				a.workspaceDir,
				filepath.Join(a.workspaceDir, "profile", "memory"),
			),
		},
		{
			Role: constant.RoleUser,
			Content: fmt.Sprintf(`## 要归档的任务信息
任务ID: %s
任务名称: %s
任务描述: %s
优先级: %s
任务执行历史记录:
%s
`,
				taskInfo.ID,
				taskInfo.Name,
				taskInfo.Description,
				taskInfo.Priority,
				historyContent,
			),
		},
	}
	msgs = append(msgs, history...)

	ctx := context.Background()
	availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
	for i := 0; i < 10; i++ {
		resp, err := a.provider.Chat(ctx, a.model, msgs, availableTools, nil)
		if err != nil {
			slog.Error("Failed to summary task memory", "error", err)
			return
		}
		if len(resp.Message.ToolCalls) == 0 {
			break
		}
		msgs = append(msgs, llm.Message{
			Role:      constant.RoleAssistant,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})
		// 执行所有的工具调用
		for _, tc := range resp.Message.ToolCalls {
			toolResult, err := a.executeToolCall(ctx, tc, taskInfo.Channel, taskInfo.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
			}

			msgs = append(msgs, llm.Message{
				Role:       constant.RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
			})
		}
	}
}
