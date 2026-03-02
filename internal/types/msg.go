package types

import "time"

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
	ToolCalls []ToolCall
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
	Result    string
}
