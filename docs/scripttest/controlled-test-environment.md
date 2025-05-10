# Controlled Test Environment for MCP Script Tests

This document explains how to implement a controlled environment for MCP script tests using a custom `TestMain` function. This setup ensures tests run in a predictable, isolated environment regardless of the host system configuration.

## Overview

When testing CLI applications, the environment can significantly impact the test results. Variables like the current working directory, PATH, HOME directory, and more can cause tests to behave differently on different systems or for different users.

The MCP project uses a custom `TestMain` function to create a controlled environment for scripttest tests. This ensures that tests run in a clean, isolated environment every time.

## Implementation

Here's a complete implementation of a controlled test environment for MCP script tests:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

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

	// Build the test binary in the current directory
	binaryName := filepath.Base(pwd) // Use directory name as binary name
	if err := exec.Command("go", "build", "-o", binaryName).Run(); err != nil {
		fmt.Printf("Failed to build %s: %v\n", binaryName, err)
		os.Exit(1)
	}
	*cleanupFiles = append(*cleanupFiles, filepath.Join(pwd, binaryName))

	// Set a minimal PATH that only includes the current directory
	os.Setenv("PATH", pwd)
	fmt.Printf("Modified PATH: %s\n", os.Getenv("PATH"))

	// Set up a controlled temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-test-*", binaryName))
	if err != nil {
		fmt.Printf("Failed to create temporary directory: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("TMPDIR", tempDir)
	*cleanupFiles = append(*cleanupFiles, tempDir)

	// Create a dummy HOME directory to avoid interference from user configuration
	fakeHome := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(fakeHome, 0755); err != nil {
		fmt.Printf("Failed to create fake home directory: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("HOME", fakeHome)

	fmt.Printf("Test environment ready: PATH=%s, HOME=%s, TMPDIR=%s\n", 
		os.Getenv("PATH"), os.Getenv("HOME"), os.Getenv("TMPDIR"))

	return origEnv
}

// cleanupEnvironment restores the original environment and removes temporary files
func cleanupEnvironment(origEnv map[string]string, cleanupFiles []string) {
	// Restore original environment
	for k, v := range origEnv {
		os.Setenv(k, v)
	}

	// Remove temporary files and directories
	for _, file := range cleanupFiles {
		// Check if it's a directory
		info, err := os.Stat(file)
		if err == nil && info.IsDir() {
			os.RemoveAll(file)
		} else {
			os.Remove(file)
		}
	}
}

func TestMyCommand(t *testing.T) {
	// Run all scripttest files in the testdata/scripts directory
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
```

## Key Components

### 1. TestMain Function

The `TestMain` function is a special function recognized by the Go testing framework. It runs before any tests in the package and can be used to set up the test environment.

```go
func TestMain(m *testing.M) {
    // Set up test environment
    // ...
    
    // Run tests
    code := m.Run()
    
    // Clean up
    // ...
    
    os.Exit(code)
}
```

### 2. Setting Up the Environment

The `setupTestEnvironment` function creates a controlled environment for testing:

1. **Saves the original environment**: Stores the original values of environment variables to restore them later.

2. **Builds the test binary**: Compiles the current package into a binary with the same name.

3. **Sets a minimal PATH**: Sets the PATH to include only the directory with the test binary.

4. **Creates a controlled temporary directory**: Sets up a dedicated temporary directory for the test.

5. **Creates a dummy HOME directory**: To avoid interference from user configuration files.

### 3. Cleaning Up

The `cleanupEnvironment` function ensures that the test environment is cleaned up after tests:

1. **Restores the original environment**: Sets environment variables back to their original values.

2. **Removes temporary files and directories**: Deletes any files or directories created during testing.

## Advantages

This controlled environment approach offers several advantages:

1. **Isolation**: Tests run in a clean environment, isolated from the host system.

2. **Reproducibility**: Tests produce the same results regardless of the host system configuration.

3. **Predictability**: Tests have a consistent starting state every time.

4. **Safety**: Tests can't accidentally modify important system files or directories.

5. **Cleanliness**: All temporary files and directories are automatically cleaned up.

## Example Usage

To use this approach in your MCP command package:

1. Copy the `TestMain`, `setupTestEnvironment`, and `cleanupEnvironment` functions to your test file.

2. Customize the `binaryName` variable to match your command name if needed.

3. Create script tests in the `testdata/scripts/` directory.

4. Run your tests with `go test ./cmd/yourcommand`.

## Extending the Environment

You can extend the controlled environment to include additional setup steps:

1. **Adding dependencies**: If your command depends on other binaries, you can build and add them to the PATH.

2. **Setting up configuration files**: You can create configuration files in the fake HOME directory.

3. **Setting up test data**: You can populate the test directory with test data files.

## See Also

- [ScriptTest Environment Guide](./scripttest-environment.md)
- [Script Test Examples](./scripttest-examples.md)
- [MCP Test Framework](../development/testing.md)