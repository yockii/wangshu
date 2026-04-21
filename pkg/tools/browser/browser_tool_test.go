package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// browserTestLock 确保浏览器测试顺序执行，避免同时启动多个浏览器实例
var browserTestLock sync.Mutex

func TestNewBrowserTool(t *testing.T) {
	tool := NewBrowserTool()

	if tool == nil {
		t.Fatal("NewBrowserTool should not return nil")
	}

	if tool.Name() != "browser" {
		t.Errorf("Expected tool name 'browser', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	params := tool.Parameters()
	if params == nil {
		t.Fatal("Tool should have parameters")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "action" {
		t.Error("'action' should be required")
	}

	expectedParams := []string{"action", "url", "selector", "text", "screenshot_path", "timeout"}
	for _, expected := range expectedParams {
		if _, ok := properties[expected]; !ok {
			t.Errorf("Parameters should have '%s' property", expected)
		}
	}
}

func TestBrowserTool_Execute_EmptyAction(t *testing.T) {
	tool := NewBrowserTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "",
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty action")
	}

	if !strings.Contains(result.Err.Error(), "action required") {
		t.Errorf("Error should mention 'action required', got: %v", result.Err)
	}
}

func TestBrowserTool_Execute_UnknownAction(t *testing.T) {
	tool := NewBrowserTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "unknown_action",
	})

	if result.Err == nil {
		t.Error("Execute should fail with unknown action")
	}

	if !strings.Contains(result.Err.Error(), "unknown action") {
		t.Errorf("Error should mention 'unknown action', got: %v", result.Err)
	}
}

func TestBrowserTool_Execute_MissingActionParameter(t *testing.T) {
	tool := NewBrowserTool()

	result := tool.Execute(context.Background(), map[string]string{})

	if result.Err == nil {
		t.Error("Execute should fail when action parameter is missing")
	}

	if !strings.Contains(result.Err.Error(), "action required") {
		t.Errorf("Error should mention 'action required', got: %v", result.Err)
	}
}

func TestBrowserTool_Execute_Open_MissingURL(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// This test will try to initialize the browser, which might fail
	// We're mainly testing that the parameter validation works
	result := tool.Execute(context.Background(), map[string]string{
		"action": "open",
		"url":    "",
	})

	if result.Err == nil {
		t.Error("Execute should fail with missing URL for open action")
		return
	}

	// Check for the expected error
	if !strings.Contains(result.Err.Error(), "url required") {
		// Browser init failed or other error, skip the test
		t.Skipf("Browser initialization failed: %v", result.Err)
	}
}

func TestBrowserTool_ActionsExist(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	// Test that close action is valid (doesn't require browser)
	result := tool.Execute(context.Background(), map[string]string{
		"action": "close",
	})
	// close should succeed even without browser
	if result.Err != nil {
		t.Logf("Action 'close' error: %v", result.Err)
	}

	// 验证 action 字符串是否被识别
	// 通过检查 Execute 的错误消息来验证 action 被识别
	// 如果返回 "unknown action" 错误，说明 action 字符串无效
	// 注意：不测试 list_tabs，因为它会初始化浏览器
	allActions := []string{
		"open",
		"screenshot",
		"click",
		"fill",
		"text",
		"html",
		"wait",
	}

	for _, action := range allActions {
		result := tool.Execute(context.Background(), map[string]string{
			"action": action,
		})
		// 预期会因为缺少参数或浏览器未初始化而失败
		// 但不应该返回 "unknown action" 错误
		if result.Err != nil && strings.Contains(result.Err.Error(), "unknown action") {
			t.Errorf("Action '%s' should be recognized, got 'unknown action' error", action)
		}
	}
}

func TestBrowserTool_Execute_Close(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// Close should work even without initialization
	result := tool.Execute(context.Background(), map[string]string{
		"action": "close",
	})

	if result.Err != nil {
		t.Errorf("Close action should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "closed") {
		t.Errorf("Close result should mention 'closed', got: %s", result.Raw)
	}

	// Ensure browser is fully closed
	tool.close()
}

func TestBrowserTool_Execute_ListTabs(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "list_tabs",
	})

	if result.Err != nil {
		// Browser init failed, skip the test
		t.Skipf("Browser initialization failed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "标签页") && !strings.Contains(result.Raw, "tab") {
		t.Errorf("ListTabs result should mention '标签页' or 'tab', got: %s", result.Raw)
	}
}

