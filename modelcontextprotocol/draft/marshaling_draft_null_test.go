package draft

import (
	"encoding/json"
	"testing"
)

func TestDraftUnmarshalJSONNull(t *testing.T) {
	tests := []struct {
		name        string
		target      interface{}
		expectError bool
	}{
		{
			name:        "ContentList with null",
			target:      &ContentList{},
			expectError: false,
		},
		{
			name:        "CallToolResult with null",
			target:      &CallToolResult{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte("null"), tt.target)
			if (err != nil) != tt.expectError {
				t.Errorf("UnmarshalJSON() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
