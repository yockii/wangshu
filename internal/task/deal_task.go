package task

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

	"github.com/yockii/yoclaw/internal/tools/task"
	"github.com/yockii/yoclaw/internal/types"
	"github.com/yockii/yoclaw/pkg/bus"
	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/skills"
	"github.com/yockii/yoclaw/pkg/tools"
)

func (tm *TaskManager) run() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasksDir := filepath.Join(tm.workspace, constant.DirTasks)
	os.MkdirAll(tasksDir, 0755)

	taskFiles, err := os.ReadDir(tasksDir)
	if err != nil {
		slog.Error("Failed to read tasks directory", "error", err)
		return
	}
	var mainTask *task.TaskInfo
	for _, taskFile := range taskFiles {
		if !taskFile.IsDir() {
			continue
		}
		taskFilePath := filepath.Join(tasksDir, taskFile.Name(), constant.TaskInfoFileName)
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

		if taskInfo.Status == constant.TaskStatusRemove {
			// 删除该任务目录
			err = os.RemoveAll(filepath.Join(tasksDir, taskFile.Name()))
			if err != nil {
				slog.Error("Failed to remove task directory", "task", taskFile.Name(), "error", err)
			}
			// 清理关系记录
			taskRelationsFilePath := filepath.Join(tm.workspace, constant.DirTasks, constant.TaskRelationsFileName)
			relations := task.TaskRelations{}
			if _, err := os.Stat(taskRelationsFilePath); err == nil {
				data, err = os.ReadFile(taskRelationsFilePath)
				if err != nil {
					slog.Error("Failed to read task relations file", "error", err)
					continue
				}
				err = json.Unmarshal(data, &relations)
				if err != nil {
					slog.Error("Failed to unmarshal task relations file", "error", err)
					continue
				}
				// 删除子任务关系
				for key, r := range relations.Relations {
					if r.RootID == taskInfo.ID || key == taskInfo.ID {
						// 删除子任务关系
						delete(relations.Relations, key)
					}
				}
				// 更新关系文件
				data, err = json.Marshal(relations)
				if err != nil {
					slog.Error("Failed to marshal task relations file", "error", err)
					continue
				}
				err = os.WriteFile(taskRelationsFilePath, data, 0644)
				if err != nil {
					slog.Error("Failed to write task relations file", "error", err)
					continue
				}
			}

			continue
		} else if taskInfo.Status == constant.TaskStatusCompleted || taskInfo.Status == constant.TaskStatusCancelled || taskInfo.Status == constant.TaskStatusFailed {
			continue
		}

		if mainTask == nil {
			mainTask = &taskInfo
			if mainTask.Priority == constant.TaskPriorityUrgent {
				break
			}
		} else {
			if taskInfo.Priority == constant.TaskPriorityUrgent && mainTask.Priority != constant.TaskPriorityUrgent {
				mainTask = &taskInfo
				break
			}
			if taskInfo.Priority == constant.TaskPriorityHigh && (mainTask.Priority == constant.TaskPriorityNormal || mainTask.Priority == constant.TaskPriorityLow) {
				mainTask = &taskInfo
			} else if taskInfo.Priority == constant.TaskPriorityNormal && mainTask.Priority == constant.TaskPriorityLow {
				mainTask = &taskInfo
			}
		}
	}

	if mainTask != nil {
		resp, finished, err := tm.dealTask(mainTask, tasksDir)
		mainTask.LastResult = resp
		if err != nil {
			slog.Error("Failed to deal task", "task", mainTask.ID, "error", err)
			mainTask.LastResult += "\n" + err.Error()
		}
		// 更新任务文件
		data, err := json.Marshal(mainTask)
		if err != nil {
			slog.Error("Failed to marshal task file", "task", mainTask.ID, "error", err)
		}
		err = os.WriteFile(filepath.Join(tm.workspace, constant.DirTasks, mainTask.ID, constant.TaskInfoFileName), data, 0644)
		if err != nil {
			slog.Error("Failed to write task file", "task", mainTask.ID, "error", err)
		}

		if finished {
			if resp != constant.TaskTagCompleted {
				bus.Default().PublishOutbound(bus.OutboundMessage{
					Channel: mainTask.Channel,
					ChatID:  mainTask.ChatID,
					Content: resp,
				})
			}

			// 完成主任务，进行总结
			tm.summaryMainTask(mainTask)
		}
	}
}

