// Package bashext provides bash command extensions for mcpscripttest.
package bashext

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rsc.io/script"
)

// DefaultCommands returns the default bash commands for mcpscripttest.
func DefaultCommands() map[string]script.Cmd {
	return map[string]script.Cmd{
		"bash": bashCmd,
	}
}

// DefaultConditions returns the default bash conditions for mcpscripttest.
func DefaultConditions() map[string]script.Cond {
	return map[string]script.Cond{
		// Add any bash-specific conditions here if needed
	}
}

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

		// Check if coverage is enabled via environment variable from script state
		coverageEnabled := false
		var coverageDir string
		if goCoverDir, hasGoCoverDir := s.LookupEnv("GOCOVERDIR"); hasGoCoverDir && goCoverDir != "" {
			coverageEnabled = true
			coverageDir = goCoverDir
		}

		// If coverage is enabled, set up the trace file
		var bashTraceFile string
		if coverageEnabled {
			// Create a unique trace file for this bash command
			bashTraceFile = filepath.Join(coverageDir, fmt.Sprintf("bash-trace-%d.sh", time.Now().UnixNano()))
		}

		// Build the command
		cmd := exec.CommandContext(ctx, "/bin/bash", "-c", bashCommand)
		cmd.Dir = s.Getwd()

		// Copy environment from script state
		for _, env := range s.Environ() {
			cmd.Env = append(cmd.Env, env)
		}

		// Set up coverage tracing if enabled
		if coverageEnabled && bashTraceFile != "" {
			// Enable bash tracing to capture script execution
			cmd.Env = append(cmd.Env, "BASH_XTRACEFD=2") // Send xtrace to stderr
			cmd.Env = append(cmd.Env, "PS4=+ ")          // Simple trace prefix

			// Wrap the command to enable tracing
			tracedCommand := fmt.Sprintf("set -x; %s", bashCommand)
			cmd.Args = []string{"/bin/bash", "-c", tracedCommand}
		}

		// Set up environment from script state
		cmd.Env = s.Environ()

		// Handle any pending stdin
		if pendingStdin := getPendingStdin(s); pendingStdin != "" {
			cmd.Stdin = strings.NewReader(pendingStdin)
			clearPendingStdin(s)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start bash command: %w", err)
		}

		// Return wait function
		return func(s *script.State) (stdout, stderr string, err error) {
			err = cmd.Wait()

			// If coverage is enabled and we have a trace file, save it
			if coverageEnabled && bashTraceFile != "" {
				// The trace output would be in stderr, but we're not capturing it separately
				// In a more sophisticated implementation, you'd capture and process the trace
			}

			return "", "", err
		}, nil
	},
)

// Stdin management functions (these would need to be shared with core or moved to a common package)
func getPendingStdin(s *script.State) string {
	// This is a simplified version - in the actual implementation this would access
	// the shared stdin store from the internal package
	return ""
}

func clearPendingStdin(s *script.State) {
	// This is a simplified version - in the actual implementation this would clear
	// the shared stdin store from the internal package
}
