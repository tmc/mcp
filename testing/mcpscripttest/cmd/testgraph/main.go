// testgraph combines testcallgraph with digraph for advanced graph analysis
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/testing/mcpscripttest/testcallgraph"
)

var (
	testFile   string
	outputFile string
	format     string
	query      string
	fromTests  bool
	toPrograms bool
	verbose    bool
)

func init() {
	flag.StringVar(&testFile, "test", "", "Test file or directory to analyze")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "digraph", "Output format: digraph, dot, json")
	flag.StringVar(&query, "query", "", "Digraph query (e.g., 'somepath', 'allpaths')")
	flag.BoolVar(&fromTests, "from-tests", false, "Start paths from test files")
	flag.BoolVar(&toPrograms, "to-programs", false, "End paths at programs")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "testgraph - Combine testcallgraph with digraph analysis\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  testgraph [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate digraph format\n")
		fmt.Fprintf(os.Stderr, "  testgraph -test tests/ > graph.digraph\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Find all paths from tests to a program\n")
		fmt.Fprintf(os.Stderr, "  testgraph -test tests/ | digraph allpaths test.txt mcpdiff\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Find programs not reached by any test\n")
		fmt.Fprintf(os.Stderr, "  testgraph -test tests/ | digraph sources | grep -v test.txt\n")
	}
}

type Graph struct {
	Nodes map[string]*Node
	Edges []Edge
}

type Node struct {
	ID       string
	Type     string // "test", "program", "function"
	Path     string
	Metadata map[string]interface{}
}

type Edge struct {
	From     string
	To       string
	Type     string
	Metadata map[string]interface{}
}

func main() {
	flag.Parse()

	if testFile == "" && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if testFile == "" && flag.NArg() > 0 {
		testFile = flag.Arg(0)
	}

	// Build the graph
	graph, err := buildGraph(testFile)
	if err != nil {
		log.Fatalf("Error building graph: %v", err)
	}

	// Setup output
	var out io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	// Output the graph
	switch format {
	case "json":
		outputJSON(out, graph)
	case "dot":
		outputDot(out, graph)
	default:
		outputDigraph(out, graph)
	}

	// If a query is specified, run it
	if query != "" {
		runQuery(graph, query)
	}
}

func buildGraph(testPath string) (*Graph, error) {
	graph := &Graph{
		Nodes: make(map[string]*Node),
		Edges: []Edge{},
	}

	// Get all test files
	files, err := getTestFiles(testPath)
	if err != nil {
		return nil, err
	}

	stitcher := testcallgraph.NewEnhancedStitcher()

	for _, file := range files {
		if verbose {
			log.Printf("Analyzing %s...", file)
		}

		// Add test file node
		testID := filepath.Base(file)
		graph.Nodes[testID] = &Node{
			ID:   testID,
			Type: "test",
			Path: file,
			Metadata: map[string]interface{}{
				"fullPath": file,
			},
		}

		// Analyze the test
		if err := stitcher.AnalyzeScriptTest(file); err != nil {
			log.Printf("Error analyzing %s: %v", file, err)
			continue
		}

		// Get connections
		edges := stitcher.CreateCallGraphConnections(file)

		for _, edge := range edges {
			// Extract program name from the To field
			programID := extractProgramID(edge.To)

			// Add program node if not exists
			if _, exists := graph.Nodes[programID]; !exists {
				graph.Nodes[programID] = &Node{
					ID:   programID,
					Type: "program",
					Path: edge.To,
					Metadata: map[string]interface{}{
						"mainPath": edge.To,
					},
				}
			}

			// Add edge
			graph.Edges = append(graph.Edges, Edge{
				From: testID,
				To:   programID,
				Type: edge.EdgeType,
				Metadata: map[string]interface{}{
					"line":     extractLine(edge.From),
					"isServer": edge.IsServer,
				},
			})
		}
	}

	return graph, nil
}

func getTestFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(p, ".txt") {
			files = append(files, p)
		}
		return nil
	})

	return files, err
}

func extractProgramID(to string) string {
	// Extract from "cmd/mcpdiff/main.go:main"
	parts := strings.Split(to, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return to
}

func extractLine(from string) int {
	// Extract from "test.txt:42"
	parts := strings.Split(from, ":")
	if len(parts) > 1 {
		var line int
		fmt.Sscanf(parts[1], "%d", &line)
		return line
	}
	return 0
}

func outputDigraph(w io.Writer, graph *Graph) {
	// Output in digraph format (simple adjacency list)
	for _, edge := range graph.Edges {
		fmt.Fprintf(w, "%s %s\n", edge.From, edge.To)
	}
}

func outputDot(w io.Writer, graph *Graph) {
	fmt.Fprintf(w, "digraph testgraph {\n")
	fmt.Fprintf(w, "  rankdir=LR;\n")
	fmt.Fprintf(w, "  node [shape=box];\n\n")

	// Define nodes
	for id, node := range graph.Nodes {
		style := ""
		if node.Type == "test" {
			style = "style=filled,fillcolor=lightblue"
		} else if node.Type == "program" {
			style = "style=filled,fillcolor=lightgreen"
		}
		fmt.Fprintf(w, "  \"%s\" [%s];\n", id, style)
	}

	fmt.Fprintf(w, "\n")

	// Define edges
	for _, edge := range graph.Edges {
		label := edge.Type
		if isServer, ok := edge.Metadata["isServer"].(bool); ok && isServer {
			label += " [SERVER]"
		}
		fmt.Fprintf(w, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", edge.From, edge.To, label)
	}

	fmt.Fprintf(w, "}\n")
}

func outputJSON(w io.Writer, graph *Graph) {
	data := map[string]interface{}{
		"nodes": graph.Nodes,
		"edges": graph.Edges,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}

func runQuery(graph *Graph, query string) {
	// Create a temporary file with the graph in digraph format
	tmpfile, err := os.CreateTemp("", "graph-*.digraph")
	if err != nil {
		log.Fatalf("Error creating temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write graph to temp file
	outputDigraph(tmpfile, graph)
	tmpfile.Close()

	// Run digraph command
	cmdParts := strings.Fields(query)
	if len(cmdParts) == 0 {
		return
	}

	allArgs := append([]string{cmdParts[0]}, cmdParts[1:]...)
	cmd := exec.Command("digraph", allArgs...)
	cmd.Stdin, err = os.Open(tmpfile.Name())
	if err != nil {
		log.Fatalf("Error opening temp file: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error running digraph: %v\n%s", err, output)
	}

	fmt.Print(string(output))
}
