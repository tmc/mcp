// testcallgraph analyzes test scripts to create call graph edges between tests and programs
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmc/mcp/exp/callgraph/testcallgraph"
)

var (
	testFile   string
	outputFile string
	format     string
	verbose    bool
	analyze    string
	listCmds   bool
	showStats  bool
	bashMode   bool
)

func init() {
	flag.StringVar(&testFile, "test", "", "Test file or directory to analyze")
	flag.StringVar(&testFile, "t", "", "Test file or directory to analyze (shorthand)")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&outputFile, "o", "", "Output file (shorthand)")
	flag.StringVar(&format, "format", "text", "Output format: text, json, dot")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&verbose, "v", false, "Verbose output (shorthand)")
	flag.StringVar(&analyze, "analyze", "", "Analyze specific program connections")
	flag.BoolVar(&listCmds, "list-commands", false, "List all custom commands found")
	flag.BoolVar(&showStats, "stats", false, "Show statistics")
	flag.BoolVar(&bashMode, "bash", false, "Enable bash script analysis and coverage")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "testcallgraph - Analyze test scripts to create call graph edges\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph [options] <test-file-or-dir>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph test.txt                    # Analyze single file\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph -format json test.txt       # Output as JSON\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph -analyze mcpdiff tests/     # Find all mcpdiff calls\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph -stats tests/               # Show statistics\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph -format dot -o graph.dot    # Generate Graphviz\n")
	}
}

type Result struct {
	TestFile     string                           `json:"test_file"`
	Executions   []testcallgraph.ProgramExecution `json:"executions"`
	Edges        []testcallgraph.CallGraphEdge    `json:"edges"`
	Programs     map[string]int                   `json:"programs"`      // program -> count
	CommandTypes map[string]int                   `json:"command_types"` // command type -> count
}

func main() {
	flag.Parse()

	// If no flags and no args, show help
	if testFile == "" && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// Get test file from args if not specified via flag
	if testFile == "" && flag.NArg() > 0 {
		testFile = flag.Arg(0)
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

	// Process files
	files, err := getTestFiles(testFile)
	if err != nil {
		log.Fatalf("Error finding test files: %v", err)
	}

	if len(files) == 0 {
		log.Fatal("No test files found")
	}

	// Choose the appropriate stitcher based on bash mode
	var stitcher interface{}
	if bashMode {
		stitcher = testcallgraph.NewBashStitcher()
	} else {
		stitcher = testcallgraph.NewEnhancedStitcher()
	}
	results := make([]*Result, 0)

	for _, file := range files {
		if verbose {
			fmt.Fprintf(os.Stderr, "Analyzing %s...\n", file)
		}

		result, err := analyzeFile(stitcher, file)
		if err != nil {
			log.Printf("Error analyzing %s: %v", file, err)
			continue
		}

		results = append(results, result)
	}

	// Output results
	switch format {
	case "json":
		outputJSON(out, results)
	case "dot":
		outputDot(out, results)
	default:
		outputText(out, results)
	}

	// Show statistics if requested
	if showStats {
		printStats(results)
	}

	// Show bash coverage report if in bash mode
	if bashMode {
		if bs, ok := stitcher.(*testcallgraph.BashStitcher); ok {
			fmt.Println(bs.GetBashCoverageReport())
		}
	}
}

func getTestFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	// Find all .txt files in directory
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

func analyzeFile(stitcher interface{}, file string) (*Result, error) {
	// Handle different stitcher types
	switch s := stitcher.(type) {
	case *testcallgraph.EnhancedStitcher:
		if err := s.AnalyzeScriptTest(file); err != nil {
			return nil, err
		}

		executions := s.TestToProgramMap[file]
		edges := s.CreateCallGraphConnections(file)

		result := &Result{
			TestFile:     file,
			Executions:   executions,
			Edges:        edges,
			Programs:     make(map[string]int),
			CommandTypes: make(map[string]int),
		}

		// Count programs and command types
		for _, exec := range executions {
			result.Programs[exec.Program]++
			result.CommandTypes[exec.ExecutedBy]++
		}

		return result, nil

	case *testcallgraph.BashStitcher:
		if err := s.AnalyzeScriptTest(file); err != nil {
			return nil, err
		}

		// Get both regular and bash executions
		executions := s.TestToProgramMap[file]
		edges := s.CreateBashCallGraph(file)

		result := &Result{
			TestFile:     file,
			Executions:   executions,
			Edges:        edges,
			Programs:     make(map[string]int),
			CommandTypes: make(map[string]int),
		}

		// Count programs and command types
		for _, exec := range executions {
			result.Programs[exec.Program]++
			result.CommandTypes[exec.ExecutedBy]++
		}

		// Also count bash scripts
		for _, bashExec := range s.BashScriptMap[file] {
			result.Programs[bashExec.ScriptPath]++
			result.CommandTypes["bash:"+bashExec.ExecutedBy]++
		}

		return result, nil

	default:
		return nil, fmt.Errorf("unknown stitcher type: %T", stitcher)
	}
}

func outputText(w io.Writer, results []*Result) {
	if analyze != "" {
		outputAnalysis(w, results, analyze)
		return
	}

	if listCmds {
		outputCommandList(w, results)
		return
	}

	for _, result := range results {
		fmt.Fprintf(w, "=== %s ===\n", result.TestFile)

		if len(result.Executions) == 0 {
			fmt.Fprintf(w, "No program executions found\n\n")
			continue
		}

		fmt.Fprintf(w, "\nProgram Executions:\n")
		for _, exec := range result.Executions {
			serverStr := ""
			if exec.IsServer {
				serverStr = " [SERVER]"
			}
			fmt.Fprintf(w, "  Line %d: %s -> %s (%s)%s\n",
				exec.Line, exec.Command, exec.Program, exec.ExecutedBy, serverStr)
		}

		fmt.Fprintf(w, "\nCall Graph Edges:\n")
		for _, edge := range result.Edges {
			fmt.Fprintf(w, "  %s\n", edge)
		}

		fmt.Fprintf(w, "\n")
	}
}

func outputJSON(w io.Writer, results []*Result) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(results)
}

