package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
	actiontypes "github.com/yockii/wangshu/pkg/types"
	"github.com/yockii/wangshu/pkg/utils"
)

type BrowserTool struct {
	basic.SimpleTool
	pw          *playwright.Playwright
	browser     playwright.Browser
	context     playwright.BrowserContext
	page        playwright.Page
	mu          sync.RWMutex
	initialized bool
}

func NewBrowserTool() *BrowserTool {
	tool := new(BrowserTool)
	tool.Name_ = constant.ToolNameBrowser
	tool.Desc_ = `浏览器自动化工具，使用 Playwright 控制 Chromium 浏览器。

支持的操作：
- open: 打开网页，自动收集并返回页面元素信息
- screenshot: 截图，保存页面截图
- click: 点击元素，返回点击后的元素信息
- fill: 填充表单，返回填充后的元素信息
- text: 获取元素文本内容
- html: 获取页面HTML内容（支持分页获取）
  * format: "full"(完整HTML) | "body"(body内容，默认) | "inner"(innerHTML) | "text"(纯文本)
  * start: 起始位置（字符偏移），用于分页获取大型页面，默认0
  * max_length: 每次最大获取长度，默认50000字符
  返回内容包含当前范围、总长度、下次获取的start位置等信息
- wait: 等待元素出现
- close: 关闭浏览器
- list_tabs: 列出所有标签页
- run_task: 执行任务脚本（自动化任务编排）
  * script: JSON格式的任务脚本
  * script_file: 脚本文件路径（与script二选一）
  * keep_browser_open: 任务完成后是否保持浏览器打开，默认false
  * variables: JSON格式的变量映射，用于替换脚本中的${var_name}

每次操作（除close/list_tabs/run_task外）都会自动返回当前页面的可交互元素信息，包括：
  元素类型、选择器（id/class/name/xpath/data属性）、可见性、可编辑性等。

特别说明：
- html action 支持分页获取大型页面：第一次调用（start=0）获取前50000字符，返回提示"下次获取: start=50000"
- 根据返回的提示，使用新的start值继续获取：{"action":"html","start":50000}
- 这样可以完整获取任意大小的页面内容，不会丢失数据
- 所有操作都会返回元素信息，帮助大模型理解页面结构`
	tool.Desc_ = `Browser automation tool (Playwright). Actions: open(url), click(selector), fill(selector,text), text(selector), html(), screenshot(path), wait(selector), close(), list_tabs(), run_task().

After each operation (open, click, fill, wait), the tool returns a compact list of interactive elements on the page, prioritized by relevance: form inputs > buttons > links. Invisible elements and empty links are filtered out. By default, up to 30 elements are returned.

Parameters for controlling element collection:
- include_elements: set to false to skip element collection (saves tokens when not needed)
- max_elements: maximum number of elements to return (default: 30, max: 100)
- element_types: comma-separated filter, e.g. "input,button,select" to only show form elements
- search_elements: keyword to filter elements by text, placeholder, value, href, selector, etc. Case-insensitive. Useful for finding specific elements on complex pages.

The browser auto-restarts if it was closed or crashed. Use include_elements=false when you only need the action result. Use search_elements to quickly locate elements by keyword (e.g. "login", "submit", "email").`
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"open", "screenshot", "close", "click", "fill", "text", "html", "wait", "list_tabs", "run_task"},
				"description": "操作类型: open(打开页面), screenshot(截图), close(关闭), click(点击), fill(填充), text(获取文本), html(获取页面HTML), wait(等待元素), list_tabs(列出标签), run_task(执行任务脚本)",
			},
			"url":               map[string]any{"type": "string", "description": "要打开的URL (open action)"},
			"selector":          map[string]any{"type": "string", "description": "CSS选择器 (click/fill/text/wait action)"},
			"text":              map[string]any{"type": "string", "description": "要填充的文本 (fill action)"},
			"screenshot_path":   map[string]any{"type": "string", "description": "截图保存路径 (screenshot action)"},
			"timeout":           map[string]any{"type": "number", "description": "超时时间（毫秒）"},
			"format":            map[string]any{"type": "string", "description": "HTML格式: full(完整HTML), body(body内容), inner(body内部HTML), text(只文本) (html action, 默认: body)"},
			"start":             map[string]any{"type": "number", "description": "起始位置（字符偏移），用于分页获取大型页面 (html action, 默认: 0)"},
			"max_length":        map[string]any{"type": "number", "description": "最大获取长度，默认50000 (html action)"},
			"script":            map[string]any{"type": "string", "description": "JSON格式的任务脚本 (run_task action)"},
			"script_file":       map[string]any{"type": "string", "description": "任务脚本文件路径 (run_task action，与script二选一)"},
			"keep_browser_open": map[string]any{"type": "boolean", "description": "任务完成后是否保持浏览器打开，默认false (run_task action)"},
			"variables":         map[string]any{"type": "string", "description": "JSON格式的变量映射，用于替换脚本中的${var_name} (run_task action)"},
			"include_elements":  map[string]any{"type": "boolean", "description": "是否返回页面元素信息，默认true。设为false可节省token (open/click/fill/text/wait action)"},
			"max_elements":      map[string]any{"type": "number", "description": "最大返回元素数量，默认30，最大100 (open/click/fill/text/wait action)"},
			"element_types":     map[string]any{"type": "string", "description": "逗号分隔的元素类型过滤，如\"input,button,select\"只返回表单元素 (open/click/fill/text/wait action)"},
			"search_elements":   map[string]any{"type": "string", "description": "关键词搜索元素，匹配text/placeholder/value/href/selector等，不区分大小写 (open/click/fill/text/wait action)"},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

