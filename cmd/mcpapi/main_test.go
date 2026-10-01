package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpcatalog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScript(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			state, err := script.NewState(context.Background(), t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := state.ExtractFiles(archive); err != nil {
				t.Fatal(err)
			}
			engine := script.NewEngine()
			engine.Cmds["mcpapi"] = script.Command(script.CmdUsage{Summary: "run mcpapi and check exit code", Args: "code args..."}, func(s *script.State, args ...string) (script.WaitFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("missing expected exit code")
				}
				want, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, err
				}
				command := append([]string(nil), args[1:]...)
				if len(command) == 3 && command[0] == "diff" {
					command[1] = s.Path(command[1])
					command[2] = s.Path(command[2])
				}
				return func(*script.State) (string, string, error) {
					var out, errout bytes.Buffer
					got := run(s.Context(), command, &out, &errout)
					var err error
					if got != want {
						err = fmt.Errorf("exit %d, want %d", got, want)
					}
					return out.String(), errout.String(), err
				}, nil
			})
			scripttest.Run(t, engine, state, path, strings.NewReader(string(archive.Comment)))
		})
	}
}

func TestSnapshot(t *testing.T) {
	t.Setenv("MCPAPI_TEST_SERVER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	code := run(context.Background(), []string{"snapshot", "-scope", "p", "-principal", "local", "-client", "default", "--", executable, "-test.run=^TestServerHelper$"}, &out, &errout)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errout.String())
	}
	s, err := mcpcatalog.Read(&out)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tools) != 1 || s.Tools[0].Name != "search" {
		t.Fatalf("got %+v", s.Tools)
	}
}
func TestServerHelper(t *testing.T) {
	if os.Getenv("MCPAPI_TEST_SERVER") != "1" {
		return
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "search", InputSchema: json.RawMessage(`{"type":"object"}`)}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, fmt.Errorf("snapshot must not call tools")
	})
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
