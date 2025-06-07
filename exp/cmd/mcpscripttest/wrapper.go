// Package main provides a wrapper for running mcpscripttest without go test
package main

import (
	"os"
	"testing"
)

// Initialize testing framework for standalone execution
func init() {
	// Mock the testing framework for standalone usage
	testing.Init()
}

// standaloneRunner wraps main() to handle testing initialization
func standaloneRunner() {
	// The testing package requires initialization
	// We provide a minimal implementation that doesn't require go test
	os.Exit(runMain())
}

func runMain() int {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && err.Error() == "testing: Short called before Init" {
				// Ignore this specific panic
				return
			}
			panic(r)
		}
	}()

	main()
	return 0
}
