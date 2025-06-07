package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// Command-line flags
var (
	testPath       = flag.String("run", "testdata/*.txt", "Path to test file or directory")
	conformance    = flag.String("conformance", "", "Run conformance tests. Optional version value selects a specific version suite (e.g. '2025-03-26')")
	verbose        = flag.Bool("v", false, "Verbose output")
	showHelp       = flag.Bool("help", false, "Show help message")
	enableCoverage = flag.Bool("coverage", false, "Enable coverage instrumentation")
	debugOnFailure = flag.Bool("debug", false, "Enable debug shell on test failure")
	httpPort       = flag.Int("http-port", 8765, "Starting port number for HTTP tests")
	extendedTests  = flag.Bool("extended", false, "Run extended tests")
)

func main() {
	// Parse flags before --
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	// Get the server command after --
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: No server command provided. Use -- to separate flags from the server command.")
		printHelp()
		os.Exit(1)
		return
	}

	// Set base environment
	os.Setenv("MCP_CONFORMANCE", "true")
	os.Setenv("MCP_HTTP_PORT", strconv.Itoa(*httpPort))

	if *verbose {
		os.Setenv("MCP_VERBOSE", "true")
	}

	if *extendedTests {
		os.Setenv("MCP_EXTENDED_TESTS", "true")
	}

	// Detect capabilities of the server
	capabilities, err := detectServerCapabilities(args)
	if err != nil {
		fmt.Printf("Error detecting server capabilities: %v\n", err)
		os.Exit(1)
		return
	}

	// Set environment variables based on detected capabilities
	for capability, supported := range capabilities {
		if supported {
			fmt.Printf("Server supports: %s\n", capability)
		} else {
			disableVar := "MCP_DISABLE_" + strings.ToUpper(capability)
			os.Setenv(disableVar, "true")
			fmt.Printf("Server does not support: %s\n", capability)
		}
	}

	// Enable coverage if requested
	if *enableCoverage {
		coverDir := filepath.Join(os.TempDir(), "mcp-coverage-"+time.Now().Format("20060102-150405"))
		err := os.MkdirAll(coverDir, 0755)
		if err != nil {
			fmt.Printf("Failed to create coverage directory: %v\n", err)
		} else {
			os.Setenv("GOCOVERDIR", coverDir)
			fmt.Printf("Coverage data will be collected in: %s\n", coverDir)
		}
	}

	// Create mcpscripttest options
	options := mcpscripttest.DefaultOptions()
	options.DebugMode = *debugOnFailure

	// Store the server command for tests to use
	os.Setenv("MCP_SERVER_COMMAND", strings.Join(args, " "))

	// Create coverage options
	coverageOpts := &mcpscripttest.CoverageOptions{
		PerTestSubdir: true,
		VerboseOutput: *verbose,
	}

	// Process conformance flag if provided
	if *conformance != "" {
		// Conformance flag takes precedence over run flag
		version := *conformance
		if version == "true" || version == "" {
			// Default to the latest version if none specified
			version = "2025-03-26"
		}

		fmt.Printf("Running %s conformance tests\n", version)

		// Set the protocol version environment variable
		os.Setenv("MCP_CONFORMANCE_VERSION", version)

		// For earlier versions, use version-specific directory if it exists
		conformancePaths := []string{
			filepath.Join("testdata", "mcp_conformance", version),
			filepath.Join("testdata", "mcp_conformance"),
		}

		for _, path := range conformancePaths {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				*testPath = filepath.Join(path, "*.txt")
				break
			}
		}
	}

	// Resolve test path
	resolvedPath := *testPath
	info, err := os.Stat(resolvedPath)
	if err != nil {
		// Check if it might be a pattern
		if strings.Contains(resolvedPath, "*") {
			// It's a pattern, so we're good to go
		} else if !strings.HasSuffix(resolvedPath, ".txt") {
			// Check if it might be a test file without .txt extension
			testWithExt := resolvedPath + ".txt"
			if _, err := os.Stat(testWithExt); err == nil {
				resolvedPath = testWithExt
			} else {
				// Check in standard locations
				stdPaths := []string{
					filepath.Join("testdata", "mcp_conformance", resolvedPath+".txt"),
					filepath.Join("testdata", "mcp_conformance", resolvedPath),
				}

				for _, path := range stdPaths {
					if _, err := os.Stat(path); err == nil {
						resolvedPath = path
						break
					}
				}
			}
		}
	} else if info.IsDir() {
		// It's a directory, so use pattern matching
		resolvedPath = filepath.Join(resolvedPath, "*.txt")
	}

	fmt.Printf("Running MCP conformance tests from: %s\n", resolvedPath)
	fmt.Printf("Using server command: %s\n", strings.Join(args, " "))
	fmt.Printf("Coverage enabled: %v\n", *enableCoverage)

	// Setup the test runner
	runner := &mcpscripttest.TestRunner{
		Options:      options,
		CoverageOpts: coverageOpts,
		Verbose:      *verbose,
	}

	// Run the tests using standalone runner to avoid testing.Short() panic
	failures := runner.RunTestsStandalone(resolvedPath)

	// Report results
	if failures == 0 {
		fmt.Println("\nALL TESTS PASSED")
	} else {
		fmt.Printf("\n%d TESTS FAILED\n", failures)
		os.Exit(1)
	}
}

