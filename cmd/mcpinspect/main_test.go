package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	input := `mcp-send {"jsonrpc":"2.0","id":1,"method":"ping"}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{}}`
	for _, tt := range []struct {
		name        string
		args        []string
		input, want string
		fail        bool
	}{
		{"help", []string{"-h"}, "", "", false},
		{"human", nil, input, "request=1", false},
		{"json", []string{"-json"}, input, `"requestRecord":1`, false},
		{"truncated", nil, "mcp-recv {", "", true},
		{"extra files", []string{"a", "b"}, "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			err := run(tt.args, strings.NewReader(tt.input), &out, &stderr)
			if (err != nil) != tt.fail {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output: %s", out.String())
			}
		})
	}
}