func (tm *TaskManager) dealTask(taskInfo *task.TaskInfo, parentDir string) (resp string, finished bool, err error) {
	taskDir := filepath.Join(parentDir, taskInfo.ID)
	changeLogFile := filepath.Join(taskDir, constant.TaskChangeLogFileName)
	changeLog := task.ChangeLog{}
	if _, err := os.Stat(changeLogFile); err == nil {
		data, err := os.ReadFile(changeLogFile)
		if err != nil {
			slog.Error("Failed to read change log file", "task", taskInfo.ID, "error", err)
		} else {
			err = json.Unmarshal(data, &changeLog)
			if err != nil {
				slog.Error("Failed to unmarshal change log file", "task", taskInfo.ID, "error", err)
			}
		}
	}
	now := time.Now()
	var notifyContent strings.Builder
	for _, log := range changeLog.Entries {
		if !log.Notified {
			log.Notified = true
			log.NotifiedAt = now
			notifyContent.WriteString(log.Content + "\n")
		}
	}

	if notifyContent.String() != "" {
		// 进行任务内容变更处理
		err = tm.changeTask(taskInfo, notifyContent.String())
		if err != nil {
			slog.Error("Failed to change task", "task", taskInfo.ID, "error", err)
			return "", false, err
		}

		// 变更处理完成，则更新changelog
		data, err := json.Marshal(changeLog)
		if err != nil {
			slog.Error("Failed to marshal change log file", "task", taskInfo.ID, "error", err)
			return "", false, err
		}
		err = os.WriteFile(changeLogFile, data, 0644)
		if err != nil {
			slog.Error("Failed to write change log file", "task", taskInfo.ID, "error", err)
			return "", false, err
		}
	} else {
		// 执行任务, 如果失败，不做任何改变，只进行日志输出错误信息后，等待下次重试
		// 如果成功，则做相对应的处理
		//  处理当前任务，需要先看subtasks.json，里面有子任务未完成的，去处理，如果都完成了，整个所有子任务的完成反馈，作为本任务的下一个user消息发送给大模型处理

		// 检查当前任务目录下是否有subtasks.json
		subtasksInfo := task.SubtasksRecord{
			Subtasks: map[string]*task.SubtaskInfo{},
		}
		subtasksFilePath := filepath.Join(taskDir, constant.TaskSubtasksInfoFileName)
		if _, err = os.Stat(subtasksFilePath); err == nil {
			data, err := os.ReadFile(subtasksFilePath)
			if err != nil {
				slog.Error("Failed to read subtasks file", "task", taskInfo.ID, "error", err)
				return "", false, err
			}
			if err = json.Unmarshal(data, &subtasksInfo); err != nil {
				slog.Error("Failed to unmarshal subtasks", "task", taskInfo.ID, "error", err)
				return "", false, err
			}
		}

		var dealSubtask *task.SubtaskInfo
		for _, subtask := range subtasksInfo.Subtasks {
			if subtask.Status == constant.TaskStatusCompleted || subtask.Status == constant.TaskStatusCancelled || subtask.Status == constant.TaskStatusFailed {
				continue
			}
			dealSubtask = subtask
			break
		}
		if dealSubtask != nil {
			subtaskInfoFilePath := filepath.Join(taskDir, dealSubtask.ID, constant.TaskInfoFileName)
			subtaskInfo := task.TaskInfo{}
			subTaskData, err := os.ReadFile(subtaskInfoFilePath)
			if err != nil {
				slog.Error("Failed to read subtask file", "task", dealSubtask.ID, "error", err)
				return "", false, err
			}
			if err = json.Unmarshal(subTaskData, &subtaskInfo); err != nil {
				slog.Error("Failed to unmarshal subtask", "task", dealSubtask.ID, "error", err)
				return "", false, err
			}

			resp, finished, err = tm.dealTask(&subtaskInfo, taskDir)
			if err != nil {
				slog.Error("Failed to do subtask", "task", subtaskInfo.ID, "error", err)
				return "", false, err
			}
			return resp, finished, err
		} else {
			// 没有子任务，则执行自己
			resp, finished, err = tm.doTask(taskInfo, taskDir)
			if err != nil {
				slog.Error("Failed to do task", "task", taskInfo.ID, "error", err)
				taskInfo.LastResult = err.Error()
			} else {
				taskInfo.LastResult = resp
				if finished {
					taskInfo.Status = constant.TaskStatusCompleted
				}
			}
		}
	}

	taskFilePath := filepath.Join(taskDir, constant.TaskInfoFileName)
	data, err := json.Marshal(taskInfo)
	if err != nil {
		slog.Error("Failed to marshal task file", "task", taskInfo.ID, "error", err)
		return
	}
	err = os.WriteFile(taskFilePath, data, 0644)
	if err != nil {
		slog.Error("Failed to write task file", "task", taskInfo.ID, "error", err)
		return
	}

	if taskInfo.Status == constant.TaskStatusCompleted && taskInfo.ParentID != "" {
		// 任务总结
		resp, err = tm.summaryTask(taskInfo, taskDir)
		if err != nil {
			slog.Error("Failed to summary task memory", "task", taskInfo.ID, "error", err)
			return
		}

		// 更新父级目录下的subtasks.json
		stRecord := task.SubtasksRecord{
			Subtasks: map[string]*task.SubtaskInfo{},
		}
		subtasksFilePath := filepath.Join(parentDir, constant.TaskSubtasksInfoFileName)
		if _, err = os.Stat(subtasksFilePath); err == nil {
			data, err = os.ReadFile(subtasksFilePath)
			if err != nil {
				slog.Error("Failed to read subtasks file", "task", taskInfo.ParentID, "error", err)
				return
			}
			if err = json.Unmarshal(data, &stRecord); err != nil {
				slog.Error("Failed to unmarshal subtasks", "task", taskInfo.ParentID, "error", err)
				return
			}
		} else if !os.IsNotExist(err) {
			slog.Error("Failed to read subtasks file", "task", taskInfo.ParentID, "error", err)
			return
		}
		if subtask, ok := stRecord.Subtasks[taskInfo.ID]; ok {
			subtask.Status = constant.TaskStatusCompleted
			subtask.UpdatedAt = now
			subtask.Summary = resp
			// 写入文件
			data, err = json.Marshal(stRecord)
			if err != nil {
				slog.Error("Failed to marshal subtasks file", "task", taskInfo.ParentID, "error", err)
				return
			}
			err = os.WriteFile(subtasksFilePath, data, 0644)
			if err != nil {
				slog.Error("Failed to write subtasks file", "task", taskInfo.ParentID, "error", err)
				return
			}
		}
	}
	return
}

