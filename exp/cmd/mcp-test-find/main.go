package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/changemanagement"
)

func main() {
	var (
		changeFile = flag.String("change", "", "Change analysis JSON file")
		codebase   = flag.String("codebase", ".", "Root directory of codebase")
		output     = flag.String("output", "-", "Output file (default: stdout)")
		format     = flag.String("format", "json", "Output format: json or text")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *changeFile == "" {
		log.Fatal("Please provide a change analysis file via -change")
	}

	// Load change analysis
	analysisData, err := os.ReadFile(*changeFile)
	if err != nil {
		log.Fatalf("Failed to read change file: %v", err)
	}

	var analysis changemanagement.AnalysisResult
	if err := json.Unmarshal(analysisData, &analysis); err != nil {
		log.Fatalf("Failed to parse change analysis: %v", err)
	}

	// Create test finder
	finder := changemanagement.NewTestFinder(*codebase)

	// Find affected tests
	result, err := finder.FindAffectedTests(&analysis)
	if err != nil {
		log.Fatalf("Failed to find tests: %v", err)
	}

	// Output results
	var outputData []byte
	switch *format {
	case "json":
		outputData, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
	case "text":
		outputData = []byte(formatAsText(result))
	default:
		log.Fatalf("Unknown format: %s", *format)
	}

	// Write output
	if *output == "-" {
		fmt.Println(string(outputData))
	} else {
		err = os.WriteFile(*output, outputData, 0644)
		if err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
		if *verbose {
			fmt.Printf("Test analysis written to %s\n", *output)
		}
	}
}

func formatAsText(result *changemanagement.TestFinderResult) string {
	var sb strings.Builder

	sb.WriteString("Affected Tests Analysis\n")
	sb.WriteString("======================\n\n")

	if len(result.DefinitelyAffected) > 0 {
		sb.WriteString("Definitely Affected:\n")
		for _, test := range result.DefinitelyAffected {
			sb.WriteString(fmt.Sprintf("  - %s\n", test))
		}
		sb.WriteString("\n")
	}

	if len(result.PossiblyAffected) > 0 {
		sb.WriteString("Possibly Affected:\n")
		for _, test := range result.PossiblyAffected {
			sb.WriteString(fmt.Sprintf("  - %s\n", test))
		}
		sb.WriteString("\n")
	}

	if len(result.RelatedTests) > 0 {
		sb.WriteString("Related Tests:\n")
		for _, test := range result.RelatedTests {
			sb.WriteString(fmt.Sprintf("  - %s\n", test))
		}
		sb.WriteString("\n")
	}

	if len(result.NewTestsNeeded) > 0 {
		sb.WriteString("New Tests Needed:\n")
		for _, test := range result.NewTestsNeeded {
			sb.WriteString(fmt.Sprintf("  - %s\n", test))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Total tests found: %d\n", 
		len(result.DefinitelyAffected) + len(result.PossiblyAffected) + len(result.RelatedTests)))

	return sb.String()
}