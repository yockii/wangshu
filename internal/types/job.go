package types

import "time"

type BasicJobInfo struct {
	ID          string     `json:"id"`
	Schedule    string     `json:"schedule"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Channel     string     `json:"channel"`
	ChatID      string     `json:"chat_id"`
	Once        bool       `json:"once"` // 是否只执行一次
}
