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

// TaskExecutor is a function that executes a cron task (kept for backwards compatibility but no longer used)
type TaskExecutor func(ctx context.Context, sessionID, prompt string) (string, error)

// TaskCreator is a function that creates a task for cron execution
type TaskCreator func(name, description, ownerID, agentName string) (taskID string, err error)

// CronEventHandler handles cron events (fired when tasks are due)
type CronEventHandler interface {
	// OnCronTaskDue is called when a cron task is due for execution
	OnCronTaskDue(task *Task) error
}

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

// Manager manages cron tasks for an agent (lightweight recorder - no longer executes tasks)
type Manager struct {
	workspace    string
	tasks        map[string]*Task // taskID -> Task
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	agentName    string  // Track which agent this manager belongs to
	taskCreator  TaskCreator // Function to create tasks for cron execution (deprecated, use eventHandler)
	eventHandler CronEventHandler // New: event-based approach
}

// NewManager creates a new cron manager
func NewManager(workspace string, executor TaskExecutor, taskCreator TaskCreator) *Manager {
	// Create cron directory
	cronDir := filepath.Join(workspace, "cron")
	os.MkdirAll(cronDir, 0755)

	ctx, cancel := context.WithCancel(context.Background())

	mgr := &Manager{
		workspace:   workspace,
		tasks:       make(map[string]*Task),
		ctx:         ctx,
		cancel:      cancel,
		taskCreator: taskCreator,
	}

	// Load existing tasks
	if err := mgr.loadTasks(); err != nil {
		fmt.Printf("Failed to load cron tasks: %v\n", err)
	}

	// Start periodic scanner
	go mgr.scanPeriodically()

	return mgr
}

// SetAgentName sets the agent name for this cron manager
func (m *Manager) SetAgentName(name string) {
	m.agentName = name
}

// Stop stops the cron manager
func (m *Manager) Stop() {
	m.cancel()
}

// SetEventHandler sets the event handler for cron events
func (m *Manager) SetEventHandler(handler CronEventHandler) {
	m.eventHandler = handler
}

// scanPeriodically scans for tasks to run every minute
func (m *Manager) scanPeriodically() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.scanAndExecute()
		}
	}
}

// scanAndExecute scans for tasks that need to run and executes them
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
			// Execute the task by creating a Task in TaskManager
			go m.executeCronTask(task)

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

// executeCronTask executes a cron task by creating a task via the event handler
func (m *Manager) executeCronTask(task *Task) {
	fmt.Printf("[Cron] Executing task '%s'\n", task.Name)

	// Try event handler first (new approach)
	if m.eventHandler != nil {
		if err := m.eventHandler.OnCronTaskDue(task); err != nil {
			fmt.Printf("[Cron] Failed to handle cron task %s: %v\n", task.Name, err)
		}
		return
	}

	// Fall back to taskCreator (legacy, for compatibility)
	if m.taskCreator != nil {
		agentName := m.agentName
		if agentName == "" {
			agentName = "default"
		}

		taskID, err := m.taskCreator(
			fmt.Sprintf("[Cron] %s", task.Name),
			task.Description,
			task.OwnerID,
			agentName,
		)
		if err != nil {
			fmt.Printf("[Cron] Failed to create task for %s: %v\n", task.Name, err)
			return
		}

		fmt.Printf("[Cron] Task '%s' started as task %s\n", task.Name, taskID)

		// Notify the owner if specified
		if task.OwnerID != "" {
			message := fmt.Sprintf("🔔 定时任务已启动\n任务: %s\n时间: %s\n任务ID: %s",
				task.Name, time.Now().Format("2006-01-02 15:04:05"), taskID)
			if err := notifyOwner(task.OwnerID, message); err != nil {
				fmt.Printf("Failed to notify owner: %v\n", err)
			}
		}
		return
	}

	fmt.Printf("[Cron] No event handler or task creator configured for task %s\n", task.Name)
}

// notifyOwner notifies the task owner
func notifyOwner(ownerID, message string) error {
	return notification.GetManager().Notify(ownerID, message)
}

// GetPendingTasks returns a list of tasks that should be executed
func (m *Manager) GetPendingTasks() ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	pending := make([]*Task, 0)

	for _, task := range m.tasks {
		if !task.Enabled {
			continue
		}
		if task.NextRun != nil && now.After(*task.NextRun) {
			pending = append(pending, task)
		}
	}

	return pending, nil
}

// UpdateTaskRunTimes updates the last run and next run times for a task
func (m *Manager) UpdateTaskRunTimes(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found")
	}

	now := time.Now()
	task.LastRun = &now
	task.UpdatedAt = now

	nextRun, err := m.calculateNextRun(task.Schedule)
	if err != nil {
		task.Enabled = false
		return fmt.Errorf("failed to calculate next run: %w", err)
	}
	task.NextRun = &nextRun

	return m.saveTask(task)
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
