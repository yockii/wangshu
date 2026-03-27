package tools

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/yockii/wangshu/pkg/tools/types"
)

// mockTool 是一个用于测试的简单工具实现
type mockTool struct {
	name        string
	description string
	parameters  map[string]any
	executeFunc func(ctx context.Context, params map[string]string) *types.ToolResult
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Parameters() map[string]any {
	return m.parameters
}

func (m *mockTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return types.NewToolResult().WithRaw("mock result")
}

// newMockTool 创建一个新的mock工具
func newMockTool(name string) *mockTool {
	return &mockTool{
		name:        name,
		description: fmt.Sprintf("Mock tool %s", name),
		parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param1": map[string]any{
					"type":        "string",
					"description": "Test parameter",
				},
			},
		},
	}
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry should not return nil")
	}

	if reg.tools == nil {
		t.Error("NewRegistry should initialize tools map")
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()
	tool := newMockTool("test_tool")

	reg.Register(tool)

	// 验证工具已注册
	if _, ok := reg.Get("test_tool"); !ok {
		t.Error("Tool should be registered")
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	reg := NewRegistry()
	tool1 := newMockTool("test_tool")
	tool1.description = "First tool"

	tool2 := newMockTool("test_tool")
	tool2.description = "Second tool"

	reg.Register(tool1)
	reg.Register(tool2)

	// 验证第二个工具覆盖了第一个
	tool, _ := reg.Get("test_tool")
	if tool.Description() != "Second tool" {
		t.Error("Second tool should overwrite the first")
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()
	tool := newMockTool("test_tool")
	reg.Register(tool)

	// 测试获取存在的工具
	retrieved, ok := reg.Get("test_tool")
	if !ok {
		t.Error("Should find existing tool")
	}
	if retrieved.Name() != "test_tool" {
		t.Error("Retrieved tool should have correct name")
	}

	// 测试获取不存在的工具
	_, ok = reg.Get("non_existent")
	if ok {
		t.Error("Should not find non-existent tool")
	}
}

func TestRegistry_Execute(t *testing.T) {
	reg := NewRegistry()

	// 创建一个会返回特定结果的工具
	expectedResult := "test result"
	tool := &mockTool{
		name:        "test_tool",
		description: "Test tool",
		parameters:  map[string]any{},
		executeFunc: func(ctx context.Context, params map[string]string) *types.ToolResult {
			return types.NewToolResult().WithRaw(expectedResult)
		},
	}
	reg.Register(tool)

	// 测试执行存在的工具
	result := reg.Execute(context.Background(), "test_tool", map[string]string{"key": "value"})
	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}
	if result.Raw != expectedResult {
		t.Errorf("Execute should return expected result, got: %s", result.Raw)
	}

	// 测试执行不存在的工具
	result = reg.Execute(context.Background(), "non_existent", nil)
	if result.Err == nil {
		t.Error("Execute should fail for non-existent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	// 注册多个工具
	tools := []*mockTool{
		newMockTool("tool1"),
		newMockTool("tool2"),
		newMockTool("tool3"),
	}

	for _, tool := range tools {
		reg.Register(tool)
	}

	// 获取工具列表
	list := reg.List()

	// 验证列表长度
	if len(list) != 3 {
		t.Errorf("List should return 3 tools, got: %d", len(list))
	}

	// 验证所有工具都在列表中
	toolSet := make(map[string]bool)
	for _, name := range list {
		toolSet[name] = true
	}

	for _, tool := range tools {
		if !toolSet[tool.name] {
			t.Errorf("Tool %s should be in the list", tool.name)
		}
	}
}

func TestRegistry_GetSummaries(t *testing.T) {
	reg := NewRegistry()

	tool1 := newMockTool("tool1")
	tool1.description = "Description 1"

	tool2 := newMockTool("tool2")
	tool2.description = "Description 2"

	reg.Register(tool1)
	reg.Register(tool2)

	// 获取工具摘要
	summaries := reg.GetSummaries()

	// 验证摘要数量
	if len(summaries) != 2 {
		t.Errorf("GetSummaries should return 2 summaries, got: %d", len(summaries))
	}

	// 验证摘要内容
	if summaries["tool1"] != "Description 1" {
		t.Errorf("Summary for tool1 incorrect, got: %s", summaries["tool1"])
	}

	if summaries["tool2"] != "Description 2" {
		t.Errorf("Summary for tool2 incorrect, got: %s", summaries["tool2"])
	}
}

func TestRegistry_GetProviderDefs(t *testing.T) {
	reg := NewRegistry()

	tool1 := newMockTool("tool1")
	tool2 := newMockTool("tool2")

	reg.Register(tool1)
	reg.Register(tool2)

	// 获取Provider定义
	defs := reg.GetProviderDefs()

	// 验证定义数量
	if len(defs) != 2 {
		t.Errorf("GetProviderDefs should return 2 definitions, got: %d", len(defs))
	}

	// 验证定义格式
	for _, def := range defs {
		if def.Type != "function" {
			t.Errorf("Definition type should be 'function', got: %s", def.Type)
		}

		if def.Function.Name == "" {
			t.Error("Definition should have a name")
		}

		if def.Function.Description == "" {
			t.Error("Definition should have a description")
		}

		if def.Function.Parameters == nil {
			t.Error("Definition should have parameters")
		}
	}
}

func TestRegistry_GetSelectedToolsInProviderDefs(t *testing.T) {
	reg := NewRegistry()

	tool1 := newMockTool("tool1")
	tool2 := newMockTool("tool2")
	tool3 := newMockTool("tool3")

	reg.Register(tool1)
	reg.Register(tool2)
	reg.Register(tool3)

	// 获取选定工具的定义
	defs := reg.GetSelectedToolsInProviderDefs("tool1", "tool3")

	// 验证只返回了选定的工具
	if len(defs) != 2 {
		t.Errorf("GetSelectedToolsInProviderDefs should return 2 definitions, got: %d", len(defs))
	}

	// 验证返回的是tool1和tool3
	names := make(map[string]bool)
	for _, def := range defs {
		names[def.Function.Name] = true
	}

	if !names["tool1"] || !names["tool3"] {
		t.Error("Should return tool1 and tool3")
	}

	if names["tool2"] {
		t.Error("Should not return tool2")
	}
}

func TestRegistry_ExecuteWithContext(t *testing.T) {
	reg := NewRegistry()

	// 创建ContextualTool
	ctxTool := &mockContextualTool{
		name:        "ctx_tool",
		description: "Contextual tool",
		parameters:  map[string]any{},
	}

	reg.Register(ctxTool)

	// 测试ExecuteWithContext
	result := reg.ExecuteWithContext(
		context.Background(),
		"ctx_tool",
		map[string]any{"param1": "value1"},
		&types.ToolContext{},
		"test_channel",
		"test_chat_id",
	)

	if result.Err != nil {
		t.Errorf("ExecuteWithContext should succeed: %v", result.Err)
	}

	if result.Raw != "contextual result: value1" {
		t.Errorf("ExecuteWithContext should return correct result, got: %s", result.Raw)
	}

	// 测试不存在的工具
	result = reg.ExecuteWithContext(
		context.Background(),
		"non_existent",
		nil,
		nil,
		"",
		"",
	)

	if result.Err == nil {
		t.Error("ExecuteWithContext should fail for non-existent tool")
	}
}

func TestArgsToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
		{
			name: "string values",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "value",
				"int":    123,
				"float":  45.67,
				"bool":   true,
				"null":   nil,
			},
			expected: map[string]string{
				"string": "value",
				"int":    "123",
				"float":  "45.67",
				"bool":   "true",
				"null":   "<nil>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := argsToStringMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("argsToStringMap length mismatch, got: %d, expected: %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("argsToStringMap[%s] = %s, expected: %s", k, result[k], v)
				}
			}
		})
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()
	toolCount := 100

	// 并发注册工具
	var wg sync.WaitGroup
	for i := 0; i < toolCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			tool := newMockTool(fmt.Sprintf("tool%d", index))
			reg.Register(tool)
		}(i)
	}

	wg.Wait()

	// 验证所有工具都已注册
	list := reg.List()
	if len(list) != toolCount {
		t.Errorf("Should have %d tools, got: %d", toolCount, len(list))
	}

	// 并发读取
	for i := 0; i < toolCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("tool%d", index)
			_, ok := reg.Get(name)
			if !ok {
				t.Errorf("Should find tool %s", name)
			}
		}(i)
	}

	wg.Wait()
}

// mockContextualTool 是一个实现了ContextualTool接口的mock工具
type mockContextualTool struct {
	name        string
	description string
	parameters  map[string]any
}

func (m *mockContextualTool) Name() string {
	return m.name
}

func (m *mockContextualTool) Description() string {
	return m.description
}

func (m *mockContextualTool) Parameters() map[string]any {
	return m.parameters
}

func (m *mockContextualTool) Execute(ctx context.Context, params map[string]string) *types.ToolResult {
	// 基本Tool接口实现
	return types.NewToolResult().WithRaw("basic execute")
}

func (m *mockContextualTool) ExecuteWithContext(ctx context.Context, params map[string]string, toolCtx *types.ToolContext) *types.ToolResult {
	return types.NewToolResult().WithRaw(fmt.Sprintf("contextual result: %s", params["param1"]))
}
