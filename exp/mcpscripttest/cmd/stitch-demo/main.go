package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func main() {
	var (
		testFile  string
		showTypes bool
		simple    bool
	)

	flag.StringVar(&testFile, "test", "", "Scripttest file to analyze")
	flag.StringVar(&testFile, "f", "", "Scripttest file to analyze (shorthand)")
	flag.BoolVar(&showTypes, "types", false, "Show command types")
	flag.BoolVar(&simple, "simple", false, "Use simple stitcher (only handles exec)")
	flag.Parse()

	// If no test file specified, create a demo
	if testFile == "" {
		runDemo()
		return
	}

	// Analyze the test file
	if simple {
		runSimpleStitcher(testFile)
	} else {
		runEnhancedStitcher(testFile, showTypes)
	}
}

func runDemo() {
	fmt.Println("=== Stitching Demo ===")
	fmt.Println("\nNo test file specified, running demo...")
	fmt.Println("\nExample test content:")
	
	demoContent := `exec mcpdiff --help
mcpdiff trace1.json trace2.json
mcp-spy -- mcpdiff test
mcp-server-start demo -- go run ./server/main.go
exec echo "test"`

	fmt.Println(demoContent)
	fmt.Println("\nAnalyzing with simple stitcher:")
	fmt.Println("------------------------------")
	
	// Simple stitcher
	simple := &testcallgraph.SimpleStitcher{}
	simpleConns := simple.AnalyzeAndStitch("demo.txt", demoContent)
	
	for _, conn := range simpleConns {
		fmt.Printf("Line %d: %s -> %s\n", conn.TestLine, conn.Program, conn.MainPath)
	}
	
	fmt.Println("\nAnalyzing with enhanced stitcher:")
	fmt.Println("---------------------------------")
	
	// Enhanced stitcher
	enhanced := testcallgraph.NewEnhancedStitcher()
	// Manually populate for demo
	enhanced.TestToProgramMap["demo.txt"] = []testcallgraph.ProgramExecution{
		{Program: "mcpdiff", Line: 1, ExecutedBy: "exec"},
		{Program: "mcpdiff", Line: 2, ExecutedBy: "mcpdiff"},
		{Program: "mcp-spy", Line: 3, ExecutedBy: "mcp-spy"},
		{Program: "server", Line: 4, ExecutedBy: "mcp-server-start", IsServer: true},
		{Program: "echo", Line: 5, ExecutedBy: "exec"},
	}
	
	edges := enhanced.CreateCallGraphConnections("demo.txt")
	for _, edge := range edges {
		fmt.Printf("%s\n", edge)
	}
	
	fmt.Println("\nTo analyze a real file, use:")
	fmt.Println("  stitch-demo -test <file>")
	fmt.Println("  stitch-demo -test <file> -types  # Show command types")
	fmt.Println("  stitch-demo -test <file> -simple # Use simple stitcher")
}

func runSimpleStitcher(testFile string) {
	fmt.Printf("=== Simple Stitcher Analysis ===\n")
	fmt.Printf("File: %s\n\n", testFile)
	
	content, err := os.ReadFile(testFile)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	
	stitcher := &testcallgraph.SimpleStitcher{}
	connections := stitcher.AnalyzeAndStitch(testFile, string(content))
	
	if len(connections) == 0 {
		fmt.Println("No connections found (only 'exec' commands are detected)")
		return
	}
	
	fmt.Println("Connections found:")
	for _, conn := range connections {
		fmt.Printf("  Line %d: %s -> %s\n", conn.TestLine, conn.Program, conn.MainPath)
	}
	
	fmt.Printf("\nTotal: %d connections\n", len(connections))
}

func runEnhancedStitcher(testFile string, showTypes bool) {
	fmt.Printf("=== Enhanced Stitcher Analysis ===\n")
	fmt.Printf("File: %s\n\n", testFile)
	
	stitcher := testcallgraph.NewEnhancedStitcher()
	
	if err := stitcher.AnalyzeScriptTest(testFile); err != nil {
		log.Fatalf("Error analyzing file: %v", err)
	}
	
	executions := stitcher.TestToProgramMap[testFile]
	if len(executions) == 0 {
		fmt.Println("No program executions found")
		return
	}
	
	// Group by program
	programMap := make(map[string][]testcallgraph.ProgramExecution)
	for _, exec := range executions {
		programMap[exec.Program] = append(programMap[exec.Program], exec)
	}
	
	fmt.Printf("Found %d programs executed:\n\n", len(programMap))
	
	for prog, execs := range programMap {
		fmt.Printf("%s:\n", prog)
		for _, exec := range execs {
			if showTypes {
				serverStr := ""
				if exec.IsServer {
					serverStr = " [SERVER]"
				}
				fmt.Printf("  Line %d: %s (%s)%s\n", exec.Line, exec.Command, exec.ExecutedBy, serverStr)
			} else {
				fmt.Printf("  Line %d: %s\n", exec.Line, strings.TrimSpace(exec.Command))
			}
		}
		fmt.Println()
	}
	
	// Show call graph edges
	fmt.Println("Call Graph Edges:")
	fmt.Println("-----------------")
	edges := stitcher.CreateCallGraphConnections(testFile)
	for _, edge := range edges {
		fmt.Printf("%s\n", edge)
	}
	
	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("- %d total executions\n", len(executions))
	fmt.Printf("- %d unique programs\n", len(programMap))
	
	execCount := 0
	customCount := 0
	serverCount := 0
	
	for _, exec := range executions {
		if exec.ExecutedBy == "exec" {
			execCount++
		} else {
			customCount++
		}
		if exec.IsServer {
			serverCount++
		}
	}
	
	fmt.Printf("- %d exec commands\n", execCount)
	fmt.Printf("- %d custom commands\n", customCount)
	fmt.Printf("- %d server processes\n", serverCount)
}