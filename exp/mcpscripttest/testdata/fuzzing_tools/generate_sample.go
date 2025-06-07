package main

import (
	"fmt"
	"time"

	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

func main() {
	// Create a basic fuzzing generator
	seed := time.Now().UnixNano()
	generator := fuzzing.NewFuzzGenerator(seed, fuzzing.Options{
		MinCommands:          3,
		MaxCommands:          5,
		IncludeExec:          true,
		IncludeStdinCommands: true,
	})

	// Generate a sample script
	script := generator.Generate()

	fmt.Println("Generated script:")
	fmt.Println(script)
}
