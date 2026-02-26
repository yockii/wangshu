package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yockii/yoclaw/internal/notification"
	"github.com/yockii/yoclaw/pkg/llm"
	"github.com/yockii/yoclaw/pkg/tools"
)

// AgentExecutor is an interface that defines what TaskManager needs from an agent
type AgentExecutor interface {
	GetTools() *tools.Registry
	CallProvider(ctx context.Context, sessionID string, msgs []llm.Message) (*llm.ChatResponse, error)
	GetLLMProvider() (llm.Provider, string) // Returns LLM provider and model
}


// Status represents the status of a task
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// TaskMeta represents task metadata stored in task.json
type TaskMeta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Schedule    string    `json:"schedule,omitempty"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	OwnerID     string    `json:"owner_id"`
	AgentName   string    `json:"agent_name"` // Which agent should execute this task
	Metadata    map[string]string `json:"metadata,omitempty"`
	mu          sync.RWMutex
	cancelFunc  context.CancelFunc
	ctx         context.Context
}

// TaskMessage represents a message in the task log (similar to session.Message)
type TaskMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Timestamp  time.Time       `json:"timestamp"`
	ToolCalls  []ToolCallInfo  `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"` // For tool result messages
}

// ToolCallInfo represents a tool call in task log
type ToolCallInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result,omitempty"`
}

// Manager manages long-running tasks
type Manager struct {
	runningTasks        map[string]*TaskMeta        // taskID -> task metadata
	agentWorkspaces     map[string]string           // agentName -> workspace path
	taskAgents          map[string]string           // taskID -> agentName mapping for quick lookup
	agentExecutors      map[string]AgentExecutor    // agentName -> executor for running tasks
	mu                  sync.RWMutex
	zombieTaskThreshold time.Duration               // zombie task detection threshold (default 5 minutes)
}

var globalManager *Manager

// GetManager returns the global task manager
func GetManager() *Manager {
	return globalManager
}

// Initialize creates and initializes the global task manager
func Initialize(agents map[string]AgentExecutor, workspaces map[string]string) (*Manager, error) {
	if globalManager != nil {
		return globalManager, nil
	}

	// Build agent workspace mapping
	agentWorkspaces := make(map[string]string)
	agentExecutors := make(map[string]AgentExecutor)
	for name, ag := range agents {
		ws := workspaces[name]
		agentWorkspaces[name] = ws
		agentExecutors[name] = ag

		// Ensure tasks directory exists for each agent
		tasksDir := filepath.Join(ws, "tasks")
		if err := os.MkdirAll(tasksDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create tasks directory for agent %s: %w", name, err)
		}
	}

	mgr := &Manager{
		runningTasks:        make(map[string]*TaskMeta),
		agentWorkspaces:     agentWorkspaces,
		taskAgents:          make(map[string]string),
		agentExecutors:      agentExecutors,
		zombieTaskThreshold: 5 * time.Minute, // default 5 minutes
	}

	// Scan task folders from all agents and resume any interrupted running tasks
	if err := mgr.scanAndResumeTasks(); err != nil {
		return nil, fmt.Errorf("failed to scan tasks: %w", err)
	}

	globalManager = mgr

	// Start periodic checker (every minute)
	go mgr.periodicCheck()

	return mgr, nil
}

// getTaskDir returns the task directory for a given task (based on its agent)
func (m *Manager) getTaskDir(task *TaskMeta) string {
	ws, exists := m.agentWorkspaces[task.AgentName]
	if !exists {
		return ""
	}
	return filepath.Join(ws, "tasks", task.ID)
}

// getTaskDirByIDs returns the task directory given taskID and agentName
func (m *Manager) getTaskDirByIDs(taskID, agentName string) string {
	ws, exists := m.agentWorkspaces[agentName]
	if !exists {
		return ""
	}
	return filepath.Join(ws, "tasks", taskID)
}

