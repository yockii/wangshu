package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yockii/wangshu/pkg/constant"
	"gopkg.in/yaml.v3"
)

type Action struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Description  string            `yaml:"description"`
	Capabilities []string          `yaml:"capabilities"`
	Inputs       map[string]string `yaml:"inputs"`  // key->type，可在执行时做类型检查
	Outputs      map[string]string `yaml:"outputs"` // key->type
	Config       ActionConfig      `yaml:"config"`
	Steps        []Step            `yaml:"steps"`
	Doc          string            `yaml:"doc"` // Markdown 说明
}

type ActionConfig struct {
	MaxSteps int    `yaml:"max_steps"`
	MaxLoop  int    `yaml:"max_loop"`
	Timeout  string `yaml:"timeout"`
}

func (a *Action) Execute(inputs map[string]any) (*ExecutionContext, error) {
	if err := a.ValidateCapabilities(); err != nil {
		return nil, err
	}

	ctx := NewExecutionContext(context.Background(), inputs)
	maxSteps := a.Config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 50
	}

	stepCount := 0
	for _, step := range a.Steps {
		step.ctx = ctx
		err := step.Do()
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.ID, err)
		}
		stepCount++
		if stepCount >= maxSteps {
			return ctx, errors.New("max steps exceeded")
		}
	}
	return ctx, nil
}

func (a *Action) ValidateCapabilities() error {
	for _, cap := range a.Capabilities {
		if _, ok := toolMapper[cap]; !ok {
			return fmt.Errorf("capability %s not supported", cap)
		}
	}
	return nil
}

func ParseActionFromMarkdown(md string) (*Action, error) {
	matches := constant.MdFrontmatterReg.FindStringSubmatch(md)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid frontmatter")
	}
	action := &Action{}
	if err := yaml.NewDecoder(strings.NewReader(matches[1])).Decode(action); err != nil {
		return nil, fmt.Errorf("unmarshal frontmatter failed: %w", err)
	}
	action.Doc = md[len(matches[1]):]
	return action, nil
}
