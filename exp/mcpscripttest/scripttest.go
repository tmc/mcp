// Package mcpscripttest provides testing utilities for MCP tools using script-based testing.
// It wraps the rsc.io/script/scripttest package to provide script-based testing for MCP.
package mcpscripttest

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

var defaultInheritEnv = []string{"USER", "HOME", "PATH"}

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
}

// DefaultOptions returns the default options
func DefaultOptions() *MCPScripttestOptions {
	return &MCPScripttestOptions{
		AdditionalEnvVars:         nil,
		CustomCommands:            make(map[string]script.Cmd),
		CustomConditions:          make(map[string]script.Cond),
		IncludeDefaultMCPCommands: true,
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
			// Create context for the test
			ctx := context.Background()
			scripttest.Test(t, ctx, engine, env, scriptFile)
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
	e.Cmds["mcpspy"] = mcpSpyCmd // Alias
	e.Cmds["mcpdiff"] = mcpDiffCmd

	// Add utility commands
	e.Cmds["stdout"] = stdoutVerifyCmd
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

// execCmd returns a standard command runner
func execCmd(name string) func(*script.State, ...string) (script.WaitFunc, error) {
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

// execCmdAsync returns an async command runner
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
