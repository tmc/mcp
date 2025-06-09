// Package mcptestutil provides comprehensive testing utilities for MCP implementations.
package mcptestutil

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

// TestTimeout is the default timeout for test operations.
const TestTimeout = 30 * time.Second

// TestConfig provides configuration for test utilities.
type TestConfig struct {
	Timeout    time.Duration
	WorkingDir string
	EnvVars    map[string]string
}

// DefaultTestConfig returns a sensible default test configuration.
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Timeout:    TestTimeout,
		WorkingDir: "",
		EnvVars:    make(map[string]string),
	}
}

// WithTimeout returns a new config with the specified timeout.
func (c TestConfig) WithTimeout(timeout time.Duration) TestConfig {
	c.Timeout = timeout
	return c
}

// WithWorkingDir returns a new config with the specified working directory.
func (c TestConfig) WithWorkingDir(dir string) TestConfig {
	c.WorkingDir = dir
	return c
}

// WithEnv returns a new config with additional environment variables.
func (c TestConfig) WithEnv(key, value string) TestConfig {
	if c.EnvVars == nil {
		c.EnvVars = make(map[string]string)
	}
	c.EnvVars[key] = value
	return c
}

// TestContext creates a test context with the configured timeout.
func (c TestConfig) TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.Timeout)
}

// PipeConnection creates a bidirectional pipe connection for testing.
type PipeConnection struct {
	Client net.Conn
	Server net.Conn
}

// Close closes both ends of the pipe connection.
func (p *PipeConnection) Close() error {
	err1 := p.Client.Close()
	err2 := p.Server.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// NewPipeConnection creates a new pipe connection for testing.
func NewPipeConnection() *PipeConnection {
	client, server := net.Pipe()
	return &PipeConnection{
		Client: client,
		Server: server,
	}
}

// TempDir creates a temporary directory for testing.
type TempDir struct {
	Path string
	t    *testing.T
}

// NewTempDir creates a new temporary directory.
func NewTempDir(t *testing.T, prefix string) *TempDir {
	t.Helper()
	path, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return &TempDir{
		Path: path,
		t:    t,
	}
}

// Cleanup removes the temporary directory.
func (td *TempDir) Cleanup() {
	td.t.Helper()
	if err := os.RemoveAll(td.Path); err != nil {
		td.t.Errorf("Failed to remove temp dir %s: %v", td.Path, err)
	}
}

// MockTransport provides a mock transport for testing.
type MockTransport struct {
	ReadFunc  func() ([]byte, error)
	WriteFunc func([]byte) error
	CloseFunc func() error
}

// Read implements the Read method for io.Reader.
func (m *MockTransport) Read(p []byte) (int, error) {
	if m.ReadFunc == nil {
		return 0, io.EOF
	}
	data, err := m.ReadFunc()
	if err != nil {
		return 0, err
	}
	copy(p, data)
	return len(data), nil
}

// Write implements the Write method for io.Writer.
func (m *MockTransport) Write(p []byte) (int, error) {
	if m.WriteFunc == nil {
		return len(p), nil
	}
	return len(p), m.WriteFunc(p)
}

// Close implements the Close method for io.Closer.
func (m *MockTransport) Close() error {
	if m.CloseFunc == nil {
		return nil
	}
	return m.CloseFunc()
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// AssertEqual fails the test if actual != expected.
func AssertEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual fails the test if actual == expected.
func AssertNotEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual == expected {
		t.Fatalf("Expected values to be different, but both were %v", actual)
	}
}

// AssertContains fails the test if s does not contain substr.
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("Expected %q to contain %q", s, substr)
	}
}

// AssertNotContains fails the test if s contains substr.
func AssertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("Expected %q to not contain %q", s, substr)
	}
}

// Eventually retries a condition until it's true or timeout is reached.
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("Condition not met within %v", timeout)
}

// MCPScriptEngine provides a script testing engine with MCP commands.
type MCPScriptEngine struct {
	engine *script.Engine
}

// NewMCPScriptEngine creates a new script engine with MCP commands.
func NewMCPScriptEngine() *MCPScriptEngine {
	engine := script.NewEngine()
	
	// Add MCP commands
	engine.Cmds["mcp-replay"] = mcpReplayCmd
	engine.Cmds["mcp-spy"] = mcpSpyCmd
	engine.Cmds["mcp-start"] = mcpStartCmd
	engine.Cmds["mcp-test"] = mcpTestCmd
	engine.Cmds["mcp-verify"] = mcpVerifyCmd
	
	return &MCPScriptEngine{engine: engine}
}

// Test runs script tests with MCP commands configured.
func (e *MCPScriptEngine) Test(t *testing.T, pattern string) {
	t.Helper()
	scripttest.Test(t, 
		context.Background(),
		e.engine,
		getTestEnvironment(),
		pattern)
}

// getTestEnvironment returns environment variables for testing.
func getTestEnvironment() []string {
	defaultInheritEnv := []string{"USER", "HOME", "PATH"}
	env := make(map[string]string)
	
	for _, key := range defaultInheritEnv {
		if val, ok := os.LookupEnv(key); ok {
			env[key] = val
		}
	}
	
	if inherit := os.Getenv("MCPSCRIPTTEST_ENV_INHERIT"); inherit != "" {
		for _, key := range strings.Split(inherit, ",") {
			key = strings.TrimSpace(key)
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	}
	
	var result []string
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// Command definitions for MCP tools
var (
	mcpReplayCmd = script.Command(
		script.CmdUsage{
			Summary: "replay MCP recordings",
			Args:    "recording [flags]",
		},
		execCmd("mcp-replay"),
	)

	mcpSpyCmd = script.Command(
		script.CmdUsage{
			Summary: "spy on MCP traffic",
			Args:    "[flags]",
		},
		execCmd("mcp-spy"),
	)

	mcpStartCmd = script.Command(
		script.CmdUsage{
			Summary: "start MCP components",
			Args:    "[flags]",
			Detail:  []string{"Starts MCP components in the background"},
			Async:   true,
		},
		execCmdAsync("mcp-start"),
	)

	mcpTestCmd = script.Command(
		script.CmdUsage{
			Summary: "run MCP tests",
			Args:    "[flags]",
		},
		execCmd("mcp-test"),
	)

	mcpVerifyCmd = script.Command(
		script.CmdUsage{
			Summary: "verify MCP recordings",
			Args:    "recording [flags]",
		},
		execCmd("mcp-verify"),
	)
)

// execCmd returns a standard command runner.
func execCmd(name string) func(*script.State, ...string) (script.WaitFunc, error) {
	return func(s *script.State, args ...string) (script.WaitFunc, error) {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(s.Context(), path, args...)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		stdout, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		return func(*script.State) (string, string, error) {
			return string(stdout), "", nil
		}, nil
	}
}

// execCmdAsync returns an async command runner.
func execCmdAsync(name string) func(*script.State, ...string) (script.WaitFunc, error) {
	return func(s *script.State, args ...string) (script.WaitFunc, error) {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(s.Context(), path, args...)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, err
		}

		if err := cmd.Start(); err != nil {
			return nil, err
		}

		return func(s *script.State) (string, string, error) {
			outBytes, err := io.ReadAll(stdout)
			if err != nil {
				return "", "", err
			}
			errBytes, err := io.ReadAll(stderr)
			if err != nil {
				return "", "", err
			}
			err = cmd.Wait()
			return string(outBytes), string(errBytes), err
		}, nil
	}
}