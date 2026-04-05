package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

func TestSplitShellLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{name: "simple", line: "echo --message hello", want: []string{"echo", "--message", "hello"}},
		{name: "quoted", line: `echo --message "hello world"`, want: []string{"echo", "--message", "hello world"}},
		{name: "escaped", line: `echo --message hello\ world`, want: []string{"echo", "--message", "hello world"}},
	}
	for _, tt := range tests {
		got, err := splitShellLine(tt.line)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestInteractiveShellCompletesCommands(t *testing.T) {
	sh := newTestShell()
	line, pos, ok := sh.autoComplete("ec", len("ec"), '\t')
	if !ok {
		t.Fatal("expected completion")
	}
	if line != "echo " {
		t.Fatalf("line=%q", line)
	}
	if pos != len("echo ") {
		t.Fatalf("pos=%d", pos)
	}
}

func TestInteractiveShellListsAmbiguousCompletions(t *testing.T) {
	sh := newTestShell()
	var out bytes.Buffer
	sh.writer = &out
	line, pos, ok := sh.autoComplete("", 0, '\t')
	if !ok {
		t.Fatal("expected completion")
	}
	if line != "" || pos != 0 {
		t.Fatalf("line=%q pos=%d", line, pos)
	}
	text := out.String()
	if !strings.Contains(text, "echo") || !strings.Contains(text, "tools") {
		t.Fatalf("suggestions=%q", text)
	}
}

func TestRunLineHelpBuiltin(t *testing.T) {
	sh := newTestShell()
	var out bytes.Buffer
	sh.writer = &out
	exit, err := sh.runLine(context.Background(), "help")
	if err != nil {
		t.Fatal(err)
	}
	if exit {
		t.Fatal("exit=true")
	}
	text := out.String()
	if !strings.Contains(text, "Builtins:") || !strings.Contains(text, "echo") {
		t.Fatalf("help output=%q", text)
	}
}

func TestRunLineToolsBuiltin(t *testing.T) {
	sh := newTestShell()
	var out bytes.Buffer
	sh.writer = &out
	exit, err := sh.runLine(context.Background(), "tools")
	if err != nil {
		t.Fatal(err)
	}
	if exit {
		t.Fatal("exit=true")
	}
	if !strings.Contains(out.String(), "echo") {
		t.Fatalf("tools output=%q", out.String())
	}
}

func TestRunLineNormalizesRepeatedBuiltin(t *testing.T) {
	sh := newTestShell()
	var out bytes.Buffer
	sh.writer = &out
	exit, err := sh.runLine(context.Background(), "helphelp")
	if err != nil {
		t.Fatal(err)
	}
	if exit {
		t.Fatal("exit=true")
	}
	if !strings.Contains(out.String(), "Builtins:") {
		t.Fatalf("output=%q", out.String())
	}
}

func TestDecorateShellErrorSuggestsNearestCommand(t *testing.T) {
	root := newRootCommand(testApp(&fakeBackend{}).opts, testApp(&fakeBackend{}))
	err := decorateShellError(errors.New(`unknown command "hep" for "mcpsh"`), "hep", root)
	if err == nil {
		t.Fatal("decorateShellError returned nil")
	}
	if !strings.Contains(err.Error(), "try: help") {
		t.Fatalf("error=%q", err.Error())
	}
}

func TestInteractiveShellCompletesFlagNames(t *testing.T) {
	sh := newTestShell()
	line, _, ok := sh.autoComplete("echo --mes", len("echo --mes"), '\t')
	if !ok {
		t.Fatal("expected completion")
	}
	if line != "echo --message " {
		t.Fatalf("line=%q", line)
	}
}

func TestInteractiveShellDoesNotCompleteHiddenStartupFlags(t *testing.T) {
	sh := newTestShell()
	root := sh.newRoot()
	cmd := findSubcommand(root, "echo")
	if cmd == nil {
		t.Fatal("missing echo command")
	}
	names := flagNames(root, cmd)
	for _, name := range names {
		if name == "--cmd" || name == "--spy-ui" || name == "--timeout" {
			t.Fatalf("unexpected hidden startup flag %q in completion names", name)
		}
	}
}

func TestInteractiveShellCompletesFlagValues(t *testing.T) {
	sh := newTestShell()
	line, _, ok := sh.autoComplete("echo --mode l", len("echo --mode l"), '\t')
	if !ok {
		t.Fatal("expected completion")
	}
	if line != "echo --mode loud " {
		t.Fatalf("line=%q", line)
	}
}

func TestExecuteShellLine(t *testing.T) {
	backend := &fakeBackend{
		result: &mcp.CallToolResult{
			Content: []any{map[string]any{"type": "text", "text": "done"}},
		},
	}
	app := testApp(backend)
	root := newRootCommand(app.opts, app)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetContext(context.Background())
	if err := executeShellLine(root, `echo --message "hello world" --mode loud`); err != nil {
		t.Fatal(err)
	}
	var args map[string]any
	if err := json.Unmarshal(backend.call.Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if args["message"] != "hello world" || args["mode"] != "loud" {
		t.Fatalf("args=%v", args)
	}
}

func TestReloadTools(t *testing.T) {
	backend := &fakeBackend{
		tools: []mcp.Tool{{Name: "echo"}},
	}
	app := &app{
		servers: []*serverConn{{
			name:    "test",
			backend: backend,
			tools:   []mcp.Tool{{Name: "old"}},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	if err := app.reloadTools(context.Background()); err != nil {
		t.Fatal(err)
	}
	tools := app.allTools()
	if len(tools) != 1 || tools[0].tool.Name != "echo" {
		t.Fatalf("tools=%v", tools)
	}
}

func TestFormatSuggestions(t *testing.T) {
	got := formatSuggestions([]string{"echo", "tools"})
	if got != "echo\ntools" {
		t.Fatalf("got %q", got)
	}
}

func TestHideInteractiveFlags(t *testing.T) {
	root := newRootCommand(bootstrapOptions{}, testApp(&fakeBackend{}))
	hideInteractiveFlags(root)
	for _, name := range []string{"cmd", "http", "sse", "timeout", "raw", "server-stderr", "spy-ui"} {
		flag := root.PersistentFlags().Lookup(name)
		if flag == nil {
			t.Fatalf("missing flag %q", name)
		}
		if !flag.Hidden {
			t.Fatalf("flag %q is not hidden", name)
		}
	}
}

func newTestShell() *interactiveShell {
	backend := &fakeBackend{}
	app := testApp(backend)
	return &interactiveShell{
		newRoot: func() *cobra.Command {
			root := newRootCommand(app.opts, app)
			hideInteractiveFlags(root)
			root.SetContext(context.Background())
			return root
		},
		prompt: shellPrompt(app),
	}
}

func testApp(backend *fakeBackend) *app {
	return &app{
		servers: []*serverConn{{
			name:    "fake",
			backend: backend,
			info:    &mcp.InitializeResult{ServerInfo: mcp.Implementation{Name: "fake", Version: "1.0.0"}},
			tools: []mcp.Tool{
				{
					Name:        "echo",
					Description: "Echo a message",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"},"mode":{"type":"string","enum":["loud","quiet"]}},"required":["message"]}`),
				},
			},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
}
