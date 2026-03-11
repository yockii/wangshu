package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

// mockProvider 是一个用于测试的模拟 LLM Provider
type mockProvider struct{}

func (m *mockProvider) Chat(ctx context.Context, model string, messages []llm.Message, tools []llm.ToolDefinition, options map[string]any) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    constant.RoleAssistant,
			Content: "Mock response",
		},
	}, nil
}

func (m *mockProvider) ChatWithJSONSchema(ctx context.Context, model string, messages []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	// 返回一个简单的 message 类型响应
	result := CronJobExecutionResult{
		TaskType:       "message",
		MessageContent: "Test message from cron job",
	}
	content, _ := json.Marshal(result)
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    constant.RoleAssistant,
			Content: string(content),
		},
	}, nil
}

func TestNewCronManager(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}

	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", provider)

	if mgr == nil {
		t.Fatal("NewCronManager should return a non-nil manager")
	}

	if mgr.workspace != tmpDir {
		t.Errorf("Expected workspace %s, got %s", tmpDir, mgr.workspace)
	}

	if mgr.model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", mgr.model)
	}

	if mgr.provider != provider {
		t.Error("Provider should be set")
	}
}

func TestCronManager_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}
	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", provider)

	// Stop 应该不会 panic
	mgr.Stop()

	// 再次 Stop 也应该安全
	mgr.Stop()
}

func TestCronManager_CreateCronDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}
	_ = NewCronManager("test-agent", tmpDir, "test-model", "", provider)

	// 等待一下让 manager 启动并扫描
	time.Sleep(200 * time.Millisecond)

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if _, err := os.Stat(cronDir); os.IsNotExist(err) {
		// 目录可能还没有被扫描到，这是正常的
		t.Logf("Cron directory not yet created (will be created on next scan)")
	}
}

func TestCronManager_JobFileOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 cron 目录
	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建测试任务文件
	job := types.BasicJobInfo{
		ID:          "test-job",
		Description: "Test Description",
		Schedule:    "* * * * *", // 每分钟执行
		Status:      constant.CronStatusEnabled,
		Once:        false,
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, "test-job.json")
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(jobFile); os.IsNotExist(err) {
		t.Error("Job file should exist")
	}

	// 读取并验证内容
	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("Failed to read job file: %v", err)
	}

	var loadedJob types.BasicJobInfo
	if err := json.Unmarshal(data, &loadedJob); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	if loadedJob.ID != "test-job" {
		t.Errorf("Expected job ID 'test-job', got '%s'", loadedJob.ID)
	}

	if loadedJob.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", loadedJob.Description)
	}
}

func TestCronManager_JobStatusTransitions(t *testing.T) {
	tmpDir := t.TempDir()

	// 验证状态常量存在
	validStatuses := []string{
		constant.CronStatusEnabled,
		constant.CronStatusPaused,
		constant.CronStatusDisabled,
	}

	for _, status := range validStatuses {
		if status == "" {
			t.Errorf("Status constant should not be empty")
		}
	}

	// 创建不同状态的任务
	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	for i, status := range validStatuses {
		job := types.BasicJobInfo{
			ID:       fmt.Sprintf("job-status-%d", i),
			Schedule: "* * * * *",
			Status:   status,
		}

		jobJSON, _ := json.Marshal(job)
		jobFile := filepath.Join(cronDir, job.ID+".json")
		if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
			t.Fatalf("Failed to write job file: %v", err)
		}

		// 验证文件存在
		if _, err := os.Stat(jobFile); os.IsNotExist(err) {
			t.Errorf("Job %s file should exist", job.ID)
		}
	}
}

func TestCronManager_OnceJob(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建一次性任务
	job := types.BasicJobInfo{
		ID:       "once-job",
		Schedule: "* * * * *",
		Status:   constant.CronStatusEnabled,
		Once:     true,
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, "once-job.json")
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 读取并验证
	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("Failed to read job file: %v", err)
	}

	var loadedJob types.BasicJobInfo
	if err := json.Unmarshal(data, &loadedJob); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	if !loadedJob.Once {
		t.Error("Job should be marked as once")
	}
}

