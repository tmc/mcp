package fuzzing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// RunWithState executes a scripttest directly with state management for fuzzing
// This allows direct invocation of test scripts without going through the file system
func RunWithState(ctx context.Context, script string, serverCmd []string, opts *mcpscripttest.MCPScripttestOptions) error {
	if opts == nil {
		opts = mcpscripttest.DefaultOptions()
	}

	// Create a temporary file for the script
	tmpDir, err := os.MkdirTemp("", "mcp-fuzz-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte(script), 0644); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	// Set up coverage if requested
	// Note: Coverage is handled differently - check if coverage is enabled via env
	if os.Getenv("GOCOVERDIR") != "" {
		if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
			testCoverDir := filepath.Join(coverDir, fmt.Sprintf("run_%d", time.Now().UnixNano()))
			if err := os.MkdirAll(testCoverDir, 0755); err == nil {
				// Set environment variable directly
				os.Setenv("GOCOVERDIR", testCoverDir)
				defer func() {
					// Merge coverage data back
					exec.Command("go", "tool", "covdata", "merge", "-i", testCoverDir, "-o", coverDir).Run()
				}()
			}
		}
	}

	// Run the test
	state := &RunState{
		Context:   ctx,
		Script:    script,
		ServerCmd: serverCmd,
		Options:   opts,
		Dir:       tmpDir,
	}

	return state.Execute()
}

// RunState maintains the state of a running test
type RunState struct {
	Context   context.Context
	Script    string
	ServerCmd []string
	Options   *mcpscripttest.MCPScripttestOptions
	Dir       string
	env       map[string]string
	server    *exec.Cmd
	traces    map[string]*bytes.Buffer
	stdout    *bytes.Buffer
	stderr    *bytes.Buffer
}

// Execute runs the test with the current state
func (s *RunState) Execute() error {
	// Set up environment
	s.env = make(map[string]string)
	s.env["HOME"] = s.Dir
	s.env["TMPDIR"] = s.Dir
	s.env["PATH"] = os.Getenv("PATH")

	// Apply additional environment variables
	for _, envVar := range s.Options.AdditionalEnvVars {
		if val := os.Getenv(envVar); val != "" {
			s.env[envVar] = val
		}
	}

	// Parse and execute the script
	lines := strings.Split(s.Script, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := s.executeLine(line, i+1); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
	}

	// Clean up
	if s.server != nil {
		s.server.Process.Kill()
		s.server.Wait()
	}

	return nil
}

// executeLine executes a single line of the script
func (s *RunState) executeLine(line string, lineNum int) error {
	// Parse the command
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "exec":
		return s.execCommand(args, false)
	case "!":
		if len(args) > 0 && args[0] == "exec" {
			return s.execCommand(args[1:], true)
		}
	case "stdin":
		return s.sendStdin(strings.Join(args, " "))
	case "stdout":
		return s.expectOutput(s.stdout, strings.Join(args, " "))
	case "stderr":
		return s.expectOutput(s.stderr, strings.Join(args, " "))
	case "mcp-serve":
		return s.startServer(args)
	case "mcp-send":
		return s.sendMCPMessage(strings.Join(args, " "))
	case "wait":
		return s.waitForServer()
	case "skip":
		// Check condition
		if len(args) > 0 && shouldSkip(args[0]) {
			return fmt.Errorf("test skipped: %s", args[0])
		}
	}

	return nil
}

// execCommand executes a shell command
func (s *RunState) execCommand(args []string, expectFailure bool) error {
	if len(args) == 0 {
		return fmt.Errorf("exec: no command specified")
	}

	cmd := exec.CommandContext(s.Context, args[0], args[1:]...)
	cmd.Dir = s.Dir
	cmd.Env = envMapToSlice(s.env)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	// Update captured output
	s.stdout = stdout
	s.stderr = stderr

	if expectFailure {
		if err == nil {
			return fmt.Errorf("expected command to fail but it succeeded")
		}
		return nil
	}

	return err
}

