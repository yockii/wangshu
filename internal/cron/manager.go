package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/yockii/yoclaw/internal/notification"
)

// TaskExecutor is a function that executes a cron task
type TaskExecutor func(ctx context.Context, sessionID, prompt string) (string, error)

// Task represents a cron task stored in workspace
type Task struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Schedule    string    `json:"schedule"`
	Enabled     bool      `json:"enabled"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	OwnerID     string    `json:"owner_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Manager manages cron tasks for an agent
type Manager struct {
	agentName  string
	workspace  string
	tasks      map[string]*Task // taskID -> Task
	mu         sync.RWMutex
	cron       *cron.Cron
	stopCh     chan struct{}
	executor   TaskExecutor // Function to execute tasks
}

// NewManager creates a new cron manager for an agent
func NewManager(agentName, workspace string, executor TaskExecutor) *Manager {
	// Create cron directory
	cronDir := filepath.Join(workspace, "cron")
	os.MkdirAll(cronDir, 0755)

	mgr := &Manager{
		agentName: agentName,
		workspace: workspace,
		tasks:     make(map[string]*Task),
		cron:      cron.New(cron.WithSeconds()),
		stopCh:    make(chan struct{}),
		executor:  executor,
	}

	// Load existing tasks
	if err := mgr.loadTasks(); err != nil {
		fmt.Printf("Failed to load cron tasks for agent %s: %v\n", agentName, err)
	}

	return mgr
}

// Start begins the cron scanner
func (m *Manager) Start() {
	m.cron.Start()

	// Start periodic scanner (every minute)
	go m.scanPeriodically()
}

// Stop stops the cron manager
func (m *Manager) Stop() {
	close(m.stopCh)
	m.cron.Stop()
}

// scanPeriodically scans for tasks to run every minute
func (m *Manager) scanPeriodically() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.scanAndExecute()
		case <-m.stopCh:
			return
		}
	}
}

// scanAndExecute scans for tasks that need to run
func (m *Manager) scanAndExecute() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for _, task := range m.tasks {
		if !task.Enabled {
			continue
		}

		// Check if next run time is reached
		if task.NextRun != nil && now.After(*task.NextRun) {
			// Execute the task
			go m.executeTask(task)

			// Update last run time
			now := time.Now()
			task.LastRun = &now
			task.UpdatedAt = now

			// Calculate next run time
			nextRun, err := m.calculateNextRun(task.Schedule)
			if err != nil {
				fmt.Printf("Failed to calculate next run for task %s: %v\n", task.Name, err)
				task.Enabled = false
			} else {
				task.NextRun = &nextRun
			}

			// Save updated task
			if err := m.saveTask(task); err != nil {
				fmt.Printf("Failed to save task %s: %v\n", task.Name, err)
			}
		}
	}
}

// executeTask executes a cron task by sending it to the agent
func (m *Manager) executeTask(task *Task) {
	fmt.Printf("[Cron] Executing task '%s' for agent '%s'\n", task.Name, m.agentName)

	if m.executor == nil {
		fmt.Printf("[Cron] No executor configured for agent %s\n", m.agentName)
		return
	}

	// Create a unique session ID for this cron task
	sessionID := fmt.Sprintf("cron_%s_%s", task.ID, time.Now().Format("20060102_150405"))

	// Build the prompt for the agent
	prompt := fmt.Sprintf("执行定时任务: %s\n描述: %s\n\n请执行这个任务。", task.Name, task.Description)
	if task.Description == "" {
		prompt = fmt.Sprintf("执行定时任务: %s\n\n请执行这个任务。", task.Name)
	}

	// Execute using the provided executor
	ctx := context.Background()
	response, err := m.executor(ctx, sessionID, prompt)

	if err != nil {
		fmt.Printf("[Cron] Task '%s' failed: %v\n", task.Name, err)
		return
	}

	fmt.Printf("[Cron] Task '%s' completed: %s\n", task.Name, response)

	// Notify the owner if specified
	if task.OwnerID != "" {
		// Use notification manager to send result
		message := fmt.Sprintf("🔔 定时任务已执行\n任务: %s\n时间: %s\n结果: %s",
			task.Name, time.Now().Format("2006-01-02 15:04:05"), response)
		if err := notification.GetManager().Notify(task.OwnerID, message); err != nil {
			fmt.Printf("Failed to notify owner: %v\n", err)
		}
	}
}

// AddTask adds a new cron task
func (m *Manager) AddTask(name, description, schedule, ownerID string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate schedule
	if err := m.validateSchedule(schedule); err != nil {
		return nil, fmt.Errorf("invalid schedule: %w", err)
	}

	// Calculate next run time
	nextRun, err := m.calculateNextRun(schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next run: %w", err)
	}

	task := &Task{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Schedule:    schedule,
		Enabled:     true,
		NextRun:     &nextRun,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}

	// Save to file
	if err := m.saveTask(task); err != nil {
		return nil, err
	}

	m.tasks[task.ID] = task

	return task, nil
}

// RemoveTask removes a cron task
func (m *Manager) RemoveTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found")
	}

	// Remove file
	taskFile := m.getTaskFilePath(task.ID)
	if err := os.Remove(taskFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(m.tasks, taskID)

	return nil
}

// EnableTask enables a cron task
func (m *Manager) EnableTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found")
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	// Recalculate next run
	nextRun, err := m.calculateNextRun(task.Schedule)
	if err != nil {
		return err
	}
	task.NextRun = &nextRun

	return m.saveTask(task)
}

// DisableTask disables a cron task
func (m *Manager) DisableTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found")
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	return m.saveTask(task)
}

// ListTasks returns all tasks
func (m *Manager) ListTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

// getTaskFilePath returns the file path for a task
func (m *Manager) getTaskFilePath(taskID string) string {
	return filepath.Join(m.workspace, "cron", taskID+".json")
}

// saveTask saves a task to file
func (m *Manager) saveTask(task *Task) error {
	taskFile := m.getTaskFilePath(task.ID)

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(taskFile, data, 0644)
}

// loadTasks loads all tasks from workspace
func (m *Manager) loadTasks() error {
	cronDir := filepath.Join(m.workspace, "cron")

	entries, err := os.ReadDir(cronDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cron directory yet
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskFile := filepath.Join(cronDir, entry.Name())
		data, err := os.ReadFile(taskFile)
		if err != nil {
			fmt.Printf("Failed to read task file %s: %v\n", taskFile, err)
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			fmt.Printf("Failed to parse task file %s: %v\n", taskFile, err)
			continue
		}

		// Recalculate next run time if task is enabled
		if task.Enabled {
			nextRun, err := m.calculateNextRun(task.Schedule)
			if err != nil {
				fmt.Printf("Failed to calculate next run for task %s: %v\n", task.Name, err)
				task.Enabled = false
			} else {
				task.NextRun = &nextRun
			}
		}

		m.tasks[task.ID] = &task
	}

	return nil
}

// validateSchedule validates a cron schedule expression
func (m *Manager) validateSchedule(schedule string) error {
	_, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(schedule)
	return err
}

// calculateNextRun calculates the next run time for a schedule
func (m *Manager) calculateNextRun(schedule string) (time.Time, error) {
	scheduleParser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := scheduleParser.Parse(schedule)
	if err != nil {
		return time.Time{}, err
	}

	return sched.Next(time.Now()), nil
}
