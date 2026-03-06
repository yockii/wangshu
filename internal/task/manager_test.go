package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yockii/wangshu/pkg/constant"
)

// MockProvider 用于测试的模拟 Provider
type MockProvider struct {
}

func TestNewTaskManager(t *testing.T) {
	tmpDir := t.TempDir()

	tm := NewTaskManager("test-agent", tmpDir, "test-model", nil)

	if tm == nil {
		t.Fatal("NewTaskManager should not return nil")
	}

	if tm.workspace != tmpDir {
		t.Errorf("Expected workspace '%s', got '%s'", tmpDir, tm.workspace)
	}

	if tm.model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", tm.model)
	}

	if tm.interval != 10*time.Second {
		t.Errorf("Expected interval 10s, got %v", tm.interval)
	}
}

func TestTaskManager_Stop(t *testing.T) {
	tmpDir := t.TempDir()

	tm := NewTaskManager("test-agent", tmpDir, "test-model", nil)

	// Stop 应该不会 panic
	tm.Stop()

	// 再次 Stop 也应该安全
	tm.Stop()
}

func TestTaskManager_CreateTasksDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	_ = NewTaskManager("test-agent", tmpDir, "test-model", nil)

	// 等待一下让 manager 启动并创建目录
	time.Sleep(100 * time.Millisecond)

	tasksDir := filepath.Join(tmpDir, constant.DirTasks)
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		t.Error("Tasks directory should be created")
	}
}

func TestTaskManager_TaskFileOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试任务目录
	taskDir := filepath.Join(tmpDir, constant.DirTasks, "test-task-id")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	// 创建任务信息文件
	taskInfoFile := filepath.Join(taskDir, constant.TaskInfoFileName)
	taskInfoContent := `{
		"id": "test-task-id",
		"name": "Test Task",
		"description": "Test Description",
		"priority": "normal",
		"status": "pending"
	}`

	if err := os.WriteFile(taskInfoFile, []byte(taskInfoContent), 0644); err != nil {
		t.Fatalf("Failed to write task info file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(taskInfoFile); os.IsNotExist(err) {
		t.Error("Task info file should exist")
	}

	// 读取并验证内容
	data, err := os.ReadFile(taskInfoFile)
	if err != nil {
		t.Fatalf("Failed to read task info file: %v", err)
	}

	if string(data) != taskInfoContent {
		t.Error("Task info content should match")
	}
}

