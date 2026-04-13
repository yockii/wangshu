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

type StructuredResponse struct {
	Content           string `json:"content" jsonschema:"title=对话内容,description=给用户的直接回复文本"`
	Emotion           string `json:"emotion" jsonschema:"description=当前的情绪状态,enum=happy,enum=sad,enum=angry,enum=neutral,enum=excited"`
	InternalMonologue string `json:"internal_monologue,omitempty" jsonschema:"description=内心独白，仅AI内部思考，不展示给用户"`
}
