// Command testcallgraph analyzes call graphs for mcpscripttest tests
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

var (
	algo      = flag.String("algo", "rta", "Call graph algorithm (static, cha, rta, vta)")
	format    = flag.String("format", "dot", "Output format (dot, json, text)")
	proximity = flag.String("proximity", "", "Find tests closest to specified location (file:line)")
	packages  = flag.String("packages", "", "Comma-separated list of packages to analyze")
	test      = flag.Bool("test", false, "Include test code in analysis")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `testcallgraph: analyze call graphs for mcpscripttest tests

Usage:
  testcallgraph [flags] test_file.txt

Examples:
  # Generate call graph for a test
  testcallgraph -packages ./cmd/mcpdiff,./cmd/mcpcat coverage_test.txt

  # Find tests closest to uncovered code
  testcallgraph -proximity parser.go:89 coverage_test.txt

  # Output in different formats
  testcallgraph -format json coverage_test.txt

Flags:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	testFile := flag.Arg(0)

	// Parse packages
	var pkgs []string
	if *packages != "" {
		pkgs = strings.Split(*packages, ",")
	} else {
		// Default to common MCP packages
		pkgs = []string{
			"./cmd/mcpdiff",
			"./cmd/mcpcat", 
			"./cmd/mcpspy",
		}
	}

	// Handle proximity analysis
	if *proximity != "" {
		if err := analyzeProximity(testFile, *proximity, pkgs); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Regular call graph analysis
	if err := analyzeCallGraph(testFile, pkgs); err != nil {
		log.Fatal(err)
	}
}

func analyzeCallGraph(testFile string, packages []string) error {
	// Build call graph
	graph, err := testcallgraph.Build(testFile, packages...)
	if err != nil {
		return fmt.Errorf("building call graph: %w", err)
	}

	// Parse and trace test file
	tests := parseTestFile(testFile)
	for _, test := range tests {
		trace, err := graph.TraceExecution(test)
		if err != nil {
			log.Printf("Warning: failed to trace %s: %v", test, err)
			continue
		}
		graph.Traces[test.String()] = trace
	}

	// Generate output
	switch *format {
	case "dot":
		viz := &testcallgraph.Visualizer{Graph: graph}
		fmt.Print(viz.GenerateDOT())
	case "json":
		// TODO: Implement JSON output
		fmt.Println("{}")
	case "text":
		// TODO: Implement text output
		fmt.Println("Call graph:")
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return nil
}

func analyzeProximity(testFile, target string, packages []string) error {
	// Parse target location
	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid target format, expected file:line")
	}

	targetLoc := testcallgraph.SourceLocation{
		File: parts[0],
		Line: parseLineNumber(parts[1]),
	}

	// Build call graph
	graph, err := testcallgraph.Build(testFile, packages...)
	if err != nil {
		return fmt.Errorf("building call graph: %w", err)
	}

	// Analyze proximity
	analyzer := &testcallgraph.ProximityAnalyzer{Graph: graph}
	result, err := analyzer.FindClosestTest(targetLoc)
	if err != nil {
		return fmt.Errorf("proximity analysis: %w", err)
	}

	// Display results
	fmt.Printf("Proximity Analysis for %s\n", target)
	fmt.Printf("=================================\n\n")

	if result.Distance < 0 {
		fmt.Println("No path found from any test to target location")
		return nil
	}

	fmt.Printf("Closest test: %s\n", result.Test)
	fmt.Printf("Distance: %d function calls\n", result.Distance)
	fmt.Printf("\nSuggested modification:\n")
	fmt.Printf("  (Analysis of how to modify test to reach target)\n")

	return nil
}

func parseTestFile(file string) []testcallgraph.TestLocation {
	// TODO: Implement actual scripttest file parsing
	// For now, return dummy data
	return []testcallgraph.TestLocation{
		{File: file, Line: 5, Cmd: "exec mcpdiff file1.mcp file2.mcp"},
		{File: file, Line: 10, Cmd: "exec mcpcat -color=never file1.mcp"},
	}
}

func parseLineNumber(s string) int {
	var line int
	fmt.Sscanf(s, "%d", &line)
	return line
}