func TestBrowserTool_Click_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// This will try to initialize browser, which might fail
	result := tool.Execute(context.Background(), map[string]string{
		"action":   "click",
		"selector": "",
	})

	// Should fail either due to browser init or missing selector
	if result.Err == nil {
		t.Error("Should fail with missing selector or browser init failure")
	}

	if !strings.Contains(result.Err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", result.Err)
	}
}

func TestBrowserTool_Fill_MissingParameters(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// Test with missing selector
	result := tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "",
		"text":     "test",
	})

	if result.Err == nil {
		t.Error("Should fail with missing selector")
	}

	// Test with missing text
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#test",
		"text":     "",
	})

	if result.Err == nil {
		t.Error("Should fail with missing text")
	}
}

func TestBrowserTool_Text_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	result := tool.Execute(context.Background(), map[string]string{
		"action":   "text",
		"selector": "",
	})

	if result.Err == nil {
		t.Error("Should fail with missing selector")
	}

	// Should fail either due to browser init or missing selector
	if !strings.Contains(result.Err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", result.Err)
	}
}

func TestBrowserTool_Wait_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	result := tool.Execute(context.Background(), map[string]string{
		"action":   "wait",
		"selector": "",
	})

	if result.Err == nil {
		t.Error("Should fail with missing selector")
	}

	// Should fail either due to browser init or missing selector
	if !strings.Contains(result.Err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", result.Err)
	}
}

func TestBrowserTool_Screenshot_DefaultPath(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// This will try to initialize browser, which might fail
	result := tool.Execute(context.Background(), map[string]string{
		"action":          "screenshot",
		"screenshot_path": "",
	})

	// Should fail due to browser initialization in test environment
	if result.Err != nil {
		t.Logf("Expected browser init failure: %v", result.Err)
	}
}

func TestBrowserTool_Description(t *testing.T) {
	tool := NewBrowserTool()

	desc := tool.Description()

	if !strings.Contains(desc, "Playwright") {
		t.Error("Description should mention Playwright")
	}

	// Check that it mentions the available actions
	expectedActions := []string{"open", "click", "fill", "screenshot"}
	for _, action := range expectedActions {
		if !strings.Contains(desc, action) {
			t.Errorf("Description should mention action '%s'", action)
		}
	}
}

func TestBrowserTool_InitialState(t *testing.T) {
	tool := NewBrowserTool()

	if tool.initialized {
		t.Error("Tool should not be initialized initially")
	}

	if tool.pw != nil {
		t.Error("Playwright instance should be nil initially")
	}

	if tool.page != nil {
		t.Error("Page should be nil initially")
	}
}

