package network

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/yockii/yoclaw/pkg/tools/basic"
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
	tool.Desc_ = "Browser automation tool (Playwright installed, ready to use). Actions: open(url), click(selector), fill(selector,text), text(selector), html(), screenshot(path), wait(selector), close()."
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
	return "Opened: " + url, nil
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
	return "Screenshot saved: " + path, nil
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
	return "Clicked: " + selector, nil
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
	return fmt.Sprintf("Filled %s", selector), nil
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
	return "Waited for: " + selector, nil
}
