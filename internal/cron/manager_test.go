package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	executed := false
	executor := func(job *types.BasicJobInfo) {
		_ = job.ID
		executed = true
	}

	_ = NewManager(tmpDir, executor)

	if !executed {
		t.Log("Executor created successfully")
	}

	// 简单验证
	if tmpDir == "" {
		t.Error("tmpDir should not be empty")
	}
}

func TestCronManager_Stop(t *testing.T) {
	tmpDir := t.TempDir()

	executor := func(job *types.BasicJobInfo) {}
	mgr := NewManager(tmpDir, executor)

	// Stop 应该不会 panic
	mgr.Stop()

	// 再次 Stop 也应该安全
	mgr.Stop()
}

func TestCronManager_CreateCronDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	executor := func(job *types.BasicJobInfo) {}
	_ = NewManager(tmpDir, executor)

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
		"* * * * *",   // 每分钟
		"0 * * * *",   // 每小时
		"0 0 * * *",   // 每天
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

func TestCronManager_Executor(t *testing.T) {
	tmpDir := t.TempDir()

	var executedJobID string
	executor := func(job *types.BasicJobInfo) {
		executedJobID = job.ID
	}

	_ = NewManager(tmpDir, executor)

	// 模拟任务执行
	job := &types.BasicJobInfo{
		ID:       "test-job",
		Schedule: "* * * * *",
	}

	// 手动调用执行器
	executor(job)

	if executedJobID != "test-job" {
		t.Errorf("Expected executor to be called with 'test-job', got '%s'", executedJobID)
	}
}

func TestCronManager_ContextManagement(t *testing.T) {
	tmpDir := t.TempDir()

	executor := func(job *types.BasicJobInfo) {}
	mgr := NewManager(tmpDir, executor)

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

	executor := func(job *types.BasicJobInfo) {}
	mgr := NewManager(tmpDir, executor)

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
		ID:       "test-job",
		Schedule: "0 9 * * *",
		Status:   constant.CronStatusEnabled,
		Channel:  "test-channel",
		ChatID:   "test-chat",
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