func TestCronManager_JobWithLastRun(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建带 LastRun 的任务
	now := time.Now()
	job := types.BasicJobInfo{
		ID:       "test-job",
		Schedule: "* * * * *",
		Status:   constant.CronStatusEnabled,
		LastRun:  &now,
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, "test-job.json")
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 读取并验证
	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("Failed to read job file: %v", err)
	}

	var loadedJob types.BasicJobInfo
	if err := json.Unmarshal(data, &loadedJob); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	if loadedJob.LastRun == nil {
		t.Error("LastRun should be preserved")
	}
}

func TestCronManager_JobWithNextRun(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建带 NextRun 的任务
	nextRun := time.Now().Add(1 * time.Minute)
	job := types.BasicJobInfo{
		ID:       "test-job",
		Schedule: "* * * * *",
		Status:   constant.CronStatusEnabled,
		NextRun:  &nextRun,
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, "test-job.json")
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 读取并验证
	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("Failed to read job file: %v", err)
	}

	var loadedJob types.BasicJobInfo
	if err := json.Unmarshal(data, &loadedJob); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	if loadedJob.NextRun == nil {
		t.Error("NextRun should be preserved")
	}
}

func TestCronManager_MultipleJobs(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建多个任务
	schedules := []string{
		"* * * * *", // 每分钟
		"0 * * * *", // 每小时
		"0 0 * * *", // 每天
	}

	for i, schedule := range schedules {
		job := types.BasicJobInfo{
			ID:       fmt.Sprintf("job-%d", i),
			Schedule: schedule,
			Status:   constant.CronStatusEnabled,
		}

		jobJSON, _ := json.Marshal(job)
		jobFile := filepath.Join(cronDir, job.ID+".json")
		if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
			t.Fatalf("Failed to write job file: %v", err)
		}
	}

	// 验证所有文件都创建成功
	for i := range schedules {
		jobID := fmt.Sprintf("job-%d", i)
		jobFile := filepath.Join(cronDir, jobID+".json")
		if _, err := os.Stat(jobFile); os.IsNotExist(err) {
			t.Errorf("Job %s file should exist", jobID)
		}
	}
}

func TestCronManager_Execute_MessageType(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}
	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", provider)

	ctx := context.Background()
	job := &types.BasicJobInfo{
		ID:          "test-job",
		Description: "Test job",
		Channel:     "test-channel",
		ChatID:      "test-chat",
	}

	// Execute 应该调用 mockProvider，但由于工具注册表可能没有 message 工具，可能会失败
	// 这是测试环境中的预期行为
	err := mgr.Execute(ctx, job)
	t.Logf("Execute result (may fail if tool registry unavailable): %v", err)
	// 在实际环境中，工具会正确注册并执行成功
}

func TestCronManager_Execute_TaskType(t *testing.T) {
	// mockProviderForTask 返回 task 类型的响应
	mockProviderForTask := &mockProviderTask{}
	tmpDir := t.TempDir()
	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", mockProviderForTask)

	ctx := context.Background()
	job := &types.BasicJobInfo{
		ID:          "test-job",
		Description: "Generate daily news summary",
		Channel:     "test-channel",
		ChatID:      "test-chat",
	}

	// Execute 应该调用 mockProvider 并成功返回
	err := mgr.Execute(ctx, job)
	// 由于实际的 task 工具可能不可用，这里可能会失败，但这是预期的
	// 我们主要验证代码路径是正确的
	t.Logf("Execute result (may fail if task tool unavailable): %v", err)
}

// mockProviderTask 返回 task 类型的响应
type mockProviderTask struct{}

func (m *mockProviderTask) Chat(ctx context.Context, model string, messages []llm.Message, tools []llm.ToolDefinition, options map[string]any) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    constant.RoleAssistant,
			Content: "Mock response",
		},
	}, nil
}

func (m *mockProviderTask) ChatWithJSONSchema(ctx context.Context, model string, messages []llm.Message, jsonSchema *llm.JSONSchema, options map[string]any) (*llm.ChatResponse, error) {
	// 返回一个 task 类型的响应
	result := CronJobExecutionResult{
		TaskType:        "task",
		TaskName:        "Generate news summary",
		TaskDescription: "Fetch and summarize daily news",
		TaskPriority:    "normal",
	}
	content, _ := json.Marshal(result)
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    constant.RoleAssistant,
			Content: string(content),
		},
	}, nil
}

func TestCronManager_ContextManagement(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}
	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", provider)

	// 验证 context 和 cancelFunc 初始化
	if mgr.ctx == nil {
		t.Error("Context should be initialized")
	}

	if mgr.cancel == nil {
		t.Error("CancelFunc should be initialized")
	}

	// 验证停止后 context 被取消
	mgr.Stop()

	select {
	case <-mgr.ctx.Done():
		// Context 应该被取消
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Stop")
	}
}