func (tm *TaskManager) changeTask(taskInfo *task.TaskInfo, notifyContent string) error {
	msgs := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: constant.TaskChangePrompt,
		},
		{
			Role: constant.RoleUser,
			Content: fmt.Sprintf(`## 要更新的任务信息
任务ID: %s
任务名称: %s
任务描述: %s
优先级: %s
变更内容:
%s
`, taskInfo.ID,
				taskInfo.Name,
				taskInfo.Description,
				taskInfo.Priority,
				notifyContent),
		},
	}
	ctx := context.Background()
	availableTools := tools.GetDefaultToolRegistry().GetSelectedToolsInProviderDefs(constant.ToolNameTask)
	for i := 0; i < 10; i++ {
		resp, err := tm.provider.Chat(ctx, tm.model, msgs, availableTools, nil)
		if err != nil {
			slog.Error("Failed to summary task memory", "error", err)
			return err
		}
		if len(resp.Message.ToolCalls) == 0 {
			break
		}
		msgs = append(msgs, llm.Message{
			Role:      constant.RoleAssistant,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})
		for _, tc := range resp.Message.ToolCalls {
			toolResult, err := tm.executeToolCall(ctx, tc, taskInfo.Channel, taskInfo.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
			}

			msgs = append(msgs, llm.Message{
				Role:      constant.RoleTool,
				Content:   toolResult,
				ToolCalls: []llm.ToolCall{tc},
			})
		}
	}

	return nil
}

