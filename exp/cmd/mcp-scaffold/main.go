// mcp-scaffold generates complete MCP project scaffolding with best practices
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tmc/mcp/exp/cmd/mcp-scaffold/internal/scaffold"
)

const (
	version = "0.1.0"
	usage   = `mcp-scaffold - MCP project scaffolding tool

Usage:
  mcp-scaffold [command] [flags] <project-name>

Commands:
  init       Initialize a new MCP project
  server     Create a new MCP server project
  client     Create a new MCP client project
  tool       Add a new tool to existing project
  plugin     Create a new MCP plugin project

Global flags:
  -lang string      Target language (go, typescript, python, rust, java)
  -template string  Project template (basic, advanced, enterprise)
  -output string    Output directory (default: current directory)
  -author string    Author name
  -license string   License type (MIT, Apache-2.0, BSD-3-Clause)
  -ci string        CI/CD system (github, gitlab, jenkins)
  -verbose          Enable verbose output
  -dry-run          Preview generated files without writing
  -help             Show help information
  -version          Show version information

Examples:
  # Create a new Go MCP server project
  mcp-scaffold server -lang go -template advanced my-server

  # Create a TypeScript client project
  mcp-scaffold client -lang typescript -template basic my-client

  # Initialize a new project in current directory
  mcp-scaffold init -lang go -template enterprise

  # Add a tool to existing project
  mcp-scaffold tool -lang go calculate-tool

  # Create a plugin project
  mcp-scaffold plugin -lang go -template basic my-plugin
`
)

type globalFlags struct {
	language string
	template string
	output   string
	author   string
	license  string
	ci       string
	verbose  bool
	dryRun   bool
	help     bool
	version  bool
}

func main() {
	var gf globalFlags

	flag.StringVar(&gf.language, "lang", "go", "Target language")
	flag.StringVar(&gf.template, "template", "basic", "Project template")
	flag.StringVar(&gf.output, "output", ".", "Output directory")
	flag.StringVar(&gf.author, "author", "", "Author name")
	flag.StringVar(&gf.license, "license", "MIT", "License type")
	flag.StringVar(&gf.ci, "ci", "github", "CI/CD system")
	flag.BoolVar(&gf.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&gf.dryRun, "dry-run", false, "Preview without writing")
	flag.BoolVar(&gf.help, "help", false, "Show help information")
	flag.BoolVar(&gf.version, "version", false, "Show version information")

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if gf.help {
		fmt.Print(usage)
		return
	}

	if gf.version {
		fmt.Printf("mcp-scaffold version %s\n", version)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: command required\n\n")
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := args[0]
	var projectName string
	if len(args) > 1 {
		projectName = args[1]
	}

	// Create scaffolder
	config := &scaffold.Config{
		Language:    gf.language,
		Template:    gf.template,
		Output:      gf.output,
		Author:      gf.author,
		License:     gf.license,
		CI:          gf.ci,
		Verbose:     gf.verbose,
		DryRun:      gf.dryRun,
		ProjectName: projectName,
	}

	scaffolder, err := scaffold.New(config)
	if err != nil {
		log.Fatalf("Failed to create scaffolder: %v", err)
	}

	ctx := context.Background()

	// Execute command
	switch command {
	case "init":
		err = scaffolder.Init(ctx)
	case "server":
		err = scaffolder.CreateServer(ctx)
	case "client":
		err = scaffolder.CreateClient(ctx)
	case "tool":
		err = scaffolder.AddTool(ctx)
	case "plugin":
		err = scaffolder.CreatePlugin(ctx)
	default:
		log.Fatalf("Unknown command: %s", command)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}

	if gf.verbose {
		fmt.Printf("Successfully executed %s command\n", command)
	}
}
