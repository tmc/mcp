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
		outputDir  = flag.String("output", "docs", "Output directory for documentation")
		format     = flag.String("format", "markdown", "Documentation format: markdown or html")
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

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create documentation generator
	generator := changemanagement.NewDocumentationGenerator()

	// Generate documentation
	docs, err := generator.GenerateDocs(&analysis, *format)
	if err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	// Save documentation files
	for _, doc := range docs {
		outputPath := filepath.Join(*outputDir, doc.Filename)
		
		// Create subdirectories if needed
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
			continue
		}
		
		err := os.WriteFile(outputPath, []byte(doc.Content), 0644)
		if err != nil {
			log.Printf("Failed to write %s: %v", outputPath, err)
			continue
		}
		
		if *verbose {
			fmt.Printf("Created: %s\n", outputPath)
		}
	}

	fmt.Printf("Generated %d documentation files in %s\n", len(docs), *outputDir)
}