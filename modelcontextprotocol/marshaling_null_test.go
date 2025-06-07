package modelcontextprotocol

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSONNull(t *testing.T) {
	tests := []struct {
		name        string
		target      interface{}
		expectError bool
	}{
		{
			name:        "EmbeddedResource with null",
			target:      &EmbeddedResource{},
			expectError: false,
		},
		{
			name:        "PromptMessage with null",
			target:      &PromptMessage{},
			expectError: false,
		},
		{
			name:        "SamplingMessage with null",
			target:      &SamplingMessage{},
			expectError: false,
		},
		{
			name:        "CreateMessageResult with null",
			target:      &CreateMessageResult{},
			expectError: false,
		},
		{
			name:        "CompleteRequestParams with null",
			target:      &CompleteRequestParams{},
			expectError: false,
		},
		{
			name:        "ReadResourceResult with null",
			target:      &ReadResourceResult{},
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
