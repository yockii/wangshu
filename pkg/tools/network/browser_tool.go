package network

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/yockii/wangshu/pkg/tools/basic"
)

type BrowserTool struct {
	basic.SimpleTool
	pw          *playwright.Playwright
	page        playwright.Page
	mu          sync.RWMutex
	initialized bool
}

func NewBrowserTool() *BrowserTool {
	tool := new(BrowserTool)
	tool.Name_ = "browser"
	tool.Desc_ = "Browser automation tool (Playwright). Actions: open(url), click(selector), fill(selector,text), text(selector), html(), screenshot(path), wait(selector), close(). IMPORTANT: After each operation (open, click, fill, wait, screenshot), the tool automatically returns complete information about all interactive elements on the page (inputs, buttons, links, etc.) with ALL available selectors (id, name, class, xpath, data-*) and attributes. Use this information to choose the most reliable selector for your next action. Do NOT guess selectors - analyze the returned element data first."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{"open", "screenshot", "close", "click", "fill", "text", "html", "wait", "list_tabs"},
			},
			"url":             map[string]any{"type": "string"},
			"selector":        map[string]any{"type": "string"},
			"text":            map[string]any{"type": "string"},
			"screenshot_path": map[string]any{"type": "string"},
			"timeout":         map[string]any{"type": "number"},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *BrowserTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action required")
	}

	if !t.initialized && action != "close" {
		if err := t.init(); err != nil {
			return "", err
		}
	}

	switch action {
	case "open":
		return t.open(params)
	case "screenshot":
		return t.screenshot(params)
	case "close":
		return t.close()
	case "click":
		return t.click(params)
	case "fill":
		return t.fill(params)
	case "text":
		return t.getText(params)
	case "html":
		return t.getHTML(params)
	case "wait":
		return t.wait(params)
	case "list_tabs":
		return "1 tab open", nil
	default:
		return "", fmt.Errorf("unknown action")
	}
}

func (t *BrowserTool) init() error {
	pw, err := playwright.Run()
	if err != nil {
		if installErr := playwright.Install(); installErr != nil {
			return fmt.Errorf("playwright install failed: %w", installErr)
		}
		pw, err = playwright.Run()
		if err != nil {
			return err
		}
	}

	var browser playwright.Browser
	var launchErr error

	if runtime.GOOS == "windows" {
		browser, launchErr = pw.Chromium.Launch(
			playwright.BrowserTypeLaunchOptions{
				Channel:  playwright.String("msedge"),
				Headless: playwright.Bool(false),
			},
		)
		if launchErr != nil {
			fmt.Printf("msedge not available, falling back to chromium: %v\n", launchErr)
			browser, launchErr = pw.Chromium.Launch(
				playwright.BrowserTypeLaunchOptions{
					Headless: playwright.Bool(false),
				},
			)
		}
	} else {
		browser, launchErr = pw.Chromium.Launch(
			playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(false),
			},
		)
	}

	if launchErr != nil {
		return launchErr
	}

	page, err := browser.NewPage()
	if err != nil {
		return err
	}

	t.page = page
	t.initialized = true
	return nil
}

func (t *BrowserTool) open(params map[string]string) (string, error) {
	url := params["url"]
	if url == "" {
		return "", fmt.Errorf("url required")
	}
	_, err := t.page.Goto(url)
	if err != nil {
		return "", err
	}

	result := "Opened: " + url
	return t.appendElementInfo(result), nil
}

func (t *BrowserTool) screenshot(params map[string]string) (string, error) {
	path := params["screenshot_path"]
	if path == "" {
		path = fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	}
	_, err := t.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	if err != nil {
		return "", err
	}

	result := "Screenshot saved: " + path
	return t.appendElementInfo(result), nil
}

func (t *BrowserTool) close() (string, error) {
	if t.page != nil {
		t.page.Close()
	}
	if t.pw != nil {
		t.pw.Stop()
	}
	t.initialized = false
	return "Browser closed", nil
}

