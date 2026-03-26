package action

type Output struct {
	Structured any    `json:"structured"`
	Raw        string `json:"raw"`
}

type ToolFunc func(params map[string]any) (Output, error)

var toolMapper = make(map[string]ToolFunc)

func init() {
	toolMapper["web.search"] = func(params map[string]any) (Output, error) {
		return Output{
			Structured: []any{
				map[string]any{"title": "结果1", "url": "https://a.com"},
				map[string]any{"title": "结果2", "url": "https://b.com"},
			},
			Raw: "1. 结果1(https://a.com)\n2. 结果2(https://b.com)",
		}, nil
	}
	toolMapper["web.fetch"] = func(params map[string]any) (Output, error) {
		return Output{
			Structured: map[string]any{
				"url":     "https://www.example.com",
				"title":   "Example Title",
				"content": "Example Content",
			},
			Raw: "Example Content",
		}, nil
	}
	toolMapper["llm.generate"] = func(params map[string]any) (Output, error) {
		return Output{
			Raw: "Example Content",
		}, nil
	}
}
