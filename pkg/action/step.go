package action

import (
	"bytes"
	"errors"
	"html/template"
)

type Step struct {
	ID       string         `yaml:"id"`
	Use      string         `yaml:"use"`
	With     map[string]any `yaml:"with"`
	ForEach  string         `yaml:"foreach"`
	MaxLoop  int            `yaml:"max_loop"`
	AssignTo string         `yaml:"assign_to"`
	Retry    int            `yaml:"retry"`

	ctx *ExecutionContext
}

func NewStep(ctx *ExecutionContext) *Step {
	return &Step{
		ctx: ctx,
	}
}

func (s *Step) Do() error {
	if s.ID == "" {
		return errors.New("id is required")
	}
	if s.Use == "" {
		return errors.New("use is required")
	}

	if s.ForEach != "" {
		// 从 context 或上一步输出中获取列表
		list, err := s.resolveForEachList(s.ForEach)
		if err != nil {
			return err
		}

		maxLoop := s.MaxLoop
		if maxLoop <= 0 {
			maxLoop = 10
		}

		var results []Output
		for i, item := range list {
			if i >= maxLoop {
				break
			}

			out, err := s.callTool(item)
			if err != nil {
				return err
			}
			results = append(results, out)
		}

		finalOutput := Output{
			Structured: results,
			Raw:        "", // 可根据需要拼接 raw
		}
		s.ctx.SetStepOutput(s.ID, finalOutput)
		if s.AssignTo != "" {
			s.ctx.SetVariable(s.AssignTo, finalOutput)
		}

		return nil
	}

	// 单次执行
	output, err := s.callTool(nil)
	if err != nil {
		return err
	}

	s.ctx.SetStepOutput(s.ID, output)
	if s.AssignTo != "" {
		s.ctx.SetVariable(s.AssignTo, output)
	}

	return nil
}

func (s *Step) callTool(item any) (Output, error) {
	tool, ok := toolMapper[s.Use]
	if !ok {
		return Output{}, errors.New("use is not a valid tool")
	}
	// 模板替换
	stepWith := make(map[string]any, len(s.With))
	for k, v := range s.With {
		stepWith[k] = v
	}
	if item != nil {
		stepWith = s.substituteItem(stepWith, item)
	} else {
		stepWith = s.substituteItem(stepWith, nil)
	}
	var output Output
	var err error
	for attempt := 0; attempt <= s.Retry; attempt++ {
		output, err = tool(stepWith)
		if err == nil {
			break
		}
	}
	return output, err
}

func (s *Step) resolveForEachList(foreach string) ([]any, error) {
	val, err := s.ctx.GetVariable(foreach)
	if err != nil {
		return nil, err
	}

	list, ok := val.([]any)
	if !ok {
		return nil, errors.New("foreach variable is not a list")
	}
	return list, nil
}

// substituteItem 替换模板 {{item.xxx}} {{input.xxx}} {{steps.xxx.structured.xxx}}
func (s *Step) substituteItem(input map[string]any, item any) map[string]any {
	res := make(map[string]any, len(input))
	for k, v := range input {
		str, ok := v.(string)
		if ok {
			// 使用模板引擎替换
			tmpl, err := template.New("").Parse(str)
			if err != nil {
				res[k] = v
				continue
			}
			var buf bytes.Buffer
			_ = tmpl.Execute(&buf, map[string]any{
				"item":  item,
				"input": s.ctx.Inputs,
				"steps": s.ctx.Steps,
			})
			res[k] = buf.String()
		} else {
			res[k] = v
		}
	}
	return res
}
