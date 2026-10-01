package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/internal/mcptracediff"
)

func TestCallExitCodes(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"base":       "mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\nmcp-recv {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n",
		"changed":    "mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\nmcp-recv {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"changed\":true}}\n",
		"incomplete": "mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		right string
		want  int
	}{{"base", 0}, {"changed", 1}, {"incomplete", 2}, {"missing", 2}} {
		t.Run(tt.right, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if got := diffCalls(filepath.Join(dir, "base"), filepath.Join(dir, tt.right), &out, &stderr, mcptracediff.Options{}, false); got != tt.want {
				t.Fatalf("exit=%d want %d; stdout=%s stderr=%s", got, tt.want, out.String(), stderr.String())
			}
		})
	}
}
