package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "basic recv",
			input: `mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000.000`,
			contains: []string{
				"\033[32mmcp-recv\033[0m", // green prefix
				"initialize",
				"\033[90m # 1000.000\033[0m", // gray timestamp
			},
		},
		{
			name:  "basic send",
			input: `mcp-send {"jsonrpc":"2.0","result":{"ok":true},"id":1} # 1001.000`,
			contains: []string{
				"\033[96mmcp-send\033[0m", // bright cyan prefix
				"result",
				"\033[90m # 1001.000\033[0m", // gray timestamp
			},
		},
		{
			name:  "shadow send",
			input: `mcp-send-shadow {"jsonrpc":"2.0","result":{"ok":true},"id":1} # 1001.100`,
			contains: []string{
				"\033[90mmcp-send-shadow\033[0m", // gray prefix
				"\033[90m{",                      // gray JSON content
				"\033[90m # 1001.100\033[0m",     // gray timestamp
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			colorize(tt.input)

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stderr = oldStderr

			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output missing expected content %q\nGot: %q", expected, output)
				}
			}
		})
	}
}

func TestHighlightJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		isRecv   bool
		contains []string
	}{
		{
			name:   "method highlighting",
			json:   `{"jsonrpc":"2.0","method":"test","id":1}`,
			isRecv: true,
			contains: []string{
				"\"\033[1mmethod\033[0m\"",
				"\"\033[1mtest\033[0m\"",
			},
		},
		{
			name:   "result highlighting",
			json:   `{"jsonrpc":"2.0","result":{"ok":true},"id":1}`,
			isRecv: false,
			contains: []string{
				"\"\033[1mresult\033[0m\"",
			},
		},
		{
			name:   "id highlighting",
			json:   `{"jsonrpc":"2.0","id":42}`,
			isRecv: false,
			contains: []string{
				"\"\033[1mid\033[0m\"",
				"42",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := highlightJSON(tt.json, tt.isRecv)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output missing expected content %q\nGot: %q", expected, output)
				}
			}
		})
	}
}
