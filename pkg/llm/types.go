package llm

type Message struct {
	Role      string
	Content   string         // 纯文本内容（向后兼容）
	Contents  []ContentBlock // 多内容块（可选，用于多模态消息）
	ToolCalls []ToolCall
}

type ContentBlock struct {
	Type      string // "text", "image"
	Text      string // 文本内容（Type="text"时使用）
	ImageData string // 图片base64数据（Type="image"时使用）
	MediaType string // 图片MIME类型，如 "image/png", "image/jpeg"
}

func NewTextContent(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

func NewImageContent(base64Data, mediaType string) ContentBlock {
	return ContentBlock{Type: "image", ImageData: base64Data, MediaType: mediaType}
}

func (m *Message) HasImage() bool {
	for _, c := range m.Contents {
		if c.Type == "image" {
			return true
		}
	}
	return false
}

func (m *Message) GetTextContent() string {
	if m.Content != "" {
		return m.Content
	}
	for _, c := range m.Contents {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON string
}

type ToolDefinition struct {
	Type     string                 `json:"type"` // "function"
	Function ToolFunctionDefinition `json:"function"`
}

type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatResponse struct {
	Message      Message
	Usage        Usage
	FinishReason string
}

type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

// JSONSchema represents a JSON Schema for structured output
type JSONSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
	Strict      bool           `json:"strict,omitempty"`
}
