package tools

import (
	"context"
	"fmt"
	"time"
)

func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&SleepTool{})
	registry.Register(&GetTimeTool{})

}

// SleepTool pauses execution for a specified duration
type SleepTool struct{}

func (t *SleepTool) Name() string {
	return "sleep"
}

func (t *SleepTool) Description() string {
	return "Pauses execution for a specified number of seconds. Useful for testing delays."
}

func (t *SleepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"seconds": map[string]interface{}{
				"type":        "number",
				"description": "Number of seconds to sleep",
			},
		},
		"required": []string{"seconds"},
	}
}

func (t *SleepTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	secondsStr, ok := params["seconds"]
	if !ok {
		return "", fmt.Errorf("missing required parameter: seconds")
	}

	var seconds float64
	n, err := fmt.Sscanf(secondsStr, "%f", &seconds)
	if err != nil || n != 1 {
		return "", fmt.Errorf("invalid seconds value: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(time.Duration(seconds*1000) * time.Millisecond):
		return fmt.Sprintf("Slept for %v seconds", seconds), nil
	}
}

// GetTimeTool returns the current time
type GetTimeTool struct{}

func (t *GetTimeTool) Name() string {
	return "get_time"
}

func (t *GetTimeTool) Description() string {
	return "Returns the current time in ISO 8601 format. Useful for timestamping or time-aware operations."
}

func (t *GetTimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *GetTimeTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