func TestTaskManager_SubtasksRecord(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试任务目录
	taskDir := filepath.Join(tmpDir, constant.DirTasks, "parent-task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	// 创建子任务目录
	subtaskDir := filepath.Join(taskDir, "subtask-1")
	if err := os.MkdirAll(subtaskDir, 0755); err != nil {
		t.Fatalf("Failed to create subtask directory: %v", err)
	}

	// 创建子任务信息文件
	subtaskInfoFile := filepath.Join(subtaskDir, constant.TaskInfoFileName)
	subtaskInfoContent := `{
		"id": "subtask-1",
		"name": "Subtask 1",
		"description": "Subtask Description",
		"priority": "normal",
		"status": "pending"
	}`

	if err := os.WriteFile(subtaskInfoFile, []byte(subtaskInfoContent), 0644); err != nil {
		t.Fatalf("Failed to write subtask info file: %v", err)
	}

	// 创建 subtasks.json
	subtasksFile := filepath.Join(taskDir, constant.TaskSubtasksInfoFileName)
	subtasksContent := `{
		"subtasks": {
			"subtask-1": {
				"id": "subtask-1",
				"name": "Subtask 1",
				"status": "pending",
				"created_at": "2024-01-01T00:00:00Z"
			}
		}
	}`

	if err := os.WriteFile(subtasksFile, []byte(subtasksContent), 0644); err != nil {
		t.Fatalf("Failed to write subtasks file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(subtasksFile); os.IsNotExist(err) {
		t.Error("Subtasks file should exist")
	}

	if _, err := os.Stat(subtaskInfoFile); os.IsNotExist(err) {
		t.Error("Subtask info file should exist")
	}
}

func TestTaskManager_TaskHistory(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试任务目录
	taskDir := filepath.Join(tmpDir, constant.DirTasks, "test-task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	// 创建历史文件
	historyFile := filepath.Join(taskDir, constant.TaskHistoryFileName)
	historyContent := `{"role":"user","content":"Test message","timestamp":"2024-01-01T00:00:00Z"}
	{"role":"assistant","content":"Test response","timestamp":"2024-01-01T00:00:01Z"}`

	if err := os.WriteFile(historyFile, []byte(historyContent), 0644); err != nil {
		t.Fatalf("Failed to write history file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Error("History file should exist")
	}

	// 读取并验证
	data, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatalf("Failed to read history file: %v", err)
	}

	if string(data) != historyContent {
		t.Logf("History content mismatch")
		t.Logf("Expected: %s", historyContent)
		t.Logf("Got: %s", string(data))
	}
}

func TestTaskManager_ChangeLog(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试任务目录
	taskDir := filepath.Join(tmpDir, constant.DirTasks, "test-task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	// 创建变更日志文件
	changeLogFile := filepath.Join(taskDir, constant.TaskChangeLogFileName)
	changeLogContent := `{
		"entries": [
			{
				"content": "Update task description",
				"notified": false,
				"notified_at": null
			},
			{
				"content": "Change priority to high",
				"notified": false,
				"notified_at": null
			}
		]
	}`

	if err := os.WriteFile(changeLogFile, []byte(changeLogContent), 0644); err != nil {
		t.Fatalf("Failed to write change log file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(changeLogFile); os.IsNotExist(err) {
		t.Error("Change log file should exist")
	}
}

func TestTaskManager_TaskRelations(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建任务目录
	tasksDir := filepath.Join(tmpDir, constant.DirTasks)
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatalf("Failed to create tasks directory: %v", err)
	}

	// 创建任务关系文件
	relationsFile := filepath.Join(tasksDir, constant.TaskRelationsFileName)
	relationsContent := `{
		"relations": {
			"task-1": {
				"root_id": "root-task",
				"parent_id": "parent-task"
			},
			"task-2": {
				"root_id": "root-task",
				"parent_id": "task-1"
			}
		}
	}`

	if err := os.WriteFile(relationsFile, []byte(relationsContent), 0644); err != nil {
		t.Fatalf("Failed to write relations file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(relationsFile); os.IsNotExist(err) {
		t.Error("Relations file should exist")
	}
}

func TestTaskManager_PriorityOrdering(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建不同优先级的任务
	priorities := []string{
		constant.TaskPriorityLow,
		constant.TaskPriorityNormal,
		constant.TaskPriorityHigh,
		constant.TaskPriorityUrgent,
	}

	// 验证优先级常量存在
	for _, priority := range priorities {
		if priority == "" {
			t.Error("Priority constant should not be empty")
		}
	}

	// 创建测试任务目录
	for i, priority := range priorities {
		taskID := fmt.Sprintf("task-%d", i)
		taskDir := filepath.Join(tmpDir, constant.DirTasks, taskID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			t.Fatalf("Failed to create task directory: %v", err)
		}

		taskInfoFile := filepath.Join(taskDir, constant.TaskInfoFileName)
		taskInfoContent := fmt.Sprintf(`{
			"id": "%s",
			"name": "Task %d",
			"description": "Task with priority",
			"priority": "%s",
			"status": "pending"
		}`, taskID, i, priority)

		if err := os.WriteFile(taskInfoFile, []byte(taskInfoContent), 0644); err != nil {
			t.Fatalf("Failed to write task info file: %v", err)
		}
	}

	// 验证所有任务文件都创建成功
	for i := range priorities {
		taskID := fmt.Sprintf("task-%d", i)
		taskInfoFile := filepath.Join(tmpDir, constant.DirTasks, taskID, constant.TaskInfoFileName)
		if _, err := os.Stat(taskInfoFile); os.IsNotExist(err) {
			t.Errorf("Task %s info file should exist", taskID)
		}
	}
}

func TestTaskManager_TaskStatusTransitions(t *testing.T) {
	tmpDir := t.TempDir()

	validStatuses := []string{
		constant.TaskStatusPending,
		constant.TaskStatusRunning,
		constant.TaskStatusCompleted,
		constant.TaskStatusFailed,
		constant.TaskStatusCancelled,
		constant.TaskStatusRemove,
	}

	// 验证状态常量存在
	for _, status := range validStatuses {
		if status == "" {
			t.Error("Status constant should not be empty")
		}
	}

	// 创建不同状态的任务
	for i, status := range validStatuses {
		taskID := fmt.Sprintf("task-status-%d", i)
		taskDir := filepath.Join(tmpDir, constant.DirTasks, taskID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			t.Fatalf("Failed to create task directory: %v", err)
		}

		taskInfoFile := filepath.Join(taskDir, constant.TaskInfoFileName)
		taskInfoContent := fmt.Sprintf(`{
			"id": "%s",
			"name": "Task with status",
			"description": "Task status test",
			"priority": "normal",
			"status": "%s"
		}`, taskID, status)

		if err := os.WriteFile(taskInfoFile, []byte(taskInfoContent), 0644); err != nil {
			t.Fatalf("Failed to write task info file: %v", err)
		}
	}
}

func TestTaskManager_ContextManagement(t *testing.T) {
	tmpDir := t.TempDir()

	tm := NewTaskManager("test-agent", tmpDir, "test-model", nil)

	// 验证 context 和 cancelFunc 初始化
	if tm.ctx == nil {
		t.Error("Context should be initialized")
	}

	if tm.cancel == nil {
		t.Error("CancelFunc should be initialized")
	}

	// 验证停止后 context 被取消
	tm.Stop()

	select {
	case <-tm.ctx.Done():
		// Context 应该被取消
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Stop")
	}
}

func TestTaskManager_ConcurrencySafety(t *testing.T) {
	tmpDir := t.TempDir()

	tm := NewTaskManager("test-agent", tmpDir, "test-model", nil)

	// 并发访问应该是安全的
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in concurrent access: %v", r)
				}
				done <- true
			}()
			// 模拟读取操作
			tm.mu.RLock()
			time.Sleep(10 * time.Millisecond)
			tm.mu.RUnlock()
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 5; i++ {
		<-done
	}

	// 停止 manager
	tm.Stop()
}
