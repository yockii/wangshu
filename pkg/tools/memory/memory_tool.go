package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yockii/yoclaw/pkg/constant"
	"github.com/yockii/yoclaw/pkg/tools/basic"
)

type MemoryTool struct {
	basic.SimpleTool
}

func NewMemoryTool() *MemoryTool {
	tool := new(MemoryTool)
	tool.Name_ = "memory"
	tool.Desc_ = "Search and retrieve stored memories. Supports searching by content and retrieving specific memory entries."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: search, get, list",
				"enum":        []string{"search", "get", "list"},
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (for 'search' action)",
			},
			"date": map[string]any{
				"type":        "string",
				"description": "Memory date in YYYY-MM-DD format (for 'get' action). Defaults to today.",
			},
			"days_back": map[string]any{
				"type":        "number",
				"description": "Number of days back to search (default: 7)",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of results to return (default: 10)",
			},
		},
		"required": []string{"action"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (t *MemoryTool) execute(ctx context.Context, params map[string]string) (string, error) {
	action := params["action"]
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "search":
		return t.searchMemory(params)
	case "get":
		return t.getMemory(params)
	case "list":
		return t.listMemories(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *MemoryTool) searchMemory(params map[string]string) (string, error) {
	query := params["query"]
	if query == "" {
		return "", fmt.Errorf("query is required for search action")
	}

	daysBack := 7
	if daysStr := params["days_back"]; daysStr != "" {
		var n int
		if _, err := fmt.Sscanf(daysStr, "%d", &n); err == nil && n > 0 {
			daysBack = n
		}
	}

	limit := 10
	if limitStr := params["limit"]; limitStr != "" {
		var n int
		if _, err := fmt.Sscanf(limitStr, "%d", &n); err == nil && n > 0 {
			limit = n
		}
	}

	workspaceDir := params[constant.ToolCallParamWorkspace]

	if workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not set")
	}

	memoryDir := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory)
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return "No memories found (memory directory does not exist)", nil
	}

	// Search through recent memory files
	results := []string{}
	now := time.Now()

	for i := 0; i < daysBack; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		memoryFile := filepath.Join(memoryDir, dateStr+constant.ExtMD)

		content, err := os.ReadFile(memoryFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to read memory file %s: %w", memoryFile, err)
			}
			continue
		}

		// Search for the query in the content
		if t.containsQuery(string(content), query) {
			// Extract relevant snippets
			snippets := t.extractSnippets(string(content), query, 3)
			result := fmt.Sprintf("📅 %s\n", dateStr)
			for _, snippet := range snippets {
				result += fmt.Sprintf("  %s\n", snippet)
			}
			results = append(results, result)

			if len(results) >= limit {
				break
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No memories found matching '%s' in the past %d days", query, daysBack), nil
	}

	output := fmt.Sprintf("Found %d memories matching '%s':\n\n", len(results), query)
	output += strings.Join(results, "\n")
	return output, nil
}

func (t *MemoryTool) getMemory(params map[string]string) (string, error) {
	dateStr := params["date"]
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	workspaceDir := params[constant.ToolCallParamWorkspace]

	if workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not set")
	}

	memoryFile := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory, dateStr+constant.ExtMD)

	content, err := os.ReadFile(memoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("No memory found for %s", dateStr), nil
		}
		return "", fmt.Errorf("failed to read memory file: %w", err)
	}

	return string(content), nil
}

func (t *MemoryTool) listMemories(params map[string]string) (string, error) {
	daysBack := 30
	if daysStr := params["days_back"]; daysStr != "" {
		var n int
		if _, err := fmt.Sscanf(daysStr, "%d", &n); err == nil && n > 0 {
			daysBack = n
		}
	}

	workspaceDir := params[constant.ToolCallParamWorkspace]

	if workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not set")
	}

	memoryDir := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory)
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return "No memories found (memory directory does not exist)", nil
	}

	// List memory files
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return "", fmt.Errorf("failed to read memory directory: %w", err)
	}

	// Filter and sort memory files
	memories := []string{}
	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a .md file matching date format
		name := entry.Name()
		if !strings.HasSuffix(name, constant.ExtMD) {
			continue
		}

		dateStr := strings.TrimSuffix(name, constant.ExtMD)
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Check if within days back
		daysDiff := int(now.Sub(date).Hours() / 24)
		if daysDiff > daysBack {
			continue
		}

		// Get file size
		info, _ := entry.Info()
		size := info.Size()

		memories = append(memories, fmt.Sprintf("📅 %s (%d bytes)", dateStr, size))
	}

	if len(memories) == 0 {
		return fmt.Sprintf("No memories found in the past %d days", daysBack), nil
	}

	// Sort by date (reverse)
	for i := 0; i < len(memories)-1; i++ {
		for j := i + 1; j < len(memories); j++ {
			if memories[i] < memories[j] {
				memories[i], memories[j] = memories[j], memories[i]
			}
		}
	}

	output := fmt.Sprintf("Memories from the past %d days (%d total):\n\n", daysBack, len(memories))
	output += strings.Join(memories, "\n")
	return output, nil
}

