package browser

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
)

func (e *TaskEngine) actionOpen(step Step) error {
	url, ok := step.Params["url"].(string)
	if !ok || url == "" {
		return &StepError{StepID: step.ID, Message: "缺少 url 参数"}
	}

	_, err := e.tool.page.Goto(url)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "打开页面失败: " + err.Error()}
	}

	e.tool.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	return nil
}

func (e *TaskEngine) actionClick(step Step) error {
	locator, err := e.resolveLocator(step)
	if err != nil {
		return err
	}

	err = locator.Click()
	if err != nil {
		return &StepError{StepID: step.ID, Message: "点击失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionFill(step Step) error {
	value, ok := step.Params["value"].(string)
	if !ok {
		return &StepError{StepID: step.ID, Message: "缺少 value 参数"}
	}

	timeout := float64(30000)
	switch t := step.Params["timeout"].(type) {
	case float64:
		timeout = t
	case int:
		timeout = float64(t)
	}

	locator, err := e.resolveLocator(step)
	if err != nil {
		return err
	}

	err = locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "等待元素可见失败: " + err.Error()}
	}

	err = locator.Click()
	if err != nil {
		return &StepError{StepID: step.ID, Message: "聚焦元素失败: " + err.Error()}
	}

	err = locator.Fill(value)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "填充失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionWait(step Step) error {
	timeout := float64(30000)
	switch t := step.Params["timeout"].(type) {
	case float64:
		timeout = t
	case int:
		timeout = float64(t)
	}

	// 如果指定了 duration，直接等待指定时间（毫秒）
	switch duration := step.Params["duration"].(type) {
	case float64:
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil
	case int:
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil
	}

	// 如果没有选择器参数，也直接等待 timeout 时间
	hasSelector := step.Params["selector"] != nil ||
		step.Params["text"] != nil ||
		step.Params["label"] != nil ||
		step.Params["role"] != nil ||
		step.Params["placeholder"] != nil ||
		step.Params["testid"] != nil

	if !hasSelector {
		time.Sleep(time.Duration(timeout) * time.Millisecond)
		return nil
	}

	// 等待元素
	locator, err := e.resolveLocator(step)
	if err != nil {
		return err
	}

	// 支持指定等待状态：visible（默认）、hidden、attached、detached
	state := playwright.WaitForSelectorStateVisible
	if s, ok := step.Params["state"].(string); ok {
		switch s {
		case "hidden":
			state = playwright.WaitForSelectorStateHidden
		case "attached":
			state = playwright.WaitForSelectorStateAttached
		case "detached":
			state = playwright.WaitForSelectorStateDetached
		}
	}

	err = locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   state,
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "等待元素超时: " + err.Error()}
	}

	return nil
}

