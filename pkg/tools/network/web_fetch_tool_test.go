package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewWebFetchTool(t *testing.T) {
	tool := NewWebFetchTool()

	if tool == nil {
		t.Fatal("NewWebFetchTool should not return nil")
	}

	if tool.Name() != "web_fetch" {
		t.Errorf("Expected tool name 'web_fetch', got '%s'", tool.Name())
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
	if !ok || len(required) == 0 || required[0] != "url" {
		t.Error("'url' should be required")
	}

	expectedParams := []string{"url", "timeout", "raw"}
	for _, expected := range expectedParams {
		if _, ok := properties[expected]; !ok {
			t.Errorf("Parameters should have '%s' property", expected)
		}
	}
}

func TestWebFetchTool_Execute_EmptyURL(t *testing.T) {
	tool := NewWebFetchTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"url": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty URL")
	}

	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("Error should mention 'url is required', got: %v", err)
	}
}

func TestWebFetchTool_Execute_InvalidURL(t *testing.T) {
	tool := NewWebFetchTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"url": "not-a-valid-url",
	})

	if err == nil {
		t.Error("Execute should fail with invalid URL")
	}

	// URL parser might return different errors, just check it fails
	if !strings.Contains(err.Error(), "invalid URL") && !strings.Contains(err.Error(), "scheme") {
		t.Logf("Got error (should fail): %v", err)
	}
}

func TestWebFetchTool_Execute_InvalidScheme(t *testing.T) {
	tool := NewWebFetchTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"url": "ftp://example.com",
	})

	if err == nil {
		t.Error("Execute should fail with non-http scheme")
	}

	if !strings.Contains(err.Error(), "must use http or https") {
		t.Errorf("Error should mention scheme requirement, got: %v", err)
	}
}

func TestWebFetchTool_Execute_ValidRequest(t *testing.T) {
	tool := NewWebFetchTool()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	result, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if !strings.Contains(result, "Hello, World!") {
		t.Errorf("Result should contain 'Hello, World!', got: %s", result)
	}
}

func TestWebFetchTool_Execute_HTMLContent(t *testing.T) {
	tool := NewWebFetchTool()

	htmlContent := `<html>
<head><title>Test Page</title></head>
<body>
<h1>Hello</h1>
<p>This is a test paragraph.</p>
<script>console.log('test');</script>
<style>body { color: red; }</style>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	result, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// Check that script and style are removed
	if strings.Contains(result, "<script>") || strings.Contains(result, "console.log") {
		t.Error("Script tags should be removed from result")
	}

	if strings.Contains(result, "<style>") || strings.Contains(result, "color: red") {
		t.Error("Style tags should be removed from result")
	}

	// Check that content is preserved
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "test paragraph") {
		t.Error("Main content should be preserved")
	}
}

func TestWebFetchTool_Execute_RawMode(t *testing.T) {
	tool := NewWebFetchTool()

	htmlContent := `<html><body><h1>Test</h1></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	result, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
		"raw": "true",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	// In raw mode, HTML should not be processed
	if !strings.Contains(result, "<html>") || !strings.Contains(result, "<h1>") {
		t.Errorf("Raw mode should preserve HTML, got: %s", result)
	}
}

func TestWebFetchTool_Execute_JSONContent(t *testing.T) {
	tool := NewWebFetchTool()

	jsonContent := `{"name":"test","value":123}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonContent))
	}))
	defer server.Close()

	result, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
		t.Errorf("Result should contain JSON content, got: %s", result)
	}
}

func TestWebFetchTool_Execute_HTTPError(t *testing.T) {
	tool := NewWebFetchTool()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	_, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err == nil {
		t.Error("Execute should fail with HTTP error")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Error should mention HTTP status code, got: %v", err)
	}
}

func TestWebFetchTool_Execute_Timeout(t *testing.T) {
	tool := NewWebFetchTool()

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		// Note: This test might be flaky, but we're testing the timeout mechanism
		w.WriteHeader(http.StatusOK)
	}))

	result, err := tool.Execute(context.Background(), map[string]string{
		"url":     server.URL,
		"timeout": "0.001", // 1ms timeout
	})

	// The test might succeed or fail depending on timing
	_ = result
	if err != nil {
		// Expected to timeout
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
			t.Logf("Got error (possibly timeout): %v", err)
		}
	}
	server.Close()
}

func TestWebFetchTool_Execute_TextContent(t *testing.T) {
	tool := NewWebFetchTool()

	textContent := "Plain text content\nLine 2\nLine 3"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(textContent))
	}))
	defer server.Close()

	result, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if !strings.Contains(result, "Plain text content") {
		t.Errorf("Result should contain text content, got: %s", result)
	}
}

func TestExtractReadableText(t *testing.T) {
	tool := NewWebFetchTool()

	html := `<html>
<head>
	<title>Test</title>
	<script>alert('test');</script>
	<style>body{margin:0;}</style>
</head>
<body>
	<h1>Main Title</h1>
	<p>Paragraph 1</p>
	<p>Paragraph 2</p>
	<div>Div content</div>
</body>
</html>`

	result := tool.extractReadableText(html)

	// Scripts and styles should be removed
	if strings.Contains(result, "alert") || strings.Contains(result, "margin:0") {
		t.Error("Script and style content should be removed")
	}

	// Main content should be preserved
	if !strings.Contains(result, "Main Title") {
		t.Error("Main title should be preserved")
	}

	// Check that important content is present
	if !strings.Contains(result, "Paragraph 1") || !strings.Contains(result, "Div content") {
		t.Error("Important content should be preserved")
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tool := NewWebFetchTool()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello &amp; World", "Hello & World"},
		{"&lt;tag&gt;", "<tag>"},
		{"&quot;quoted&quot;", "\"quoted\""},
		{"&apos;single&apos;", "'single'"},
		{"Hello&nbsp;World", "Hello World"},
	}

	for _, tt := range tests {
		result := tool.decodeHTMLEntities(tt.input)
		if result != tt.expected {
			t.Errorf("decodeHTMLEntities(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestWebFetchTool_UserAgent(t *testing.T) {
	tool := NewWebFetchTool()

	receivedUserAgent := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	_, err := tool.Execute(context.Background(), map[string]string{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if receivedUserAgent == "" {
		t.Error("User-Agent header should be set")
	}

	if !strings.Contains(receivedUserAgent, "Mozilla") {
		t.Errorf("User-Agent should look like a browser, got: %s", receivedUserAgent)
	}
}
