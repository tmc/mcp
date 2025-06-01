// Command claude-proxy provides a test implementation of the Claude CLI
// that can be used in script tests. It mimics basic Claude CLI functionality
// without requiring the actual Claude executable or its dependencies.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Models supported by the proxy
var supportedModels = []string{
	"claude-3-opus-20240229",
	"claude-3-sonnet-20240229",
	"claude-3-haiku-20240307",
}

// Tools supported by the proxy
var supportedTools = []string{
	"bash",
	"read_file",
	"write_file",
	"mcp",
}

// Mock MCP servers
var mcpServers = []string{
	"mock-time-server",
	"mock-echo-server",
	"mock-file-server",
}

// Main entry point for the proxy
func main() {
	// Configure logging
	log.SetPrefix("claude-proxy: ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// No arguments
	if len(os.Args) < 2 {
		fmt.Println("Claude AI assistant CLI (proxy for testing)")
		os.Exit(0)
	}

	// Process command-line arguments
	switch os.Args[1] {
	case "--version", "-v":
		// Version command
		fmt.Println("claude version 1.0.0 (proxy for testing)")

	case "capabilities":
		// Capabilities command
		capabilities := map[string]interface{}{
			"models": supportedModels,
			"tools":  supportedTools,
		}
		json, err := json.Marshal(capabilities)
		if err != nil {
			log.Fatalf("Error creating capabilities JSON: %v", err)
		}
		fmt.Println(string(json))

	case "mcp":
		// MCP commands
		if len(os.Args) < 3 {
			fmt.Println("PROXY - Claude MCP command requires a subcommand")
			os.Exit(1)
		}

		switch os.Args[2] {
		case "list":
			// List MCP servers
			fmt.Println("MOCK - Claude MCP list command")
			fmt.Println("Available MCP servers:")
			for _, server := range mcpServers {
				fmt.Printf("  - %s\n", server)
			}

		case "connect":
			if len(os.Args) < 4 {
				fmt.Println("PROXY - Claude MCP connect requires a server name")
				os.Exit(1)
			}
			// Connect to MCP server
			serverName := os.Args[3]
			fmt.Printf("MOCK - Claude MCP connecting to server: %s\n", serverName)

		default:
			// Handle other MCP commands
			subcommand := os.Args[2]
			args := strings.Join(os.Args[3:], " ")
			fmt.Printf("MOCK - Claude MCP command: %s %s\n", subcommand, args)
		}

	default:
		// For all other commands, treat as a prompt
		prompt := strings.Join(os.Args[1:], " ")
		fmt.Printf("MOCK - Claude would have processed: %s\n", prompt)
	}
}
