package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

// TestWorkflow drives real binaries and checks the red-before-green contract.
func TestWorkflow(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(protocol, func(t *testing.T) { testWorkflow(t, protocol) })
	}
}

func testWorkflow(t *testing.T, protocol string) {
	work := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binaries := make(map[string]string)
	for _, tool := range []struct{ name, dir, pkg string }{
		{"workflow", filepath.Join(root, "examples"), "./workflow"},
		{"mcpinspect", root, "./cmd/mcpinspect"},
		{"mcpfixture", root, "./cmd/mcpfixture"},
		{"mcpapi", root, "./cmd/mcpapi"},
		{"mcpdiff", root, "./exp/cmd/mcpdiff"},
	} {
		binary := filepath.Join(work, tool.name)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, tool.pkg)
		cmd.Dir = tool.dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("build %s: %v\n%s", tool.name, err, output)
		}
		binaries[tool.name] = binary
	}
	run := func(want int, name string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binaries[name], args...)
		cmd.Dir = work
		output, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%s %v: %v", name, args, err)
			}
			code = exit.ExitCode()
		}
		if code != want {
			t.Fatalf("%s %v exit %d, want %d\n%s", name, args, code, want, output)
		}
		return string(output)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(work, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if got := run(0, "workflow", "record", "-protocol", protocol, "bad.mcp"); got != "6\n" {
		t.Fatalf("bad result = %q", got)
	}
	inspection := run(0, "mcpinspect", "bad.mcp")
	if !strings.Contains(inspection, "tools/call") {
		t.Fatalf("inspection lacks tool call: %s", inspection)
	}
	observed := run(0, "mcpfixture", "-record", fmt.Sprint(callRecord(t, filepath.Join(work, "bad.mcp"))), "bad.mcp")
	if !strings.Contains(observed, `"text": "6"`) {
		t.Fatalf("fixture lost observed bug: %s", observed)
	}
	write("observed.txt", observed)
	run(0, "workflow", "expect", "-text", "9", "observed.txt", "regression.json")
	failure := run(1, "mcpfixture", "check", "regression.json", "--", binaries["workflow"], "server")
	if !strings.Contains(failure, "observed result changed") {
		t.Fatalf("red regression wrong failure: %s", failure)
	}
	run(0, "mcpfixture", "check", "regression.json", "--", binaries["workflow"], "server", "-fixed")
	if got := run(0, "workflow", "record", "-fixed", "-protocol", protocol, "good.mcp"); got != "9\n" {
		t.Fatalf("fixed result = %q", got)
	}
	run(1, "mcpdiff", "-calls", "bad.mcp", "good.mcp")
	run(0, "mcpdiff", "-calls", "good.mcp", "good.mcp")
	labels := []string{"snapshot", "-scope", "local", "-principal", "anonymous", "-client", "workflow", "--", binaries["workflow"], "server"}
	write("before.json", run(0, "mcpapi", labels...))
	write("after.json", run(0, "mcpapi", append(labels, "-fixed")...))
	run(0, "mcpapi", "diff", "before.json", "after.json")
}

func callRecord(t *testing.T, path string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := mcptrace.NewReader(file)
	for index := 1; ; index++ {
		record, err := reader.Read()
		if err == io.EOF {
			t.Fatal("trace has no tool call")
		}
		if err != nil {
			t.Fatal(err)
		}
		var message struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(record.Message, &message); err != nil {
			t.Fatal(err)
		}
		if message.Method == "tools/call" && record.Direction == "send" {
			return index
		}
	}
}