// resolveLocator 解析定位器，使用 Playwright 原生 Locator API
// 优先级：selector > label > text > role > placeholder > testid > alt > title > near
// within 参数可限定所有定位方式的搜索范围
// within_index 参数可指定第几个 within 元素
// index 参数可指定第几个元素：first（默认）、last、或数字索引（从0开始）
func (e *TaskEngine) resolveLocator(step Step) (playwright.Locator, error) {
	page := e.tool.page

	// 获取搜索范围（within 参数对所有定位方式生效）
	var scopeLocator playwright.Locator
	if within, ok := step.Params["within"].(string); ok && within != "" {
		scopeLocator = page.Locator(within)
		// 支持 within_index 选择第几个 within 容器
		scopeLocator = e.applyWithinIndex(scopeLocator, step)
	} else {
		scopeLocator = page.Locator("body")
	}

	var locator playwright.Locator

	// 1. 直接指定 selector
	if selector, ok := step.Params["selector"].(string); ok && selector != "" {
		locator = scopeLocator.Locator(selector)
		return e.applyIndex(locator, step), nil
	}

	// 2. 通过 label 文本定位表单元素（推荐）
	if label, ok := step.Params["label"].(string); ok && label != "" {
		selector := fmt.Sprintf(
			"input:right-of(:text('%s')), input:below(:text('%s')), textarea:right-of(:text('%s')), textarea:below(:text('%s'))",
			label, label, label, label,
		)
		locator = scopeLocator.Locator(selector)
		return e.applyIndex(locator, step), nil
	}

	// 3. 通过文本内容定位
	if text, ok := step.Params["text"].(string); ok && text != "" {
		locator = scopeLocator.GetByText(text)
		return e.applyIndex(locator, step), nil
	}

	// 4. 通过 ARIA role 定位
	if role, ok := step.Params["role"].(string); ok && role != "" {
		name, _ := step.Params["role_name"].(string)
		ariaRole := playwright.AriaRole(role)
		if name != "" {
			locator = scopeLocator.GetByRole(ariaRole, playwright.LocatorGetByRoleOptions{
				Name: playwright.String(name),
			})
		} else {
			locator = scopeLocator.GetByRole(ariaRole)
		}
		return e.applyIndex(locator, step), nil
	}

	// 5. 通过 placeholder 定位
	if placeholder, ok := step.Params["placeholder"].(string); ok && placeholder != "" {
		locator = scopeLocator.GetByPlaceholder(placeholder)
		return e.applyIndex(locator, step), nil
	}

	// 6. 通过 data-testid 定位
	if testid, ok := step.Params["testid"].(string); ok && testid != "" {
		locator = scopeLocator.GetByTestId(testid)
		return e.applyIndex(locator, step), nil
	}

	// 7. 通过 alt 文本定位
	if alt, ok := step.Params["alt"].(string); ok && alt != "" {
		locator = scopeLocator.GetByAltText(alt)
		return e.applyIndex(locator, step), nil
	}

	// 8. 通过 title 属性定位
	if title, ok := step.Params["title"].(string); ok && title != "" {
		locator = scopeLocator.GetByTitle(title)
		return e.applyIndex(locator, step), nil
	}

	// 9. 布局选择器：near（在某个元素附近）
	if near, ok := step.Params["near"].(string); ok && near != "" {
		tag, _ := step.Params["tag"].(string)
		if tag != "" {
			locator = scopeLocator.Locator(fmt.Sprintf("%s:near(:text('%s'))", tag, near))
		} else {
			locator = scopeLocator.Locator(fmt.Sprintf(":near(:text('%s'))", near))
		}
		return e.applyIndex(locator, step), nil
	}

	return nil, &StepError{StepID: step.ID, Message: "缺少定位参数（selector/label/text/role/placeholder/testid/alt/title/near）"}
}

// applyWithinIndex 根据索引参数选择 within 容器
func (e *TaskEngine) applyWithinIndex(locator playwright.Locator, step Step) playwright.Locator {
	indexParam := step.Params["within_index"]

	// 字符串类型：first 或 last
	if idx, ok := indexParam.(string); ok {
		switch idx {
		case "last":
			return locator.Last()
		case "first":
			return locator.First()
		}
	}

	// 数字类型：指定索引（从0开始）
	switch idx := indexParam.(type) {
	case int:
		return locator.Nth(idx)
	case float64:
		return locator.Nth(int(idx))
	}

	// 默认返回第一个
	return locator.First()
}

// applyIndex 根据索引参数选择元素
func (e *TaskEngine) applyIndex(locator playwright.Locator, step Step) playwright.Locator {
	indexParam := step.Params["index"]

	// 字符串类型：first 或 last
	if idx, ok := indexParam.(string); ok {
		switch idx {
		case "last":
			return locator.Last()
		case "first":
			return locator.First()
		}
	}

	// 数字类型：指定索引（从0开始）
	switch idx := indexParam.(type) {
	case int:
		return locator.Nth(idx)
	case float64:
		return locator.Nth(int(idx))
	}

	// 默认返回第一个
	return locator.First()
}

func (e *TaskEngine) actionWaitForUser(step Step) error {
	if step.Detect == nil {
		return &StepError{StepID: step.ID, Message: "缺少 detect 条件"}
	}

	timeout := 300000
	if step.Timeout > 0 {
		timeout = step.Timeout
	}

	if step.Description != "" {
		fmt.Printf("\n⏳ %s\n", step.Description)
	}

	switch step.Detect.Condition {
	case "url_changed":
		return e.waitForURLChange(step, timeout)
	case "url_contains":
		return e.waitForURLContains(step, timeout)
	case "element_appear":
		return e.waitForElementAppear(step, timeout)
	case "element_disappear":
		return e.waitForElementDisappear(step, timeout)
	case "manual_confirm":
		return e.waitForManualConfirm(step, timeout)
	default:
		return &StepError{StepID: step.ID, Message: "未知的检测条件: " + step.Detect.Condition}
	}
}