func outputDot(w io.Writer, results []*Result) {
	fmt.Fprintf(w, "digraph testcallgraph {\n")
	fmt.Fprintf(w, "  rankdir=LR;\n")
	fmt.Fprintf(w, "  node [shape=box];\n\n")

	// Collect all unique nodes
	testNodes := make(map[string]bool)
	programNodes := make(map[string]bool)

	for _, result := range results {
		testNodes[filepath.Base(result.TestFile)] = true
		for _, edge := range result.Edges {
			programNodes[edge.To] = true
		}
	}

	// Define test nodes
	fmt.Fprintf(w, "  // Test files\n")
	for node := range testNodes {
		fmt.Fprintf(w, "  \"%s\" [style=filled,fillcolor=lightblue];\n", node)
	}

	// Define program nodes
	fmt.Fprintf(w, "\n  // Programs\n")
	for node := range programNodes {
		fmt.Fprintf(w, "  \"%s\" [style=filled,fillcolor=lightgreen];\n", node)
	}

	// Add edges
	fmt.Fprintf(w, "\n  // Edges\n")
	for _, result := range results {
		testNode := filepath.Base(result.TestFile)
		for _, edge := range result.Edges {
			label := edge.EdgeType
			if edge.IsServer {
				label += " [SERVER]"
			}
			fmt.Fprintf(w, "  \"%s\" -> \"%s\" [label=\"%s\"];\n",
				testNode, edge.To, label)
		}
	}

	fmt.Fprintf(w, "}\n")
}

func outputAnalysis(w io.Writer, results []*Result, program string) {
	fmt.Fprintf(w, "=== Analysis: %s ===\n\n", program)

	found := false
	for _, result := range results {
		var matches []testcallgraph.ProgramExecution
		for _, exec := range result.Executions {
			if exec.Program == program {
				matches = append(matches, exec)
			}
		}

		if len(matches) > 0 {
			found = true
			fmt.Fprintf(w, "%s:\n", result.TestFile)
			for _, exec := range matches {
				fmt.Fprintf(w, "  Line %d: %s (%s)\n",
					exec.Line, exec.Command, exec.ExecutedBy)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	if !found {
		fmt.Fprintf(w, "No occurrences of %s found\n", program)
	}
}

func outputCommandList(w io.Writer, results []*Result) {
	commands := make(map[string]int)

	for _, result := range results {
		for _, exec := range result.Executions {
			commands[exec.ExecutedBy]++
		}
	}

	fmt.Fprintf(w, "=== Custom Commands Found ===\n\n")

	// Sort by frequency
	type cmdCount struct {
		cmd   string
		count int
	}
	var sorted []cmdCount
	for cmd, count := range commands {
		sorted = append(sorted, cmdCount{cmd, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	for _, cc := range sorted {
		fmt.Fprintf(w, "%s: %d occurrences\n", cc.cmd, cc.count)
	}
}

func printStats(results []*Result) {
	totalFiles := len(results)
	totalExecs := 0
	programs := make(map[string]int)
	commands := make(map[string]int)
	serverCount := 0

	for _, result := range results {
		totalExecs += len(result.Executions)
		for prog, count := range result.Programs {
			programs[prog] += count
		}
		for cmd, count := range result.CommandTypes {
			commands[cmd] += count
		}
		for _, exec := range result.Executions {
			if exec.IsServer {
				serverCount++
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n=== Statistics ===\n")
	fmt.Fprintf(os.Stderr, "Files analyzed: %d\n", totalFiles)
	fmt.Fprintf(os.Stderr, "Total executions: %d\n", totalExecs)
	fmt.Fprintf(os.Stderr, "Unique programs: %d\n", len(programs))
	fmt.Fprintf(os.Stderr, "Server processes: %d\n", serverCount)
	fmt.Fprintf(os.Stderr, "\nTop programs:\n")

	// Sort programs by frequency
	type progCount struct {
		prog  string
		count int
	}
	var sortedProgs []progCount
	for prog, count := range programs {
		sortedProgs = append(sortedProgs, progCount{prog, count})
	}
	sort.Slice(sortedProgs, func(i, j int) bool {
		return sortedProgs[i].count > sortedProgs[j].count
	})

	for i := 0; i < 5 && i < len(sortedProgs); i++ {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", sortedProgs[i].prog, sortedProgs[i].count)
	}
}
