// coverage-by-program analyzes test coverage per program based on dependency graph
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
	"sort"
	"strings"
)

var (
	depGraphFile  string
	coverageDir   string
	outputFile    string
	format        string
	showUncovered bool
	showStats     bool
	verbose       bool
)

func init() {
	flag.StringVar(&depGraphFile, "depgraph", "", "Dependency graph file (JSON)")
	flag.StringVar(&coverageDir, "coverage", "", "Coverage data directory")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "text", "Output format: text, json, markdown")
	flag.BoolVar(&showUncovered, "show-uncovered", false, "Show uncovered programs")
	flag.BoolVar(&showStats, "stats", false, "Show detailed statistics")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "coverage-by-program - Analyze test coverage per program\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  coverage-by-program [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Analyze coverage by program\n")
		fmt.Fprintf(os.Stderr, "  coverage-by-program -depgraph deps.json -coverage /tmp/coverage\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Show only uncovered programs\n")
		fmt.Fprintf(os.Stderr, "  coverage-by-program -depgraph deps.json -coverage /tmp/coverage -show-uncovered\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Output as markdown report\n")
		fmt.Fprintf(os.Stderr, "  coverage-by-program -depgraph deps.json -coverage /tmp/coverage -format markdown\n")
	}
}

type DepGraph struct {
	Nodes       map[string]DepNode     `json:"nodes"`
	Edges       []DepEdge              `json:"edges"`
	NodesByType map[string][]string    `json:"nodesByType"`
}

type DepNode struct {
	ID         string                 `json:"ID"`
	Type       string                 `json:"Type"`
	Properties map[string]interface{} `json:"Properties,omitempty"`
	InDegree   int                   `json:"InDegree"`
	OutDegree  int                   `json:"OutDegree"`
}

type DepEdge struct {
	From       string                 `json:"From"`
	To         string                 `json:"To"`
	Weight     int                   `json:"Weight"`
	Properties map[string]interface{} `json:"Properties,omitempty"`
}

type ProgramCoverage struct {
	Program        string
	Package        string
	CoveragePercent float64
	TotalLines     int
	CoveredLines   int
	Tests          []string
	Files          []FileCoverage
}

type FileCoverage struct {
	File           string
	CoveragePercent float64
	TotalLines     int
	CoveredLines   int
}

