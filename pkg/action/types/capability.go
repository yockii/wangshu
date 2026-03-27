package types

type ActionOutput struct {
	Status  string `json:"status"` // success / failed
	Message string `json:"message"`
	Data    any    `json:"data"`  // 不同的capability有不同的数据结构
	Trace   any    `json:"trace"` // 内部调试跟踪信息
}

func NewActionOutput(status string, message string, data any, trace any) *ActionOutput {
	return &ActionOutput{
		Status:  status,
		Message: message,
		Data:    data,
		Trace:   trace,
	}
}
