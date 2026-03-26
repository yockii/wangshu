package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type WebFetchTool struct {
	basic.SimpleTool
}

func NewWebFetchTool() *WebFetchTool {
	tool := new(WebFetchTool)
	tool.Name_ = "web_fetch"
	tool.Desc_ = "Fetch and extract readable content from a URL. Handles HTML, plain text, and JSON responses."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Request timeout in seconds (default: 15)",
			},
			"raw": map[string]any{
				"type":        "boolean",
				"description": "If true, return raw content instead of extracting readable text (default: false)",
			},
		},
		"required": []string{"url"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *WebFetchTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	targetURL := params["url"]
	if targetURL == "" {
		return types.NewToolResult().WithError(fmt.Errorf("url is required"))
	}

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("invalid URL: %w", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return types.NewToolResult().WithError(fmt.Errorf("URL must use http or https scheme"))
	}

	timeout := 15 * time.Second
	if timeoutStr := params["timeout"]; timeoutStr != "" {
		var duration float64
		if _, err := fmt.Sscanf(timeoutStr, "%f", &duration); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	raw := params["raw"] == "true"

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create request: %w", err))
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.NewToolResult().WithError(fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read response: %w", err))
	}

	contentType := resp.Header.Get("Content-Type")

	bodyStr := string(body)

	if raw {
		return types.NewToolResult().WithRaw(bodyStr).WithStructured(map[string]any{
			"data": bodyStr,
		})
	}

	// Extract readable content based on content type
	if strings.Contains(contentType, "text/html") {
		text := t.extractReadableText(bodyStr)
		return types.NewToolResult().WithRaw(text).WithStructured(map[string]any{
			"data": text,
		})
	} else if strings.Contains(contentType, "application/json") {
		text := t.formatJSON(bodyStr)
		return types.NewToolResult().WithRaw(text).WithStructured(map[string]any{
			"data": text,
		})
	} else if strings.Contains(contentType, "text/") {
		return types.NewToolResult().WithRaw(bodyStr).WithStructured(map[string]any{
			"data": bodyStr,
		})
	}

	// Default: try to extract readable text
	text := t.extractReadableText(bodyStr)
	return types.NewToolResult().WithRaw(text).WithStructured(map[string]any{
		"data": text,
	})
}

// extractReadableText extracts main content from HTML
func (t *WebFetchTool) extractReadableText(html string) string {
	// Remove script and style tags
	scriptRegex := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	styleRegex := regexp.MustCompile(`<style[^>]*>.*?</style>`)

	html = scriptRegex.ReplaceAllString(html, "")
	html = styleRegex.ReplaceAllString(html, "")

	// Remove HTML comments
	commentRegex := regexp.MustCompile(`<!--.*?-->`)
	html = commentRegex.ReplaceAllString(html, "")

	// Replace common block elements with newlines
	blockRegex := regexp.MustCompile(`</(div|p|h[1-6]|li|tr|article|section)>`)
	html = blockRegex.ReplaceAllString(html, "\n")

	// Remove all HTML tags
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text := tagRegex.ReplaceAllString(html, "")

	// Decode HTML entities
	text = t.decodeHTMLEntities(text)

	// Clean up whitespace
	lines := strings.Split(text, "\n")
	cleanedLines := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and very short lines (likely navigation)
		if line != "" && len(line) > 3 {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// decodeHTMLEntities decodes common HTML entities
func (t *WebFetchTool) decodeHTMLEntities(text string) string {
	// Common HTML entities
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&apos;": "'",
		"&nbsp;": " ",
		"&#39;":  "'",
		"&#34;":  "\"",
	}

	result := text
	for entity, replacement := range replacements {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	// Handle numeric entities like &#123; or &#x1F600;
	numericRegex := regexp.MustCompile(`&#(\d+);`)
	result = numericRegex.ReplaceAllStringFunc(result, func(match string) string {
		numStr := match[2 : len(match)-1]
		var num int
		fmt.Sscanf(numStr, "%d", &num)
		return string(rune(num))
	})

	hexRegex := regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	result = hexRegex.ReplaceAllStringFunc(result, func(match string) string {
		hexStr := match[3 : len(match)-1]
		var num int64
		fmt.Sscanf(hexStr, "%x", &num)
		return string(rune(num))
	})

	return result
}

// formatJSON formats JSON content for readability
func (t *WebFetchTool) formatJSON(jsonStr string) string {
	// Simple formatting - add proper indentation
	// In production, use json.MarshalIndent
	var pretty strings.Builder
	indent := 0
	inString := false

	for i, ch := range jsonStr {
		switch ch {
		case '{', '[':
			if !inString {
				pretty.WriteRune(ch)
				pretty.WriteRune('\n')
				indent++
				pretty.WriteString(strings.Repeat("  ", indent))
			} else {
				pretty.WriteRune(ch)
			}
		case '}', ']':
			if !inString {
				pretty.WriteRune('\n')
				indent--
				pretty.WriteString(strings.Repeat("  ", indent))
				pretty.WriteRune(ch)
			} else {
				pretty.WriteRune(ch)
			}
		case ',':
			if !inString {
				pretty.WriteRune(ch)
				pretty.WriteRune('\n')
				pretty.WriteString(strings.Repeat("  ", indent))
			} else {
				pretty.WriteRune(ch)
			}
		case '"':
			// Check if it's escaped
			if i == 0 || jsonStr[i-1] != '\\' {
				inString = !inString
			}
			pretty.WriteRune(ch)
		case ':':
			if !inString {
				pretty.WriteRune(ch)
				pretty.WriteRune(' ')
			} else {
				pretty.WriteRune(ch)
			}
		default:
			pretty.WriteRune(ch)
		}
	}

	return pretty.String()
}

// fetchWithAuth fetches a URL with authentication
func (t *WebFetchTool) fetchWithAuth(ctx context.Context, targetURL, authType, authToken string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	switch authType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+authToken)
	case "basic":
		req.Header.Set("Authorization", "Basic "+authToken)
	case "apikey":
		req.Header.Set("X-API-Key", authToken)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// extractMetadata extracts metadata from HTML (title, description, etc.)
func (t *WebFetchTool) extractMetadata(html string) map[string]string {
	metadata := make(map[string]string)

	// Extract title
	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	if match := titleRegex.FindStringSubmatch(html); len(match) > 1 {
		metadata["title"] = match[1]
	}

	// Extract meta description
	descRegex := regexp.MustCompile(`<meta[^>]*name=["']description["'][^>]*content=["'](.*?)["']`)
	if match := descRegex.FindStringSubmatch(html); len(match) > 1 {
		metadata["description"] = match[1]
	}

	// Extract og:title
	ogTitleRegex := regexp.MustCompile(`<meta[^>]*property=["']og:title["'][^>]*content=["'](.*?)["']`)
	if match := ogTitleRegex.FindStringSubmatch(html); len(match) > 1 {
		metadata["og_title"] = match[1]
	}

	return metadata
}
