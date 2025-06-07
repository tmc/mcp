// Package internal provides the core testing utilities for MCP tools using script-based testing.
// It wraps the rsc.io/script/scripttest package to provide script-based testing for MCP.
package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"rsc.io/script"
	"rsc.io/script/scripttest"

	"github.com/tmc/mcp/exp/mcpscripttest/coverage"
)

var defaultInheritEnv = []string{"USER", "HOME", "PATH"}

// stdinStore stores pending stdin for the next command
var stdinStore struct {
	sync.Mutex
	pendingContent map[*script.State]string
}

func init() {
	stdinStore.pendingContent = make(map[*script.State]string)
}

// Options is an alias for MCPScripttestOptions for external API
type Options = MCPScripttestOptions

// MCPScripttestOptions defines configuration options for MCP scripttest
type MCPScripttestOptions struct {
	// Additional environment variables to inherit
	AdditionalEnvVars []string
	// Custom commands to add to the engine
	CustomCommands map[string]script.Cmd
	// Custom conditions to add to the engine
	CustomConditions map[string]script.Cond
	// Whether to include default MCP commands
	IncludeDefaultMCPCommands bool
	// Whether to run go tool deadcode at the end of tests
	RunDeadcodeCheck bool
	// Whether to enable debug shell on test failure
	DebugMode bool
	// Aggressive timeout configurations
	TimeoutConfig *TimeoutConfig
}

// TimeoutConfig defines aggressive timeout settings for different operations
type TimeoutConfig struct {
	// Default command timeout (default: 10s, was 30s)
	DefaultCommandTimeout time.Duration
	// Server startup timeout (default: 5s, was 5s)
	ServerStartupTimeout time.Duration
	// Server response timeout (default: 2s, was 5s)
	ServerResponseTimeout time.Duration
	// Bash command timeout (default: 15s, was 30s)
	BashCommandTimeout time.Duration
	// Tool execution timeout (default: 10s, was no limit)
	ToolExecutionTimeout time.Duration
	// Test overall timeout (default: 30s, was no limit)
	TestOverallTimeout time.Duration
	// Retry settings
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultTimeoutConfig returns reasonable default timeout settings
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		DefaultCommandTimeout: 30 * time.Second,       // Restored to reasonable value
		ServerStartupTimeout:  10 * time.Second,       // Increased for reliability
		ServerResponseTimeout: 10 * time.Second,       // Increased for reliability
		BashCommandTimeout:    60 * time.Second,       // Increased for complex operations
		ToolExecutionTimeout:  30 * time.Second,       // Reasonable tool execution time
		TestOverallTimeout:    120 * time.Second,      // Generous overall test limit
		MaxRetries:            3,                      // Keep retry logic
		RetryDelay:            500 * time.Millisecond, // Slightly longer retry delay
	}
}

// DefaultOptions returns the default options
func DefaultOptions() *MCPScripttestOptions {
	return &MCPScripttestOptions{
		AdditionalEnvVars:         nil,
		CustomCommands:            make(map[string]script.Cmd),
		CustomConditions:          make(map[string]script.Cond),
		IncludeDefaultMCPCommands: true,
		RunDeadcodeCheck:          true,
		DebugMode:                 false,
		TimeoutConfig:             DefaultTimeoutConfig(),
	}
}

func getTestEnvironment(additionalEnvVars []string) []string {
	env := make(map[string]string)

	// Include default environment variables
	for _, key := range defaultInheritEnv {
		if val, ok := os.LookupEnv(key); ok {
			env[key] = val
		}
	}

	// Include additional environment variables specified in options
	for _, key := range additionalEnvVars {
		if val, ok := os.LookupEnv(key); ok {
			env[key] = val
		}
	}

	// Include environment variables specified in MCPSCRIPTTEST_ENV_INHERIT
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

// Test runs script tests with MCP commands configured.
func Test(t *testing.T, pattern string, opts ...*MCPScripttestOptions) {
	t.Helper()

	// Handle options
	var options *MCPScripttestOptions
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	} else {
		options = DefaultOptions()
	}

	// Find files matching the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatalf("No script files found matching pattern: %s", pattern)
	}

	engine := NewEngine(options)
	env := getTestEnvironment(options.AdditionalEnvVars)

	// Run each script file as a subtest
	for _, file := range files {
		scriptFile := file // Capture for closure
		baseName := filepath.Base(scriptFile)
		t.Run(baseName, func(t *testing.T) {
			// Create context with overall test timeout
			ctx := context.Background()

			// Check if we should disable timeouts (when MCP_TOOL_TIMEOUT=0 is set)
			if os.Getenv("MCP_TOOL_TIMEOUT") == "0" {
				// Create a completely fresh context to avoid any inherited timeouts
				ctx = context.Background()
			}

			// Temporarily disable timeout to debug context cancellation issues
			/*
				if options.TimeoutConfig.TestOverallTimeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, options.TimeoutConfig.TestOverallTimeout)
					defer cancel()
				}
			*/
			synctestCompatibleTest(t, ctx, engine, env, scriptFile)
		})
	}
}

