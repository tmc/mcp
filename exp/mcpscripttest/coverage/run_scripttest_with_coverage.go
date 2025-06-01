// +build ignore

// This is a standalone program that shows how to run a scripttest with coverage
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Create a temporary directory for coverage
	tempDir, err := os.MkdirTemp("", "scripttest-coverage-demo")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	coverDir := filepath.Join(tempDir, "coverage")
	os.MkdirAll(coverDir, 0755)

	fmt.Printf("Running scripttest with coverage...\n")
	fmt.Printf("Coverage directory: %s\n\n", coverDir)

	// Set up environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOCOVERDIR=%s", coverDir))

	// Run the test
	cmd := exec.Command("go", "test", "-v", "-run", "TestScripttestCoverageAcrossBinaries")
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Test failed: %v", err)
	}

	fmt.Printf("\n=== Coverage Analysis ===\n")

	// List coverage files
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		log.Printf("Failed to read coverage directory: %v", err)
		return
	}

	fmt.Printf("Coverage files created:\n")
	for _, entry := range entries {
		fmt.Printf("  %s\n", entry.Name())
	}

	// Analyze coverage
	fmt.Printf("\nCoverage by package:\n")
	cmd = exec.Command("go", "tool", "covdata", "percent", "-i", coverDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}