type elementCollectOptions struct {
	includeElements bool
	maxElements     int
	elementTypes    string
	searchKeyword   string
}

func (t *BrowserTool) parseElementOptions(params map[string]string) elementCollectOptions {
	opts := elementCollectOptions{
		includeElements: true,
		maxElements:     30,
		elementTypes:    "",
	}
	if v, ok := params["include_elements"]; ok {
		opts.includeElements = v != "false"
	}
	if v, ok := params["max_elements"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			opts.maxElements = n
		}
	}
	if v, ok := params["element_types"]; ok {
		opts.elementTypes = v
	}
	if v, ok := params["search_elements"]; ok {
		opts.searchKeyword = strings.TrimSpace(v)
	}
	return opts
}

func (t *BrowserTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	action := params["action"]
	if action == "" {
		return types.NewToolResult().WithError(fmt.Errorf("action required"))
	}

	if action != "close" {
		if err := t.ensureBrowserReady(); err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("浏览器启动失败: %w", err))
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
		return t.listTabs()
	case "run_task":
		return t.runTask(params)
	default:
		return types.NewToolResult().WithError(fmt.Errorf("unknown action"))
	}
}

// detectSystemBrowsers 检测系统可用的浏览器
// 返回：(浏览器类型, 可执行路径, 是否需要安装完整浏览器)
func detectSystemBrowsers() (browserType string, executablePath string, needFullInstall bool) {
	switch runtime.GOOS {
	case "windows":
		// Windows: 使用where命令查找Microsoft Edge（更可靠，支持任意系统盘）
		// where会搜索PATH环境变量和常见安装位置
		if path, err := exec.LookPath("msedge"); err == nil {
			fmt.Printf("✓ 检测到系统浏览器: Microsoft Edge (%s)\n", path)
			return "chromium", path, false // 用chromium驱动启动系统Edge，不需要安装完整浏览器
		}

		// 备选方案：使用环境变量构建常见路径
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")

		edgePaths := []string{}
		if programFiles != "" {
			edgePaths = append(edgePaths, programFiles+`\Microsoft\Edge\Application\msedge.exe`)
		}
		if programFilesX86 != "" {
			edgePaths = append(edgePaths, programFilesX86+`\Microsoft\Edge\Application\msedge.exe`)
		}

		for _, path := range edgePaths {
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("✓ 检测到系统浏览器: Microsoft Edge\n")
				return "chromium", path, false
			}
		}

		// 没找到Edge，需要安装Chromium
		fmt.Println("✗ 未检测到Microsoft Edge，将安装Chromium")
		return "chromium", "", true

	case "darwin":
		// macOS: Safari是系统自带的，但Playwright需要WebKit驱动
		if _, err := os.Stat("/Applications/Safari.app"); err == nil {
			fmt.Println("✓ 检测到系统浏览器: Safari")
			return "webkit", "", true // WebKit驱动需要完整安装
		}
		fmt.Println("✗ 未检测到Safari，将安装Chromium")
		return "chromium", "", true

	case "linux":
		// Linux: 尝试检测常见浏览器
		browsers := []string{
			"google-chrome",
			"chromium",
			"chromium-browser",
			"firefox",
		}

		for _, cmd := range browsers {
			if path, err := exec.LookPath(cmd); err == nil {
				fmt.Printf("✓ 检测到系统浏览器: %s (%s)\n", cmd, path)
				// Linux上的系统浏览器可能能用，但为了兼容性还是安装Chromium驱动
				return "chromium", "", true
			}
		}

		// 没找到任何浏览器
		fmt.Println("✗ 未检测到任何浏览器，将安装Chromium")
		return "chromium", "", true

	default:
		fmt.Printf("⚠ 未知操作系统(%s)，将安装Chromium\n", runtime.GOOS)
		return "chromium", "", true
	}
}