// containsQuery checks if the content contains the search query
func (t *MemoryTool) containsQuery(content, query string) bool {
	// Case-insensitive search
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	return strings.Contains(lowerContent, lowerQuery)
}

// extractSnippets extracts relevant text snippets around the search query
func (t *MemoryTool) extractSnippets(content, query string, maxSnippets int) []string {
	lines := strings.Split(content, "\n")
	snippets := []string{}
	lowerQuery := strings.ToLower(query)

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), lowerQuery) {
			// Truncate long lines
			snippet := strings.TrimSpace(line)
			if len(snippet) > 200 {
				// Find the position of the query
				queryPos := strings.Index(strings.ToLower(snippet), lowerQuery)
				if queryPos > 0 {
					start := queryPos - 50
					if start < 0 {
						start = 0
					}
					end := queryPos + 150
					if end > len(snippet) {
						end = len(snippet)
					}
					snippet = "..." + snippet[start:end] + "..."
				} else {
					snippet = snippet[:200] + "..."
				}
			}
			snippets = append(snippets, snippet)
			if len(snippets) >= maxSnippets {
				break
			}
		}
	}

	return snippets
}

// SaveMemory saves a memory entry for the current day
func (t *MemoryTool) SaveMemory(workspaceDir, content string) error {
	if workspaceDir == "" {
		return fmt.Errorf("workspace directory not set")
	}

	memoryDir := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory)
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	memoryFile := filepath.Join(memoryDir, dateStr+constant.ExtMD)

	// Append to existing file or create new
	file, err := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open memory file: %w", err)
	}
	defer file.Close()

	// Add timestamp
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("\n## [%s] %s\n\n%s\n", timestamp, time.Now().Format("2006-01-02"), content)

	if _, err := file.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write memory: %w", err)
	}

	return nil
}

// SearchByPattern searches memories using regex pattern
func (t *MemoryTool) SearchByPattern(workspaceDir, pattern string, daysBack int) (string, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	if workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not set")
	}

	memoryDir := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory)
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return "No memories found", nil
	}

	results := []string{}
	now := time.Now()

	for i := 0; i < daysBack; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		memoryFile := filepath.Join(memoryDir, dateStr+constant.ExtMD)

		content, err := os.ReadFile(memoryFile)
		if err != nil {
			continue
		}

		if regex.Match(content) {
			results = append(results, fmt.Sprintf("📅 %s: Match found", dateStr))
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No memories found matching pattern: %s", pattern), nil
	}

	return strings.Join(results, "\n"), nil
}

// GetMemoryStats returns statistics about stored memories
func (t *MemoryTool) GetMemoryStats(workspaceDir string) (string, error) {
	if workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not set")
	}

	memoryDir := filepath.Join(workspaceDir, constant.DirProfile, constant.DirMemory)
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return "Memory directory does not exist", nil
	}

	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return "", fmt.Errorf("failed to read memory directory: %w", err)
	}

	totalSize := int64(0)
	count := 0

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), constant.ExtMD) {
			info, _ := entry.Info()
			totalSize += info.Size()
			count++
		}
	}

	stats := "Memory Statistics:\n"
	stats += fmt.Sprintf("  Total entries: %d\n", count)
	stats += fmt.Sprintf("  Total size: %d bytes\n", totalSize)
	if count > 0 {
		stats += fmt.Sprintf("  Average size: %d bytes\n", totalSize/int64(count))
	}

	return stats, nil
}