// detectServerCapabilities probes the provided server to determine what features it supports
func detectServerCapabilities(args []string) (map[string]bool, error) {
	capabilities := make(map[string]bool)

	// Default capabilities
	capabilities["stdio"] = true // Assume stdio is always supported
	capabilities["http"] = false
	capabilities["sse"] = false
	capabilities["websocket"] = false
	capabilities["tools"] = false
	capabilities["resources"] = false
	capabilities["prompts"] = false
	capabilities["logging"] = false
	capabilities["batch"] = true // Assume batch is supported by default

	// Try to start the server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return capabilities, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return capabilities, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the server
	if err := cmd.Start(); err != nil {
		return capabilities, fmt.Errorf("failed to start server: %v", err)
	}

	// Read stdout and stderr in goroutines
	stdoutCh := make(chan string, 100)
	stderrCh := make(chan string, 100)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutCh <- scanner.Text()
		}
		close(stdoutCh)
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrCh <- scanner.Text()
		}
		close(stderrCh)
	}()

	// Wait for output for a short time to see if we can detect HTTP support
	timeout := time.After(5 * time.Second)

	httpDetected := false
	sseDetected := false

detectionLoop:
	for {
		select {
		case line, ok := <-stdoutCh:
			if !ok {
				break detectionLoop
			}
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				httpDetected = true
			}
			if strings.Contains(line, "SSE") || strings.Contains(line, "Server-Sent Events") {
				sseDetected = true
			}
		case line, ok := <-stderrCh:
			if !ok {
				break detectionLoop
			}
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				httpDetected = true
			}
			if strings.Contains(line, "SSE") || strings.Contains(line, "Server-Sent Events") {
				sseDetected = true
			}
		case <-timeout:
			break detectionLoop
		case <-ctx.Done():
			break detectionLoop
		}
	}

	// If we didn't detect HTTP from output, try connecting to the port
	if !httpDetected {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", *httpPort), 1*time.Second)
		if err == nil {
			conn.Close()
			httpDetected = true
		}

		// Make a quick HTTP request to check if it responds
		if !httpDetected {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d", *httpPort))
			if err == nil {
				resp.Body.Close()
				httpDetected = true
			}
		}
	}

	// Set detected capabilities
	capabilities["http"] = httpDetected
	capabilities["sse"] = sseDetected

	// Clean up
	cmd.Process.Kill()
	cmd.Wait()

	// If we want to do a more thorough test, we could also probe for capabilities
	// by sending requests to the server, but for now this simple detection is enough

	return capabilities, nil
}

func printHelp() {
	fmt.Println("mcpscripttest - MCP Protocol Conformance Test Runner")
	fmt.Println("\nThis tool runs conformance tests against an MCP server")
	fmt.Println("\nUsage:")
	fmt.Println("  mcpscripttest [options] -- <server command>")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  Run all conformance tests against a stdio server:")
	fmt.Println("    mcpscripttest -- go run ./cmd/my-mcp-server")
	fmt.Println("\n  Run a specific test:")
	fmt.Println("    mcpscripttest -run 01_base_messaging -- go run ./cmd/my-mcp-server")
	fmt.Println("\n  Run tests with coverage:")
	fmt.Println("    mcpscripttest -coverage -- go run ./cmd/my-mcp-server")
	fmt.Println("\n  Run tests with verbose output:")
	fmt.Println("    mcpscripttest -v -- go run ./cmd/my-mcp-server")
	fmt.Println("\n  Run tests against an HTTP server on a specific port:")
	fmt.Println("    mcpscripttest -http-port 9000 -- go run ./cmd/my-mcp-server --http")
}
