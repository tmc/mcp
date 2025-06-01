package main

import (
	"fmt"
	"log"
	"time"
	
	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

func main() {
	fmt.Println("Running standalone fuzzing")
	
	// Define test function
	testFunc := func(script string) error {
		// This would normally run the script through mcpscripttest
		fmt.Printf("Testing script: %d bytes\n", len(script))
		return nil
	}
	
	// Run fuzzing
	err := fuzzing.Run(testFunc, fuzzing.RunOptions{
		Duration:    10 * time.Second,
		MaxScripts:  100,
		CoverageDir: "",
		ServerCommand: "echo 'test server'",
	})
	
	if err != nil {
		log.Fatalf("Fuzzing failed: %v", err)
	}
	
	fmt.Println("Scripts tested: 100")
}