func (t *BrowserTool) click(params map[string]string) (string, error) {
	selector := params["selector"]
	if selector == "" {
		return "", fmt.Errorf("selector required")
	}
	err := t.page.Click(selector)
	if err != nil {
		return "", err
	}

	result := "Clicked: " + selector

	// 等待可能的导航或页面更新
	t.page.WaitForTimeout(500)

	return t.appendElementInfo(result), nil
}

func (t *BrowserTool) fill(params map[string]string) (string, error) {
	selector := params["selector"]
	text := params["text"]
	if selector == "" || text == "" {
		return "", fmt.Errorf("selector and text required")
	}
	err := t.page.Fill(selector, text)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Filled: %s with '%s'", selector, text)
	return t.appendElementInfo(result), nil
}

func (t *BrowserTool) getText(params map[string]string) (string, error) {
	selector := params["selector"]
	if selector == "" {
		return "", fmt.Errorf("selector required")
	}
	text, err := t.page.InnerText(selector)
	if err != nil {
		return "", err
	}
	return text, nil
}

func (t *BrowserTool) getHTML(params map[string]string) (string, error) {
	html, err := t.page.Content()
	if err != nil {
		return "", err
	}
	return html, nil
}

func (t *BrowserTool) wait(params map[string]string) (string, error) {
	selector := params["selector"]
	if selector == "" {
		return "", fmt.Errorf("selector required")
	}
	_, err := t.page.WaitForSelector(selector)
	if err != nil {
		return "", err
	}
	result := "Waited for: " + selector
	if elements := t.collectElements(); elements != "" {
		result += "\n\n" + elements
	}
	return result, nil
}

// ElementInfo 包含元素的所有选择器和属性信息
type ElementInfo struct {
	// 基本信息
	Tag        string `json:"tag"`
	Visible    bool   `json:"visible"`
	Enabled    bool   `json:"enabled"`
	Editable   bool   `json:"editable"`

	// 各种选择器
	IDSelector    string            `json:"id_selector,omitempty"`
	NameSelector  string            `json:"name_selector,omitempty"`
	ClassSelector string            `json:"class_selector,omitempty"`
	XPathSelector string            `json:"xpath_selector,omitempty"`
	DataSelectors map[string]string `json:"data_selectors,omitempty"` // data-testid, data-test-id等

	// 元素属性
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	Text        string `json:"text,omitempty"`
	Href        string `json:"href,omitempty"`
	ARIALabel   string `json:"aria_label,omitempty"`

	// 表单特定属性
	ReadOnly bool `json:"readonly,omitempty"`
	Required bool `json:"required,omitempty"`
	Checked  bool `json:"checked,omitempty"`
}