// getBrowserDataDir 获取浏览器专用用户数据目录
// 从配置文件读取，默认为 ~/.wangshu/browser_profile
func getBrowserDataDir() (string, error) {
	// 从配置获取浏览器数据目录
	var dataDir string
	if config.DefaultCfg != nil {
		dataDir = config.DefaultCfg.Browser.DataDir
	}

	if dataDir == "" {
		// 配置为空或未初始化，使用默认值
		dataDir = "~/.wangshu/browser_profile"
	}

	// 展开路径中的 ~ 为用户主目录
	expandedPath := utils.ExpandPath(dataDir)

	// 确保目录存在
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create browser profile directory: %w", err)
	}

	// 转换为绝对路径
	absDir, err := filepath.Abs(expandedPath)
	if err != nil {
		return absDir, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absDir, nil
}

func (t *BrowserTool) init() error {
	// 尝试启动Playwright
	browserType, executablePath, needFullInstall := detectSystemBrowsers()
	pw, err := playwright.Run()
	if err != nil {
		if !needFullInstall {
			// 系统有浏览器，只安装驱动
			fmt.Println("安装Playwright驱动（跳过浏览器下载）...")
			if installErr := playwright.Install(&playwright.RunOptions{
				SkipInstallBrowsers: true, // ← 关键：只装驱动，不装浏览器
				Verbose:             true,
			}); installErr != nil {
				return fmt.Errorf("playwright driver install failed: %w", installErr)
			}
		} else {
			// 需要安装完整浏览器
			fmt.Printf("安装Playwright浏览器和驱动: %s\n", browserType)
			if installErr := playwright.Install(&playwright.RunOptions{
				Browsers: []string{browserType},
				Verbose:  true,
			}); installErr != nil {
				return fmt.Errorf("playwright install failed: %w", installErr)
			}
		}

		// 再次尝试启动
		pw, err = playwright.Run()
		if err != nil {
			return fmt.Errorf("playwright run failed after install: %w", err)
		}
	}
	t.pw = pw

	// 获取浏览器专用用户数据目录（持久化、隔离）
	browserDataDir, err := getBrowserDataDir()
	if err != nil {
		return fmt.Errorf("failed to get browser data directory: %w", err)
	}
	fmt.Printf("浏览器数据目录: %s\n", browserDataDir)

	var browser playwright.Browser
	var context playwright.BrowserContext
	var launchErr error

	// 根据检测结果启动浏览器（使用持久化上下文）
	if runtime.GOOS == "windows" {
		// Windows: 优先使用系统Edge
		if executablePath != "" {
			context, launchErr = pw.Chromium.LaunchPersistentContext(
				browserDataDir,
				playwright.BrowserTypeLaunchPersistentContextOptions{
					Channel:        playwright.String("msedge"),
					ExecutablePath: playwright.String(executablePath),
					Headless:       playwright.Bool(false),
				},
			)
			if launchErr != nil {
				fmt.Printf("系统Edge启动失败: %v，回退到Chromium\n", launchErr)
				context, launchErr = pw.Chromium.LaunchPersistentContext(
					browserDataDir,
					playwright.BrowserTypeLaunchPersistentContextOptions{
						Headless: playwright.Bool(false),
					},
				)
			} else {
				fmt.Println("✓ 已启动系统浏览器: Microsoft Edge（持久化环境）")
			}
		} else {
			context, launchErr = pw.Chromium.LaunchPersistentContext(
				browserDataDir,
				playwright.BrowserTypeLaunchPersistentContextOptions{
					Headless: playwright.Bool(false),
				},
			)
		}

	} else if runtime.GOOS == "darwin" {
		// macOS: 使用WebKit（Safari）
		context, launchErr = pw.WebKit.LaunchPersistentContext(
			browserDataDir,
			playwright.BrowserTypeLaunchPersistentContextOptions{
				Headless: playwright.Bool(false),
			},
		)
		if launchErr != nil {
			fmt.Println("WebKit启动失败，回退到Chromium")
			context, launchErr = pw.Chromium.LaunchPersistentContext(
				browserDataDir,
				playwright.BrowserTypeLaunchPersistentContextOptions{
					Headless: playwright.Bool(false),
				},
			)
		}

	} else {
		// Linux和其他系统: 使用Chromium
		context, launchErr = pw.Chromium.LaunchPersistentContext(
			browserDataDir,
			playwright.BrowserTypeLaunchPersistentContextOptions{
				Headless: playwright.Bool(false),
			},
		)
	}

	if launchErr != nil {
		return launchErr
	}

	// 存储context
	t.context = context

	// 从context获取browser
	browser = context.Browser()
	t.browser = browser

	// 从context获取或创建第一个page
	pages := context.Pages()
	if len(pages) > 0 {
		t.page = pages[0]
	} else {
		page, err := context.NewPage()
		if err != nil {
			return err
		}
		t.page = page
	}

	if launchErr != nil {
		return launchErr
	}
	t.browser = browser

	if t.page == nil {
		page, err := browser.NewPage()
		if err != nil {
			return err
		}

		t.page = page
	}
	t.initialized = true
	return nil
}

