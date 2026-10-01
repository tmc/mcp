package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rsc.io/script"
)

// bashCmd runs a bash command with the -c flag
var bashCmd = script.Command(
	script.CmdUsage{
		Summary: "run a bash command",
		Args:    "'command'",
		Detail: []string{
			"Runs a bash command with the -c flag",
			"Example: bash 'echo hello world'",
			"Example: bash 'cat file.txt | grep foo'",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bash command requires exactly one argument (the command string)")
		}

		// Get the command to run
		bashCommand := args[0]

		// Create the bash command with -c flag
		ctx := s.Context()
		// Temporarily disable timeout to debug context cancellation issues
		/*
			timeout := 60 * time.Second // Reasonable default timeout
			if timeoutStr := os.Getenv("MCP_BASH_TIMEOUT"); timeoutStr != "" {
				if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
					timeout = parsedTimeout
				}
			}
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
		*/

		// Check if coverage is enabled via environment variable from script state
		coverageEnabled := false
		var coverageDir string

		// Look through the script's environment for coverage settings
		for _, env := range s.Environ() {
			if strings.HasPrefix(env, "MCP_BASH_COVERAGE=") && strings.TrimPrefix(env, "MCP_BASH_COVERAGE=") == "1" {
				coverageEnabled = true
			}
			if strings.HasPrefix(env, "MCP_BASH_COVERAGE_DIR=") {
				coverageDir = strings.TrimPrefix(env, "MCP_BASH_COVERAGE_DIR=")
			}
		}

		var traceFile *os.File
		var tracePath string

		if coverageEnabled {
			// Create a trace file for this bash execution
			traceDir := coverageDir
			if traceDir == "" {
				traceDir = filepath.Join(os.TempDir(), "mcp-bash-coverage")
			}
			os.MkdirAll(traceDir, 0755)

			// Generate unique trace filename
			testName := "unknown"
			// Try to get test name from environment
			for _, env := range s.Environ() {
				if strings.HasPrefix(env, "TEST_NAME=") {
					testName = strings.ReplaceAll(strings.TrimPrefix(env, "TEST_NAME="), "/", "_")
					break
				}
			}
			tracePath = filepath.Join(traceDir, fmt.Sprintf("bash-%s-%d.trace", testName, time.Now().UnixNano()))

			var err error
			traceFile, err = os.Create(tracePath)
			if err != nil {
				s.Logf("Warning: Failed to create bash trace file: %v", err)
				coverageEnabled = false
			}
		}

		// Prepare the command
		cmdStr := bashCommand
		if coverageEnabled && traceFile != nil {
			// Enable tracing with PS4 to include script name and line numbers
			// Use FD 3 which is the first extra file descriptor
			s.Logf("Enabling bash coverage, trace file: %s", tracePath)
			cmdStr = fmt.Sprintf("PS4='+($0:$LINENO): '; set -x; export BASH_XTRACEFD=3; exec 3>>%s; %s", tracePath, bashCommand)
		}

		cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		// Pass the trace file to the subprocess if coverage is enabled
		if coverageEnabled && traceFile != nil {
			// ExtraFiles starts at file descriptor 3
			cmd.ExtraFiles = []*os.File{traceFile}
		}

		// Set up stdin if available
		stdinStore.Lock()
		stdinContent, hasStdin := stdinStore.pendingContent[s]
		if hasStdin {
			delete(stdinStore.pendingContent, s)
		}
		stdinStore.Unlock()

		if hasStdin {
			cmd.Stdin = strings.NewReader(stdinContent)
		}

		// Run the command and capture output
		output, err := cmd.CombinedOutput()

		// Close trace file if we opened one
		if traceFile != nil {
			traceFile.Close()
			// Record the trace file location for later analysis
			if err == nil && tracePath != "" {
				s.Setenv("LAST_BASH_TRACE", tracePath)
				s.Logf("Trace file closed: %s", tracePath)
			}
		}

		return func(s *script.State) (string, string, error) {
			if err != nil {
				// If there's an error, return the output in stderr
				return "", string(output), err
			}
			// Return the output in stdout
			return string(output), "", nil
		}, nil
	},
)

// mcpProbeCmd is a command for probing MCP servers
var mcpProbeCmd = script.Command(
	script.CmdUsage{
		Summary: "probe MCP servers for information",
		Args:    "[flags] command",
		Detail: []string{
			"Probes MCP servers to get information about their capabilities",
			"Example: mcp-probe --timeout 5s mcp-serve -- echo-server",
		},
	},
	execCmd("mcp-probe"),
)