// CreateTask creates a new task
func (m *Manager) CreateTask(name, description, ownerID, agentName, schedule string) (*TaskMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskID := uuid.New().String()
	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}

	// Create task directory
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create task directory: %w", err)
	}

	task := &TaskMeta{
		ID:          taskID,
		Name:        name,
		Description: description,
		Status:      StatusPending,
		Priority:    5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		OwnerID:     ownerID,
		AgentName:   agentName,
		Schedule:    schedule,
		Metadata:    make(map[string]string),
	}

	// Save task metadata
	if err := m.saveTaskMeta(task); err != nil {
		os.RemoveAll(taskDir)
		return nil, err
	}

	// Initialize empty messages log
	if err := m.initMessagesLog(taskID); err != nil {
		os.RemoveAll(taskDir)
		return nil, err
	}

	// Record creation event
	if err := m.appendTaskEvent(taskID, "system", fmt.Sprintf("Task created by %s", ownerID)); err != nil {
		os.RemoveAll(taskDir)
		return nil, err
	}

	// Store taskID -> agentName mapping
	m.taskAgents[taskID] = agentName

	return task, nil
}

// ExecuteTask starts executing a task with agent interaction
func (m *Manager) ExecuteTask(taskID string) error {
	m.mu.Lock()
	_, exists := m.runningTasks[taskID]
	if exists {
		m.mu.Unlock()
		return fmt.Errorf("task %s is already running", taskID)
	}

	// Get agent name from mapping
	agentName, ok := m.taskAgents[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}

	// Load task metadata
	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		m.mu.Unlock()
		return fmt.Errorf("agent %s not found", agentName)
	}

	taskMeta, err := m.loadTaskMeta(taskDir)
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to load task %s: %w", taskID, err)
	}

	// Mark as running
	taskMeta.Status = StatusRunning
	now := time.Now()
	taskMeta.StartedAt = &now
	taskMeta.UpdatedAt = now

	if err := m.saveTaskMeta(taskMeta); err != nil {
		m.mu.Unlock()
		return err
	}

	m.runningTasks[taskID] = taskMeta
	m.mu.Unlock()

	// Create context for this task
	ctx, cancel := context.WithCancel(context.Background())
	taskMeta.ctx = ctx
	taskMeta.cancelFunc = cancel

	// Start task execution in background
	go m.runTask(taskMeta)

	return nil
}