func element2ActionElement(element ElementInfo) actiontypes.ElementInfo {
	return actiontypes.ElementInfo{
		Tag:            element.Tag,
		Enabled:        element.Enabled,
		Editable:       element.Editable,
		Selector:       element.Selector,
		SelectorUnique: element.SelectorUnique,
		Type:           element.Type,
		Placeholder:    element.Placeholder,
		Value:          element.Value,
		Text:           element.Text,
		Href:           element.Href,
		ReadOnly:       element.ReadOnly,
		Required:       element.Required,
		Checked:        element.Checked,
	}
}

func elements2ActionElements(elements []ElementInfo) []actiontypes.ElementInfo {
	actionElements := make([]actiontypes.ElementInfo, 0, len(elements))
	for _, element := range elements {
		actionElements = append(actionElements, element2ActionElement(element))
	}
	return actionElements
}

func (t *BrowserTool) open(params map[string]string) *types.ToolResult {
	url := params["url"]
	if url == "" {
		return types.NewToolResult().WithError(fmt.Errorf("url required"))
	}
	_, err := t.page.Goto(url)
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	opts := t.parseElementOptions(params)
	if !opts.includeElements {
		return types.NewToolResult().WithRaw("Opened: " + url).WithStructured(
			actiontypes.NewBrowserOpenData(url, nil))
	}

	elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
	result := "Opened: " + url
	return types.NewToolResult().WithRaw(t.appendElementInfo(result, elements)).WithStructured(
		actiontypes.NewBrowserOpenData(url, elements2ActionElements(elements)))
}

func (t *BrowserTool) screenshot(params map[string]string) *types.ToolResult {
	path := params["screenshot_path"]
	if path == "" {
		path = fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	}
	_, err := t.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	result := "Screenshot saved: " + path
	return types.NewToolResult().WithRaw(result).WithStructured(
		actiontypes.NewBrowserScreenshotData(path))
}

func (t *BrowserTool) close() *types.ToolResult {
	// Close page first
	if t.page != nil {
		if err := t.page.Close(); err != nil {
			fmt.Printf("Error closing page: %v\n", err)
		}
		t.page = nil
	}

	// Close context (this also closes the browser for persistent contexts)
	if t.context != nil {
		if err := t.context.Close(); err != nil {
			fmt.Printf("Error closing context: %v\n", err)
		}
		t.context = nil
	}

	// Clear browser reference
	t.browser = nil

	// Finally stop playwright
	if t.pw != nil {
		if err := t.pw.Stop(); err != nil {
			fmt.Printf("Error stopping playwright: %v\n", err)
		}
		t.pw = nil
	}

	t.initialized = false
	return types.NewToolResult().WithRaw("Browser closed")
}

func (t *BrowserTool) click(params map[string]string) *types.ToolResult {
	selector := params["selector"]
	if selector == "" {
		return types.NewToolResult().WithError(fmt.Errorf("selector required"))
	}
	err := t.page.Click(selector)
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	result := "Clicked: " + selector

	time.Sleep(500 * time.Millisecond)

	opts := t.parseElementOptions(params)
	if !opts.includeElements {
		return types.NewToolResult().WithRaw(result).WithStructured(
			actiontypes.NewBrowserClickData(nil))
	}

	elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
	return types.NewToolResult().WithRaw(t.appendElementInfo(result, elements)).WithStructured(
		actiontypes.NewBrowserClickData(elements2ActionElements(elements)))
}

func (t *BrowserTool) fill(params map[string]string) *types.ToolResult {
	selector := params["selector"]
	text := params["text"]
	if selector == "" || text == "" {
		return types.NewToolResult().WithError(fmt.Errorf("selector and text required"))
	}
	err := t.page.Fill(selector, text)
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	result := fmt.Sprintf("Filled: %s with '%s'", selector, text)

	opts := t.parseElementOptions(params)
	if !opts.includeElements {
		return types.NewToolResult().WithRaw(result).WithStructured(
			actiontypes.NewBrowserFillData(nil))
	}

	elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
	return types.NewToolResult().WithRaw(t.appendElementInfo(result, elements)).WithStructured(
		actiontypes.NewBrowserFillData(elements2ActionElements(elements)))
}

func (t *BrowserTool) getText(params map[string]string) *types.ToolResult {
	selector := params["selector"]
	if selector == "" {
		return types.NewToolResult().WithError(fmt.Errorf("selector required"))
	}
	text, err := t.page.InnerText(selector)
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	opts := t.parseElementOptions(params)
	if !opts.includeElements {
		return types.NewToolResult().WithRaw(text).WithStructured(
			actiontypes.NewBrowserTextData(nil))
	}

	elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
	return types.NewToolResult().WithRaw(t.appendElementInfo(text, elements)).WithStructured(
		actiontypes.NewBrowserTextData(elements2ActionElements(elements)))
}

