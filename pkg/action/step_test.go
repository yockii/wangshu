package action_test

import (
	"context"
	"testing"

	"github.com/yockii/wangshu/pkg/action"
)

func TestStep_Do(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		wantErr bool
	}{
		{
			name:    "测试web.search步骤",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var s = action.NewStep(action.NewExecutionContext(context.Background(), map[string]any{"query": "golang"}))
			s.ID = "search"
			s.Use = "web.search"
			s.With = map[string]any{"query": "{{input.query}}"}

			gotErr := s.Do()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Do() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Do() succeeded unexpectedly")
			}
		})
	}
}