func main() {
	flag.Parse()
	
	if depGraphFile == "" || coverageDir == "" {
		flag.Usage()
		os.Exit(1)
	}
	
	// Load dependency graph
	depgraph, err := loadDepGraph(depGraphFile)
	if err != nil {
		log.Fatalf("Error loading dependency graph: %v", err)
	}
	
	// Analyze coverage by program
	coverage, err := analyzeCoverage(depgraph, coverageDir)
	if err != nil {
		log.Fatalf("Error analyzing coverage: %v", err)
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
	
	// Output results
	switch format {
	case "json":
		outputJSON(out, coverage)
	case "markdown":
		outputMarkdown(out, coverage)
	default:
		outputText(out, coverage)
	}
}

func loadDepGraph(filename string) (*DepGraph, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var depgraph DepGraph
	if err := json.Unmarshal(data, &depgraph); err != nil {
		return nil, err
	}
	
	return &depgraph, nil
}

func analyzeCoverage(depgraph *DepGraph, coverageDir string) ([]ProgramCoverage, error) {
	// Map programs to their tests
	programTests := make(map[string][]string)
	for _, edge := range depgraph.Edges {
		if depgraph.Nodes[edge.From].Type == "test" && depgraph.Nodes[edge.To].Type == "program" {
			programTests[edge.To] = append(programTests[edge.To], edge.From)
		}
	}
	
	// Get coverage data using go tool covdata
	coverageData, err := getCoverageData(coverageDir)
	if err != nil {
		return nil, err
	}
	
	// Analyze coverage for each program
	var results []ProgramCoverage
	for program, tests := range programTests {
		node := depgraph.Nodes[program]
		
		// Extract package from program metadata
		packageName := extractPackage(node.Properties)
		
		pc := ProgramCoverage{
			Program: program,
			Package: packageName,
			Tests:   tests,
		}
		
		// Get coverage for this package
		if pkgCoverage, ok := coverageData[packageName]; ok {
			pc.CoveragePercent = pkgCoverage.Percent
			pc.TotalLines = pkgCoverage.TotalLines
			pc.CoveredLines = pkgCoverage.CoveredLines
			pc.Files = pkgCoverage.Files
		}
		
		results = append(results, pc)
	}
	
	// Add uncovered programs if requested
	if showUncovered {
		for _, program := range depgraph.NodesByType["program"] {
			if _, hasCoverage := programTests[program]; !hasCoverage {
				node := depgraph.Nodes[program]
				packageName := extractPackage(node.Properties)
				
				pc := ProgramCoverage{
					Program:        program,
					Package:        packageName,
					CoveragePercent: 0,
					Tests:          []string{},
				}
				
				results = append(results, pc)
			}
		}
	}
	
	// Sort by coverage percentage
	sort.Slice(results, func(i, j int) bool {
		if results[i].CoveragePercent == results[j].CoveragePercent {
			return results[i].Program < results[j].Program
		}
		return results[i].CoveragePercent > results[j].CoveragePercent
	})
	
	return results, nil
}

func extractPackage(properties map[string]interface{}) string {
	if mainPath, ok := properties["mainPath"].(string); ok {
		// Extract from "cmd/mcpdiff/main.go:main"
		parts := strings.Split(mainPath, "/")
		if len(parts) > 2 {
			return strings.Join(parts[:len(parts)-1], "/")
		}
	}
	return ""
}

type PackageCoverage struct {
	Package      string
	Percent      float64
	TotalLines   int
	CoveredLines int
	Files        []FileCoverage
}

func getCoverageData(coverageDir string) (map[string]PackageCoverage, error) {
	// Use go tool covdata percent to get package coverage
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i", coverageDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage data: %v", err)
	}
	
	coverage := make(map[string]PackageCoverage)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Parse line like: "github.com/tmc/mcp/cmd/mcpdiff coverage: 82.3% of statements"
		parts := strings.Fields(line)
		if len(parts) >= 4 && parts[1] == "coverage:" {
			pkg := parts[0]
			var percent float64
			fmt.Sscanf(parts[2], "%f%%", &percent)
			
			coverage[pkg] = PackageCoverage{
				Package: pkg,
				Percent: percent,
			}
		}
	}
	
	// Get detailed file coverage
	for pkg := range coverage {
		fileCoverage, err := getFileCoverage(coverageDir, pkg)
		if err == nil {
			pc := coverage[pkg]
			pc.Files = fileCoverage
			coverage[pkg] = pc
		}
	}
	
	return coverage, nil
}

func getFileCoverage(coverageDir, pkg string) ([]FileCoverage, error) {
	// Use go tool covdata textfmt to get detailed coverage
	cmd := exec.Command("go", "tool", "covdata", "textfmt", "-i", coverageDir, "-pkg", pkg)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Parse the profile format
	var files []FileCoverage
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			file := parts[0]
			// Extract file path
			if idx := strings.Index(file, ":"); idx > 0 {
				file = file[:idx]
			}
			
			// Group by file (simplified)
			found := false
			for i, fc := range files {
				if fc.File == file {
					files[i].TotalLines++
					if parts[2] != "0" {
						files[i].CoveredLines++
					}
					found = true
					break
				}
			}
			
			if !found {
				fc := FileCoverage{File: file, TotalLines: 1}
				if parts[2] != "0" {
					fc.CoveredLines = 1
				}
				files = append(files, fc)
			}
		}
	}
	
	// Calculate percentages
	for i := range files {
		if files[i].TotalLines > 0 {
			files[i].CoveragePercent = float64(files[i].CoveredLines) / float64(files[i].TotalLines) * 100
		}
	}
	
	return files, nil
}

