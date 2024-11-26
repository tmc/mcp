package mcptest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp"
	"golang.org/x/tools/txtar"
	"rsc.io/script"
)

// RunTXTARFile runs an MCP test from a txtar file
func RunTXTARFile(ctx context.Context, filename, dir string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %v", err)
	}

	// Parse the txtar file
	ar := txtar.Parse(content)

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "mcptest-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract files
	for _, f := range ar.Files {
		path := filepath.Join(tmpDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating directory %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, f.Data, 0644); err != nil {
			return fmt.Errorf("writing file %s: %v", f.Name, err)
		}
	}

	// Create script engine
	eng := script.NewEngine()
	eng.Cmds["mcp"] = script.Command(script.CmdUsage{
		Summary: "run MCP command",
	}, func(s *script.State, args ...string) (script.WaitFunc, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("usage: mcp <command> <method> [params]")
		}
		output, err := runMCPCommand(tmpDir, args...)
		if err != nil {
			return nil, err
		}
		return func(*script.State) (string, string, error) {
			return output, "", nil
		}, nil
	})

	// Add default commands
	for name, cmd := range script.DefaultCmds() {
		eng.Cmds[name] = cmd
	}

	// Add default conditions
	for name, cond := range script.DefaultConds() {
		eng.Conds[name] = cond
	}

	state, err := script.NewState(ctx, tmpDir, os.Environ())
	if err != nil {
		return fmt.Errorf("creating script state: %v", err)
	}

	var buf bytes.Buffer
	return eng.Execute(state, string(ar.Comment), bufio.NewReader(strings.NewReader("")), &buf)
}

func runMCPCommand(dir string, args ...string) (string, error) {
	cmd := args[0]
	method := args[1]
	var params json.RawMessage
	if len(args) > 2 {
		params = []byte(args[2])
	}

	// Start server
	server := exec.Command(cmd)
	server.Dir = dir
	stderr := &strings.Builder{} // Capture stderr for error reporting
	server.Stderr = stderr

	stdin, err := server.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("creating stdin pipe: %v", err)
	}

	stdout, err := server.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("creating stdout pipe: %v", err)
	}

	if err := server.Start(); err != nil {
		return "", fmt.Errorf("starting server: %v", err)
	}

	// Ensure server is cleaned up
	defer func() {
		if err := server.Process.Kill(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to kill server: %v\n", err)
		}
		if err := server.Wait(); err != nil {
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Fprintf(os.Stderr, "warning: server exited with error: %v\n", err)
			}
		}
	}()

	// Create client
	client := mcp.NewClient(struct {
		io.ReadWriteCloser
	}{
		ReadWriteCloser: rwc{stdin, stdout},
	})
	defer client.Close()

	var result json.RawMessage
	var callErr error

	switch method {
	case "initialize":
		var initArgs mcp.InitializeArgs
		if err := json.Unmarshal(params, &initArgs); err != nil {
			return "", fmt.Errorf("parsing initialize args: %v", err)
		}
		reply, err := client.Initialize(context.Background(), initArgs.ClientInfo)
		if err != nil {
			callErr = err
		} else {
			result, err = json.Marshal(reply)
			if err != nil {
				return "", fmt.Errorf("marshaling response: %v", err)
			}
		}

	case "tools/list":
		tools, err := client.ListTools(context.Background())
		if err != nil {
			callErr = err
		} else {
			result, err = json.Marshal(map[string][]mcp.Tool{"tools": tools})
			if err != nil {
				return "", fmt.Errorf("marshaling response: %v", err)
			}
		}

	case "tools/call":
		var callArgs struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &callArgs); err != nil {
			return "", fmt.Errorf("parsing call args: %v", err)
		}
		toolResult, err := client.CallTool(context.Background(), callArgs.Name, callArgs.Arguments)
		if err != nil {
			callErr = err
		} else {
			result, err = json.Marshal(toolResult)
			if err != nil {
				return "", fmt.Errorf("marshaling response: %v", err)
			}
		}

	default:
		return "", fmt.Errorf("unknown method: %s", method)
	}

	if callErr != nil {
		return "", callErr
	}

	// Pretty print JSON for comparison
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, result, "", "  "); err != nil {
		return "", fmt.Errorf("formatting response: %v", err)
	}

	return prettyJSON.String(), nil
}

type rwc struct {
	io.WriteCloser
	io.Reader
}

func (rwc) Close() error { return nil }
