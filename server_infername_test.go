//go:build !windows
// +build !windows

package mcp

import (
	"os"
	"testing"
)

// TestInferServerName tests the inferServerName function
func TestInferServerName(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "binary name",
			args:     []string{"myserver"},
			expected: "myserver",
		},
		{
			name:     "with path",
			args:     []string{"/usr/bin/myserver"},
			expected: "myserver",
		},
		{
			name:     "multiple args",
			args:     []string{"myserver", "--flag", "value"},
			expected: "myserver",
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			result := inferServerName()
			if result != tt.expected {
				t.Errorf("inferServerName() = %s, want %s", result, tt.expected)
			}
		})
	}
}