func (e *TaskEngine) waitForURLChange(step Step, timeout int) error {
	fromURL := step.Detect.From
	toURL := step.Detect.To

	start := time.Now()
	for {
		currentURL := e.tool.page.URL()

		if toURL != "" && currentURL == toURL {
			return nil
		}
		if fromURL != "" && currentURL != fromURL {
			return nil
		}

		if time.Since(start).Milliseconds() >= int64(timeout) {
			return &StepError{StepID: step.ID, Message: "等待 URL 变化超时"}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (e *TaskEngine) waitForURLContains(step Step, timeout int) error {
	value := step.Detect.Value
	if value == "" {
		return &StepError{StepID: step.ID, Message: "url_contains 需要指定 value"}
	}

	start := time.Now()
	for {
		currentURL := e.tool.page.URL()
		if contains(currentURL, value) {
			return nil
		}

		if time.Since(start).Milliseconds() >= int64(timeout) {
			return &StepError{StepID: step.ID, Message: "等待 URL 包含指定值超时"}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (e *TaskEngine) waitForElementAppear(step Step, timeout int) error {
	selector := step.Detect.Selector
	if selector == "" {
		return &StepError{StepID: step.ID, Message: "element_appear 需要指定 selector"}
	}

	locator := e.tool.page.Locator(selector)
	err := locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(timeout)),
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "等待元素出现超时: " + err.Error()}
	}
	return nil
}

func (e *TaskEngine) waitForElementDisappear(step Step, timeout int) error {
	selector := step.Detect.Selector
	if selector == "" {
		return &StepError{StepID: step.ID, Message: "element_disappear 需要指定 selector"}
	}

	locator := e.tool.page.Locator(selector)
	err := locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(float64(timeout)),
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "等待元素消失超时: " + err.Error()}
	}
	return nil
}

func (e *TaskEngine) waitForManualConfirm(step Step, timeout int) error {
	fmt.Println("\n按 Enter 键继续...")
	confirmChan := make(chan bool)

	go func() {
		var input string
		fmt.Scanln(&input)
		confirmChan <- true
	}()

	select {
	case <-confirmChan:
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return &StepError{StepID: step.ID, Message: "等待用户确认超时"}
	}
}

func (e *TaskEngine) actionExtract(step Step) error {
	if len(step.Fields) == 0 {
		return &StepError{StepID: step.ID, Message: "缺少 fields 配置"}
	}

	data, err := e.extractFields(step.Fields)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "提取数据失败: " + err.Error()}
	}

	for k, v := range data {
		e.result.Data[k] = v
	}

	return nil
}

func (e *TaskEngine) extractFields(fields map[string]FieldConfig) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for name, config := range fields {
		if config.Type == "list" {
			listData, err := e.extractList(config)
			if err != nil {
				return nil, fmt.Errorf("提取列表 %s 失败: %w", name, err)
			}
			result[name] = listData
		} else {
			value, err := e.extractSingleValue(config)
			if err != nil {
				return nil, fmt.Errorf("提取字段 %s 失败: %w", name, err)
			}
			result[name] = value
		}
	}

	return result, nil
}

func (e *TaskEngine) extractSingleValue(config FieldConfig) (string, error) {
	// 特殊处理：剪贴板
	if config.Attr == "clipboard" {
		return e.readClipboard()
	}

	selector := e.resolveFieldSelector(config)
	locator := e.tool.page.Locator(selector)

	var value string
	var err error

	switch config.Attr {
	case "text", "":
		value, err = locator.InnerText()
	case "value":
		value, err = locator.InputValue()
	case "html", "innerHTML":
		value, err = locator.InnerHTML()
	case "src", "href", "alt", "title", "placeholder":
		value, err = locator.GetAttribute(config.Attr)
	default:
		value, err = locator.GetAttribute(config.Attr)
	}

	if err != nil {
		return "", err
	}
	return value, nil
}