// runTask executes the task using agent's runLoop
func (m *Manager) runTask(task *TaskMeta) {
	defer func() {
		// Remove from running tasks
		m.mu.Lock()
		delete(m.runningTasks, task.ID)
		m.mu.Unlock()
	}()

	// Get the assigned agent executor
	ag, exists := m.agentExecutors[task.AgentName]
	if !exists {
		m.failTask(task.ID, fmt.Sprintf("agent %s not found", task.AgentName))
		return
	}

	taskDir := m.getTaskDir(task)

	// Notify user that task is starting
	m.notifyTaskUser(task.ID, fmt.Sprintf("📋 任务开始执行\n任务: %s\n描述: %s", task.Name, task.Description))

	// Prepare task prompt with task context
	systemPrompt := fmt.Sprintf(`You are executing a long-running task.

Task: %s
Description: %s
Task ID: %s

You should work on this task step by step. Use available tools to complete it.
Report your progress regularly.
When the task is complete, report the final result clearly.

The task workspace is: %s

Previous messages have been loaded from the task log. Continue from where we left off.`,
		task.Name, task.Description, task.ID, taskDir)

	// Load previous messages from task log
	taskMessages, err := m.loadTaskMessages(task.ID)
	if err != nil {
		m.failTask(task.ID, fmt.Sprintf("failed to load task messages: %v", err))
		return
	}

	// Build initial messages with system prompt
	msgs := make([]llm.Message, 0, len(taskMessages)+1)
	msgs = append(msgs, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Convert task messages to LLM format using the helper function
	msgs = append(msgs, convertToLLMMessages(taskMessages)...)

	// If this is a new task, add the initial user message
	if len(taskMessages) == 0 {
		initialMessage := fmt.Sprintf("Please execute this task: %s\n\nDescription: %s",
			task.Name, task.Description)
		msgs = append(msgs, llm.Message{
			Role:    "user",
			Content: initialMessage,
		})

		// Save initial message to task log
		m.appendTaskEvent(task.ID, "user", initialMessage)
	}

	// Create context for this task
	ctx := task.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Execute task loop with max iterations
	maxIterations := 20 // Limit iterations for long-running tasks
	var finalContent string

	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(1 * time.Minute)
	defer heartbeatTicker.Stop()

	for i := 0; i < maxIterations; i++ {
		// Check if task was cancelled or paused
		select {
		case <-ctx.Done():
			m.appendTaskEvent(task.ID, "system", "Task was cancelled or paused")
			return
		case <-heartbeatTicker.C:
			// Update heartbeat
			task.mu.Lock()
			task.UpdatedAt = time.Now()
			task.mu.Unlock()
			m.saveTaskMeta(task)
			continue
		default:
		}

		// Call LLM
		resp, err := ag.CallProvider(ctx, task.ID, msgs)
		if err != nil {
			m.failTask(task.ID, fmt.Sprintf("LLM call failed (iteration %d): %v", i+1, err))
			return
		}

		// Save assistant message
		assistantMsg := TaskMessage{
			Role:      resp.Message.Role,
			Content:   resp.Message.Content,
			Timestamp: time.Now(),
		}

		// Save tool calls if any
		for _, tc := range resp.Message.ToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, ToolCallInfo{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}

		m.saveTaskMessage(task.ID, assistantMsg)

		msgs = append(msgs, llm.Message{
			Role:      resp.Message.Role,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})

		// If no tool calls, task is complete
		if len(resp.Message.ToolCalls) == 0 {
			finalContent = resp.Message.Content
			break
		}

		// Execute tool calls
		for _, tc := range resp.Message.ToolCalls {
			// Notify user about tool call
			m.notifyTaskUser(task.ID, fmt.Sprintf("🔧 执行工具: %s", tc.Name))

			// Execute tool
			toolResult, err := m.executeToolCall(ctx, ag, tc, task)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
			}

			// Save tool result with ToolCallID
			resultMsg := TaskMessage{
				Role:       "tool",
				Content:    toolResult,
				Timestamp:  time.Now(),
				ToolCallID: tc.ID,
			}
			m.saveTaskMessage(task.ID, resultMsg)

			// Add tool result to messages
			msgs = append(msgs, llm.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
			})
		}

		// Notify progress periodically
		if (i+1)%3 == 0 {
			m.notifyTaskUser(task.ID, fmt.Sprintf("⏳ 任务进行中... (已完成 %d 轮迭代)", i+1))
		}
	}

	// Task completed successfully
	m.completeTask(task.ID, finalContent)
	m.notifyTaskUser(task.ID, fmt.Sprintf("✅ 任务完成\n任务: %s\n结果: %s", task.Name, finalContent))
}

// executeToolCall executes a tool call for a task
func (m *Manager) executeToolCall(ctx context.Context, ag AgentExecutor, tc llm.ToolCall, task *TaskMeta) (string, error) {
	// Parse tool arguments
	var args map[string]interface{}
	if tc.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}

	if args == nil {
		args = make(map[string]interface{})
	}

	// Get workspace for this task
	ws := m.agentWorkspaces[task.AgentName]
	if ws == "" {
		return "", fmt.Errorf("workspace not found for agent %s", task.AgentName)
	}
	args[tools.ToolCallParamWorkspace] = ws

	// Get tool registry
	toolsRegistry := ag.GetTools()

	// Create ToolContext for tools that need LLM access
	llmProvider, model := ag.GetLLMProvider()
	toolCtx := tools.NewToolContext(
		task.AgentName,
		task.OwnerID,
		ws,
		task.ID,
		"", // channel - not used in task execution
		"", // chatID - not used in task execution
		llmProvider,
		model,
	)

	// Execute tool with context
	result := toolsRegistry.ExecuteWithContext(ctx, tc.Name, args, toolCtx, "", "")

	if result.IsError {
		return result.ForLLM, fmt.Errorf("tool execution failed: %s", result.ForLLM)
	}

	return result.ForLLM, nil
}

