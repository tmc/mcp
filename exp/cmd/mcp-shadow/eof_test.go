package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

func TestEOFDrainsResponses(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "mcp-shadow")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	path := filepath.Join(dir, "trace.mcp")
	cmd := exec.CommandContext(ctx, binary, "-primary", "cat", "-shadow", "cat", "-compare", "-q", "-o", path)
	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.String())
	}
	if stdout.String() != input {
		t.Fatalf("output = %q, want %q", stdout.String(), input)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := mcptrace.NewReader(file)
	seen := make(map[string]int)
	for i := 0; i < 3; i++ {
		record, err := reader.Read()
		if err != nil {
			t.Fatal(err)
		}
		seen[record.Direction]++
	}
	for _, direction := range []string{"recv", "send", "send-shadow"} {
		if seen[direction] != 1 {
			t.Fatalf("trace directions = %v", seen)
		}
	}
	if _, err := reader.Read(); err != io.EOF {
		t.Fatalf("trace end: %v", err)
	}
}
