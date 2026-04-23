package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup coverage environment automatically
	fmt.Fprintf(os.Stderr, "%s: Setting up coverage environment GOCOVERDIR=%s\n", os.Args[0], os.Getenv("GOCOVERDIR"))
}

func TestProcessReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		strip    bool
		expected string
	}{
		{
			name: "sort by timestamp",
			input: `mcp-send [2023-01-02 10:30:15.123] {"jsonrpc":"2.0","id":2,"method":"test"}
mcp-recv [2023-01-01 15:45:30.456] {"jsonrpc":"2.0","id":1,"result":{}}
mcp-send [2023-01-01 15:45:00.789] {"jsonrpc":"2.0","id":1,"method":"initialize"}
# Some comment line
mcp-recv [2023-01-02 10:31:00.000] {"jsonrpc":"2.0","id":2,"result":{}}`,
			strip: false,
			expected: `# Some comment line
mcp-send [2023-01-01 15:45:00.789] {"jsonrpc":"2.0","id":1,"method":"initialize"}
mcp-recv [2023-01-01 15:45:30.456] {"jsonrpc":"2.0","id":1,"result":{}}
mcp-send [2023-01-02 10:30:15.123] {"jsonrpc":"2.0","id":2,"method":"test"}
mcp-recv [2023-01-02 10:31:00.000] {"jsonrpc":"2.0","id":2,"result":{}}
`,
		},
		{
			name: "strip timestamps",
			input: `mcp-send [2023-01-02 10:30:15.123] {"jsonrpc":"2.0","id":2,"method":"test"}
mcp-recv [2023-01-01 15:45:30.456] {"jsonrpc":"2.0","id":1,"result":{}}
# Some comment line
mcp-send [2023-01-01 15:45:00.789] {"jsonrpc":"2.0","id":1,"method":"initialize"}`,
			strip: true,
			expected: `mcp-send [TIMESTAMP] {"jsonrpc":"2.0","id":2,"method":"test"}
mcp-recv [TIMESTAMP] {"jsonrpc":"2.0","id":1,"result":{}}
# Some comment line
mcp-send [TIMESTAMP] {"jsonrpc":"2.0","id":1,"method":"initialize"}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			// Set strip flag for this test
			origStrip := *stripTimestamp
			*stripTimestamp = tt.strip
			defer func() { *stripTimestamp = origStrip }()

			processReader(input, &output)

			if output.String() != tt.expected {
				t.Errorf("processReader() output = %v, want %v", output.String(), tt.expected)
			}
		})
	}
}
