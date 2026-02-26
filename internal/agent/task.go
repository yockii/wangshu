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
	"time"

	"github.com/yockii/yoclaw/internal/session"
	"github.com/yockii/yoclaw/internal/tools/task"
	"github.com/yockii/yoclaw/pkg/bus"
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
		if dealTask.Status == "completed" {
			// 发送任务完成消息
			bus.Default().PublishOutbound(bus.OutboundMessage{
				Channel: dealTask.Channel,
				ChatID:  dealTask.ChatID,
				Content: lastMsg,
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
			Role: "system",
			Content: fmt.Sprintf(TaskExecutionPrompt,
				skillsXML,
				a.workspaceDir,
				runtimeInfo,
			),
		},
		{
			Role: "user",
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
		Role:      "assistant",
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
		a.appendTaskHistory(taskInfo, llm.Message{
			Role:       "tool",
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
		msgs = append(msgs, llm.Message{
			Role:       "tool",
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
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
