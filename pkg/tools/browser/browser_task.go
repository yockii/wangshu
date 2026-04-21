package browser

import (
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

var varPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

type TaskScript struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Steps       []Step `json:"steps"`
}

type Step struct {
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Detect      *DetectCondition       `json:"detect,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Fields      map[string]FieldConfig `json:"fields,omitempty"`
	Check       *CheckCondition        `json:"check,omitempty"`
	Then        []Step                 `json:"then,omitempty"`
	Else        []Step                 `json:"else,omitempty"`
	OnError     string                 `json:"on_error,omitempty"`
}

type DetectCondition struct {
	Condition string `json:"condition"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
}

type CheckCondition struct {
	Selector     string `json:"selector,omitempty"`
	Label        string `json:"label,omitempty"`
	Text         string `json:"text,omitempty"`
	Role         string `json:"role,omitempty"`
	RoleName     string `json:"role_name,omitempty"`
	TestID       string `json:"testid,omitempty"`
	Placeholder  string `json:"placeholder,omitempty"`
	Exists       *bool  `json:"exists,omitempty"`
	Visible      *bool  `json:"visible,omitempty"`
	TextEquals   string `json:"text_equals,omitempty"`
	TextContains string `json:"text_contains,omitempty"`
}

type FieldConfig struct {
	Selector    string                 `json:"selector,omitempty"`
	Label       string                 `json:"label,omitempty"`
	Text        string                 `json:"text,omitempty"`
	Role        string                 `json:"role,omitempty"`
	RoleName    string                 `json:"role_name,omitempty"`
	TestID      string                 `json:"testid,omitempty"`
	Placeholder string                 `json:"placeholder,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Alt         string                 `json:"alt,omitempty"`
	Attr        string                 `json:"attr"`
	Type        string                 `json:"type,omitempty"`
	Container   string                 `json:"container,omitempty"`
	Fields      map[string]FieldConfig `json:"fields,omitempty"`
}

type TaskResult struct {
	Success       bool                   `json:"success"`
	TaskName      string                 `json:"task_name"`
	Duration      string                 `json:"duration"`
	StepsExecuted int                    `json:"steps_executed"`
	CurrentStep   string                 `json:"current_step,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Screenshot    string                 `json:"screenshot,omitempty"`
}

type TaskEngine struct {
	tool      *BrowserTool
	result    *TaskResult
	startAt   time.Time
	variables map[string]string
}

func NewTaskEngine(tool *BrowserTool) *TaskEngine {
	return &TaskEngine{
		tool: tool,
		result: &TaskResult{
			Data: make(map[string]interface{}),
		},
		variables: make(map[string]string),
	}
}

// SetVariable 设置变量
func (e *TaskEngine) SetVariable(name, value string) {
	e.variables[name] = value
}

// SetVariables 批量设置变量
func (e *TaskEngine) SetVariables(vars map[string]string) {
	for k, v := range vars {
		e.variables[k] = v
	}
}

func (e *TaskEngine) Execute(script *TaskScript) *TaskResult {
	e.startAt = time.Now()
	e.result.TaskName = script.Name
	e.result.Success = false

	defer func() {
		e.result.Duration = time.Since(e.startAt).String()
	}()

	if err := e.tool.ensureBrowserReady(); err != nil {
		e.result.Error = "初始化浏览器失败: " + err.Error()
		return e.result
	}

	e.tool.page.Context().GrantPermissions([]string{"clipboard-read"})
	e.tool.page.Context().Route("**/*", func(route playwright.Route) {
		req := route.Request()
		if strings.Contains(req.URL(), "google-analytics.com") {
			route.Abort()
			return
		}
		route.Continue()
	})

	for i, step := range script.Steps {
		// 替换步骤中的变量
		step = e.replaceStepVariables(step)

		e.result.CurrentStep = step.ID
		e.result.StepsExecuted = i + 1

		if err := e.executeStep(step); err != nil {
			e.result.Error = err.Error()
			if step.OnError == "continue" {
				continue
			}
			if screenshotPath, screenshotErr := e.takeErrorScreenshot(step.ID); screenshotErr == nil {
				e.result.Screenshot = screenshotPath
			}
			return e.result
		}
	}

	e.result.Success = true
	return e.result
}

