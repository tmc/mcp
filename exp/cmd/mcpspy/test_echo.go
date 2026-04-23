package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	testEcho = flag.Bool("test-echo", false, "Run echo tool test")
)

func init() {
	// Flag is already registered in the var declaration
}

// addTestFlags adds the test command handling
func addTestFlags() {
	// The testEcho flag is already registered in init()
}

// handleTestCommands checks if any test commands need to be run
// Returns true if a test command was handled
func handleTestCommands() bool {
	fmt.Fprintf(os.Stderr, "Checking for test commands...\n")
	if *testEcho {
		fmt.Fprintf(os.Stderr, "Running echo tool test...\n")
		TestEchoTool()
		return true
	}
	fmt.Fprintf(os.Stderr, "No test commands found\n")
	return false
}
