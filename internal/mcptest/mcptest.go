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
	"runtime"
	"strings"

	"github.com/tmc/mcp"
	"golang.org/x/tools/txtar"
	"rsc.io/script"
)

// serverState tracks the current MCP server process
type serverState struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	client *mcp.Client
}

func RunTxTarFile(ctx context.Context, filename string, output io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %v", err)
	}
	// Create script engine
	eng := script.NewEngine()
	// Track server state
	var state serverState
	defer func() {
		if state.cmd != nil {
			state.cmd.Process.Kill()
			state.cmd.Wait()
		}
	}()
	// Add MCP commands
	for name, cmd := range mcpCommands(output, &state) {
		eng.Cmds[name] = cmd
	}
	// Add default commands
	for name, cmd := range script.DefaultCmds() {
		eng.Cmds[name] = cmd
	}

	env := os.Environ()
	workdir := os.TempDir()
	s, err := script.NewState(ctx, workdir, env)
	if err != nil {
		return err
	}
	// Unpack archive.
	a, err := txtar.ParseFile(filename)
	if err != nil {
		return err
	}
	initScriptDirs(s)
	if err := s.ExtractFiles(a); err != nil {
		return err
	}
	work, _ := s.LookupEnv("WORK")
	fmt.Fprintf(output, "$WORK=%s", work)
	return eng.Execute(s, filename, bufio.NewReader(bytes.NewReader(content)), output)
}

func initScriptDirs(s *script.State) {
	must := func(err error) {}
	work := s.Getwd()
	must(s.Setenv("WORK", work))
	must(os.MkdirAll(filepath.Join(work, "tmp"), 0777))
	must(s.Setenv(tempEnvName(), filepath.Join(work, "tmp")))
}

func tempEnvName() string {
	switch runtime.GOOS {
	case "windows":
		return "TMP"
	case "plan9":
		return "TMPDIR" // actually plan 9 doesn't have one at all but this is fine
	default:
		return "TMPDIR"
	}
}

// handleMCPStart implements the mcp-start command
func handleMCPStart(s *script.State, output io.Writer, state *serverState, args ...string) (script.WaitFunc, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: mcp-start <command> [args...]")
	}

	// Kill any existing server
	if state.cmd != nil {
		state.cmd.Process.Kill()
		state.cmd.Wait()
		state.cmd = nil
		state.client = nil
	}

	cmd := exec.CommandContext(s.Context(), args[0], args[1:]...)
	cmd.Dir = s.Getwd()

	// Create pipes
	var err error
	state.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %v", err)
	}
	state.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %v", err)
	}

	// Wire stderr directly to os.Stderr for immediate feedback
	cmd.Stderr = os.Stderr

	fmt.Fprintf(output, "# Starting MCP server: %s %s\n", args[0], strings.Join(args[1:], " "))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting server: %v", err)
	}
	state.cmd = cmd

	transport := newDebugTransport(
		rwc{state.stdin, state.stdout},
		output,
	)
	// Create MCP client
	state.client = mcp.NewClient(transport)

	// Return a wait function that will clean up the server
	return func(*script.State) (string, string, error) {
		return "", "", nil
	}, nil
}

func handleMCP(s *script.State, output io.Writer, state *serverState, args ...string) (script.WaitFunc, error) {
	fmt.Fprintf(output, "# Handling MCP command: %s\n", strings.Join(args, " "))
	if state.client == nil {
		return nil, fmt.Errorf("no MCP server running, use mcp-start first")
	}

	if len(args) < 1 {
		return nil, fmt.Errorf("usage: mcp <method> [params]")
	}

	method := args[0]
	var params json.RawMessage
	if len(args) > 1 {
		params = []byte(args[1])
	}

	// Log the raw request
	req := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	reqBytes, _ := json.Marshal(req)
	fmt.Fprintf(output, "# >> %s\n", string(reqBytes))

	var result json.RawMessage
	var err error

	switch method {
	case "initialize":
		var initArgs mcp.InitializeArgs
		if err := json.Unmarshal(params, &initArgs); err != nil {
			return nil, fmt.Errorf("parsing initialize args: %v", err)
		}
		fmt.Fprintf(output, "# Calling Initialize with ClientInfo: %+v\n", initArgs.ClientInfo)
		reply, err := state.client.Initialize(s.Context(), initArgs.ClientInfo)
		if err != nil {
			fmt.Fprintf(output, "# Initialize error: %v\n", err)
			return nil, err
		}
		fmt.Fprintf(output, "# Initialize reply: %+v\n", reply)
		result, err = json.Marshal(reply)

		// ... rest of the cases ...
	}

	if err != nil {
		return nil, fmt.Errorf("executing %s: %v", method, err)
	}

	// Log the raw response
	fmt.Fprintf(output, "# << %s\n", string(result))

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, result, "", "  "); err != nil {
		return nil, fmt.Errorf("formatting response: %v", err)
	}

	return func(*script.State) (string, string, error) {
		return prettyJSON.String(), "", nil
	}, nil
}

// Add debug transport wrapper
type debugTransport struct {
	rw  io.ReadWriteCloser
	out io.Writer
}

func newDebugTransport(rw io.ReadWriteCloser, out io.Writer) *debugTransport {
	return &debugTransport{rw: rw, out: out}
}

func (d *debugTransport) Read(p []byte) (n int, err error) {
	n, err = d.rw.Read(p)
	if n > 0 {
		fmt.Fprintf(d.out, "# READ: %s\n", string(p[:n]))
	}
	if err != nil {
		fmt.Fprintf(d.out, "# READ ERR: %v\n", err)
	}
	return
}

func (d *debugTransport) Write(p []byte) (n int, err error) {
	fmt.Fprintf(d.out, "# WRITE: %s\n", string(p))
	return d.rw.Write(p)
}

func (d *debugTransport) Close() error {
	return d.rw.Close()
}

// mcpCommands returns the MCP-specific script commands
func mcpCommands(output io.Writer, state *serverState) map[string]script.Cmd {
	return map[string]script.Cmd{
		"mcp-start": script.Command(script.CmdUsage{
			Summary: "start MCP server",
			Async:   true,
		}, func(s *script.State, args ...string) (script.WaitFunc, error) {
			return handleMCPStart(s, output, state, args...)
		}),
		"mcp": script.Command(script.CmdUsage{
			Summary: "run MCP command",
		}, func(s *script.State, args ...string) (script.WaitFunc, error) {
			return handleMCP(s, output, state, args...)
		}),
	}
}

type rwc struct {
	io.WriteCloser
	io.Reader
}

func (r rwc) Close() error {
	if err := r.WriteCloser.Close(); err != nil {
		return err
	}
	if rc, ok := r.Reader.(io.Closer); ok {
		return rc.Close()
	}
	return nil
}
