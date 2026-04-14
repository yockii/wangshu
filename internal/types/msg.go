package types

import "time"

type Message struct {
	Role      string
	Content   string         // 纯文本内容（向后兼容）
	Contents  []ContentBlock // 多内容块（可选，用于多模态消息）
	Timestamp time.Time
	ToolCalls []ToolCall
}

type ContentBlock struct {
	Type      string // "text", "image"
	Text      string // 文本内容（Type="text"时使用）
	ImageData string // 图片base64数据（Type="image"时使用）
	MediaType string // 图片MIME类型，如 "image/png", "image/jpeg"
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
	Result    string
}
