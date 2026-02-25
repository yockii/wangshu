package agent

import (
	"sync"
	"time"
)

type EventType string

const (
	EventTypeToolStart EventType = "tool_start"
	EventTypeToolEnd   EventType = "tool_end"
	EventTypeToolError EventType = "tool_error"
	EventTypeTextDelta EventType = "text_delta"
	EventTypeThinking  EventType = "thinking"
	EventTypeLifecycle EventType = "lifecycle"
)

type Event struct {
	Type      EventType `json:"type"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
type ToolStartData struct {
	ToolName   string `json:"tool_name"`
	ToolCallID string `json:"tool_call_id"`
	Args       any    `json:"args,omitempty"`
}
type ToolEndData struct {
	ToolName   string `json:"tool_name"`
	ToolCallID string `json:"tool_call_id"`
	Result     string `json:"result,omitempty"`
	IsError    bool   `json:"is_error"`
}
type LifecycleData struct {
	Phase   string `json:"phase"`
	Message string `json:"message,omitempty"`
}
type EventEmitter struct {
	mu        sync.RWMutex
	listeners []chan Event
}

var globalEmitter = &EventEmitter{}

func GetEventEmitter() *EventEmitter {
	return globalEmitter
}

func (e *EventEmitter) Subscribe() chan Event {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan Event, 100)
	e.listeners = append(e.listeners, ch)
	return ch
}

func (e *EventEmitter) SubscribeToChannel(ch chan Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.listeners = append(e.listeners, ch)
}

func (e *EventEmitter) Unsubscribe(ch chan Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, listener := range e.listeners {
		if listener == ch {
			e.listeners = append(e.listeners[:i], e.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

func (e *EventEmitter) Emit(event Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	event.Timestamp = time.Now()

	for _, listener := range e.listeners {
		select {
		case listener <- event:
		default:
		}
	}
}

func EmitToolStart(sessionID, toolName, toolCallID string, args interface{}) {
	GetEventEmitter().Emit(Event{
		Type:      EventTypeToolStart,
		SessionID: sessionID,
		Data: ToolStartData{
			ToolName:   toolName,
			ToolCallID: toolCallID,
			Args:       args,
		},
	})
}

func EmitToolEnd(sessionID, toolName, toolCallID, result string, isError bool) {
	GetEventEmitter().Emit(Event{
		Type:      EventTypeToolEnd,
		SessionID: sessionID,
		Data: ToolEndData{
			ToolName:   toolName,
			ToolCallID: toolCallID,
			Result:     result,
			IsError:    isError,
		},
	})
}

func EmitLifecycle(sessionID, phase, message string) {
	GetEventEmitter().Emit(Event{
		Type:      EventTypeLifecycle,
		SessionID: sessionID,
		Data: LifecycleData{
			Phase:   phase,
			Message: message,
		},
	})
}
