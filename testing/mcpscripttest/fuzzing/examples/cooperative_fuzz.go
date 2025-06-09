package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: cooperative_fuzz <binary_path>")
	}

	binaryPath := os.Args[1]
	fmt.Printf("Cooperative fuzzing with %s\n", binaryPath)

	// Create smart generator with the binary
	config := fuzzing.SmartGeneratorConfig{
		GeneratorConfig: fuzzing.GeneratorConfig{
			IncludeExec: true,
		},
		EnableIntrospection: true,
		BinaryPaths:         []string{binaryPath},
		ValidateCommands:    true,
	}

	generator := fuzzing.NewSmartGenerator(time.Now().UnixNano(), config)

	// Generate commands using cooperative fuzzing
	for i := 0; i < 3; i++ {
		script := generator.Generate()
		fmt.Printf("Generated script %d:\n%s\n\n", i+1, script)
	}

	fmt.Println("Generated valid commands")
}
