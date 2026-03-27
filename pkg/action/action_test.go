package action_test

import (
	"os"
	"testing"

	"github.com/yockii/wangshu/pkg/action"
)

func TestAction_Execute(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		mdPath string
		// Named input parameters for target function.
		inputs  map[string]any
		wantErr bool
	}{
		{
			name:   "Action执行测试",
			mdPath: "./testdata/simple.md",
			inputs: map[string]any{
				"param1": "value1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.mdPath)
			if err != nil {
				t.Fatalf("could not read file: %v", err)
			}

			a, err := action.ParseActionFromMarkdown(string(data))
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got, gotErr := a.Execute(tt.inputs)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Execute() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Execute() succeeded unexpectedly")
			}
			t.Logf("Execute() = %v", got)
		})
	}
}