func (e *TaskEngine) readClipboard() (string, error) {
	result, err := e.tool.page.Evaluate("navigator.clipboard.readText()")
	if err != nil {
		return "", fmt.Errorf("读取剪贴板失败: %w", err)
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("剪贴板内容不是字符串")
}

func (e *TaskEngine) extractList(config FieldConfig) ([]map[string]interface{}, error) {
	if config.Container == "" {
		return nil, fmt.Errorf("list 类型需要指定 container")
	}

	containerLocator := e.tool.page.Locator(config.Container)
	count, err := containerLocator.Count()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for i := 0; i < count; i++ {
		itemLocator := containerLocator.Nth(i)
		item := make(map[string]interface{})

		for fieldName, fieldConfig := range config.Fields {
			selector := e.resolveFieldSelector(fieldConfig)
			itemLocatorField := itemLocator.Locator(selector)

			var value string
			switch fieldConfig.Attr {
			case "text", "":
				value, _ = itemLocatorField.InnerText()
			case "value":
				value, _ = itemLocatorField.InputValue()
			case "html", "innerHTML":
				value, _ = itemLocatorField.InnerHTML()
			default:
				value, _ = itemLocatorField.GetAttribute(fieldConfig.Attr)
			}

			item[fieldName] = value
		}

		results = append(results, item)
	}

	return results, nil
}

// resolveFieldSelector 解析字段选择器，支持多种定位策略
func (e *TaskEngine) resolveFieldSelector(config FieldConfig) string {
	if config.Selector != "" {
		return config.Selector
	}
	if config.Label != "" {
		return fmt.Sprintf("label:has-text('%s') >> input, label:has-text('%s') >> textarea", config.Label, config.Label)
	}
	if config.Text != "" {
		return fmt.Sprintf("text=%s", config.Text)
	}
	if config.Role != "" {
		if config.RoleName != "" {
			return fmt.Sprintf("role=%s[name='%s']", config.Role, config.RoleName)
		}
		return fmt.Sprintf("role=%s", config.Role)
	}
	if config.TestID != "" {
		return fmt.Sprintf("[data-testid='%s']", config.TestID)
	}
	if config.Placeholder != "" {
		return fmt.Sprintf("input[placeholder*='%s'], textarea[placeholder*='%s']", config.Placeholder, config.Placeholder)
	}
	if config.Title != "" {
		return fmt.Sprintf("[title*='%s']", config.Title)
	}
	if config.Alt != "" {
		return fmt.Sprintf("img[alt*='%s']", config.Alt)
	}
	return ""
}

func (e *TaskEngine) actionScreenshot(step Step) error {
	path, ok := step.Params["path"].(string)
	if !ok || path == "" {
		path = fmt.Sprintf("screenshot_%s_%d.png", step.ID, time.Now().Unix())
	}

	_, err := e.tool.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "截图失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionScroll(step Step) error {
	direction, ok := step.Params["direction"].(string)
	if !ok {
		direction = "down"
	}

	amount := 500
	switch a := step.Params["amount"].(type) {
	case float64:
		amount = int(a)
	case int:
		amount = a
	}

	var deltaY float64
	if direction == "up" {
		deltaY = float64(-amount)
	} else {
		deltaY = float64(amount)
	}

	err := e.tool.page.Mouse().Wheel(0, deltaY)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "滚动失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionHover(step Step) error {
	locator, err := e.resolveLocator(step)
	if err != nil {
		return err
	}

	err = locator.Hover()
	if err != nil {
		return &StepError{StepID: step.ID, Message: "悬停失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionSelect(step Step) error {
	locator, err := e.resolveLocator(step)
	if err != nil {
		return err
	}

	value, ok := step.Params["value"].(string)
	if !ok || value == "" {
		return &StepError{StepID: step.ID, Message: "缺少 value 参数"}
	}

	_, err = locator.SelectOption(playwright.SelectOptionValues{
		Values: &[]string{value},
	})
	if err != nil {
		return &StepError{StepID: step.ID, Message: "选择失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionClipboard(step Step) error {
	// 1. 可选：点击复制按钮（支持所有定位方式）
	if _, hasClick := step.Params["click"]; hasClick || step.Params["selector"] != nil || step.Params["text"] != nil || step.Params["label"] != nil {
		locator, err := e.resolveLocator(step)
		if err != nil {
			return &StepError{StepID: step.ID, Message: "定位复制按钮失败: " + err.Error()}
		}
		err = locator.Click()
		if err != nil {
			return &StepError{StepID: step.ID, Message: "点击复制按钮失败: " + err.Error()}
		}
		// 等待复制完成
		time.Sleep(200 * time.Millisecond)
	}

	// 2. 读取剪贴板
	value, err := e.readClipboard()
	if err != nil {
		return &StepError{StepID: step.ID, Message: err.Error()}
	}

	// 3. 存储到结果中
	fieldName := "clipboard"
	if name, ok := step.Params["field"].(string); ok && name != "" {
		fieldName = name
	}
	e.result.Data[fieldName] = value

	return nil
}

func (e *TaskEngine) actionCondition(step Step) error {
	if step.Check == nil {
		return &StepError{StepID: step.ID, Message: "缺少 check 条件"}
	}

	conditionMet, err := e.evaluateCondition(step.Check)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "条件判断失败: " + err.Error()}
	}

	var stepsToExecute []Step
	if conditionMet {
		stepsToExecute = step.Then
	} else {
		stepsToExecute = step.Else
	}

	for _, s := range stepsToExecute {
		if err := e.executeStep(s); err != nil {
			return err
		}
	}

	return nil
}

func (e *TaskEngine) evaluateCondition(check *CheckCondition) (bool, error) {
	selector := e.resolveCheckSelector(check)
	if selector == "" {
		return false, fmt.Errorf("缺少选择器参数")
	}

	locator := e.tool.page.Locator(selector)

	if check.Exists != nil {
		count, err := locator.Count()
		if err != nil {
			return false, err
		}
		exists := count > 0
		if *check.Exists {
			return exists, nil
		}
		return !exists, nil
	}

	if check.Visible != nil {
		visible, err := locator.IsVisible()
		if err != nil {
			return false, err
		}
		if *check.Visible {
			return visible, nil
		}
		return !visible, nil
	}

	if check.TextEquals != "" {
		text, err := locator.InnerText()
		if err != nil {
			return false, err
		}
		return text == check.TextEquals, nil
	}

	if check.TextContains != "" {
		text, err := locator.InnerText()
		if err != nil {
			return false, err
		}
		return contains(text, check.TextContains), nil
	}

	count, err := locator.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// resolveCheckSelector 解析检查条件的选择器
func (e *TaskEngine) resolveCheckSelector(check *CheckCondition) string {
	if check.Selector != "" {
		return check.Selector
	}
	if check.Label != "" {
		return fmt.Sprintf("label:has-text('%s') >> input", check.Label)
	}
	if check.Text != "" {
		return fmt.Sprintf("text=%s", check.Text)
	}
	if check.Role != "" {
		if check.RoleName != "" {
			return fmt.Sprintf("role=%s[name='%s']", check.Role, check.RoleName)
		}
		return fmt.Sprintf("role=%s", check.Role)
	}
	if check.TestID != "" {
		return fmt.Sprintf("[data-testid='%s']", check.TestID)
	}
	if check.Placeholder != "" {
		return fmt.Sprintf("input[placeholder*='%s']", check.Placeholder)
	}
	return ""
}

func (e *TaskEngine) actionGoto(step Step) error {
	url, ok := step.Params["url"].(string)
	if !ok || url == "" {
		return &StepError{StepID: step.ID, Message: "缺少 url 参数"}
	}

	_, err := e.tool.page.Goto(url)
	if err != nil {
		return &StepError{StepID: step.ID, Message: "跳转页面失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionBack(step Step) error {
	_, err := e.tool.page.GoBack()
	if err != nil {
		return &StepError{StepID: step.ID, Message: "后退失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) actionRefresh(step Step) error {
	_, err := e.tool.page.Reload()
	if err != nil {
		return &StepError{StepID: step.ID, Message: "刷新失败: " + err.Error()}
	}

	return nil
}

func (e *TaskEngine) takeErrorScreenshot(stepID string) (string, error) {
	path := fmt.Sprintf("error_%s_%d.png", stepID, time.Now().Unix())
	_, err := e.tool.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
