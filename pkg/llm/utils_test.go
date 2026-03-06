package llm

import (
	"testing"
)

func TestCleanJSONMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "纯JSON，无需清理",
			input:    `{"name": "test", "value": 123}`,
			expected: `{"name": "test", "value": 123}`,
		},
		{
			name:     "带json标记的代码块",
			input:    "```json\n{\"name\": \"test\", \"value\": 123}\n```",
			expected: `{"name": "test", "value": 123}`,
		},
		{
			name:     "不带json标记的代码块",
			input:    "```\n{\"name\": \"test\", \"value\": 123}\n```",
			expected: `{"name": "test", "value": 123}`,
		},
		{
			name:     "带前后空白的代码块",
			input:    "\n```json\n{\"name\": \"test\", \"value\": 123}\n```\n",
			expected: `{"name": "test", "value": 123}`,
		},
		{
			name:     "多行JSON",
			input:    "```json\n{\n  \"name\": \"test\",\n  \"value\": 123\n}\n```",
			expected: "{\n  \"name\": \"test\",\n  \"value\": 123\n}",
		},
		{
			name:     "只有前导空白",
			input:    "\n  \n{\"name\": \"test\"}",
			expected: `{"name": "test"}`,
		},
		{
			name:     "代码块后有多个换行",
			input:    "```json\n{\"name\": \"test\"}\n```\n\n\n",
			expected: `{"name": "test"}`,
		},
		{
			name:     "带额外文本的代码块",
			input:    "这是JSON数据：\n```json\n{\"name\": \"test\"}\n```",
			expected: `{"name": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJSONMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanJSONMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractJSONFromContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "标准JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Markdown包裹的JSON",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSONFromContent(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractJSONFromContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}