func outputText(w io.Writer, coverage []ProgramCoverage) {
	fmt.Fprintf(w, "Coverage by Program\n")
	fmt.Fprintf(w, "==================\n\n")
	
	for _, pc := range coverage {
		if !showUncovered && pc.CoveragePercent == 0 && len(pc.Tests) == 0 {
			continue
		}
		
		fmt.Fprintf(w, "Program: %s\n", pc.Program)
		fmt.Fprintf(w, "Package: %s\n", pc.Package)
		fmt.Fprintf(w, "Coverage: %.1f%% (%d/%d lines)\n", pc.CoveragePercent, pc.CoveredLines, pc.TotalLines)
		fmt.Fprintf(w, "Tests: %d\n", len(pc.Tests))
		
		if verbose && len(pc.Tests) > 0 {
			fmt.Fprintf(w, "  Tests:\n")
			for _, test := range pc.Tests {
				fmt.Fprintf(w, "    - %s\n", test)
			}
		}
		
		if showStats && len(pc.Files) > 0 {
			fmt.Fprintf(w, "  Files:\n")
			for _, file := range pc.Files {
				fmt.Fprintf(w, "    - %s: %.1f%% (%d/%d)\n", 
					filepath.Base(file.File), 
					file.CoveragePercent,
					file.CoveredLines,
					file.TotalLines)
			}
		}
		
		fmt.Fprintf(w, "\n")
	}
}

func outputMarkdown(w io.Writer, coverage []ProgramCoverage) {
	fmt.Fprintf(w, "# Coverage by Program\n\n")
	
	// Summary table
	fmt.Fprintf(w, "| Program | Package | Coverage | Tests |\n")
	fmt.Fprintf(w, "|---------|---------|----------|-------|\n")
	
	for _, pc := range coverage {
		if !showUncovered && pc.CoveragePercent == 0 && len(pc.Tests) == 0 {
			continue
		}
		
		fmt.Fprintf(w, "| %s | %s | %.1f%% | %d |\n",
			pc.Program,
			pc.Package,
			pc.CoveragePercent,
			len(pc.Tests))
	}
	
	fmt.Fprintf(w, "\n## Details\n\n")
	
	for _, pc := range coverage {
		if !showUncovered && pc.CoveragePercent == 0 && len(pc.Tests) == 0 {
			continue
		}
		
		fmt.Fprintf(w, "### %s\n\n", pc.Program)
		fmt.Fprintf(w, "- **Package**: `%s`\n", pc.Package)
		fmt.Fprintf(w, "- **Coverage**: %.1f%% (%d/%d lines)\n", pc.CoveragePercent, pc.CoveredLines, pc.TotalLines)
		fmt.Fprintf(w, "- **Test Count**: %d\n", len(pc.Tests))
		
		if verbose && len(pc.Tests) > 0 {
			fmt.Fprintf(w, "\n**Tests:**\n")
			for _, test := range pc.Tests {
				fmt.Fprintf(w, "- %s\n", test)
			}
		}
		
		if showStats && len(pc.Files) > 0 {
			fmt.Fprintf(w, "\n**File Coverage:**\n")
			for _, file := range pc.Files {
				fmt.Fprintf(w, "- `%s`: %.1f%% (%d/%d)\n", 
					filepath.Base(file.File), 
					file.CoveragePercent,
					file.CoveredLines,
					file.TotalLines)
			}
		}
		
		fmt.Fprintf(w, "\n")
	}
}

func outputJSON(w io.Writer, coverage []ProgramCoverage) {
	// Calculate summary statistics
	total := len(coverage)
	covered := 0
	avgCoverage := 0.0
	
	for _, pc := range coverage {
		if pc.CoveragePercent > 0 {
			covered++
		}
		avgCoverage += pc.CoveragePercent
	}
	
	if total > 0 {
		avgCoverage /= float64(total)
	}
	
	data := map[string]interface{}{
		"programs": coverage,
		"summary": map[string]interface{}{
			"totalPrograms":    total,
			"coveredPrograms":  covered,
			"averageCoverage":  avgCoverage,
			"uncoveredPrograms": total - covered,
		},
	}
	
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}