// NewEngine returns a script engine configured with MCP commands.
func NewEngine(opts ...*MCPScripttestOptions) *script.Engine {
	// Handle options
	var options *MCPScripttestOptions
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	} else {
		options = DefaultOptions()
	}

	e := script.NewEngine()

	// Add default commands and conditions from scripttest
	for k, v := range scripttest.DefaultCmds() {
		e.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		e.Conds[k] = v
	}

	// Add default MCP commands if enabled
	if options.IncludeDefaultMCPCommands {
		addDefaultMCPCommands(e)
	}

	// Add server command support
	registerServerCommands(e)

	// Add custom commands and conditions
	for k, v := range options.CustomCommands {
		e.Cmds[k] = v
	}
	for k, v := range options.CustomConditions {
		e.Conds[k] = v
	}

	return e
}

// addDefaultMCPCommands adds all the default MCP command tools to the engine
func addDefaultMCPCommands(e *script.Engine) {
	// Add all MCP commands with mcp- prefix
	e.Cmds["mcp-replay"] = mcpReplayCmd
	e.Cmds["mcp-spy"] = mcpSpyCmd
	e.Cmds["mcp-start"] = mcpStartCmd
	e.Cmds["mcp-test"] = mcpTestCmd
	e.Cmds["mcp-verify"] = mcpVerifyCmd
	e.Cmds["mcp-send"] = mcpSendCmd
	e.Cmds["mcp-recv"] = mcpRecvCmd
	e.Cmds["mcp-serve"] = mcpServeCmd
	e.Cmds["mcp-scripttest-server"] = mcpScripttestServerCmd
	e.Cmds["mcpspy"] = mcpSpyCmd // Alias
	e.Cmds["mcpdiff"] = mcpDiffCmd

	// Add the missing MCP tools
	e.Cmds["mcpcat"] = mcpcatCmd
	e.Cmds["mcp-sort"] = mcpSortCmd
	e.Cmds["mcp-shadow"] = mcpShadowCmd
	e.Cmds["mcp-probe"] = mcpProbeCmd

	// Add utility commands
	e.Cmds["stdout"] = stdoutVerifyCmd
	e.Cmds["setstdin"] = setStdinCmd
	e.Cmds["cat"] = catCmd
	e.Cmds["bash"] = bashCmd // Add bash command

	// Add shell-like variable assignment support
	e.Cmds["server_command"] = variableAssignCmd("server_command")
}

// Command definitions
var mcpReplayCmd = script.Command(
	script.CmdUsage{
		Summary: "replay MCP recordings",
		Args:    "recording [flags]",
	},
	execCmd("mcp-replay"),
)

var mcpSpyCmd = script.Command(
	script.CmdUsage{
		Summary: "spy on MCP traffic",
		Args:    "[flags]",
	},
	execCmd("mcp-spy"),
)

var mcpStartCmd = script.Command(
	script.CmdUsage{
		Summary: "start MCP components",
		Args:    "[flags]",
		Detail:  []string{"Starts MCP components in the background"},
		Async:   true,
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// Handle --help synchronously
		if len(args) > 0 && args[0] == "--help" {
			return func(*script.State) (string, string, error) {
				return "Usage: mcp-start [options]\n", "", nil
			}, nil
		}
		return execCmdAsync("mcp-start")(s, args...)
	},
)

var mcpTestCmd = script.Command(
	script.CmdUsage{
		Summary: "run MCP tests",
		Args:    "[flags]",
	},
	execCmd("mcp-test"),
)

