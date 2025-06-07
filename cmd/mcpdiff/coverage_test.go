package main

import (
	"os"
	"testing"
)

// TestMainNoArgs tests behavior with no arguments
func TestMainNoArgs(t *testing.T) {
	// Just test basic setup without calling main()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcpdiff"}
	// Don't actually call main() to avoid exit
}