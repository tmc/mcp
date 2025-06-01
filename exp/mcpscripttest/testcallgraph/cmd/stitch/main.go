package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func main() {
	var testFile string
	flag.StringVar(&testFile, "test", "", "Scripttest file to analyze")
	flag.Parse()

	if testFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -test <scripttest_file>\n", os.Args[0])
		os.Exit(1)
	}

	stitcher := testcallgraph.NewStitcher()
	
	// Analyze the test file and stitch to programs
	connections, err := stitcher.StitchTestToPrograms(testFile)
	if err != nil {
		log.Fatalf("Failed to stitch: %v", err)
	}

	if len(connections) == 0 {
		fmt.Println("No program connections found")
		return
	}

	fmt.Printf("Found %d program connections:\n", len(connections))
	for _, conn := range connections {
		fmt.Printf("  %s\n", conn)
	}
}