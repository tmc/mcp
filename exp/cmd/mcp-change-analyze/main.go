package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/mcp/exp/changemanagement"
)

func main() {
	var (
		description = flag.String("description", "", "Natural language change description")
		file        = flag.String("file", "", "Read description from file")
		output      = flag.String("output", "-", "Output file (default: stdout)")
		format      = flag.String("format", "json", "Output format: json, yaml, or text")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Get change description
	var changeDesc string
	if *description != "" {
		changeDesc = *description
	} else if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		changeDesc = string(data)
	} else {
		// Read from stdin if no description provided
		if flag.NArg() > 0 {
			changeDesc = strings.Join(flag.Args(), " ")
		} else {
			log.Fatal("Please provide a change description via -description, -file, or stdin")
		}
	}

	// Create analyzer
	analyzer := changemanagement.NewChangeAnalyzer()

	// Analyze the change
	analysis, err := analyzer.AnalyzeChange(changeDesc)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Output results
	var outputData []byte
	switch *format {
	case "json":
		outputData, err = json.MarshalIndent(analysis, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
	case "yaml":
		// TODO: Implement YAML output
		log.Fatal("YAML output not yet implemented")
	case "text":
		outputData = []byte(formatAsText(analysis))
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
			fmt.Printf("Analysis written to %s\n", *output)
		}
	}
}

func formatAsText(analysis *changemanagement.AnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("Change Analysis Results\n")
	sb.WriteString("======================\n\n")

	sb.WriteString(fmt.Sprintf("Type: %s\n", analysis.Type))
	sb.WriteString(fmt.Sprintf("Category: %s\n", analysis.Category))
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", analysis.RiskLevel))
	sb.WriteString(fmt.Sprintf("Breaking Change: %v\n", analysis.Breaking))
	sb.WriteString(fmt.Sprintf("Confidence: %.2f\n\n", analysis.Confidence))

	if len(analysis.Components) > 0 {
		sb.WriteString("Affected Components:\n")
		for _, comp := range analysis.Components {
			sb.WriteString(fmt.Sprintf("  - %s\n", comp))
		}
		sb.WriteString("\n")
	}

	if len(analysis.Requirements.Functional) > 0 {
		sb.WriteString("Functional Requirements:\n")
		for _, req := range analysis.Requirements.Functional {
			sb.WriteString(fmt.Sprintf("  - %s\n", req))
		}
		sb.WriteString("\n")
	}

	if len(analysis.Requirements.NonFunctional) > 0 {
		sb.WriteString("Non-Functional Requirements:\n")
		for _, req := range analysis.Requirements.NonFunctional {
			sb.WriteString(fmt.Sprintf("  - %s\n", req))
		}
		sb.WriteString("\n")
	}

	if len(analysis.AffectedAreas) > 0 {
		sb.WriteString("Affected Areas:\n")
		for _, area := range analysis.AffectedAreas {
			sb.WriteString(fmt.Sprintf("  - %s\n", area))
		}
		sb.WriteString("\n")
	}

	if len(analysis.Recommendations) > 0 {
		sb.WriteString("Recommendations:\n")
		for _, rec := range analysis.Recommendations {
			sb.WriteString(fmt.Sprintf("  - %s (confidence: %.2f)\n", rec.Suggestion, rec.Confidence))
		}
	}

	return sb.String()
}