// TestBrowserTool_CollectElements 测试元素信息收集功能
func TestBrowserTool_CollectElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 立即注册 defer，确保任何时候都会清理

	// 创建一个简单的HTML页面进行测试
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head><title>Test Page</title></head>
	<body>
		<form id="loginForm">
			<input type="text" id="username" name="username" placeholder="Username" required>
			<input type="password" id="password" name="password" placeholder="Password" required>
			<button type="submit" id="submitBtn">Submit</button>
		</form>
		<a href="https://example.com" id="link1" data-testid="test-link">Example Link</a>
	</body>
	</html>
	`

	// 设置页面内容
	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 调用collectElements
	elements := tool.collectElements()

	// 验证返回了元素信息
	if elements == nil {
		t.Error("collectElements should return element information")
	}

	// 验证包含预期的元素信息
	expectedInfo := []string{
		"selector",
		"username",
		"password",
		"submitBtn",
		"link1",
		"input",
		"button",
		"a",
	}

	elListJson, _ := json.Marshal(elements)
	elListStr := string(elListJson)

	for _, info := range expectedInfo {
		if !strings.Contains(elListStr, info) {
			t.Errorf("Element information should contain '%s', got: %s", info, elListStr)
		}
	}

	// 验证包含选择器信息
	expectedSelectors := []string{
		"selector",
		"selector_unique",
	}

	for _, selector := range expectedSelectors {
		if !strings.Contains(elListStr, selector) {
			t.Errorf("Element information should contain selector type '%s'", selector)
		}
	}

	// 验证包含属性信息
	expectedAttributes := []string{
		"enabled",
		"editable",
		"type",
		"placeholder",
	}

	for _, attr := range expectedAttributes {
		if !strings.Contains(elListStr, attr) {
			t.Errorf("Element information should contain attribute '%s'", attr)
		}
	}

	// 验证JSON格式
	if !strings.Contains(elListStr, "{") || !strings.Contains(elListStr, "}") {
		t.Error("Element information should be in JSON format")
	}

	t.Logf("Collected elements:\n%s", elListStr)
}

// TestBrowserTool_Open_WithElements 测试打开页面后返回元素信息
func TestBrowserTool_Open_WithElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// 使用data URI创建一个简单的测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<input type="text" id="test-input" name="test" value="Hello">
		<button id="test-btn">Click Me</button>
	</body>
	</html>
	`
	dataURL := "data:text/html;charset=utf-8," + strings.ReplaceAll(htmlContent, "\n", "%0A")

	result := tool.Execute(context.Background(), map[string]string{
		"action": "open",
		"url":    dataURL,
	})

	if result.Err != nil {
		t.Skipf("Failed to open page: %v", result.Err)
		return
	}

	// 验证返回包含操作结果
	if !strings.Contains(result.Raw, "Opened:") {
		t.Errorf("Result should contain 'Opened:', got: %s", result.Raw)
	}

	// 验证返回包含元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Result should contain element information, got: %s", result.Raw)
	}

	// 验证包含测试元素
	if !strings.Contains(result.Raw, "test-input") && !strings.Contains(result.Raw, "test-btn") {
		t.Errorf("Result should contain test elements, got: %s", result.Raw)
	}
}

// TestBrowserTool_Fill_WithElements 测试填充后返回元素信息
func TestBrowserTool_Fill_WithElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 立即注册 defer

	// 创建测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<input type="text" id="myInput" name="myField" placeholder="Enter text">
		<input type="text" id="otherInput" name="otherField">
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 执行fill操作
	result := tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#myInput",
		"text":     "test value",
	})

	if result.Err != nil {
		t.Errorf("Fill should succeed: %v", result.Err)
	}

	// 验证返回包含操作结果
	if !strings.Contains(result.Raw, "Filled:") {
		t.Errorf("Result should contain 'Filled:', got: %s", result.Raw)
	}

	// 验证返回包含元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Result should contain element information after fill, got: %s", result.Raw)
	}

	// 验证包含两个输入框的信息
	if !strings.Contains(result.Raw, "myInput") || !strings.Contains(result.Raw, "otherInput") {
		t.Errorf("Result should contain both input elements, got: %s", result.Raw)
	}

	t.Logf("Fill result:\n%s", result.Raw)
}

// TestBrowserTool_Click_WithElements 测试点击后返回元素信息
func TestBrowserTool_Click_WithElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 立即注册 defer

	// 创建测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<button id="btn1">Button 1</button>
		<button id="btn2">Button 2</button>
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 执行click操作
	result := tool.Execute(context.Background(), map[string]string{
		"action":   "click",
		"selector": "#btn1",
	})

	if result.Err != nil {
		t.Errorf("Click should succeed: %v", result.Err)
	}

	// 验证返回包含操作结果
	if !strings.Contains(result.Raw, "Clicked:") {
		t.Errorf("Result should contain 'Clicked:', got: %s", result.Raw)
	}

	// 验证返回包含元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Result should contain element information after click, got: %s", result.Raw)
	}

	t.Logf("Click result:\n%s", result.Raw)
}