// notifyTaskUser notifies the user who created the task about task updates
func (m *Manager) notifyTaskUser(taskID, message string) {
	m.mu.RLock()
	task, ok := m.runningTasks[taskID]
	agentName := m.taskAgents[taskID]
	m.mu.RUnlock()

	if !ok || task == nil {
		// Task might not be in runningTasks, try to load from disk
		agentName = m.taskAgents[taskID]
		if agentName == "" {
			return
		}
		taskDir := m.getTaskDirByIDs(taskID, agentName)
		if taskDir == "" {
			return
		}
		loadedTask, err := m.loadTaskMeta(taskDir)
		if err != nil {
			return
		}
		task = loadedTask
	}

	// Try to notify the owner
	if task.OwnerID != "" {
		if err := notification.GetManager().Notify(task.OwnerID, message); err == nil {
			return // Successfully notified owner
		}
	}

	// If owner notification failed, try broadcasting to active users
	count := notification.GetManager().Broadcast(message, true)
	if count > 0 {
		fmt.Printf("Task notification sent to %d active users\n", count)
	}
}

// completeTask marks a task as completed
func (m *Manager) completeTask(taskID, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return
	}

	task, err := m.loadTaskMeta(taskDir)
	if err != nil {
		return
	}

	task.Status = StatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	m.saveTaskMeta(task)

	// Append completion event
	m.appendTaskEvent(taskID, "system", fmt.Sprintf("Task completed: %s", result))

	// Notify user if needed
	// This could integrate with notification system
}

// failTask marks a task as failed
func (m *Manager) failTask(taskID, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return
	}

	task, err := m.loadTaskMeta(taskDir)
	if err != nil {
		return
	}

	task.Status = StatusFailed
	task.UpdatedAt = time.Now()

	m.saveTaskMeta(task)

	// Append failure event
	m.appendTaskEvent(taskID, "error", fmt.Sprintf("Task failed: %s", reason))
}

// PauseTask pauses a running task
func (m *Manager) PauseTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s is not running", taskID)
	}

	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	task.Status = StatusPaused
	task.UpdatedAt = time.Now()
	m.saveTaskMeta(task)

	delete(m.runningTasks, taskID)

	// Append pause event
	m.appendTaskEvent(taskID, "system", "Task paused by user")

	return nil
}

// ResumeTask resumes a paused task
func (m *Manager) ResumeTask(taskID string) error {
	m.mu.Lock()
	task, exists := m.runningTasks[taskID]
	if exists {
		m.mu.Unlock()
		return fmt.Errorf("task %s is already running", taskID)
	}
	m.mu.Unlock()

	// Get agent name from mapping
	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Load and update task
	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return fmt.Errorf("agent %s not found", agentName)
	}

	task, err := m.loadTaskMeta(taskDir)
	if err != nil {
		return err
	}

	if task.Status != StatusPaused {
		return fmt.Errorf("task %s is not paused (current status: %s)", taskID, task.Status)
	}

	// Resume by executing again
	return m.ExecuteTask(taskID)
}

// CancelTask cancels a task and cleans up
func (m *Manager) CancelTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel if running
	if task, exists := m.runningTasks[taskID]; exists {
		if task.cancelFunc != nil {
			task.cancelFunc()
		}
		delete(m.runningTasks, taskID)
	}

	// Get agent name from mapping
	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Mark as cancelled and clean up
	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return fmt.Errorf("agent %s not found", agentName)
	}

	task, err := m.loadTaskMeta(taskDir)
	if err == nil {
		task.Status = StatusCancelled
		task.UpdatedAt = time.Now()
		m.saveTaskMeta(task)
		m.appendTaskEvent(taskID, "system", "Task cancelled by user")

		// Delete task folder
		os.RemoveAll(taskDir)

		// Remove from mapping
		delete(m.taskAgents, taskID)
	}

	return nil
}