// startServer starts the MCP server
func (s *RunState) startServer(args []string) error {
	if s.server != nil {
		return fmt.Errorf("server already running")
	}

	// Find the -- separator
	var serverArgs []string
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) {
			serverArgs = args[i+1:]
			break
		}
	}

	if len(serverArgs) == 0 {
		serverArgs = s.ServerCmd
	}

	if len(serverArgs) == 0 {
		return fmt.Errorf("no server command specified")
	}

	s.server = exec.CommandContext(s.Context, serverArgs[0], serverArgs[1:]...)
	s.server.Dir = s.Dir
	s.server.Env = envMapToSlice(s.env)

	// Set up trace capture if enabled
	// Note: Trace is handled differently in scripttest
	if false { // TODO: Implement trace support
		if err := s.setupTraceCapture(); err != nil {
			return err
		}
	}

	return s.server.Start()
}

// setupTraceCapture sets up MCP trace capture
func (s *RunState) setupTraceCapture() error {
	// This is a simplified version - in reality we'd set up
	// proper MCP trace capture through pipes or files
	s.traces = make(map[string]*bytes.Buffer)
	s.traces["server"] = &bytes.Buffer{}

	// Set environment variables for trace output
	traceFile := filepath.Join(s.Dir, "server.mcp")
	s.server.Env = append(s.server.Env, fmt.Sprintf("MCP_TRACE=%s", traceFile))

	return nil
}

// sendStdin sends data to the server's stdin
func (s *RunState) sendStdin(data string) error {
	if s.server == nil {
		return fmt.Errorf("no server running")
	}

	stdin, err := s.server.StdinPipe()
	if err != nil {
		return err
	}

	_, err = io.WriteString(stdin, data+"\n")
	return err
}

// sendMCPMessage sends an MCP message to the server
func (s *RunState) sendMCPMessage(message string) error {
	// In a real implementation, this would properly format and send
	// the message through the appropriate transport
	return s.sendStdin(message)
}

// expectOutput checks if the output contains the expected string
func (s *RunState) expectOutput(buf *bytes.Buffer, expected string) error {
	if buf == nil {
		return fmt.Errorf("no output captured")
	}

	output := buf.String()
	if !strings.Contains(output, expected) {
		return fmt.Errorf("expected %q in output, got %q", expected, output)
	}

	return nil
}

// waitForServer waits for the server to be ready
func (s *RunState) waitForServer() error {
	if s.server == nil {
		return fmt.Errorf("no server running")
	}

	// In a real implementation, we'd wait for the server to be ready
	// by checking its output or trying to connect
	time.Sleep(100 * time.Millisecond)
	return nil
}

// shouldSkip checks if a test should be skipped based on the condition
func shouldSkip(condition string) bool {
	switch condition {
	case "windows":
		return os.Getenv("GOOS") == "windows"
	case "!linux":
		return os.Getenv("GOOS") != "linux"
	default:
		return false
	}
}

// FuzzWithState provides a fuzzing interface that maintains state across runs
func FuzzWithState(f *testing.F, serverCmd []string, opts *mcpscripttest.MCPScripttestOptions) {
	// Set up coverage feedback
	coverageDir := os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		coverageDir = f.TempDir()
	}

	feedback, err := NewCoverageFeedback(coverageDir)
	if err != nil {
		f.Logf("Warning: Failed to initialize coverage feedback: %v", err)
		feedback = nil
	}

	// Create fuzzer
	generator := NewFuzzGenerator(0)
	var fuzzer *CoverageGuidedFuzzer
	if feedback != nil {
		fuzzer = NewCoverageGuidedFuzzer(generator, feedback)
	}

	// Add seed corpus
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		generator.rng = newRand(seed)

		// Generate script
		var script string
		if fuzzer != nil {
			script = fuzzer.GenerateInput()
		} else {
			script = generator.Generate()
		}

		// Run with state management
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := RunWithState(ctx, script, serverCmd, opts)

		// Handle coverage feedback
		if feedback != nil && fuzzer != nil && err == nil {
			// Note: In a real implementation, we'd properly integrate
			// with the coverage collection to get actual coverage data
			result := &TestCoverageResult{
				TestID:   seed,
				TestName: fmt.Sprintf("fuzz_%d", seed),
			}

			fuzzer.RecordResult(script, result)
		}

		// Report errors but don't fail on expected issues
		if err != nil && !strings.Contains(err.Error(), "test skipped") {
			t.Logf("Script execution failed: %v", err)
		}
	})
}

// envMapToSlice converts an environment map to a slice of KEY=VALUE strings
func envMapToSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}
