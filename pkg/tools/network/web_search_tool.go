package network

import (
	"context"
	"encoding/json"
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

type WebSearchTool struct {
	basic.SimpleTool
}

func NewWebSearchTool() *WebSearchTool {
	tool := new(WebSearchTool)
	tool.Name_ = "web_search"
	tool.Desc_ = "Search the web for information. Supports DuckDuckGo, Baidu (China-friendly), and can automatically choose based on timezone."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
			"num_results": map[string]any{
				"type":        "number",
				"description": "Number of results to return (default: 10)",
			},
			"engine": map[string]any{
				"type":        "string",
				"description": "Search engine: baidu, auto (chooses based on timezone)",
				"enum":        []string{"baidu", "auto"},
			},
		},
		"required": []string{"query"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *WebSearchTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	query := params["query"]
	if query == "" {
		return types.NewToolResult().WithError(fmt.Errorf("query is required"))
	}

	engine := params["engine"]
	if engine == "" {
		engine = "baidu"
	}

	numResults := 10
	if numStr := params["num_results"]; numStr != "" {
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil && n > 0 && n <= 50 {
			numResults = n
		}
	}

	// Auto-detect engine based on timezone if requested
	if engine == "auto" {
		engine = t.detectBestEngine()
	}

	switch engine {
	case "duckduckgo":
		return t.searchDuckDuckGo(ctx, query, numResults)
	case "baidu":
		return t.searchBaidu(ctx, query, numResults)
	default:
		return types.NewToolResult().WithError(fmt.Errorf("unsupported search engine: %s", engine))
	}
}

// detectBestEngine chooses the best search engine based on system timezone
func (t *WebSearchTool) detectBestEngine() string {
	// Get current timezone
	timezone := time.Now().Location().String()

	// China timezone - use Baidu
	if strings.Contains(timezone, "Asia/Shanghai") ||
		strings.Contains(timezone, "Asia/Chongqing") ||
		strings.Contains(timezone, "Asia/Harbin") ||
		strings.Contains(timezone, "Asia/Urumqi") ||
		strings.Contains(timezone, "Asia/Hong_Kong") ||
		strings.Contains(timezone, "Asia/Macau") ||
		strings.Contains(timezone, "Asia/Taipei") {
		return "baidu"
	}

	// Default to DuckDuckGo for rest of the world
	return "duckduckgo"
}

// searchDuckDuckGo performs a search using DuckDuckGo (HTML parsing, no API)
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, numResults int) *types.ToolResult {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create request: %w", err))
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("search request failed: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read response: %w", err))
	}

	results := t.parseDuckDuckGoResults(string(body), numResults)

	if len(results) == 0 {
		return types.NewToolResult().WithRaw("No results found")
	}

	output := fmt.Sprintf("Found %d results for '%s':\n\n", len(results), query)
	for i, result := range results {
		output += fmt.Sprintf("%d. %s\n", i+1, result.Title)
		output += fmt.Sprintf("   %s\n", result.URL)
		if result.Snippet != "" {
			output += fmt.Sprintf("   %s\n\n", result.Snippet)
		} else {
			output += "\n"
		}
	}

	return types.NewToolResult().WithRaw(output).WithStructured(map[string]any{
		"data": results,
	})
}

// searchBaidu performs a search using Baidu (HTML parsing, China-friendly)
func (t *WebSearchTool) searchBaidu(ctx context.Context, query string, numResults int) *types.ToolResult {
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d", url.QueryEscape(query), numResults)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create request: %w", err))
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("search request failed: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to read response: %w", err))
	}

	results := t.parseBaiduResults(string(body), numResults)

	if len(results) == 0 {
		return types.NewToolResult().WithRaw("No results found")
	}

	output := fmt.Sprintf("找到 %d 个关于 '%s' 的结果:\n\n", len(results), query)
	for i, result := range results {
		output += fmt.Sprintf("%d. %s\n", i+1, result.Title)
		output += fmt.Sprintf("   %s\n", result.URL)
		if result.Snippet != "" {
			output += fmt.Sprintf("   %s\n\n", result.Snippet)
		} else {
			output += "\n"
		}
	}

	return types.NewToolResult().WithRaw(output).WithStructured(map[string]any{
		"data": results,
	})
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// parseDuckDuckGoResults parses DuckDuckGo HTML response
func (t *WebSearchTool) parseDuckDuckGoResults(html string, maxResults int) []SearchResult {
	results := []SearchResult{}

	// DuckDuckGo uses class="result__a" for links
	// Pattern: <a class="result__a" href="...">Title</a>
	linkRegex := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	matches := linkRegex.FindAllStringSubmatch(html, -1)

	for i, match := range matches {
		if i >= maxResults {
			break
		}

		if len(match) >= 3 {
			result := SearchResult{
				URL: match[1],
				// Strip HTML tags from title
				Title: stripHTML(match[2]),
			}

			// Try to find snippet nearby (class="result__snippet")
			// This is simplified - in production, use a proper HTML parser
			results = append(results, result)
		}
	}

	return results
}