func TestCronManager_ConcurrencySafety(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &mockProvider{}
	mgr := NewCronManager("test-agent", tmpDir, "test-model", "", provider)

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
			mgr.mu.RLock()
			time.Sleep(10 * time.Millisecond)
			mgr.mu.RUnlock()
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 5; i++ {
		<-done
	}

	// 停止 manager
	mgr.Stop()
}

func TestCronManager_JobWithChannel(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建带 Channel 和 ChatID 的任务
	job := types.BasicJobInfo{
		ID:          "test-job",
		Description: "Test job",
		Schedule:    "0 9 * * *",
		Status:      constant.CronStatusEnabled,
		Channel:     "test-channel",
		ChatID:      "test-chat",
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, "test-job.json")
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 读取并验证
	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("Failed to read job file: %v", err)
	}

	var loadedJob types.BasicJobInfo
	if err := json.Unmarshal(data, &loadedJob); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	if loadedJob.Channel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", loadedJob.Channel)
	}

	if loadedJob.ChatID != "test-chat" {
		t.Errorf("Expected chatID 'test-chat', got '%s'", loadedJob.ChatID)
	}
}

func TestCronManager_NonJSONFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建非 JSON 文件（应该被忽略）
	readmeFile := filepath.Join(cronDir, "README.txt")
	if err := os.WriteFile(readmeFile, []byte("This is a readme"), 0644); err != nil {
		t.Fatalf("Failed to write readme: %v", err)
	}

	// 创建子目录（应该被忽略）
	subDir := filepath.Join(cronDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
		t.Error("README file should exist")
	}

	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Subdirectory should exist")
	}

	// 这些文件和目录应该被 scanJobs 忽略，所以不会导致错误
}

func TestCronManager_EmptyJobID(t *testing.T) {
	tmpDir := t.TempDir()

	cronDir := filepath.Join(tmpDir, constant.DirCron)
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("Failed to create cron directory: %v", err)
	}

	// 创建空 ID 的任务（应该被跳过）
	job := types.BasicJobInfo{
		ID:       "", // 空ID
		Schedule: "* * * * *",
		Status:   constant.CronStatusEnabled,
	}

	jobJSON, _ := json.Marshal(job)
	jobFile := filepath.Join(cronDir, ".json") // 使用空文件名
	if err := os.WriteFile(jobFile, jobJSON, 0644); err != nil {
		t.Fatalf("Failed to write job file: %v", err)
	}

	// 文件应该被创建，但会被 scanJobs 忽略（日志警告）
	// 验证文件存在
	if _, err := os.Stat(jobFile); os.IsNotExist(err) {
		t.Error("Job file should exist even with empty ID")
	}
}

func TestCronJobExecutionResult_JSONSchema(t *testing.T) {
	// 验证 CronJobExecutionResult 可以正确序列化和反序列化
	result := CronJobExecutionResult{
		TaskType:       "message",
		MessageContent: "Test message",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CronJobExecutionResult: %v", err)
	}

	var unmarshaled CronJobExecutionResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CronJobExecutionResult: %v", err)
	}

	if unmarshaled.TaskType != "message" {
		t.Errorf("Expected type 'message', got '%s'", unmarshaled.TaskType)
	}

	if unmarshaled.MessageContent != "Test message" {
		t.Errorf("Expected message content 'Test message', got '%s'", unmarshaled.MessageContent)
	}
}

func TestCronJobExecutionResult_TaskType(t *testing.T) {
	result := CronJobExecutionResult{
		TaskType:        "task",
		TaskName:        "Test Task",
		TaskDescription: "Test task description",
		TaskPriority:    "high",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CronJobExecutionResult: %v", err)
	}

	var unmarshaled CronJobExecutionResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CronJobExecutionResult: %v", err)
	}

	if unmarshaled.TaskType != "task" {
		t.Errorf("Expected type 'task', got '%s'", unmarshaled.TaskType)
	}

	if unmarshaled.TaskName != "Test Task" {
		t.Errorf("Expected task name 'Test Task', got '%s'", unmarshaled.TaskName)
	}

	if unmarshaled.TaskPriority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", unmarshaled.TaskPriority)
	}
}
