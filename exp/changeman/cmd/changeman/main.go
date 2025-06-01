package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tmc/mcp/exp/changeman"
)

func main() {
	var (
		projectRoot = flag.String("root", ".", "Project root directory")
		description = flag.String("desc", "", "Change description to analyze")
	)
	flag.Parse()

	if *description == "" {
		fmt.Fprintf(os.Stderr, "Error: change description is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create change manager
	manager := changeman.NewChangeManager(*projectRoot)

	// Analyze the change
	analysis, err := manager.AnalyzeChange(*description)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing change: %v\n", err)
		os.Exit(1)
	}

	// Print the analysis
	fmt.Println(analysis.Summary())
}