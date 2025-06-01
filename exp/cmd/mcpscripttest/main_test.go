package main

import (
	"os"
	"testing"
)

func TestConformance(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set up test args
	os.Args = []string{"mcpscripttest", "-conformance"}
	
	// Add server command if provided
	if len(oldArgs) > 1 {
		os.Args = append(os.Args, oldArgs[1:]...)
	}

	// Call main function
	main()
}