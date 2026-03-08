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

	result, err := tool.Execute(context.Background(), map[string]string{
		"action": "current",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	expected := "Current version: " + constant.Version
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestVersionTool_Execute_Latest(t *testing.T) {
	tool := NewVersionTool()

	result, err := tool.Execute(context.Background(), map[string]string{
		"action": "latest",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	expectedPrefix := "Latest version:"
	if len(result) < len(expectedPrefix) || result[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected result to start with '%s', got '%s'", expectedPrefix, result)
	}
}

func TestVersionTool_Execute_Check(t *testing.T) {
	tool := NewVersionTool()

	result, err := tool.Execute(context.Background(), map[string]string{
		"action": "check",
	})

	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	if constant.Version == "dev" {
		expected := "Development version detected. Cannot compare with latest release."
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	}
}

func TestVersionTool_Execute_InvalidAction(t *testing.T) {
	tool := NewVersionTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"action": "invalid",
	})

	if err == nil {
		t.Error("Execute should fail with invalid action")
	}

	expectedError := "invalid action: invalid"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestVersionTool_Execute_MissingAction(t *testing.T) {
	tool := NewVersionTool()

	_, err := tool.Execute(context.Background(), map[string]string{})

	if err == nil {
		t.Error("Execute should fail when action parameter is missing")
	}

	expectedError := "invalid action: "
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestVersionTool_Execute_EmptyAction(t *testing.T) {
	tool := NewVersionTool()

	_, err := tool.Execute(context.Background(), map[string]string{
		"action": "",
	})

	if err == nil {
		t.Error("Execute should fail with empty action")
	}

	expectedError := "invalid action: "
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestVersionTool_Execute_AllActions(t *testing.T) {
	tool := NewVersionTool()

	actions := []string{"current", "latest", "check"}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]string{
				"action": action,
			})

			if err != nil {
				t.Errorf("Execute should succeed for action '%s': %v", action, err)
			}

			if result == "" {
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

	result, err := tool.getCurrentVersion()

	if err != nil {
		t.Errorf("getCurrentVersion should succeed: %v", err)
	}

	expected := "Current version: " + constant.Version
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestVersionTool_GetLatestVersion(t *testing.T) {
	tool := NewVersionTool()

	result, err := tool.getLatestVersion(context.Background())

	if err != nil {
		t.Errorf("getLatestVersion should succeed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	expectedPrefix := "Latest version:"
	if len(result) < len(expectedPrefix) || result[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected result to start with '%s', got '%s'", expectedPrefix, result)
	}
}

func TestVersionTool_CheckVersion(t *testing.T) {
	tool := NewVersionTool()

	result, err := tool.checkVersion(context.Background())

	if err != nil {
		t.Errorf("checkVersion should succeed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	if constant.Version == "dev" {
		expected := "Development version detected. Cannot compare with latest release."
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	}
}

func TestVersionTool_ContextCancellation(t *testing.T) {
	tool := NewVersionTool()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tool.getLatestVersion(ctx)

	if err == nil {
		t.Error("getLatestVersion should fail with cancelled context")
	}

	if constant.Version != "dev" {
		_, err = tool.checkVersion(ctx)

		if err == nil {
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