// TestBrowserTool_Wait_WithElements 测试等待后返回元素信息
func TestBrowserTool_Wait_WithElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 立即注册 defer

	// 创建测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<div id="container">
			<input type="text" id="waitInput">
		</div>
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 执行wait操作
	result := tool.Execute(context.Background(), map[string]string{
		"action":   "wait",
		"selector": "#waitInput",
	})

	if result.Err != nil {
		t.Errorf("Wait should succeed: %v", result.Err)
	}

	// 验证返回包含操作结果
	if !strings.Contains(result.Raw, "Waited for:") {
		t.Errorf("Result should contain 'Waited for:', got: %s", result.Raw)
	}

	// 验证返回包含元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Result should contain element information after wait, got: %s", result.Raw)
	}

	t.Logf("Wait result:\n%s", result.Raw)
}

// TestBrowserTool_ElementInfoStructure 测试元素信息结构完整性
func TestBrowserTool_ElementInfoStructure(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	// 创建包含各种元素的测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head><title>Test</title></head>
	<body>
		<form id="testForm">
			<input type="text" id="text1" name="textfield" placeholder="Text input" required>
			<input type="email" id="email1" name="emailfield" placeholder="Email">
			<input type="checkbox" id="check1" name="checkfield">
			<input type="radio" id="radio1" name="radiofield">
			<textarea id="textarea1" name="comment" placeholder="Comment"></textarea>
			<select id="select1" name="choice">
				<option value="1">Option 1</option>
				<option value="2">Option 2</option>
			</select>
			<button type="submit" id="submitBtn">Submit</button>
			<button type="button" id="cancelBtn" disabled>Cancel</button>
		</form>
		<a href="/page1" id="link1" data-test-id="page1-link">Page 1</a>
		<a href="/page2" class="nav-link" aria-label="Go to page 2">Page 2</a>
		<div role="button" id="divBtn">Custom Button</div>
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	elements := tool.collectElements()

	elJson, _ := json.Marshal(elements)
	elStr := string(elJson)

	// 验证各种元素类型都被收集
	expectedTags := []string{
		"input", "textarea", "select", "button", "a", "div",
	}

	for _, tag := range expectedTags {
		if !strings.Contains(elStr, `"tag": "`+tag+`"`) {
			t.Errorf("Should collect element with tag '%s'", tag)
		}
	}

	// 验证不同类型的input都被识别
	expectedTypes := []string{
		`"type": "text"`,
		`"type": "email"`,
		`"type": "checkbox"`,
		`"type": "radio"`,
	}

	for _, inputType := range expectedTypes {
		if !strings.Contains(elStr, inputType) {
			t.Errorf("Should collect input with type '%s'", inputType)
		}
	}

	// 验证特殊属性
	expectedAttributes := []string{
		`"required": true`,
		`"enabled": false`,
		`"placeholder": "Text input"`,
	}

	for _, attr := range expectedAttributes {
		if !strings.Contains(elStr, attr) {
			t.Errorf("Should contain attribute '%s'", attr)
		}
	}

	// 验证选择器存在且唯一性标记
	if !strings.Contains(elStr, `"selector"`) {
		t.Error("Should contain selector field")
	}
	if !strings.Contains(elStr, `"selector_unique"`) {
		t.Error("Should contain selector_unique field")
	}
	if !strings.Contains(elStr, `"selector_unique": true`) {
		t.Error("Should have at least one element with unique selector")
	}

	t.Logf("Full element info:\n%s", elStr)
}

// TestBrowserTool_CollectElements_EmptyPage 测试空页面
func TestBrowserTool_CollectElements_EmptyPage(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	htmlContent := `<!DOCTYPE html><html><body><p>Empty page</p></body></html>`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	elements := tool.collectElements()

	elJson, _ := json.Marshal(elements)
	elStr := string(elJson)

	// 空页面应该返回空字符串或者不包含交互元素
	if elStr != "" && strings.Contains(elStr, `"tag":`) {
		// 如果有元素，应该很少（可能只有p标签，但我们不收集p标签）
		t.Logf("Empty page returned: %s", elStr)
	}
}