var mcpVerifyCmd = script.Command(
	script.CmdUsage{
		Summary: "verify MCP recordings",
		Args:    "recording [flags]",
	},
	execCmd("mcp-verify"),
)

var mcpSendCmd = script.Command(
	script.CmdUsage{
		Summary: "send MCP messages",
		Args:    "message [flags]",
	},
	execCmd("mcp-send"),
)

var mcpRecvCmd = script.Command(
	script.CmdUsage{
		Summary: "receive MCP messages",
		Args:    "[flags]",
	},
	execCmd("mcp-recv"),
)

var mcpDiffCmd = script.Command(
	script.CmdUsage{
		Summary: "compare MCP files",
		Args:    "file1 file2 [flags]",
	},
	execCmd("mcpdiff"),
)

var mcpcatCmd = script.Command(
	script.CmdUsage{
		Summary: "colorize MCP trace files",
		Args:    "[flags] file",
	},
	execCmd("mcpcat"),
)

var mcpSortCmd = script.Command(
	script.CmdUsage{
		Summary: "sort MCP trace files by timestamp",
		Args:    "[flags] file",
	},
	execCmd("mcp-sort"),
)

var mcpShadowCmd = script.Command(
	script.CmdUsage{
		Summary: "shadow MCP traffic to test server implementations",
		Args:    "[flags] --primary cmd --shadow cmd",
	},
	execCmd("mcp-shadow"),
)

// stdoutVerifyCmd verifies stdout contains specific text or matches a pattern
var stdoutVerifyCmd = script.Command(
	script.CmdUsage{
		Summary: "verify stdout contains text",
		Args:    "pattern",
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		if len(args) != 1 {
			return nil, script.ErrUsage
		}
		pattern := args[0]
		stdout := s.Stdout()
		if !strings.Contains(stdout, pattern) {
			return nil, fmt.Errorf("stdout did not contain expected text: %q", pattern)
		}
		return nil, nil
	},
)

// execCmd returns a standard command runner with aggressive timeouts
func execCmd(name string) func(*script.State, ...string) (script.WaitFunc, error) {
	return func(s *script.State, args ...string) (script.WaitFunc, error) {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}

		// Apply aggressive timeout for tool execution
		ctx := s.Context()
		timeout := 10 * time.Second // Default aggressive timeout
		if timeoutStr := os.Getenv("MCP_TOOL_TIMEOUT"); timeoutStr != "" {
			if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = parsedTimeout
			}
		}

		// Apply timeout only if it's not zero (zero means no timeout)
		var cancel context.CancelFunc
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
		} else {
			// No timeout requested
			cancel = func() {} // no-op cancel function
		}

		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return nil, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return nil, err
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return nil, err
		}

		return func(s *script.State) (string, string, error) {
			defer cancel()
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

// execCmdAsync returns an async command runner with aggressive timeouts
func execCmdAsync(name string) func(*script.State, ...string) (script.WaitFunc, error) {
	return func(s *script.State, args ...string) (script.WaitFunc, error) {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}

		// Apply aggressive timeout for async tool execution
		ctx := s.Context()
		timeout := 10 * time.Second // Default aggressive timeout
		if timeoutStr := os.Getenv("MCP_ASYNC_TIMEOUT"); timeoutStr != "" {
			if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = parsedTimeout
			}
		}

		// Apply timeout only if it's not zero (zero means no timeout)
		var cancel context.CancelFunc
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
		} else {
			// No timeout requested
			cancel = func() {} // no-op cancel function
		}

		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return nil, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return nil, err
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return nil, err
		}

		return func(s *script.State) (string, string, error) {
			defer cancel()
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

// Add the missing command definitions

var mcpServeCmd = script.Command(
	script.CmdUsage{
		Summary: "start MCP server from command",
		Args:    "-- command [args...]",
		Detail:  []string{"Start an MCP server using the provided command after the -- separator"},
	},
	execCmd("mcp-serve"),
)

var mcpScripttestServerCmd = script.Command(
	script.CmdUsage{
		Summary: "run a scriptable MCP server for testing",
		Args:    "[flags]",
	},
	execCmdAsync("mcp-scripttest-server"),
)

// setStdinCmd directly sets stdin content from a string for the next command
var setStdinCmd = script.Command(
	script.CmdUsage{
		Summary: "prepare stdin content for next command",
		Args:    "['text']",
		Detail: []string{
			"With text argument, uses that text as stdin for the next command",
			"With no arguments, uses the previous command's stdout as stdin",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		var content string

		if len(args) == 0 {
			content = s.Stdout()
		} else {
			// Join all args with spaces
			content = strings.Join(args, " ")
		}

		// Store the content for the next command's stdin
		stdinStore.Lock()
		stdinStore.pendingContent[s] = content
		stdinStore.Unlock()

		return nil, nil
	},
)

// catCmd implements a simple cat command
var catCmd = script.Command(
	script.CmdUsage{
		Summary: "concatenate and print",
		Args:    "[text...]",
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// If args are provided, output them
		if len(args) > 0 {
			return func(*script.State) (string, string, error) {
				return strings.Join(args, " ") + "\n", "", nil
			}, nil
		}

		// Otherwise, check for stdin content
		stdinStore.Lock()
		content, ok := stdinStore.pendingContent[s]
		if ok {
			delete(stdinStore.pendingContent, s)
		}
		stdinStore.Unlock()

		if ok {
			return func(*script.State) (string, string, error) {
				return content, "", nil
			}, nil
		}

		// No input, return empty
		return func(*script.State) (string, string, error) {
			return "", "", nil
		}, nil
	},
)

// variableAssignCmd creates a command that handles variable assignment syntax
func variableAssignCmd(varName string) script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: fmt.Sprintf("assign value to %s variable", varName),
			Args:    "= value",
			Detail:  []string{fmt.Sprintf("Sets the %s variable to the given value", varName)},
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) < 2 || args[0] != "=" {
				return nil, fmt.Errorf("invalid syntax: expected '%s = value'", varName)
			}

			// Join all args after = to handle values with spaces
			value := strings.Join(args[1:], " ")

			// Expand environment variables in the value
			value = os.ExpandEnv(value)

			// Store in environment for later use
			os.Setenv(varName, value)

			// Return function to print the assignment
			return func(*script.State) (string, string, error) {
				return fmt.Sprintf("%s=%s\n", varName, value), "", nil
			}, nil
		},
	)
}

