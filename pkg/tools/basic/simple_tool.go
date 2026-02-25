package basic

import "context"

// SimpleTool is a helper for creating simple tools
type SimpleTool struct {
	Name_    string
	Desc_    string
	Params_  map[string]interface{}
	ExecFunc func(ctx context.Context, params map[string]string) (string, error)
}

// Name returns the tool name
func (t *SimpleTool) Name() string {
	return t.Name_
}

// Description returns the tool description
func (t *SimpleTool) Description() string {
	return t.Desc_
}

// Parameters returns the tool parameters schema
func (t *SimpleTool) Parameters() map[string]interface{} {
	return t.Params_
}

// Execute runs the tool
func (t *SimpleTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	if t.ExecFunc == nil {
		return "", nil
	}
	return t.ExecFunc(ctx, params)
}