func (e *TaskEngine) executeStep(step Step) error {
	switch step.Action {
	case "open":
		return e.actionOpen(step)
	case "click":
		return e.actionClick(step)
	case "fill":
		return e.actionFill(step)
	case "wait":
		return e.actionWait(step)
	case "wait_for_user":
		return e.actionWaitForUser(step)
	case "extract":
		return e.actionExtract(step)
	case "screenshot":
		return e.actionScreenshot(step)
	case "scroll":
		return e.actionScroll(step)
	case "hover":
		return e.actionHover(step)
	case "select":
		return e.actionSelect(step)
	case "condition":
		return e.actionCondition(step)
	case "goto":
		return e.actionGoto(step)
	case "back":
		return e.actionBack(step)
	case "refresh":
		return e.actionRefresh(step)
	case "clipboard":
		return e.actionClipboard(step)
	default:
		return &StepError{StepID: step.ID, Message: "未知 action: " + step.Action}
	}
}

type StepError struct {
	StepID  string
	Message string
}

func (e *StepError) Error() string {
	return "步骤[" + e.StepID + "] " + e.Message
}

// replaceStepVariables 替换步骤中的所有变量
func (e *TaskEngine) replaceStepVariables(step Step) Step {
	step.ID = e.replaceVariables(step.ID)
	step.Description = e.replaceVariables(step.Description)
	step.Params = e.replaceParamsVariables(step.Params)
	step.Fields = e.replaceFieldsVariables(step.Fields)
	step.Check = e.replaceCheckVariables(step.Check)
	step.Detect = e.replaceDetectVariables(step.Detect)
	return step
}

// replaceVariables 替换字符串中的变量，格式为 ${var_name}
func (e *TaskEngine) replaceVariables(s string) string {
	if s == "" {
		return s
	}
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		submatches := varPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := submatches[1]
		defaultValue := ""
		if len(submatches) > 2 {
			defaultValue = submatches[2]
		}

		if value, ok := e.variables[varName]; ok && value != "" {
			return value
		}
		return defaultValue
	})
}

// replaceParamsVariables 替换参数中的变量
func (e *TaskEngine) replaceParamsVariables(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = e.replaceVariables(val)
		case map[string]interface{}:
			result[k] = e.replaceParamsVariables(val)
		default:
			result[k] = v
		}
	}
	return result
}

// replaceFieldsVariables 替换字段配置中的变量
func (e *TaskEngine) replaceFieldsVariables(fields map[string]FieldConfig) map[string]FieldConfig {
	if fields == nil {
		return nil
	}
	result := make(map[string]FieldConfig)
	for k, v := range fields {
		v.Selector = e.replaceVariables(v.Selector)
		v.Label = e.replaceVariables(v.Label)
		v.Text = e.replaceVariables(v.Text)
		v.Attr = e.replaceVariables(v.Attr)
		v.Container = e.replaceVariables(v.Container)
		v.TestID = e.replaceVariables(v.TestID)
		v.Placeholder = e.replaceVariables(v.Placeholder)
		v.Title = e.replaceVariables(v.Title)
		v.Alt = e.replaceVariables(v.Alt)
		v.Fields = e.replaceFieldsVariables(v.Fields)
		result[k] = v
	}
	return result
}

// replaceCheckVariables 替换检查条件中的变量
func (e *TaskEngine) replaceCheckVariables(check *CheckCondition) *CheckCondition {
	if check == nil {
		return nil
	}
	check.Selector = e.replaceVariables(check.Selector)
	check.Label = e.replaceVariables(check.Label)
	check.Text = e.replaceVariables(check.Text)
	check.Role = e.replaceVariables(check.Role)
	check.RoleName = e.replaceVariables(check.RoleName)
	check.TestID = e.replaceVariables(check.TestID)
	check.Placeholder = e.replaceVariables(check.Placeholder)
	check.TextEquals = e.replaceVariables(check.TextEquals)
	check.TextContains = e.replaceVariables(check.TextContains)
	return check
}

// replaceDetectVariables 替换检测条件中的变量
func (e *TaskEngine) replaceDetectVariables(detect *DetectCondition) *DetectCondition {
	if detect == nil {
		return nil
	}
	detect.Condition = e.replaceVariables(detect.Condition)
	detect.Selector = e.replaceVariables(detect.Selector)
	detect.Value = e.replaceVariables(detect.Value)
	detect.From = e.replaceVariables(detect.From)
	detect.To = e.replaceVariables(detect.To)
	return detect
}