// collectElements 收集页面上所有可交互元素的完整信息
// 不做任何过滤、排序或优先级判断，返回所有可用信息供LLM分析
func (t *BrowserTool) collectElements() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.page == nil {
		return ""
	}

	// 使用单一查询获取所有可交互元素
	selector := "input, textarea, select, button, a[href], [role='button'], [onclick], [tabindex]"

	// 使用JavaScript获取所有元素信息
	result, err := t.page.EvalOnSelectorAll(selector, `
		(elements) => {
			return elements.map((el, index) => {
				// 检查元素是否可见
				const rect = el.getBoundingClientRect();
				const visible = rect.width > 0 && rect.height > 0 &&
					window.getComputedStyle(el).display !== 'none' &&
					window.getComputedStyle(el).visibility !== 'hidden' &&
					el.offsetParent !== null;

				// 检查元素是否可用
				const enabled = !el.disabled && !el.hasAttribute('disabled');

				// 检查元素是否可编辑
				const editable = !el.readOnly && !el.hasAttribute('readonly');

				// 获取所有data-*属性
				const dataAttrs = {};
				for (let attr of el.attributes || []) {
					if (attr.name && attr.name.startsWith('data-')) {
						dataAttrs[attr.name] = attr.value || '';
					}
				}

				// 构建XPath
				const getXPath = (element) => {
					if (element.id && element.id !== '') {
						return '//*[@id="' + element.id + '"]';
					}
					if (element === document.body) {
						return element.tagName.toLowerCase();
					}

					const ix = Array.from(element.parentNode.children)
						.filter(child => child.tagName === element.tagName)
						.indexOf(element) + 1;

					return getXPath(element.parentNode) + '/' +
						element.tagName.toLowerCase() + '[' + ix + ']';
				};

				// 获取class选择器（取第一个非动态类名）
				let classSelector = '';
				if (el.className && typeof el.className === 'string') {
					const classes = el.className.split(/\s+/).filter(c =>
						c && !c.match(/^(css-|_|[a-f0-9]{6,})/i) && !c.includes(':')
					);
					if (classes.length > 0) {
						classSelector = el.tagName.toLowerCase() + '.' + classes[0];
					}
				}

				// 获取文本内容（限制长度）
				let textContent = '';
				if (el.textContent && typeof el.textContent === 'string') {
					textContent = el.textContent.trim().substring(0, 100);
					if (el.textContent.length > 100) {
						textContent += '...';
					}
				}

				// 获取aria-label
				const ariaLabel = el.getAttribute && el.getAttribute('aria-label') ?
					el.getAttribute('aria-label') : '';

				return {
					tag: el.tagName ? el.tagName.toLowerCase() : '',
					visible: visible,
					enabled: enabled,
					editable: editable,
					id_selector: el.id && el.id !== '' ? '#' + el.id : '',
					name_selector: el.name && el.name !== '' ? '[name="' + el.name + '"]' : '',
					class_selector: classSelector,
					xpath_selector: getXPath(el),
					data_selectors: dataAttrs,
					type: el.type || '',
					name: el.name || '',
					placeholder: el.placeholder || '',
					value: el.value || '',
					text: textContent,
					href: el.href || '',
					aria_label: ariaLabel,
					readonly: !!el.readOnly,
					required: !!el.required,
					checked: !!el.checked
				};
			});
		}
	`)

	if err != nil {
		return ""
	}

	// 解析结果
	var elements []ElementInfo
	if dataArray, ok := result.([]interface{}); ok {
		for _, item := range dataArray {
			if dataMap, ok := item.(map[string]interface{}); ok {
				info := ElementInfo{
					Tag:          getString(dataMap, "tag"),
					Visible:      getBool(dataMap, "visible"),
					Enabled:      getBool(dataMap, "enabled"),
					Editable:     getBool(dataMap, "editable"),
					IDSelector:   getString(dataMap, "id_selector"),
					NameSelector: getString(dataMap, "name_selector"),
					ClassSelector: getString(dataMap, "class_selector"),
					XPathSelector: getString(dataMap, "xpath_selector"),
					DataSelectors: make(map[string]string),
					Type:        getString(dataMap, "type"),
					Name:        getString(dataMap, "name"),
					Placeholder: getString(dataMap, "placeholder"),
					Value:       getString(dataMap, "value"),
					Text:        getString(dataMap, "text"),
					Href:        getString(dataMap, "href"),
					ARIALabel:   getString(dataMap, "aria_label"),
					ReadOnly:    getBool(dataMap, "readonly"),
					Required:    getBool(dataMap, "required"),
					Checked:     getBool(dataMap, "checked"),
				}

				// 处理data_selectors
				if dataSel, ok := dataMap["data_selectors"].(map[string]interface{}); ok {
					for k, v := range dataSel {
						if vs, ok := v.(string); ok {
							info.DataSelectors[k] = vs
						}
					}
				}

				elements = append(elements, info)
			}
		}
	}

	if len(elements) == 0 {
		return ""
	}

	// 转换为JSON格式返回给LLM
	jsonData, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		return ""
	}

	return fmt.Sprintf("Page Elements (%d found):\n%s", len(elements), string(jsonData))
}

// getString 从map中安全获取字符串值
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBool 从map中安全获取布尔值
func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// appendElementInfo 在操作结果后附加元素信息
func (t *BrowserTool) appendElementInfo(baseResult string) string {
	if elements := t.collectElements(); elements != "" {
		return baseResult + "\n\n" + elements
	}
	return baseResult
}
