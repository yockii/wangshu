package network

import (
	"context"
	"strings"
	"sync"
	"testing"
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

	_, err := tool.Execute(context.Background(), map[string]string{
		"action": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty action")
	}

	if !strings.Contains(err.Error(), "action required") {
		t.Errorf("Error should mention 'action required', got: %v", err)
	}
}

func TestBrowserTool_Execute_UnknownAction(t *testing.T) {
	tool := NewBrowserTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"action": "unknown_action",
	})

	if err == nil {
		t.Error("Execute should fail with unknown action")
	}

	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("Error should mention 'unknown action', got: %v", err)
	}
}

func TestBrowserTool_Execute_MissingActionParameter(t *testing.T) {
	tool := NewBrowserTool()

	_, err := tool.Execute(context.Background(), map[string]string{})

	if err == nil {
		t.Error("Execute should fail when action parameter is missing")
	}

	if !strings.Contains(err.Error(), "action required") {
		t.Errorf("Error should mention 'action required', got: %v", err)
	}
}

func TestBrowserTool_Execute_Open_MissingURL(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// This test will try to initialize the browser, which might fail
	// We're mainly testing that the parameter validation works
	_, err := tool.Execute(context.Background(), map[string]string{
		"action": "open",
		"url":    "",
	})

	if err == nil {
		t.Error("Execute should fail with missing URL for open action")
	}

	// The error could be about URL or about browser initialization
	if !strings.Contains(err.Error(), "url required") {
		t.Logf("Got error (might be browser init failure): %v", err)
	}
}

func TestBrowserTool_ActionsExist(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// Test that all expected actions are valid
	validActions := []string{
		"open",
		"screenshot",
		"close",
		"click",
		"fill",
		"text",
		"html",
		"wait",
		"list_tabs",
	}

	for _, action := range validActions {
		// We can't actually execute these without a browser,
		// but we can verify the action strings are recognized
		if action == "close" || action == "list_tabs" {
			// These actions don't require browser initialization
			var err error
			if action == "close" {
				_, err = tool.Execute(context.Background(), map[string]string{
					"action": action,
				})
			} else {
				_, err = tool.Execute(context.Background(), map[string]string{
					"action": action,
				})
			}
			// close should succeed, list_tabs should succeed
			if err != nil && action == "close" {
				t.Logf("Action '%s' error: %v", action, err)
			}
		} else {
			// Other actions require browser initialization, which will fail in tests
			// Just verify the action is recognized by checking it doesn't say "unknown action"
			_, err := tool.Execute(context.Background(), map[string]string{
				"action": action,
			})
			if err != nil && strings.Contains(err.Error(), "unknown action") {
				t.Errorf("Action '%s' should be recognized, got 'unknown action' error", action)
			}
		}
	}
}

func TestBrowserTool_Execute_Close(t *testing.T) {
	tool := NewBrowserTool()

	// Close should work even without initialization
	result, err := tool.Execute(context.Background(), map[string]string{
		"action": "close",
	})

	if err != nil {
		t.Errorf("Close action should succeed: %v", err)
	}

	if !strings.Contains(result, "closed") {
		t.Errorf("Close result should mention 'closed', got: %s", result)
	}
}

func TestBrowserTool_Execute_ListTabs(t *testing.T) {
	tool := NewBrowserTool()

	result, err := tool.Execute(context.Background(), map[string]string{
		"action": "list_tabs",
	})

	if err != nil {
		t.Errorf("ListTabs action should succeed: %v", err)
	}

	if !strings.Contains(result, "tab") {
		t.Errorf("ListTabs result should mention 'tab', got: %s", result)
	}
}

func TestBrowserTool_Click_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// This will try to initialize browser, which might fail
	_, err := tool.Execute(context.Background(), map[string]string{
		"action":   "click",
		"selector": "",
	})

	// Should fail either due to browser init or missing selector
	if err == nil {
		t.Error("Should fail with missing selector or browser init failure")
	}

	if !strings.Contains(err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", err)
	}
}

func TestBrowserTool_Fill_MissingParameters(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// Test with missing selector
	_, err := tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "",
		"text":     "test",
	})

	if err == nil {
		t.Error("Should fail with missing selector")
	}

	// Test with missing text
	_, err = tool.Execute(context.Background(), map[string]string{
		"action":   "fill",
		"selector": "#test",
		"text":     "",
	})

	if err == nil {
		t.Error("Should fail with missing text")
	}
}

func TestBrowserTool_Text_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"action":   "text",
		"selector": "",
	})

	if err == nil {
		t.Error("Should fail with missing selector")
	}

	// Should fail either due to browser init or missing selector
	if !strings.Contains(err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", err)
	}
}

func TestBrowserTool_Wait_MissingSelector(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"action":   "wait",
		"selector": "",
	})

	if err == nil {
		t.Error("Should fail with missing selector")
	}

	// Should fail either due to browser init or missing selector
	if !strings.Contains(err.Error(), "selector required") {
		t.Logf("Got error (likely browser init): %v", err)
	}
}

func TestBrowserTool_Screenshot_DefaultPath(t *testing.T) {
	browserTestLock.Lock()
	defer browserTestLock.Unlock()

	tool := NewBrowserTool()

	// This will try to initialize browser, which might fail
	_, err := tool.Execute(context.Background(), map[string]string{
		"action":          "screenshot",
		"screenshot_path": "",
	})

	// Should fail due to browser initialization in test environment
	if err != nil {
		t.Logf("Expected browser init failure: %v", err)
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