// parseBaiduResults parses Baidu HTML response
func (t *WebSearchTool) parseBaiduResults(html string, maxResults int) []SearchResult {
	results := []SearchResult{}

	// Baidu uses class="t" for title and class="c-abstract" for snippet
	// Pattern: <a href="...">Title</a>
	titleRegex := regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	matches := titleRegex.FindAllStringSubmatch(html, -1)

	// Also look for data-tools (Baidu's actual link format)
	dataToolsRegex := regexp.MustCompile(`data-tools='{"title":"([^"]+)","url":"([^"]+)"'`)
	dataToolsMatches := dataToolsRegex.FindAllStringSubmatch(html, -1)

	// Prefer data-tools results (more reliable)
	for i, match := range dataToolsMatches {
		if i >= maxResults {
			break
		}

		if len(match) >= 3 {
			// Decode Unicode escapes in title
			title := decodeUnicodeEscape(match[1])
			// Decode URL
			resultURL := match[2]

			results = append(results, SearchResult{
				Title: stripHTML(title),
				URL:   resultURL,
			})
		}
	}

	// If not enough results from data-tools, fall back to regular links
	if len(results) < maxResults {
		for _, match := range matches {
			if len(results) >= maxResults {
				break
			}

			if len(match) >= 3 {
				// Skip if this looks like a navigation link
				title := stripHTML(match[2])
				if len(title) < 5 || strings.Contains(title, "百度") {
					continue
				}

				results = append(results, SearchResult{
					Title: title,
					URL:   match[1],
				})
			}
		}
	}

	// Try to extract snippets (class="c-abstract")
	snippetRegex := regexp.MustCompile(`<div[^>]*class="c-abstract"[^>]*>(.*?)</div>`)
	snippetMatches := snippetRegex.FindAllStringSubmatch(html, -1)

	for i, match := range snippetMatches {
		if i < len(results) && len(match) >= 2 {
			snippet := stripHTML(match[1])
			snippet = strings.TrimSpace(snippet)
			results[i].Snippet = snippet
		}
	}

	return results
}

// stripHTML removes HTML tags from a string
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(s, "")
}

// decodeUnicodeEscape decodes Unicode escape sequences like \u003d
func decodeUnicodeEscape(s string) string {
	re := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		hex := match[2:]
		var code int
		fmt.Sscanf(hex, "%x", &code)
		return string(rune(code))
	})
}

// searchWithAPI performs search using a search API (for future use with API keys)
func (t *WebSearchTool) searchWithAPI(ctx context.Context, engine, query string, numResults int) *types.ToolResult {
	// Placeholder for API-based search
	// This can be extended to support:
	// - SerpAPI
	// - Bing Search API
	// - DuckDuckGo Instant Answer API

	return types.NewToolResult().WithRaw("API-based search not yet implemented")
}

// searchSerpAPI performs search using SerpAPI (requires API key)
func (t *WebSearchTool) searchSerpAPI(ctx context.Context, query string, numResults int, apiKey string) *types.ToolResult {
	if apiKey == "" {
		return types.NewToolResult().WithError(fmt.Errorf("SerpAPI key is required"))
	}

	searchURL := fmt.Sprintf("https://serpapi.com/search?engine=baidu&q=%s&num=%d&api_key=%s",
		url.QueryEscape(query), numResults, apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to create request: %w", err))
	}

	resp, err := client.Do(req)
	if err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("search request failed: %w", err))
	}
	defer resp.Body.Close()

	var result struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return types.NewToolResult().WithError(fmt.Errorf("failed to decode response: %w", err))
	}

	if len(result.Organic) == 0 {
		return types.NewToolResult().WithRaw("No results found")
	}

	output := fmt.Sprintf("Found %d results for '%s':\n\n", len(result.Organic), query)
	for i, r := range result.Organic {
		output += fmt.Sprintf("%d. %s\n", i+1, r.Title)
		output += fmt.Sprintf("   %s\n", r.Link)
		output += fmt.Sprintf("   %s\n\n", r.Snippet)
	}

	return types.NewToolResult().WithRaw(output).WithStructured(map[string]any{
		"data": result.Organic,
	})
}
