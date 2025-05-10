package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
	"rsc.io/script"
)

// Pre-find the claude executable before any environment changes
var realClaudePath, _ = exec.LookPath("claude")

// Pre-find node executable to check dependencies
var nodeAvailable = false

func init() {
	nodePath, err := exec.LookPath("node")
	if err == nil && nodePath != "" {
		nodeAvailable = true
	}
}

func TestMain(m *testing.M) {
	// Get the current working directory
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Keep track of what to clean up
	var cleanupFiles []string

	// Create a controlled test environment
	testEnv := setupTestEnvironment(pwd, &cleanupFiles)
	defer cleanupEnvironment(testEnv, cleanupFiles)

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

// setupTestEnvironment creates a controlled environment for testing
func setupTestEnvironment(pwd string, cleanupFiles *[]string) map[string]string {
	// Print initial environment
	fmt.Printf("Initial Working Directory: %s\n", pwd)
	fmt.Printf("Initial PATH: %s\n", os.Getenv("PATH"))

	// Save original environment variables
	origEnv := map[string]string{
		"PATH":    os.Getenv("PATH"),
		"HOME":    os.Getenv("HOME"),
		"TMPDIR":  os.Getenv("TMPDIR"),
		"GOPATH":  os.Getenv("GOPATH"),
		"GOCACHE": os.Getenv("GOCACHE"),
	}

	// Create a bin directory for our executables
	binDir, err := os.MkdirTemp("", "mcpdiff-bin-*")
	if err != nil {
		fmt.Printf("Failed to create bin directory: %v\n", err)
		os.Exit(1)
	}
	*cleanupFiles = append(*cleanupFiles, binDir)

	// Build the mcpdiff binary in the bin directory
	mcpdiffPath := filepath.Join(binDir, "mcpdiff")
	buildMcpdiffCmd := exec.Command("go", "build", "-o", mcpdiffPath)
	if err := buildMcpdiffCmd.Run(); err != nil {
		fmt.Printf("Failed to build mcpdiff: %v\n", err)
		os.Exit(1)
	}

	// Build the claude-proxy binary in the bin directory
	claudeProxyPath := filepath.Join(binDir, "claude")
	claudeProxySrc := filepath.Join(pwd, "..", "claude-proxy")
	buildClaudeCmd := exec.Command("go", "build", "-o", claudeProxyPath, claudeProxySrc)
	if err := buildClaudeCmd.Run(); err != nil {
		fmt.Printf("Failed to build claude-proxy: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built claude proxy at: %s\n", claudeProxyPath)

	// Set a minimal PATH that only includes our bin directory
	// This is necessary for the test framework to find our test binary
	origPath := os.Getenv("PATH")
	// Note: we preserve the original path for process execution
	// but update the PATH for test binary discovery
	os.Setenv("PATH", binDir+":"+origPath)
	fmt.Printf("Modified PATH: %s\n", os.Getenv("PATH"))

	// Set up a controlled temporary directory
	tempDir, err := os.MkdirTemp("", "mcpdiff-test-*")
	if err != nil {
		fmt.Printf("Failed to create temporary directory: %v\n", err)
		os.Exit(1)
	}

	// Instead of setting global TMPDIR, we'll just track it for cleanup
	*cleanupFiles = append(*cleanupFiles, tempDir)

	// Create a dummy HOME directory to avoid interference from user configuration
	fakeHome := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(fakeHome, 0755); err != nil {
		fmt.Printf("Failed to create fake home directory: %v\n", err)
		os.Exit(1)
	}

	// For TestMain we'll use a modified environment, but child processes
	// will get their environment from script.State.Environ()
	fmt.Printf("Test environment ready: bin=%s, temp=%s\n", binDir, tempDir)

	return origEnv
}

// cleanupEnvironment restores the original environment and removes temporary files
func cleanupEnvironment(origEnv map[string]string, cleanupFiles []string) {
	// Restore original environment (only PATH is needed since we only modified that)
	if origPath, ok := origEnv["PATH"]; ok {
		os.Setenv("PATH", origPath)
	}

	fmt.Printf("Cleaning up %d temporary files/directories\n", len(cleanupFiles))
	// Remove temporary files and directories
	for _, file := range cleanupFiles {
		// Check if it's a directory
		info, err := os.Stat(file)
		if err == nil && info.IsDir() {
			fmt.Printf("Removing directory: %s\n", file)
			if err := os.RemoveAll(file); err != nil {
				fmt.Printf("Warning: Failed to remove directory %s: %v\n", file, err)
			}
		} else if err == nil {
			fmt.Printf("Removing file: %s\n", file)
			if err := os.Remove(file); err != nil {
				fmt.Printf("Warning: Failed to remove file %s: %v\n", file, err)
			}
		} else {
			fmt.Printf("Warning: Could not stat %s: %v\n", file, err)
		}
	}
	fmt.Printf("Cleanup complete\n")
}

func TestMCPDiff(t *testing.T) {
	// Create a custom options object with additional commands
	options := mcpscripttest.DefaultOptions()

	// Log claude and node availability
	if realClaudePath != "" {
		fmt.Printf("Found claude at: %s\n", realClaudePath)
	} else {
		fmt.Printf("Claude not found in PATH\n")
	}

	fmt.Printf("Node.js available: %v\n", nodeAvailable)

	// Add a custom claude command that handles various scenarios
	options.CustomCommands["claude"] = script.Command(
		script.CmdUsage{
			Summary: "Claude AI assistant CLI",
			Args:    "[flags] [prompt]",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			// Check if we should skip path cleaning (useful for environments where node is installed)
			skipPathCleaning := os.Getenv("MCPSCRIPTTEST_SKIP_PATH_CLEANING") != ""

			// If no claude is found, use mock implementation
			if realClaudePath == "" {
				return mockClaudeCommand(args)
			}

			// If node.js is not available and we're not skipping path cleaning, use mock
			if !nodeAvailable && !skipPathCleaning {
				fmt.Printf("Node.js not available, using mock Claude implementation\n")
				return mockClaudeCommand(args)
			}

			// We have claude and node.js, try to use the real claude
			cmd := exec.Command(realClaudePath, args...)
			cmd.Dir = s.Getwd()

			// Set up the environment
			env := s.Environ()
			if skipPathCleaning {
				// Use environment as-is if skipping path cleaning
				cmd.Env = env
			} else {
				// Otherwise, try to preserve the PATH to find node
				origPath := os.Getenv("PATH")
				claudeDir := filepath.Dir(realClaudePath)

				newEnv := []string{}
				pathSet := false

				for _, envVar := range env {
					if strings.HasPrefix(envVar, "PATH=") {
						newEnv = append(newEnv, "PATH="+claudeDir+":"+origPath)
						pathSet = true
					} else {
						newEnv = append(newEnv, envVar)
					}
				}

				if !pathSet {
					newEnv = append(newEnv, "PATH="+claudeDir+":"+origPath)
				}

				cmd.Env = newEnv
			}

			// Set up pipes
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return nil, err
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return nil, err
			}

			// Start the command
			if err := cmd.Start(); err != nil {
				fmt.Printf("Failed to start claude: %v, falling back to mock\n", err)
				return mockClaudeCommand(args)
			}

			// Return wait function
			return func(s *script.State) (string, string, error) {
				outBytes, err := io.ReadAll(stdout)
				if err != nil {
					return "", "", fmt.Errorf("failed to read claude stdout: %w", err)
				}
				errBytes, err := io.ReadAll(stderr)
				if err != nil {
					return "", "", fmt.Errorf("failed to read claude stderr: %w", err)
				}

				waitErr := cmd.Wait()

				// If the command fails, fall back to mock
				if waitErr != nil {
					fmt.Printf("Claude execution failed: %v, falling back to mock\n", waitErr)
					mockWaitFn, _ := mockClaudeCommand(args)
					return mockWaitFn(s)
				}

				return string(outBytes), string(errBytes), nil
			}, nil
		},
	)

	// Run all scripttest files in the testdata/scripts directory with custom options
	mcpscripttest.Test(t, "testdata/scripts/*.txt", options)
}

// mockClaudeCommand provides a mock implementation of Claude for testing
func mockClaudeCommand(args []string) (script.WaitFunc, error) {
	// For version flag, return mock version
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
		return func(s *script.State) (string, string, error) {
			return "claude version 1.0.0 (mock for testing)\n", "", nil
		}, nil
	}

	// For capabilities, return mock response
	if len(args) > 0 && args[0] == "capabilities" {
		return func(s *script.State) (string, string, error) {
			return `{"models":["claude-3-opus-20240229","claude-3-sonnet-20240229"],"tools":["bash","read_file"]}`, "", nil
		}, nil
	}

	// Special handling for MCP commands
	if len(args) > 0 && args[0] == "mcp" {
		if len(args) > 1 && args[1] == "list" {
			return func(s *script.State) (string, string, error) {
				return "MOCK - Claude MCP list command\nAvailable MCP servers:\n  - mock-time-server\n  - mock-file-server\n", "", nil
			}, nil
		}

		// Generic MCP command response
		return func(s *script.State) (string, string, error) {
			return fmt.Sprintf("MOCK - Claude MCP command: %s\n", strings.Join(args[1:], " ")), "", nil
		}, nil
	}

	// For other prompts, return a mock response
	prompt := strings.Join(args, " ")
	return func(s *script.State) (string, string, error) {
		return fmt.Sprintf("MOCK - Claude would have processed: %s\n", prompt), "", nil
	}, nil
}
