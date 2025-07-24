// mcp-gen is a comprehensive code generation tool for MCP clients, servers, and boilerplate
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/codegen"
	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/config"
	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/templates"
)

const (
	version = "0.1.0"
	usage   = `mcp-gen - Multi-language MCP code generator

Usage:
  mcp-gen [command] [flags]

Commands:
  client     Generate MCP client SDK
  server     Generate MCP server stub
  types      Generate types from JSON schema
  docs       Generate documentation
  tests      Generate test suites
  plugin     Generate plugin boilerplate

Global flags:
  -lang string      Target language (go, typescript, python, rust, java)
  -output string    Output directory (default ".")
  -package string   Package/module name
  -config string    Configuration file path
  -verbose          Enable verbose output
  -dry-run          Preview generated code without writing files
  -help             Show help information
  -version          Show version information

Examples:
  # Generate Go client from MCP server
  mcp-gen client -lang go -output ./client -package github.com/user/client ./server

  # Generate TypeScript client from schema
  mcp-gen client -lang typescript -output ./ts-client schema.json

  # Generate server stub with tests
  mcp-gen server -lang go -output ./server -package myserver tools.json
  mcp-gen tests -lang go -output ./server/tests -package myserver ./server

  # Generate documentation
  mcp-gen docs -output ./docs ./server
`
)

type globalFlags struct {
	language string
	output   string
	pkg      string
	config   string
	verbose  bool
	dryRun   bool
	help     bool
	version  bool
}

func main() {
	var gf globalFlags
	
	flag.StringVar(&gf.language, "lang", "go", "Target language (go, typescript, python, rust, java)")
	flag.StringVar(&gf.output, "output", ".", "Output directory")
	flag.StringVar(&gf.pkg, "package", "", "Package/module name")
	flag.StringVar(&gf.config, "config", "", "Configuration file path")
	flag.BoolVar(&gf.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&gf.dryRun, "dry-run", false, "Preview generated code without writing files")
	flag.BoolVar(&gf.help, "help", false, "Show help information")
	flag.BoolVar(&gf.version, "version", false, "Show version information")

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if gf.help {
		fmt.Print(usage)
		return
	}

	if gf.version {
		fmt.Printf("mcp-gen version %s\n", version)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: command required\n\n")
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := args[0]
	commandArgs := args[1:]

	// Load configuration
	cfg, err := config.Load(gf.config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Apply CLI overrides
	if gf.language != "" {
		cfg.Language = gf.language
	}
	if gf.output != "" {
		cfg.Output = gf.output
	}
	if gf.pkg != "" {
		cfg.Package = gf.pkg
	}
	cfg.Verbose = gf.verbose
	cfg.DryRun = gf.dryRun

	// Create code generator
	generator, err := codegen.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	ctx := context.Background()

	// Execute command
	switch command {
	case "client":
		err = runClientCommand(ctx, generator, commandArgs)
	case "server":
		err = runServerCommand(ctx, generator, commandArgs)
	case "types":
		err = runTypesCommand(ctx, generator, commandArgs)
	case "docs":
		err = runDocsCommand(ctx, generator, commandArgs)
	case "tests":
		err = runTestsCommand(ctx, generator, commandArgs)
	case "plugin":
		err = runPluginCommand(ctx, generator, commandArgs)
	default:
		log.Fatalf("Unknown command: %s", command)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func runClientCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("client command requires input (server binary or schema file)")
	}

	input := args[0]
	
	// Check if input is a server binary or schema file
	if strings.HasSuffix(input, ".json") {
		// Generate client from schema
		return gen.GenerateClientFromSchema(ctx, input)
	} else {
		// Generate client from running server
		return gen.GenerateClientFromServer(ctx, input)
	}
}

func runServerCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("server command requires input (tools definition file)")
	}

	input := args[0]
	return gen.GenerateServerStub(ctx, input)
}

func runTypesCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("types command requires input (JSON schema file)")
	}

	input := args[0]
	return gen.GenerateTypes(ctx, input)
}

func runDocsCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("docs command requires input (server binary or schema file)")
	}

	input := args[0]
	return gen.GenerateDocs(ctx, input)
}

func runTestsCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("tests command requires input (server binary or generated code directory)")
	}

	input := args[0]
	return gen.GenerateTests(ctx, input)
}

func runPluginCommand(ctx context.Context, gen *codegen.Generator, args []string) error {
	pluginName := "mcp-plugin"
	if len(args) > 0 {
		pluginName = args[0]
	}
	
	return gen.GeneratePlugin(ctx, pluginName)
}