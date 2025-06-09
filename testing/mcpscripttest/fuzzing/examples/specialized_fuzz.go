package main

import (
	"fmt"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

func main() {
	fmt.Println("Using MCP trace generator")

	// Create MCP-focused generator
	generator := fuzzing.NewMCPTraceGenerator(time.Now().UnixNano())

	// Generate some scripts
	for i := 0; i < 5; i++ {
		script := generator.Generate()
		fmt.Printf("Script %d:\n%s\n\n", i+1, script)
	}

	fmt.Println("Scripts generated: 5")
}
