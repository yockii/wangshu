package types

type ToolResult struct {
	Structured map[string]any `json:"structured"`
	Raw        string         `json:"raw"`
	Silent     bool           `json:"silent"`
	Async      bool           `json:"async"`
	Err        error          `json:"-"`
}

func NewToolResult() *ToolResult {
	return &ToolResult{}
}

func (tr *ToolResult) WithRaw(raw string) *ToolResult {
	tr.Raw = raw
	return tr
}

func (tr *ToolResult) WithStructured(structured map[string]any) *ToolResult {
	tr.Structured = structured
	return tr
}

func (tr *ToolResult) WithSilent(silent bool) *ToolResult {
	tr.Silent = silent
	return tr
}

func (tr *ToolResult) WithAsync(async bool) *ToolResult {
	tr.Async = async
	return tr
}

func (tr *ToolResult) WithError(err error) *ToolResult {
	tr.Err = err
	return tr
}
