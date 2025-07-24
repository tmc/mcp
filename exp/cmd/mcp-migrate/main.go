// mcp-migrate provides migration and upgrade assistance for MCP projects
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tmc/mcp/exp/cmd/mcp-migrate/internal/migrate"
)

const (
	version = "0.1.0"
	usage   = `mcp-migrate - MCP migration and upgrade assistant

Usage:
  mcp-migrate [command] [flags] [source] [target]

Commands:
  analyze    Analyze project for migration opportunities
  upgrade    Upgrade MCP protocol version
  transform  Transform code between versions
  validate   Validate migration results
  plan       Create migration plan
  apply      Apply migration plan

Global flags:
  -from string      Source version (e.g., "1.0", "2024-11-05")
  -to string        Target version (e.g., "2.0", "2024-12-03")
  -lang string      Project language (go, typescript, python, rust, java)
  -path string      Project path (default: current directory)
  -config string    Migration configuration file
  -backup           Create backup before migration
  -verbose          Enable verbose output
  -dry-run          Preview migration without applying changes
  -interactive      Interactive migration mode
  -help             Show help information
  -version          Show version information

Examples:
  # Analyze project for migration opportunities
  mcp-migrate analyze -lang go -path ./my-project

  # Upgrade from protocol v1.0 to v2.0
  mcp-migrate upgrade -from 1.0 -to 2.0 -lang go -path ./my-project

  # Create migration plan
  mcp-migrate plan -from 2024-11-05 -to 2024-12-03 -lang typescript

  # Apply migration with backup
  mcp-migrate apply -backup -verbose migration-plan.json

  # Interactive migration
  mcp-migrate upgrade -interactive -lang go
`
)

type globalFlags struct {
	from        string
	to          string
	language    string
	path        string
	config      string
	backup      bool
	verbose     bool
	dryRun      bool
	interactive bool
	help        bool
	version     bool
}

func main() {
	var gf globalFlags
	
	flag.StringVar(&gf.from, "from", "", "Source version")
	flag.StringVar(&gf.to, "to", "", "Target version")
	flag.StringVar(&gf.language, "lang", "", "Project language")
	flag.StringVar(&gf.path, "path", ".", "Project path")
	flag.StringVar(&gf.config, "config", "", "Migration configuration file")
	flag.BoolVar(&gf.backup, "backup", false, "Create backup before migration")
	flag.BoolVar(&gf.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&gf.dryRun, "dry-run", false, "Preview migration without applying")
	flag.BoolVar(&gf.interactive, "interactive", false, "Interactive migration mode")
	flag.BoolVar(&gf.help, "help", false, "Show help information")
	flag.BoolVar(&gf.version, "version", false, "Show version information")

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if gf.help {
		fmt.Print(usage)
		return
	}

	if gf.version {
		fmt.Printf("mcp-migrate version %s\n", version)
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

	// Create migrator
	config := &migrate.Config{
		From:        gf.from,
		To:          gf.to,
		Language:    gf.language,
		Path:        gf.path,
		ConfigFile:  gf.config,
		Backup:      gf.backup,
		Verbose:     gf.verbose,
		DryRun:      gf.dryRun,
		Interactive: gf.interactive,
	}

	migrator, err := migrate.New(config)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}

	ctx := context.Background()

	// Execute command
	switch command {
	case "analyze":
		err = migrator.Analyze(ctx)
	case "upgrade":
		err = migrator.Upgrade(ctx)
	case "transform":
		err = migrator.Transform(ctx, commandArgs)
	case "validate":
		err = migrator.Validate(ctx)
	case "plan":
		err = migrator.CreatePlan(ctx)
	case "apply":
		err = migrator.ApplyPlan(ctx, commandArgs)
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