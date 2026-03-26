package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yockii/wangshu/pkg/constant"
)

func TestNewVersionTool(t *testing.T) {
	tool := NewVersionTool()

	if tool == nil {
		t.Fatal("NewVersionTool should not return nil")
	}

	if tool.Name() != constant.ToolNameVersion {
		t.Errorf("Expected tool name '%s', got '%s'", constant.ToolNameVersion, tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	params := tool.Parameters()
	if params == nil {
		t.Fatal("Tool should have parameters")
	}

	if params["type"] != "object" {
		t.Error("Parameters type should be 'object'")
	}

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	if _, ok := properties["action"]; !ok {
		t.Error("Parameters should have 'action' property")
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "action" {
		t.Error("'action' should be required")
	}
}

func TestVersionTool_Execute_Current(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "current",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	expected := "Current version: " + constant.Version
	if result.Raw != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.Raw)
	}
}

func TestVersionTool_Execute_Latest(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "latest",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	if result.Raw == "" {
		t.Error("Result should not be empty")
	}

	expectedPrefix := "Latest version:"
	if len(result.Raw) < len(expectedPrefix) || result.Raw[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected result to start with '%s', got '%s'", expectedPrefix, result.Raw)
	}
}

func TestVersionTool_Execute_Check(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "check",
	})

	if result.Err != nil {
		t.Errorf("Execute should succeed: %v", result.Err)
	}

	if result.Raw == "" {
		t.Error("Result should not be empty")
	}

	if constant.Version == "dev" {
		expected := "Development version detected. Cannot compare with latest release."
		if result.Raw != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result.Raw)
		}
	}
}

func TestVersionTool_Execute_InvalidAction(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "invalid",
	})

	if result.Err == nil {
		t.Error("Execute should fail with invalid action")
	}

	expectedError := "invalid action: invalid"
	if result.Err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, result.Err.Error())
	}
}

func TestVersionTool_Execute_MissingAction(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{})

	if result.Err == nil {
		t.Error("Execute should fail when action parameter is missing")
	}

	expectedError := "invalid action: "
	if result.Err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, result.Err.Error())
	}
}

func TestVersionTool_Execute_EmptyAction(t *testing.T) {
	tool := NewVersionTool()

	result := tool.Execute(context.Background(), map[string]string{
		"action": "",
	})

	if result.Err == nil {
		t.Error("Execute should fail with empty action")
	}

	expectedError := "invalid action: "
	if result.Err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, result.Err.Error())
	}
}

func TestVersionTool_Execute_AllActions(t *testing.T) {
	tool := NewVersionTool()

	actions := []string{"current", "latest", "check"}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]string{
				"action": action,
			})

			if result.Err != nil {
				t.Errorf("Execute should succeed for action '%s': %v", action, result.Err)
			}

			if result.Raw == "" {
				t.Errorf("Result should not be empty for action '%s'", action)
			}
		})
	}

	t.Run("restart", func(t *testing.T) {
		// Skip restart test as it will actually restart the process
		t.Skip("Skipping restart test as it will restart the process")
	})
}

func TestVersionTool_GetCurrentVersion(t *testing.T) {
	tool := NewVersionTool()

	result := tool.getCurrentVersion()

	if result.Err != nil {
		t.Errorf("getCurrentVersion should succeed: %v", result.Err)
	}

	expected := "Current version: " + constant.Version
	if result.Raw != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.Raw)
	}
}

func TestVersionTool_GetLatestVersion(t *testing.T) {
	tool := NewVersionTool()

	result := tool.getLatestVersion(context.Background())

	if result.Err != nil {
		t.Errorf("getLatestVersion should succeed: %v", result.Err)
	}

	if result.Raw == "" {
		t.Error("Result should not be empty")
	}

	expectedPrefix := "Latest version:"
	if len(result.Raw) < len(expectedPrefix) || result.Raw[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected result to start with '%s', got '%s'", expectedPrefix, result.Raw)
	}
}

func TestVersionTool_CheckVersion(t *testing.T) {
	tool := NewVersionTool()

	result := tool.checkVersion(context.Background())

	if result.Err != nil {
		t.Errorf("checkVersion should succeed: %v", result.Err)
	}

	if result.Raw == "" {
		t.Error("Result should not be empty")
	}

	if constant.Version == "dev" {
		expected := "Development version detected. Cannot compare with latest release."
		if result.Raw != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result.Raw)
		}
	}
}

func TestVersionTool_ContextCancellation(t *testing.T) {
	tool := NewVersionTool()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := tool.getLatestVersion(ctx)

	if result.Err == nil {
		t.Error("getLatestVersion should fail with cancelled context")
	}

	if constant.Version != "dev" {
		result = tool.checkVersion(ctx)

		if result.Err == nil {
			t.Error("checkVersion should fail with cancelled context")
		}
	}
}

func TestVersionTool_Rrestart_GetExecutablePath(t *testing.T) {
	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	if exePath == "" {
		t.Error("Executable path should not be empty")
	}

	_, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		t.Errorf("Failed to resolve symlinks: %v", err)
	}
}

func TestVersionTool_Rrestart_MethodExists(t *testing.T) {
	// Skip this test as it will actually restart the process
	t.Skip("Skipping restart method test as it will restart the process")
}
