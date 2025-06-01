package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/cmd2mcpserver"
)

func main() {
	var (
		outputDir   = flag.String("output", "", "Output directory for the generated MCP server")
		moduleName  = flag.String("module", "", "Go module name for the generated server")
		serverName  = flag.String("server", "", "Server struct name (default: based on tool name)")
		toolName    = flag.String("tool", "", "Tool name for MCP (default: based on binary name)")
		description = flag.String("desc", "", "Tool description")
		sourceDir   = flag.String("source", "", "Source directory to analyze for flags (optional)")
		dryRun      = flag.Bool("dry-run", false, "Output generated source as txtar to stdout")
		toolDef     = flag.Bool("tool-def", false, "Output just the tool definition as JSON")
		verbose     = flag.Bool("v", false, "Verbose output")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <binary-path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nConverts a Go command-line tool into an MCP server.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate MCP server:\n")
		fmt.Fprintf(os.Stderr, "  %s -output ./myserver -module github.com/user/myserver ./mycommand\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Preview generated code:\n")
		fmt.Fprintf(os.Stderr, "  %s -dry-run ./mycommand\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # View tool definition:\n")
		fmt.Fprintf(os.Stderr, "  %s -tool-def ./mycommand\n", os.Args[0])
	}
	
	flag.Parse()
	
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	
	binaryPath := flag.Arg(0)

	// Validate binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		log.Fatalf("Binary not found: %v", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	binaryPath = absPath

	// Try to analyze the binary for additional information
	analyzer := cmd2mcpserver.NewBinaryAnalyzer(binaryPath)
	binaryInfo, analyzerErr := analyzer.ExtractBinaryInfo()
	if analyzerErr == nil && *verbose {
		fmt.Printf("Binary Analysis:\n")
		fmt.Printf("  Module: %s\n", binaryInfo.ModuleInfo.ModulePath)
		fmt.Printf("  Version: %s\n", binaryInfo.ModuleInfo.ModuleVersion)
		if binaryInfo.Description != "" {
			fmt.Printf("  Description: %s\n", binaryInfo.Description)
		}
		if binaryInfo.SourcePath != "" {
			fmt.Printf("  Source: %s\n", binaryInfo.SourcePath)
		}
		fmt.Println()
	}
	
	// Derive defaults from binary name
	binaryName := filepath.Base(binaryPath)
	binaryName = strings.TrimSuffix(binaryName, filepath.Ext(binaryName))
	
	if *outputDir == "" {
		*outputDir = fmt.Sprintf("./%s-mcp-server", binaryName)
	}
	
	if *moduleName == "" {
		*moduleName = fmt.Sprintf("github.com/generated/%s-mcp-server", binaryName)
	}
	
	if *serverName == "" {
		// Convert to CamelCase
		parts := strings.Split(binaryName, "-")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		*serverName = strings.Join(parts, "")
	}
	
	if *toolName == "" {
		*toolName = binaryName
	}
	
	if *description == "" {
		// Use binary analysis description if available
		if analyzerErr == nil && binaryInfo.Description != "" {
			*description = binaryInfo.Description
		} else {
			*description = fmt.Sprintf("MCP wrapper for %s command", binaryName)
		}
	}
	
	config := &cmd2mcpserver.Config{
		BinaryPath:  binaryPath,
		OutputDir:   *outputDir,
		ModuleName:  *moduleName,
		ServerName:  *serverName,
		ToolName:    *toolName,
		Description: *description,
	}
	
	if *verbose {
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Binary: %s\n", config.BinaryPath)
		fmt.Printf("  Output: %s\n", config.OutputDir)
		fmt.Printf("  Module: %s\n", config.ModuleName)
		fmt.Printf("  Server: %s\n", config.ServerName)
		fmt.Printf("  Tool:   %s\n", config.ToolName)
		fmt.Printf("  Desc:   %s\n", config.Description)
		fmt.Println()
	}
	
	generator := cmd2mcpserver.NewGenerator(config)
	
	// If source directory provided, analyze it for flags
	sourceDirToAnalyze := *sourceDir

	// If no source directory provided but we found one from binary analysis, use that
	if sourceDirToAnalyze == "" && analyzerErr == nil && binaryInfo.SourcePath != "" {
		sourceDirToAnalyze = binaryInfo.SourcePath
		if *verbose {
			fmt.Printf("Using source directory from binary analysis: %s\n", sourceDirToAnalyze)
		}
	}

	if sourceDirToAnalyze != "" {
		extractor := cmd2mcpserver.NewFlagExtractor(sourceDirToAnalyze)
		flags, err := extractor.ExtractFlags()
		if err != nil {
			log.Fatalf("Failed to extract flags: %v", err)
		}
		generator.SetFlags(flags)
		generator.SetUsesStdin(extractor.UsesStdin())

		if *verbose {
			fmt.Printf("Extracted %d flag definitions\n", len(flags))
			if extractor.UsesStdin() {
				fmt.Println("Detected stdin usage")
			}
		}
	}
	
	if *toolDef {
		// Output just the tool definition as JSON
		toolDef := generator.GetToolDefinition()

		jsonBytes, err := json.MarshalIndent(toolDef, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal tool definition: %v", err)
		}

		fmt.Print(string(jsonBytes))
	} else if *dryRun {
		// Generate as txtar and output to stdout
		if *verbose {
			fmt.Fprintf(os.Stderr, "Generating MCP server as txtar for %s...\n", binaryName)
		}

		txtar, err := generator.GenerateTxtar()
		if err != nil {
			log.Fatalf("Failed to generate txtar: %v", err)
		}

		fmt.Print(txtar)
	} else {
		// Normal generation to filesystem
		fmt.Printf("Generating MCP server for %s...\n", binaryName)

		if err := generator.Generate(); err != nil {
			log.Fatalf("Failed to generate server: %v", err)
		}

		fmt.Printf("✓ Generated MCP server in %s\n", *outputDir)
		fmt.Printf("\nTo run the server:\n")
		fmt.Printf("  cd %s\n", *outputDir)
		fmt.Printf("  go run .\n")
	}
}