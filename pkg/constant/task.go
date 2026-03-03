package constant

const (
	TaskRelationsFileName    = "taskRelations.json"
	TaskInfoFileName         = "task.json"
	TaskChangeLogFileName    = "changeLog.json"
	TaskSubtasksInfoFileName = "subtasks.json"
	TaskHistoryFileName      = "history.jsonl"
)

const (
	TaskTagCompleted = "TASK_COMPLETED"
)

const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusCancelled = "cancelled"

	TaskStatusFailed = "failed"
	TaskStatusRemove = "remove"
)

const (
	TaskPriorityLow    = "low"
	TaskPriorityNormal = "normal"
	TaskPriorityHigh   = "high"
	TaskPriorityUrgent = "urgent"
)
const (
	TaskActionCreate    = "create"
	TaskActionList      = "list"
	TaskActionStatus    = "status"
	TaskActionCancel    = "cancel"
	TaskActionClean     = "clean"
	TaskActionRestart   = "restart"
	TaskActionUpdate    = "update"
	TaskActionAddChange = "add_change"
)