func (t *BrowserTool) getHTML(params map[string]string) *types.ToolResult {
	// 支持不同的返回格式
	// format: full（完整HTML）, body（只body内容）, inner（body innerHTML）, text（只文本内容）
	format := params["format"]
	if format == "" {
		format = "body" // 默认只返回body，更简洁
	}

	var content string
	var err error

	switch format {
	case "full":
		content, err = t.page.Content()
		if err != nil {
			return types.NewToolResult().WithError(err)
		}

	case "body":
		// 获取body的innerHTML
		content, err = t.page.Locator("body").InnerHTML()
		if err != nil {
			return types.NewToolResult().WithError(err)
		}

	case "inner":
		// 获取body的innerHTML（与body相同）
		content, err = t.page.Locator("body").InnerHTML()
		if err != nil {
			return types.NewToolResult().WithError(err)
		}

	case "text":
		// 只获取文本内容
		content, err = t.page.Locator("body").InnerText()
		if err != nil {
			return types.NewToolResult().WithError(err)
		}

	default:
		return types.NewToolResult().WithError(fmt.Errorf("unknown format: %s (use: full, body, inner, text)", format))
	}

	totalLength := len(content)

	// 支持分页获取（start 参数）
	start := 0
	if len(params["start"]) > 0 {
		if s, parseErr := strconv.Atoi(params["start"]); parseErr == nil && s >= 0 && s < totalLength {
			start = s
		} else if parseErr != nil {
			return types.NewToolResult().WithError(fmt.Errorf("invalid start parameter: %v", parseErr))
		} else if s >= totalLength {
			return types.NewToolResult().WithError(fmt.Errorf("⚠️ 起始位置超出范围 (start: %d, 总长度: %d)", s, totalLength))
		}
	}

	// 默认最大长度
	maxLength := 50000
	if len(params["max_length"]) > 0 {
		if max, parseErr := strconv.Atoi(params["max_length"]); parseErr == nil {
			maxLength = max
		}
	}

	// 计算结束位置
	end := start + maxLength
	if end > totalLength {
		end = totalLength
	}

	// 截取指定范围的内容
	content = content[start:end]

	// 构建友好的返回信息
	var result strings.Builder
	hasMore := end < totalLength
	result.WriteString(fmt.Sprintf("📄 页面内容片段\n"))
	result.WriteString(fmt.Sprintf("📊 范围: %d-%d / 总长度: %d 字符\n", start, end, totalLength))
	result.WriteString(fmt.Sprintf("📋 格式: %s\n", format))

	if hasMore {
		nextStart := end
		result.WriteString(fmt.Sprintf("➡️  下次获取: start=%d (还有 %d 字符)\n", nextStart, totalLength-nextStart))
	} else {
		result.WriteString(fmt.Sprintf("✅ 已获取完整内容\n"))
	}

	result.WriteString(fmt.Sprintf("\n"))

	if format == "text" {
		result.WriteString("📝 文本内容:\n")
	} else {
		result.WriteString("🔍 HTML内容:\n")
	}

	result.WriteString(content)

	if hasMore {
		result.WriteString(fmt.Sprintf("\n\n... (还有 %d 字符未显示，使用 start=%d 继续获取)", totalLength-end, end))
	}

	tr := types.NewToolResult()
	opts := t.parseElementOptions(params)
	if format != "text" && start == 0 && opts.includeElements {
		elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
		result.WriteString(t.appendElementInfo("", elements))
	}
	c := result.String()
	tr.WithRaw(c)
	nextStart := end
	tr.WithStructured(actiontypes.NewBrowserHTMLData(format, start, maxLength, content, nextStart))

	return tr
}

func (t *BrowserTool) wait(params map[string]string) *types.ToolResult {
	selector := params["selector"]
	if selector == "" {
		return types.NewToolResult().WithError(fmt.Errorf("selector required"))
	}
	_, err := t.page.WaitForSelector(selector)
	if err != nil {
		return types.NewToolResult().WithError(err)
	}

	result := "Waited for: " + selector

	opts := t.parseElementOptions(params)
	if !opts.includeElements {
		return types.NewToolResult().WithRaw(result).WithStructured(
			actiontypes.NewBrowserClickData(nil))
	}

	elements := t.collectElementsWithOptions(opts.maxElements, opts.elementTypes, opts.searchKeyword)
	return types.NewToolResult().WithRaw(t.appendElementInfo(result, elements)).WithStructured(
		actiontypes.NewBrowserClickData(elements2ActionElements(elements)))
}