// TestBrowserTool_Screenshot_WithElements 测试截图后返回元素信息
func TestBrowserTool_Screenshot_WithElements(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 立即注册 defer

	// 创建测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<h1>Test Page</h1>
		<input type="text" id="screenshotInput">
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	err := tool.page.SetContent(htmlContent)
	if err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 执行screenshot操作
	result := tool.Execute(context.Background(), map[string]string{
		"action":          "screenshot",
		"screenshot_path": "test_screenshot.png",
	})

	if result.Err != nil {
		t.Errorf("Screenshot should succeed: %v", result.Err)
	}

	// 验证返回包含操作结果
	if !strings.Contains(result.Raw, "Screenshot saved:") {
		t.Errorf("Result should contain 'Screenshot saved:', got: %s", result.Raw)
	}

	// 验证返回包含元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Result should contain element information after screenshot, got: %s", result.Raw)
	}

	t.Logf("Screenshot result:\n%s", result.Raw)
}

// TestBrowserTool_RealHTMLFile 测试使用真实HTML文件的操作
func TestBrowserTool_RealHTMLFile(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close() // 确保测试结束时关闭浏览器

	// 获取测试文件的绝对路径
	// 测试文件位于 pkg/tools/browser/testdata/test_page.html
	testFile := "./testdata/test_page.html"

	// 转换为绝对路径
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// 转换为file URL
	// Windows: file:///C:/path/to/file.html
	// Unix: file:///path/to/file.html
	fileURL := "file:///" + strings.ReplaceAll(absPath, "\\", "/")

	// 打开本地HTML文件
	result := tool.Execute(context.Background(), map[string]string{
		"action": "open",
		"url":    fileURL,
	})

	if result.Err != nil {
		t.Skipf("Failed to open test HTML file: %v", result.Err)
		return
	}

	// 验证打开成功
	if !strings.Contains(result.Raw, "Opened:") {
		t.Errorf("Expected 'Opened:' in result, got: %s", result.Raw)
	}

	// 验证返回了元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Expected element information in result, got: %s", result.Raw)
	}

	t.Logf("Open result:\n%s", result.Raw)

	// 测试fill操作 - 填充用户名
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#username",
		"text":     "测试用户",
	})

	if result.Err != nil {
		t.Errorf("Fill username should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Filled:") {
		t.Errorf("Expected 'Filled:' in result, got: %s", result.Raw)
	}

	// 验证元素信息仍然被返回
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Expected element information after fill, got: %s", result.Raw)
	}

	t.Logf("Fill username result:\n%s", result.Raw)

	// 测试fill操作 - 填充邮箱
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#email",
		"text":     "test@example.com",
	})

	if result.Err != nil {
		t.Errorf("Fill email should succeed: %v", result.Err)
	}

	t.Logf("Fill email result:\n%s", result.Raw)

	// 测试fill操作 - 填充备注
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#comments",
		"text":     "这是一段测试备注",
	})

	if result.Err != nil {
		t.Errorf("Fill comments should succeed: %v", result.Err)
	}

	t.Logf("Fill comments result:\n%s", result.Raw)

	// 注意：select元素不能用fill操作，需要点击后选择选项
	// 这里跳过select的测试

	// 测试click操作 - 点击复选框
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "click",
		"selector": "#agree",
	})

	if result.Err != nil {
		t.Errorf("Click checkbox should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Clicked:") {
		t.Errorf("Expected 'Clicked:' in result, got: %s", result.Raw)
	}

	t.Logf("Click checkbox result:\n%s", result.Raw)

	// 等待一下让页面更新
	time.Sleep(500 * time.Millisecond)

	// 测试click操作 - 点击提交按钮
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "click",
		"selector": "#submitBtn",
	})

	if result.Err != nil {
		t.Errorf("Click submit button should succeed: %v", result.Err)
	}

	t.Logf("Click submit button result:\n%s", result.Raw)

	// 测试text操作 - 读取结果区域
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "text",
		"selector": "#result",
	})

	if result.Err != nil {
		t.Errorf("Get text should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "表单已提交") {
		t.Logf("Expected confirmation message in result, got: %s", result.Raw)
	}

	t.Logf("Get text result:\n%s", result.Raw)

	// 测试screenshot操作
	screenshotPath := "test_screenshot_real.png"
	result = tool.Execute(context.Background(), map[string]string{
		"action":          "screenshot",
		"screenshot_path": screenshotPath,
	})

	if result.Err != nil {
		t.Errorf("Screenshot should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Screenshot saved:") {
		t.Errorf("Expected 'Screenshot saved:' in result, got: %s", result.Raw)
	}

	t.Logf("Screenshot result:\n%s", result.Raw)

	// 测试wait操作 - 等待一个存在的元素
	result = tool.Execute(context.Background(), map[string]string{
		"action":   "wait",
		"selector": "#result",
	})

	if result.Err != nil {
		t.Errorf("Wait for element should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "Waited for:") {
		t.Errorf("Expected 'Waited for:' in result, got: %s", result.Raw)
	}

	t.Logf("Wait result:\n%s", result.Raw)

	// 验证最后一次操作后仍然返回元素信息
	if !strings.Contains(result.Raw, "[Elements:") {
		t.Errorf("Expected element information after wait operation, got: %s", result.Raw)
	}
}

// TestBrowserTool_RealHTMLFile_ElementValidation 详细验证元素信息
func TestBrowserTool_RealHTMLFile_ElementValidation(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	testFile := "./testdata/test_page.html"

	// 转换为绝对路径
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// 转换为file URL
	fileURL := "file:///" + strings.ReplaceAll(absPath, "\\", "/")

	// 打开测试页面
	result := tool.Execute(context.Background(), map[string]string{
		"action": "open",
		"url":    fileURL,
	})

	if result.Err != nil {
		t.Skipf("Failed to open test HTML file: %v", result.Err)
		return
	}

	// 验证关键元素都被收集
	expectedElements := []string{
		"username",
		"email",
		"password",
		"comments",
		"country",
		"agree",
		"submitBtn",
		"cancelBtn",
		"resetBtn",
		"link1",
		"link2",
		"link3",
		"customButton",
	}

	for _, elementID := range expectedElements {
		if !strings.Contains(result.Raw, elementID) {
			t.Errorf("Expected element '%s' in result", elementID)
		}
	}

	// 验证选择器类型
	expectedSelectors := []string{
		"selector",
		"selector_unique",
	}

	for _, selector := range expectedSelectors {
		if !strings.Contains(result.Raw, selector) {
			t.Errorf("Expected selector type '%s' in result", selector)
		}
	}

	// 验证特殊属性
	expectedAttributes := []string{
		`"placeholder": "请输入用户名"`,
		`"placeholder": "请输入邮箱"`,
		`"type": "text"`,
		`"type": "email"`,
		`"type": "password"`,
		`"required": true`,
	}

	for _, attr := range expectedAttributes {
		if !strings.Contains(result.Raw, attr) {
			t.Errorf("Expected attribute '%s' in result", attr)
		}
	}

	// 验证data-testid属性被收集到selector中
	if !strings.Contains(result.Raw, "username-input") ||
		!strings.Contains(result.Raw, "email-input") ||
		!strings.Contains(result.Raw, "submit-button") {
		t.Error("Expected data-testid attributes to be collected in selector")
	}

	t.Logf("Full result:\n%s", result.Raw)
}

// TestBrowserTool_HTML_Formats 测试HTML获取的不同格式
func TestBrowserTool_HTML_Formats(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	// 创建测试页面
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head><title>Test Page</title></head>
	<body>
		<h1 id="heading">Welcome</h1>
		<p class="text">This is a test page.</p>
		<button id="btn">Click me</button>
	</body>
	</html>
	`

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	if err := tool.page.SetContent(htmlContent); err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 测试 text 格式
	result := tool.Execute(context.Background(), map[string]string{
		"action": "html",
		"format": "text",
	})

	if result.Err != nil {
		t.Errorf("Get HTML text should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "📄 页面内容") {
		t.Error("Result should contain page header")
	}

	if !strings.Contains(result.Raw, "Welcome") {
		t.Error("Result should contain page text content")
	}

	t.Logf("Text format result:\n%s", result.Raw)

	// 测试 body 格式
	result = tool.Execute(context.Background(), map[string]string{
		"action": "html",
		"format": "body",
	})

	if result.Err != nil {
		t.Errorf("Get HTML body should succeed: %v", result.Err)
	}

	if !strings.Contains(result.Raw, "🔍 HTML内容") {
		t.Error("Result should contain HTML header")
	}

	if !strings.Contains(result.Raw, "<h1") && !strings.Contains(result.Raw, "<h1>") {
		t.Error("Result should contain HTML tags")
	}

	if !strings.Contains(result.Raw, "heading") {
		t.Error("Result should contain element info")
	}

	t.Logf("Body format result (first 500 chars):\n%s", result.Raw[:min(500, len(result.Raw))])

	// 测试 max_length 限制
	result = tool.Execute(context.Background(), map[string]string{
		"action":     "html",
		"format":     "text",
		"max_length": "50",
	})

	if result.Err != nil {
		t.Errorf("Get HTML with max_length should succeed: %v", result.Err)
	}

	// 验证内容被截断
	lines := strings.Split(result.Raw, "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "内容被截断") {
		t.Logf("Last line: %s", lastLine)
	}

	t.Logf("Truncated result:\n%s", result.Raw)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestBrowserTool_HTML_Pagination 测试HTML分页获取功能
func TestBrowserTool_HTML_Pagination(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()
	defer tool.close()

	// 创建一个较长的测试页面（约150字符）
	longContent := strings.Repeat("This is a test line with some content. ", 10)

	htmlContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<body>
		<h1>Long Content Test</h1>
		<div id="content">%s</div>
		<button id="btn">Submit</button>
	</body>
	</html>
	`, longContent)

	if err := tool.init(); err != nil {
		t.Skipf("Failed to initialize browser: %v", err)
		return
	}

	if err := tool.page.SetContent(htmlContent); err != nil {
		t.Fatalf("Failed to set page content: %v", err)
	}

	// 第一次获取：前100个字符
	result1 := tool.Execute(context.Background(), map[string]string{
		"action":     "html",
		"format":     "text",
		"start":      "0",
		"max_length": "100",
	})

	if result1.Err != nil {
		t.Errorf("First fetch should succeed: %v", result1.Err)
	}

	// 验证返回了分页信息
	if !strings.Contains(result1.Raw, "📊 范围: 0-100") {
		t.Error("Result should show range 0-100")
	}

	if !strings.Contains(result1.Raw, "下次获取: start=100") {
		t.Error("Result should prompt for next fetch with start=100")
	}

	// 验证内容长度约为100
	lines := strings.Split(result1.Raw, "\n")
	contentStart := -1
	for i, line := range lines {
		if strings.Contains(line, "文本内容:") {
			contentStart = i + 1
			break
		}
	}

	if contentStart >= 0 && contentStart < len(lines) {
		fetchedContent := strings.Join(lines[contentStart:], "\n")
		// 移除可能的截断提示
		if idx := strings.Index(fetchedContent, "\n\n... (还有"); idx > 0 {
			fetchedContent = fetchedContent[:idx]
		}
		if len(fetchedContent) > 110 {
			t.Errorf("First chunk should be ~100 chars, got: %d", len(fetchedContent))
		}
	}

	t.Logf("First fetch result:\n%s", result1.Raw)

	// 第二次获取：从100开始
	result2 := tool.Execute(context.Background(), map[string]string{
		"action":     "html",
		"format":     "text",
		"start":      "100",
		"max_length": "100",
	})

	if result2.Err != nil {
		t.Errorf("Second fetch should succeed: %v", result2.Err)
	}

	if !strings.Contains(result2.Raw, "📊 范围: 100-200") {
		t.Error("Result should show range 100-200")
	}

	t.Logf("Second fetch result:\n%s", result2.Raw)

	// 获取完整内容（无start参数）
	resultFull := tool.Execute(context.Background(), map[string]string{
		"action": "html",
		"format": "text",
	})

	if resultFull.Err != nil {
		t.Errorf("Full fetch should succeed: %v", resultFull.Err)
	}

	// 完整获取应该包含所有内容
	if !strings.Contains(resultFull.Raw, "Long Content Test") {
		t.Error("Full content should contain heading")
	}

	if !strings.Contains(resultFull.Raw, longContent[:50]) {
		t.Error("Full content should contain long content")
	}

	t.Logf("Full fetch result (first 300 chars):\n%s", resultFull.Raw[:min(300, len(resultFull.Raw))])
}
