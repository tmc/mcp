package main

import (
	"fmt"
	"log"

	"github.com/tmc/mcp/exp/changeman"
)

func main() {
	// Create a change manager for the MCP project
	manager := changeman.NewChangeManager("../../..")

	// Example change descriptions
	changes := []string{
		"Fix critical bug in JSON-RPC marshaling for mcp-probe tool",
		"Add new feature to support OutputSchema in draft spec",
		"Refactor transport layer to improve SSE streaming performance",
		"Update documentation for mcpdiff shadow record handling",
		"Implement security fix for authentication in client connections",
	}

	fmt.Println("Changeman Example - Analyzing MCP Changes")
	fmt.Println("========================================\n")

	for i, changeDesc := range changes {
		fmt.Printf("Change %d: %s\n", i+1, changeDesc)
		fmt.Println("-" + fmt.Sprintf("%d", len(changeDesc)+10))

		analysis, err := manager.AnalyzeChange(changeDesc)
		if err != nil {
			log.Printf("Error analyzing change: %v\n", err)
			continue
		}

		fmt.Println(analysis.Summary())
		fmt.Println()
	}
}