// TestWithOptions runs script tests with the given options.
func TestWithOptions(t *testing.T, pattern string, opts *Options) {
	if opts == nil {
		opts = DefaultOptions()
	}
	Test(t, pattern, opts)
}

// TestWithCoverageOptions runs script tests with coverage options.
func TestWithCoverageOptions(t *testing.T, pattern string, coverageOpts *coverage.CoverageOptions, opts ...*Options) {
	// Note: coverage parameter handling would be implemented here
	// For now, just run the tests
	if len(opts) == 0 {
		opts = []*Options{DefaultOptions()}
	}
	Test(t, pattern, opts[0])
}

// isSynctestContext detects if we're running under synctest by checking for synctest context markers.
// This function uses reflection to detect synctest context types without importing testing/synctest,
// which may not be available in all builds.
func isSynctestContext(ctx context.Context) bool {
	// Check if we're running under synctest by examining the context chain
	for ctx != nil {
		// Use reflection to check the context type name to avoid import dependency
		contextType := reflect.TypeOf(ctx)
		if contextType != nil {
			typeName := contextType.String()
			// Look for synctest-related context types
			if strings.Contains(typeName, "synctest") || strings.Contains(typeName, "Synctest") {
				return true
			}
		}

		// Walk up the context chain
		if valuer, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
			// Check if there's a parent context
			if parentCtx, ok := valuer.Value(context.Background()).(context.Context); ok && parentCtx != ctx {
				ctx = parentCtx
				continue
			}
		}

		// Try to get the underlying context using reflection
		val := reflect.ValueOf(ctx)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			// Look for a Context field
			for i := 0; i < val.NumField(); i++ {
				field := val.Field(i)
				if field.Type() == reflect.TypeOf((*context.Context)(nil)).Elem() && field.CanInterface() {
					if parentCtx, ok := field.Interface().(context.Context); ok && parentCtx != ctx {
						ctx = parentCtx
						continue
					}
				}
			}
		}

		break
	}

	// Also check for environment variables that might indicate synctest
	if os.Getenv("GOEXPERIMENT") != "" && strings.Contains(os.Getenv("GOEXPERIMENT"), "synctest") {
		return true
	}

	return false
}
