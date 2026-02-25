package tools

type ToolResult struct {
	ForLLM  string `json:"for_llm"`
	ForUser string `json:"for_user"`
	Silent  bool   `json:"silent"`
	IsError bool   `json:"is_error"`
	Async   bool   `json:"async"`
	Err     error  `json:"-"`
}

func NewToolResult(forLLM string) *ToolResult {
	return &ToolResult{ForLLM: forLLM}
}

func SilentResult(forLLM string) *ToolResult {
	return &ToolResult{
		ForLLM: forLLM,
		Silent: true,
	}
}

func AsyncResult(forLLM string) *ToolResult {
	return &ToolResult{
		ForLLM:  forLLM,
		ForUser: forLLM, // 异步任务同样也设置给用户消息
		Async:   true,
	}
}

func ErrorResult(message string) *ToolResult {
	return &ToolResult{
		ForLLM:  message,
		IsError: true,
	}
}

func UserResult(content string) *ToolResult {
	return &ToolResult{
		ForLLM:  content,
		ForUser: content, // 异步任务同样也设置给用户消息
	}
}

func (tr *ToolResult) WithError(err error) *ToolResult {
	tr.Err = err
	return tr
}