func (tm *TaskManager) doTask(taskInfo *task.TaskInfo, taskDir string) (string, bool, error) {
	result := ""

	// 技能元数据加载
	skillList, err := skills.GetDefaultLoader().LoadSkills()
	if err != nil {
		slog.Error("Failed to load skills in running task", "task", taskInfo.ID, "error", err)
		return "", false, err
	}
	// 将skills转为xml字符串
	parent := types.SkillsParent{
		SkillList: skillList,
	}
	skillsXML, err := xml.Marshal(parent)
	if err != nil {
		slog.Error("Failed to marshal skills to xml in running task", "task", taskInfo.ID, "error", err)
		return "", false, err
	}

	runtimeInfo := fmt.Sprintf(
		"操作系统: %s, CPU架构: %s, 当前时间: %s",
		runtime.GOOS, runtime.GOARCH, time.Now().Local().Format(time.RFC3339),
	)

	msgs := []llm.Message{
		{
			Role: constant.RoleSystem,
			Content: fmt.Sprintf(constant.TaskExecutionPrompt,
				skillsXML,
				tm.workspace,
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
	history, err := tm.readTaskHistory(taskDir)
	if err != nil {
		return "", false, err
	}
	msgs = append(msgs, history...)

	ctx := context.Background()
	availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
	resp, err := tm.provider.Chat(ctx, tm.model, msgs, availableTools, nil)
	if err != nil {
		slog.Error("Failed to chat in running task", "task", taskInfo.ID, "error", err)
		return "", false, err
	}

	respMsg := llm.Message{
		Role:      constant.RoleAssistant,
		Content:   resp.Message.Content,
		ToolCalls: resp.Message.ToolCalls,
	}
	result = resp.Message.Content
	tm.appendTaskHistory(taskDir, respMsg)

	msgs = append(msgs, respMsg)

	isFinished := true

	for _, tc := range resp.Message.ToolCalls {
		isFinished = false
		toolResult, err := tm.executeToolCall(ctx, tc, taskInfo.Channel, taskInfo.ChatID)
		if err != nil {
			toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
		}
		result += "\n" + toolResult
		toolMsg := llm.Message{
			Role:      constant.RoleTool,
			Content:   toolResult,
			ToolCalls: []llm.ToolCall{tc},
		}
		tm.appendTaskHistory(taskDir, toolMsg)
		msgs = append(msgs, toolMsg)
	}

	if isFinished {
		// 再次检查完成标记，确保不会提前完成任务
		if !strings.Contains(result, constant.TaskTagCompleted) {
			isFinished = false
			userMsg := llm.Message{
				Role:    constant.RoleUser,
				Content: fmt.Sprintf("任务尚未完成，你必须调用工具继续执行任务。如果认为任务已完成，请输出`%s`。", constant.TaskTagCompleted),
			}
			tm.appendTaskHistory(taskDir, userMsg)
			msgs = append(msgs, userMsg)
		}
	}

	return result, isFinished, nil
}

func (tm *TaskManager) readTaskHistory(taskDir string) ([]llm.Message, error) {
	result := make([]llm.Message, 0)
	taskHistoryFile := filepath.Join(taskDir, constant.TaskHistoryFileName)
	if _, err := os.Stat(taskHistoryFile); err == nil {
		f, err := os.Open(taskHistoryFile)
		if err != nil {
			slog.Error("Failed to open task history file", "error", err)
			return result, err
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		for {
			var msg types.Message
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

func (tm *TaskManager) appendTaskHistory(taskDir string, msg llm.Message) {
	taskHistoryFile := filepath.Join(taskDir, constant.TaskHistoryFileName)

	os.MkdirAll(filepath.Dir(taskHistoryFile), 0755)

	f, err := os.OpenFile(taskHistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open task history file", "error", err)
		return
	}
	defer f.Close()

	var tcs []types.ToolCall
	for _, tc := range msg.ToolCalls {
		tcs = append(tcs, types.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		})
	}

	sMsg := types.Message{
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

func (tm *TaskManager) summaryTask(taskInfo *task.TaskInfo, taskDir string) (string, error) {
	history, err := tm.readTaskHistory(taskDir)
	if err != nil {
		slog.Error("Failed to read task history", "taskDir", taskDir, "error", err)
		return "", err
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
			Role:    constant.RoleSystem,
			Content: fmt.Sprintf(constant.TaskSummaryPrompt),
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
	resp, err := tm.provider.Chat(ctx, tm.model, msgs, nil, nil)
	if err != nil {
		slog.Error("Failed to summary task memory", "error", err)
		return "", err
	}
	return resp.Message.Content, nil
}

func (tm *TaskManager) summaryMainTask(taskInfo *task.TaskInfo) {
	taskDir := filepath.Join(tm.workspace, constant.DirTasks, taskInfo.ID)

	// 任务完成，让大模型总结任务信息并返回
	history, err := tm.readTaskHistory(taskDir)
	if err != nil {
		slog.Error("Failed to read task history", "taskDir", taskDir, "error", err)
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
			Content: fmt.Sprintf(constant.TaskSummaryArchivePrompt,
				tm.workspace,
				filepath.Join(tm.workspace, constant.DirProfile, constant.DirMemory),
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
		resp, err := tm.provider.Chat(ctx, tm.model, msgs, availableTools, nil)
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
			toolResult, err := tm.executeToolCall(ctx, tc, taskInfo.Channel, taskInfo.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
			}

			msgs = append(msgs, llm.Message{
				Role:      constant.RoleTool,
				Content:   toolResult,
				ToolCalls: []llm.ToolCall{tc},
			})
		}
	}
}
func (tm *TaskManager) executeToolCall(ctx context.Context, tc llm.ToolCall, channel, chatID string) (string, error) {
	var args map[string]any
	if tc.Arguments != "" {
		err := json.Unmarshal([]byte(tc.Arguments), &args)
		if err != nil {
			return "", fmt.Errorf("Failed to parse tool arguments: %w", err)
		}
	}

	if args == nil {
		args = make(map[string]any)
	}

	args[constant.ToolCallParamWorkspace] = tm.workspace
	args[constant.ToolCallParamChannel] = channel
	args[constant.ToolCallParamChatID] = chatID

	// Create ToolContext with agent information
	toolCtx := tools.NewToolContext(
		"",
		"", // agent owner - can be added later
		tm.workspace,
		"", // sessionID - can be passed separately if needed
		channel,
		chatID,
		tm.provider,
		tm.model,
	)

	result := tools.GetDefaultToolRegistry().ExecuteWithContext(ctx, tc.Name, args, toolCtx, channel, chatID)
	if result.IsError {
		return result.ForLLM, fmt.Errorf("Tool execution failed: %s", result.Err)
	}
	return result.ForLLM, nil
}
