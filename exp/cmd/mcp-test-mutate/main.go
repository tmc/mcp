package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tmc/mcp/exp/changemanagement"
)

func main() {
	var (
		testFile   = flag.String("test", "", "Test file to mutate")
		outputDir  = flag.String("output", "mutations", "Output directory for mutations")
		strategies = flag.String("strategies", "all", "Mutation strategies: all, reorder, fuzz, timing, error")
		count      = flag.Int("count", 10, "Number of mutations to generate")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *testFile == "" {
		log.Fatal("Please provide a test file via -test")
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Read test file
	content, err := os.ReadFile(*testFile)
	if err != nil {
		log.Fatalf("Failed to read test file: %v", err)
	}

	// Create mutator
	mutator := changemanagement.NewTestMutator()

	// Parse strategies
	var selectedStrategies []changemanagement.MutationStrategy
	if *strategies == "all" {
		selectedStrategies = []changemanagement.MutationStrategy{
			changemanagement.MutationReorder,
			changemanagement.MutationFuzz,
			changemanagement.MutationTiming,
			changemanagement.MutationError,
		}
	} else {
		// Parse specific strategies
		selectedStrategies = append(selectedStrategies, changemanagement.MutationReorder) // Default for now
	}

	// Generate mutations
	mutations, err := mutator.MutateTest(string(content), selectedStrategies, *count)
	if err != nil {
		log.Fatalf("Failed to generate mutations: %v", err)
	}

	// Save mutations
	baseFileName := filepath.Base(*testFile)
	baseFileName = baseFileName[:len(baseFileName)-len(filepath.Ext(baseFileName))]

	for i, mutation := range mutations {
		fileName := fmt.Sprintf("%s_mutation_%d_%s.txt", baseFileName, i+1, mutation.Type)
		outputPath := filepath.Join(*outputDir, fileName)

		err := os.WriteFile(outputPath, []byte(mutation.Content), 0644)
		if err != nil {
			log.Printf("Failed to write mutation %d: %v", i+1, err)
			continue
		}

		if *verbose {
			fmt.Printf("Created mutation: %s (type: %s)\n", outputPath, mutation.Type)
		}
	}

	fmt.Printf("Generated %d mutations in %s\n", len(mutations), *outputDir)
}