// GetTask retrieves task metadata
func (m *Manager) GetTask(taskID string) (*TaskMeta, error) {
	m.mu.RLock()
	agentName, ok := m.taskAgents[taskID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}

	return m.loadTaskMeta(taskDir)
}

// ListTasks lists all tasks based on filters
func (m *Manager) ListTasks(includeDeleted bool) ([]*TaskMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*TaskMeta, 0)

	// Scan all agents' workspaces
	for _, ws := range m.agentWorkspaces {
		tasksDir := filepath.Join(ws, "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			taskDir := filepath.Join(tasksDir, entry.Name())
			task, err := m.loadTaskMeta(taskDir)
			if err != nil {
				continue
			}

			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// GetTaskProgress returns detailed task progress
func (m *Manager) GetTaskProgress(taskID string) (string, error) {
	task, err := m.GetTask(taskID)
	if err != nil {
		return "", err
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	result := fmt.Sprintf("Task: %s (ID: %s)\n", task.Name, task.ID)
	result += fmt.Sprintf("Status: %s\n", task.Status)
	result += fmt.Sprintf("Agent: %s\n", task.AgentName)
	result += fmt.Sprintf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

	if task.StartedAt != nil {
		result += fmt.Sprintf("Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
		if task.CompletedAt != nil {
			duration := task.CompletedAt.Sub(*task.StartedAt)
			result += fmt.Sprintf("Completed: %s (duration: %s)\n",
				task.CompletedAt.Format("2006-01-02 15:04:05"),
				duration.Round(time.Second))
		}
	}

	if task.Schedule != "" {
		result += fmt.Sprintf("Schedule: %s\n", task.Schedule)
		if task.LastRun != nil {
			result += fmt.Sprintf("Last run: %s\n", task.LastRun.Format("2006-01-02 15:04:05"))
		}
		if task.NextRun != nil {
			result += fmt.Sprintf("Next run: %s\n", task.NextRun.Format("2006-01-02 15:04:05"))
		}
	}

	// Load and show recent events
	messages, err := m.loadTaskMessages(taskID)
	if err == nil && len(messages) > 0 {
		result += "\nRecent activity:\n"
		shown := 0
		maxToShow := 10

		// Show last messages in reverse order
		for i := len(messages) - 1; i >= 0 && shown < maxToShow; i-- {
			msg := messages[i]
			previewContent := msg.Content
			if len(previewContent) > 100 {
				previewContent = previewContent[:100] + "..."
			}
			result += fmt.Sprintf("  [%s] %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.Role,
				previewContent)
			shown++
		}
	}

	return result, nil
}

// AddTaskMessage adds a user message to a task (for user interaction during execution)
func (m *Manager) AddTaskMessage(taskID, message string) error {
	return m.appendTaskEvent(taskID, "user", message)
}

// saveTaskMeta saves task metadata to task.json
func (m *Manager) saveTaskMeta(task *TaskMeta) error {
	task.mu.RLock()
	defer task.mu.RUnlock()

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}

	taskDir := m.getTaskDir(task)
	if taskDir == "" {
		return fmt.Errorf("agent %s not found", task.AgentName)
	}
	taskFile := filepath.Join(taskDir, "task.json")
	return os.WriteFile(taskFile, data, 0644)
}

// SaveTaskMeta is the public version of saveTaskMeta
func (m *Manager) SaveTaskMeta(task *TaskMeta) error {
	return m.saveTaskMeta(task)
}

// loadTaskMeta loads task metadata from task.json
func (m *Manager) loadTaskMeta(taskDir string) (*TaskMeta, error) {
	taskFile := filepath.Join(taskDir, "task.json")
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, err
	}

	var task TaskMeta
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// getTaskMessagesPath returns the messages file path for a task
func (m *Manager) getTaskMessagesPath(taskID string) (string, error) {
	m.mu.RLock()
	agentName, ok := m.taskAgents[taskID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("task %s not found", taskID)
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return "", fmt.Errorf("agent %s not found", agentName)
	}

	return filepath.Join(taskDir, "messages.jsonl"), nil
}

// initMessagesLog creates an empty messages.jsonl file
func (m *Manager) initMessagesLog(taskID string) error {
	messagesPath, err := m.getTaskMessagesPath(taskID)
	if err != nil {
		return err
	}
	// Create empty file (or truncate if exists)
	file, err := os.Create(messagesPath)
	if err != nil {
		return err
	}
	return file.Close()
}

// loadTaskMessages loads all messages from messages.jsonl (streaming)
func (m *Manager) loadTaskMessages(taskID string) ([]TaskMessage, error) {
	messagesPath, err := m.getTaskMessagesPath(taskID)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty list
	if _, err := os.Stat(messagesPath); os.IsNotExist(err) {
		return []TaskMessage{}, nil
	}

	file, err := os.Open(messagesPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Stream decode line by line
	messages := []TaskMessage{}
	decoder := json.NewDecoder(file)
	for {
		var msg TaskMessage
		if err := decoder.Decode(&msg); err != nil {
			if err.Error() == "EOF" {
				break
			}
			// Skip malformed lines
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// saveTaskMessage appends a message to messages.jsonl (efficient append)
func (m *Manager) saveTaskMessage(taskID string, message TaskMessage) error {
	messagesPath, err := m.getTaskMessagesPath(taskID)
	if err != nil {
		return err
	}

	// Open file in append mode
	file, err := os.OpenFile(messagesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open messages file: %w", err)
	}
	defer file.Close()

	// Encode and append message as a single line
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(message); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return nil
}

// appendTaskEvent adds an event to the task log
func (m *Manager) appendTaskEvent(taskID, eventType, message string) error {
	msg := TaskMessage{
		Role:      eventType,
			Content:   message,
		Timestamp: time.Now(),
	}

	return m.saveTaskMessage(taskID, msg)
}

// convertToLLMMessages converts task messages to LLM format
func convertToLLMMessages(messages []TaskMessage) []llm.Message {
	result := make([]llm.Message, 0, len(messages))

	for i, msg := range messages {
		llmMsg := llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Add ToolCallID if present (for tool result messages)
		if msg.ToolCallID != "" {
			llmMsg.ToolCallID = msg.ToolCallID
		}

		// Convert tool calls if any
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]llm.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, llm.ToolCall{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
			llmMsg.ToolCalls = toolCalls
		}

		result[i] = llmMsg
	}

	return result
}

// scanAndResumeTasks scans task folders from all agents and resumes interrupted tasks
func (m *Manager) scanAndResumeTasks() error {
	// Scan each agent's workspace
	for agentName, ws := range m.agentWorkspaces {
		tasksDir := filepath.Join(ws, "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Printf("Warning: failed to read tasks directory for agent %s: %v\n", agentName, err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			taskDir := filepath.Join(tasksDir, entry.Name())
			task, err := m.loadTaskMeta(taskDir)
			if err != nil {
				fmt.Printf("Warning: failed to load task from %s: %v\n", taskDir, err)
				continue
			}

			// Store taskID -> agentName mapping
			m.taskAgents[task.ID] = agentName

			// If task was running, check if it's a zombie or resume it
			if task.Status == StatusRunning {
				// Check if task is a zombie
				if m.isZombieTask(task) {
					m.markTaskAsFailed(task.ID, "Task detected as zombie (no heartbeat for over 5 minutes)")
					continue
				}
				// Resume non-zombie running tasks
				fmt.Printf("Resuming interrupted task: %s (%s) for agent %s\n", task.Name, task.ID, agentName)
				m.ExecuteTask(task.ID)
			}
		}
	}

	return nil
}

// periodicCheck periodically checks for pending/incomplete tasks and recurring tasks
func (m *Manager) periodicCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()

		// Collect all tasks from all agents
		allTasks := make([]*TaskMeta, 0)
		for _, ws := range m.agentWorkspaces {
			tasksDir := filepath.Join(ws, "tasks")
			entries, err := os.ReadDir(tasksDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				taskDir := filepath.Join(tasksDir, entry.Name())
				task, err := m.loadTaskMeta(taskDir)
				if err != nil {
					continue
				}
				allTasks = append(allTasks, task)
			}
		}

		// Process tasks
		for _, task := range allTasks {
			// Resume interrupted running tasks
			if task.Status == StatusRunning {
				// Check if task is a zombie
				if _, exists := m.runningTasks[task.ID]; !exists {
					if m.isZombieTask(task) {
						m.markTaskAsFailed(task.ID, "Task detected as zombie (not in running tasks map)")
						continue
					}
					// Resume non-zombie running tasks
					fmt.Printf("Resuming task: %s\n", task.Name)
					m.mu.Unlock()
					go m.ExecuteTask(task.ID)
					m.mu.Lock()
				}
			}

			// Check for recurring tasks that need to run
			if task.Schedule != "" {
				if m.shouldRunRecurringTask(task) {
					fmt.Printf("Starting recurring task: %s\n", task.Name)

					// For recurring tasks, we might want to clone the task
					// For now, just execute it
					m.mu.Unlock()
					go m.ExecuteTask(task.ID)
					m.mu.Lock()
				}
			}
		}

		m.mu.Unlock()
	}
}

// shouldRunRecurringTask checks if a recurring task should run now
func (m *Manager) shouldRunRecurringTask(task *TaskMeta) bool {
	// Simple implementation: check if NextRun is passed
	// In production, this would parse the cron schedule
	if task.NextRun == nil {
		return false
	}

	return time.Now().After(*task.NextRun)
}

// isZombieTask checks if a task is a zombie process
func (m *Manager) isZombieTask(task *TaskMeta) bool {
	if task.Status != StatusRunning {
		return false
	}
	if task.UpdatedAt.IsZero() {
		return false
	}
	return time.Since(task.UpdatedAt) > m.zombieTaskThreshold
}

// markTaskAsFailed marks a task as failed with a reason
func (m *Manager) markTaskAsFailed(taskID, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return
	}

	task, err := m.loadTaskMeta(taskDir)
	if err != nil {
		return
	}

	task.Status = StatusFailed
	task.UpdatedAt = time.Now()

	if err := m.saveTaskMeta(task); err != nil {
		fmt.Printf("Failed to mark zombie task %s as failed: %v\n", taskID, err)
		return
	}

	m.appendTaskEvent(taskID, "error", fmt.Sprintf("Task marked as failed: %s", reason))
	fmt.Printf("Zombie task %s marked as failed: %s\n", taskID, reason)
}

// DeleteCompletedTask removes a completed task folder
func (m *Manager) DeleteCompletedTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentName, ok := m.taskAgents[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	taskDir := m.getTaskDirByIDs(taskID, agentName)
	if taskDir == "" {
		return fmt.Errorf("agent %s not found", agentName)
	}

	// Check if task is completed
	task, err := m.loadTaskMeta(taskDir)
	if err != nil {
		return err
	}

	if task.Status != StatusCompleted {
		return fmt.Errorf("task %s is not completed (status: %s)", taskID, task.Status)
	}

	// Delete the task folder and remove from mapping
	delete(m.taskAgents, taskID)
	return os.RemoveAll(taskDir)
}

// CleanupOldTasks removes completed tasks older than specified duration
func (m *Manager) CleanupOldTasks(maxAge time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	now := time.Now()

	// Scan all agents' workspaces
	for _, ws := range m.agentWorkspaces {
		tasksDir := filepath.Join(ws, "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			taskDir := filepath.Join(tasksDir, entry.Name())
			task, err := m.loadTaskMeta(taskDir)
			if err != nil {
				continue
			}

			// Delete completed tasks older than maxAge
			if task.Status == StatusCompleted && task.CompletedAt != nil {
				if now.Sub(*task.CompletedAt) > maxAge {
					if err := os.RemoveAll(taskDir); err == nil {
						delete(m.taskAgents, task.ID)
						count++
					}
				}
			}
		}
	}

	return count, nil
}
