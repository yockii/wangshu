package action

import (
	"context"
	"errors"

	"github.com/yockii/wangshu/pkg/action/types"
)

type ExecutionContext struct {
	context.Context
	Inputs    map[string]any
	Steps     map[string]*types.ActionOutput
	Variables map[string]any
}

func NewExecutionContext(ctx context.Context, inputs map[string]any) *ExecutionContext {
	return &ExecutionContext{
		Inputs:    inputs,
		Steps:     make(map[string]*types.ActionOutput),
		Variables: make(map[string]any),
		Context:   ctx,
	}
}

// 获取变量（支持 input.xxx、variables.xxx、steps.stepID）
func (ec *ExecutionContext) GetVariable(key string) (any, error) {
	// 支持类似 "input.query" 或 "steps.search.structured.results"
	if key == "" {
		return nil, errors.New("key empty")
	}

	// 简单解析前缀
	switch {
	case len(key) > 6 && key[:6] == "input.":
		k := key[6:]
		return ec.Inputs[k], nil
	case len(key) > 6 && key[:6] == "steps.":
		// 支持 steps.<stepID>
		// 简单拆分
		var stepID string
		parts := []rune(key[6:])
		for i, r := range parts {
			if r == '.' {
				stepID = string(parts[:i])
				break
			}
		}
		step, ok := ec.Steps[stepID]
		if !ok {
			return nil, nil
		}
		return step.Data, nil
	default:
		if v, ok := ec.Variables[key]; ok {
			return v, nil
		}
		return nil, nil
	}
}

func (ec *ExecutionContext) SetVariable(key string, value any) {
	ec.Variables[key] = value
}

func (ec *ExecutionContext) SetStepOutput(stepID string, output *types.ActionOutput) {
	ec.Steps[stepID] = output
}