func (t *BrowserTool) listTabs() *types.ToolResult {
	var tabs []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	var result strings.Builder
	result.WriteString("当前打开的标签页:\n\n")

	// 使用 context 获取页面列表（支持持久化上下文）
	if t.context != nil {
		ctxPages := t.context.Pages()
		for _, page := range ctxPages {
			title, err := page.Title()
			if err != nil {
				continue
			}
			tabs = append(tabs, struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			}{Title: title, URL: page.URL()})
			result.WriteString(fmt.Sprintf("- Title: %s, URL: %s\n", title, page.URL()))
		}
	} else if t.browser != nil {
		ctxList := t.browser.Contexts()
		for _, ctx := range ctxList {
			ctxPages := ctx.Pages()
			for _, page := range ctxPages {
				title, err := page.Title()
				if err != nil {
					continue
				}
				tabs = append(tabs, struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				}{Title: title, URL: page.URL()})
				result.WriteString(fmt.Sprintf("- Title: %s, URL: %s\n", title, page.URL()))
			}
		}
	}
	return types.NewToolResult().WithRaw(result.String()).WithStructured(
		actiontypes.NewBrowserListTabsData(tabs))
}

// ElementInfo 包含元素的所有选择器和属性信息
type ElementInfo struct {
	Tag      string `json:"tag"`
	Enabled  bool   `json:"enabled"`
	Editable bool   `json:"editable"`

	Selector       string `json:"selector"`
	SelectorUnique bool   `json:"selector_unique"`

	Type        string `json:"type,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	Text        string `json:"text,omitempty"`
	Href        string `json:"href,omitempty"`

	ReadOnly bool `json:"readonly,omitempty"`
	Required bool `json:"required,omitempty"`
	Checked  bool `json:"checked,omitempty"`

	Priority int `json:"-"`
}

// collectElements 收集页面上可交互元素的信息
// 智能过滤：过滤不可见元素、限制链接数量、按优先级排序
func (t *BrowserTool) collectElements() []ElementInfo {
	return t.collectElementsWithOptions(30, "")
}

// collectElementsWithOptions 带选项的元素收集
// maxElements: 最大返回元素数量
// elementTypes: 逗号分隔的元素类型过滤，如 "input,button,select"，空字符串表示所有类型
func (t *BrowserTool) collectElementsWithOptions(maxElements int, elementTypes string, searchKeyword ...string) []ElementInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.page == nil {
		return nil
	}

	selector := "input, textarea, select, button, a[href], [role='button'], [onclick], [tabindex]"

	result, err := t.page.EvalOnSelectorAll(selector, `
		(elements) => {
			const getPriority = (el) => {
				const tag = el.tagName ? el.tagName.toLowerCase() : '';
				if (tag === 'input' || tag === 'textarea' || tag === 'select') return 5;
				if (tag === 'button' || el.getAttribute('role') === 'button') return 4;
				if (tag === 'select') return 3;
				if (el.hasAttribute('onclick')) return 3;
				if (tag === 'a') return 1;
				return 2;
			};

			const isUniqueSelector = (sel) => {
				try {
					return document.querySelectorAll(sel).length === 1;
				} catch(e) {
					return false;
				}
			};

			const getUniqueSelector = (el) => {
				if (el.id && el.id !== '' && !el.id.match(/^[0-9]/)) {
					const sel = '#' + CSS.escape(el.id);
					if (isUniqueSelector(sel)) return sel;
				}
				for (let attr of ['data-testid', 'data-test-id', 'data-cy', 'data-test']) {
					const val = el.getAttribute(attr);
					if (val) {
						const sel = '[' + attr + '="' + CSS.escape(val) + '"]';
						if (isUniqueSelector(sel)) return sel;
					}
				}
				if (el.name && el.name !== '') {
					const sel = '[name="' + CSS.escape(el.name) + '"]';
					if (isUniqueSelector(sel)) return sel;
				}
				const ariaLabel = el.getAttribute('aria-label');
				if (ariaLabel) {
					const sel = '[aria-label="' + CSS.escape(ariaLabel) + '"]';
					if (isUniqueSelector(sel)) return sel;
				}
				if (el.getAttribute('role')) {
					const role = el.getAttribute('role');
					const sel = el.tagName.toLowerCase() + '[role="' + CSS.escape(role) + '"]';
					if (isUniqueSelector(sel)) return sel;
				}
				if (el.placeholder) {
					const sel = el.tagName.toLowerCase() + '[placeholder="' + CSS.escape(el.placeholder) + '"]';
					if (isUniqueSelector(sel)) return sel;
				}
				if (el.className && typeof el.className === 'string') {
					const classes = el.className.split(/\\s+/).filter(c =>
						c && !c.match(/^(css-|_|[a-f0-9]{6,})/i) && !c.includes(':')
					);
					for (const cls of classes) {
						const sel = el.tagName.toLowerCase() + '.' + CSS.escape(cls);
						if (isUniqueSelector(sel)) return sel;
					}
				}
				if (el.type) {
					const sel = el.tagName.toLowerCase() + '[type="' + CSS.escape(el.type) + '"]';
					if (isUniqueSelector(sel)) return sel;
				}
				const text = (el.textContent || '').trim();
				if (text && text.length <= 50 && el.tagName) {
					const tag = el.tagName.toLowerCase();
					const candidates = Array.from(document.querySelectorAll(tag))
						.filter(e => (e.textContent || '').trim() === text);
					if (candidates.length === 1) {
						return tag + ':text("' + text.substring(0, 30) + '")';
					}
				}
				return '';
			};

			const getXPath = (element) => {
				if (element.id && element.id !== '') {
					return '//*[@id="' + element.id + '"]';
				}
				if (element === document.body) {
					return element.tagName.toLowerCase();
				}
				try {
					const ix = Array.from(element.parentNode.children)
						.filter(child => child.tagName === element.tagName)
						.indexOf(element) + 1;
					return getXPath(element.parentNode) + '/' +
						element.tagName.toLowerCase() + '[' + ix + ']';
				} catch(e) {
					return '';
				}
			};

			const items = [];
			for (const el of elements) {
				const rect = el.getBoundingClientRect();
				const visible = rect.width > 0 && rect.height > 0 &&
					window.getComputedStyle(el).display !== 'none' &&
					window.getComputedStyle(el).visibility !== 'hidden' &&
					el.offsetParent !== null;

				if (!visible) continue;

				const tag = el.tagName ? el.tagName.toLowerCase() : '';
				const textContent = (el.textContent && typeof el.textContent === 'string')
					? el.textContent.trim().substring(0, 50)
					: '';

				if (tag === 'a' && textContent.length < 2) continue;

				const enabled = !el.disabled && !el.hasAttribute('disabled');
				const editable = !el.readOnly && !el.hasAttribute('readonly');

				const bestSelector = getUniqueSelector(el);
				const fallbackSelector = bestSelector === '' ? getXPath(el) : '';

				items.push({
					priority: getPriority(el),
					tag: tag,
					enabled: enabled,
					editable: editable,
					selector: bestSelector || fallbackSelector,
					selector_unique: bestSelector !== '',
					type: el.type || '',
					placeholder: el.placeholder || '',
					value: (el.value && tag !== 'a') ? el.value.substring(0, 50) : '',
					text: textContent,
					href: tag === 'a' ? (el.href || '') : '',
					readonly: !!el.readOnly,
					required: !!el.required,
					checked: !!el.checked
				});
			}

			items.sort((a, b) => b.priority - a.priority);
			return items;
		}
	`)

	if err != nil {
		return nil
	}

	var allElements []ElementInfo
	if dataArray, ok := result.([]interface{}); ok {
		for _, item := range dataArray {
			if dataMap, ok := item.(map[string]interface{}); ok {
				info := ElementInfo{
					Tag:            getString(dataMap, "tag"),
					Enabled:        getBool(dataMap, "enabled"),
					Editable:       getBool(dataMap, "editable"),
					Selector:       getString(dataMap, "selector"),
					SelectorUnique: getBool(dataMap, "selector_unique"),
					Type:           getString(dataMap, "type"),
					Placeholder:    getString(dataMap, "placeholder"),
					Value:          getString(dataMap, "value"),
					Text:           getString(dataMap, "text"),
					Href:           getString(dataMap, "href"),
					ReadOnly:       getBool(dataMap, "readonly"),
					Required:       getBool(dataMap, "required"),
					Checked:        getBool(dataMap, "checked"),
					Priority:       getInt(dataMap, "priority"),
				}

				allElements = append(allElements, info)
			}
		}
	}

	if len(allElements) == 0 {
		return nil
	}

	filtered := filterElementsByTypes(allElements, elementTypes)

	var kw string
	if len(searchKeyword) > 0 {
		kw = searchKeyword[0]
	}
	filtered = filterElementsByKeyword(filtered, kw)

	if len(filtered) > maxElements {
		filtered = filtered[:maxElements]
	}

	return filtered
}

// filterElementsByTypes 按元素类型过滤
func filterElementsByTypes(elements []ElementInfo, elementTypes string) []ElementInfo {
	if elementTypes == "" {
		return elements
	}

	types := strings.Split(strings.ToLower(elementTypes), ",")
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t != "" {
			typeSet[t] = true
		}
	}

	var filtered []ElementInfo
	for _, el := range elements {
		if typeSet[el.Tag] {
			filtered = append(filtered, el)
		}
	}
	return filtered
}

func filterElementsByKeyword(elements []ElementInfo, keyword string) []ElementInfo {
	if keyword == "" {
		return elements
	}
	kw := strings.ToLower(keyword)
	var filtered []ElementInfo
	for _, el := range elements {
		if strings.Contains(strings.ToLower(el.Text), kw) ||
			strings.Contains(strings.ToLower(el.Placeholder), kw) ||
			strings.Contains(strings.ToLower(el.Value), kw) ||
			strings.Contains(strings.ToLower(el.Href), kw) ||
			strings.Contains(strings.ToLower(el.Type), kw) ||
			strings.Contains(strings.ToLower(el.Selector), kw) ||
			strings.Contains(strings.ToLower(el.Tag), kw) {
			filtered = append(filtered, el)
		}
	}
	return filtered
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

// getInt 从map中安全获取整数值
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

// appendElementInfo 在操作结果后附加元素信息（紧凑格式）
func (t *BrowserTool) appendElementInfo(baseResult string, elements []ElementInfo) string {
	var result strings.Builder
	result.WriteString(baseResult)

	result.WriteString(fmt.Sprintf("\n[Elements: %d]\n", len(elements)))
	for i, el := range elements {
		parts := []string{fmt.Sprintf("%d.%s", i+1, el.Tag)}

		if el.Selector != "" {
			if el.SelectorUnique {
				parts = append(parts, el.Selector)
			} else {
				parts = append(parts, el.Selector+"(xpath)")
			}
		}
		if el.Type != "" {
			parts = append(parts, fmt.Sprintf("type=%s", el.Type))
		}
		if !el.Enabled {
			parts = append(parts, "disabled")
		}
		if el.Editable && (el.Tag == "input" || el.Tag == "textarea") {
			parts = append(parts, "editable")
		}
		if el.Placeholder != "" {
			parts = append(parts, fmt.Sprintf("placeholder=%q", truncateStr(el.Placeholder, 30)))
		}
		if el.Text != "" && el.Tag != "input" && el.Tag != "textarea" {
			parts = append(parts, fmt.Sprintf("text=%q", truncateStr(el.Text, 30)))
		}
		if el.Href != "" {
			parts = append(parts, fmt.Sprintf("href=%q", truncateStr(el.Href, 60)))
		}
		if el.Value != "" && (el.Tag == "input" || el.Tag == "textarea") {
			parts = append(parts, fmt.Sprintf("value=%q", truncateStr(el.Value, 30)))
		}
		if el.Required {
			parts = append(parts, "required")
		}
		if el.Checked {
			parts = append(parts, "checked")
		}

		result.WriteString(strings.Join(parts, " "))
		result.WriteString("\n")
	}

	return result.String()
}

// truncateStr 截断字符串
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ensureBrowserReady 确保浏览器已初始化且可用（供 TaskEngine 和 execute 使用）
// 如果浏览器已断开连接，会自动清理并重新初始化
func (t *BrowserTool) ensureBrowserReady() error {
	if !t.initialized || t.page == nil || t.context == nil {
		return t.init()
	}
	if t.browser != nil && !t.browser.IsConnected() {
		t.cleanup()
		return t.init()
	}
	return nil
}

func (t *BrowserTool) cleanup() {
	if t.page != nil {
		t.page.Close()
		t.page = nil
	}
	if t.context != nil {
		t.context.Close()
		t.context = nil
	}
	t.browser = nil
	if t.pw != nil {
		t.pw.Stop()
		t.pw = nil
	}
	t.initialized = false
}

// runTask 执行任务脚本
func (t *BrowserTool) runTask(params map[string]string) *types.ToolResult {
	scriptInput := params["script"]
	scriptFile := params["script_file"]

	if scriptInput == "" && scriptFile == "" {
		return types.NewToolResult().WithError(fmt.Errorf("缺少 script 或 script_file 参数"))
	}

	var scriptContent string
	if scriptFile != "" {
		content, err := os.ReadFile(scriptFile)
		if err != nil {
			return types.NewToolResult().WithError(fmt.Errorf("读取脚本文件失败: %w", err))
		}
		scriptContent = string(content)
	} else {
		scriptContent = scriptInput
	}

	var script TaskScript
	if err := json.Unmarshal([]byte(scriptContent), &script); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("解析任务脚本失败: %w", err))
	}

	engine := NewTaskEngine(t)

	if varsJSON, ok := params["variables"]; ok && varsJSON != "" {
		var vars map[string]string
		if err := json.Unmarshal([]byte(varsJSON), &vars); err == nil {
			engine.SetVariables(vars)
		}
	}

	result := engine.Execute(&script)

	keepBrowserOpen := false
	if v, ok := params["keep_browser_open"]; ok {
		keepBrowserOpen = strings.ToLower(v) == "true" || v == "1"
	}

	if !keepBrowserOpen {
		if closeResult := t.close(); closeResult.Err != nil {
			if result.Error == "" {
				result.Error = "关闭浏览器失败: " + closeResult.Err.Error()
			}
		}
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("序列化结果失败: %w", err))
	}

	return types.NewToolResult().WithRaw(string(resultJSON)).WithStructured(
		actiontypes.NewBrowserRunTaskData(result.Data))
}
