// Demo shows how testcallgraph enhances mcpscripttest with call graph analysis
package main

import (
	"fmt"
	"log"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func main() {
	// Example 1: Basic call graph analysis
	fmt.Println("=== Example 1: Call Graph Analysis ===")
	analyzeTestCallGraph()
	fmt.Println()

	// Example 2: Proximity analysis for uncovered code
	fmt.Println("=== Example 2: Proximity Analysis ===")
	findClosestTest()
	fmt.Println()

	// Example 3: Test optimization suggestions
	fmt.Println("=== Example 3: Test Optimization ===")
	suggestTestModifications()
}

func analyzeTestCallGraph() {
	// Build call graph for test file
	graph, err := testcallgraph.Build(
		"testdata/coverage_test.txt",
		"./cmd/mcpdiff",
		"./cmd/mcpcat",
	)
	if err != nil {
		log.Fatal(err)
	}

	// Trace a specific test line
	test := testcallgraph.TestLocation{
		File: "testdata/coverage_test.txt",
		Line: 5,
		Cmd:  "exec mcpdiff file1.mcp file2.mcp",
	}

	trace, err := graph.TraceExecution(test)
	if err != nil {
		log.Fatal(err)
	}

	// Display results
	fmt.Printf("Test: %s\n", test.Cmd)
	fmt.Printf("Execution time: %v\n", trace.Duration)
	fmt.Printf("Functions called: %d\n", len(trace.Calls))
	fmt.Println("\nCall sequence:")
	for i, call := range trace.Calls {
		fmt.Printf("%d. %s.%s -> %s.%s\n",
			i+1,
			call.Caller.Package, call.Caller.Function,
			call.Callee.Package, call.Callee.Function)
		if i >= 5 {
			fmt.Println("   ... (truncated)")
			break
		}
	}
}

func findClosestTest() {
	// Target uncovered code
	target := testcallgraph.SourceLocation{
		Package:  "github.com/tmc/mcp/internal/parser",
		Function: "handleParseError", 
		File:     "parser.go",
		Line:     89,
	}

	// Build graph and analyze
	graph, _ := testcallgraph.Build(
		"testdata/coverage_test.txt",
		"./cmd/mcpdiff",
		"./cmd/mcpcat",
	)

	// Find closest test
	analyzer := &testcallgraph.ProximityAnalyzer{Graph: graph}
	result, err := analyzer.FindClosestTest(target)
	if err != nil {
		log.Fatal(err)
	}

	// Display results
	fmt.Printf("Target: %s:%d (%s)\n", target.File, target.Line, target.Function)
	fmt.Printf("Closest test: %s:%d\n", result.Test.File, result.Test.Line)
	fmt.Printf("Distance: %d function calls\n", result.Distance)
	fmt.Printf("Test command: %s\n", result.Test.Cmd)
	
	fmt.Println("\nPath to target:")
	for i, loc := range result.Path {
		fmt.Printf("  %d. %s.%s (line %d)\n", 
			i+1, loc.Package, loc.Function, loc.Line)
	}
}

func suggestTestModifications() {
	// Example modification suggestion
	current := testcallgraph.TestLocation{
		File: "testdata/coverage_test.txt",
		Line: 5,
		Cmd:  "exec mcpdiff valid.mcp valid2.mcp",
	}

	target := testcallgraph.SourceLocation{
		Package:  "github.com/tmc/mcp/internal/parser",
		Function: "handleParseError",
		File:     "parser.go", 
		Line:     89,
	}

	fmt.Printf("Current test: %s\n", current.Cmd)
	fmt.Printf("Target: %s:%d\n", target.File, target.Line)
	fmt.Println("\nSuggested modifications:")
	
	// Modification 1: Invalid JSON
	fmt.Println("1. Replace valid.mcp with invalid JSON:")
	fmt.Println("   exec mcpdiff invalid.mcp valid2.mcp")
	fmt.Println("   stderr 'parse error'")
	fmt.Println("   -- invalid.mcp --")
	fmt.Println("   {\"unclosed\": \"quote}")
	fmt.Println("   Why: Triggers JSON parse error, calls handleParseError")
	fmt.Println()

	// Modification 2: Binary file
	fmt.Println("2. Use binary file to trigger different error:")
	fmt.Println("   exec mcpdiff binary.dat valid2.mcp")
	fmt.Println("   stderr 'invalid format'")
	fmt.Println("   -- binary.dat --")
	fmt.Println("   \\x00\\x01\\x02\\x03")
	fmt.Println("   Why: Binary content triggers format error")
	fmt.Println()

	// Modification 3: Empty file
	fmt.Println("3. Use empty file:")
	fmt.Println("   exec mcpdiff empty.mcp valid2.mcp")
	fmt.Println("   stderr 'unexpected EOF'")
	fmt.Println("   -- empty.mcp --")
	fmt.Println("   ")
	fmt.Println("   Why: Empty file triggers EOF error")
}

// Example visualization
func generateVisualization() {
	graph, _ := testcallgraph.Build(
		"testdata/coverage_test.txt",
		"./cmd/mcpdiff",
	)

	viz := &testcallgraph.Visualizer{Graph: graph}
	dot := viz.GenerateDOT()
	
	fmt.Println("Generated DOT visualization:")
	fmt.Println(dot)
	fmt.Println("\nTo render: echo '<dot>' | dot -Tsvg > callgraph.svg")
}