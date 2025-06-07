// testcallgraph-coverage combines testcallgraph analysis with coverage data
package main

import (
	"bufio"
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

	"github.com/tmc/mcp/exp/callgraph/testcallgraph"
)

var (
	testFile     string
	coverageDir  string
	outputFile   string
	format       string
	verbose      bool
	differential bool
	baseline     string
	threshold    float64
)

func init() {
	flag.StringVar(&testFile, "test", "", "Test file or directory to analyze")
	flag.StringVar(&coverageDir, "coverage", "", "Coverage data directory")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "text", "Output format: text, json")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&differential, "diff", false, "Show differential coverage")
	flag.StringVar(&baseline, "baseline", "", "Baseline coverage for differential analysis")
	flag.Float64Var(&threshold, "threshold", 0.0, "Only show tests with coverage increase above threshold")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "testcallgraph-coverage - Combine testcallgraph with coverage analysis\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph-coverage [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Analyze test with coverage\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph-coverage -test test.txt -coverage /tmp/cov\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Show differential coverage\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph-coverage -diff -baseline base.cov -coverage new.cov\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Find tests with significant coverage increase\n")
		fmt.Fprintf(os.Stderr, "  testcallgraph-coverage -diff -threshold 5.0\n")
	}
}

type CoverageData struct {
	Package  string  `json:"package"`
	Coverage float64 `json:"coverage"`
	Lines    int     `json:"lines"`
	Covered  int     `json:"covered"`
}

type TestCoverageResult struct {
	TestFile      string                        `json:"test_file"`
	Programs      []string                      `json:"programs"`
	Edges         []testcallgraph.CallGraphEdge `json:"edges"`
	Coverage      map[string]CoverageData       `json:"coverage"`
	TotalCoverage float64                       `json:"total_coverage"`
	Differential  *DifferentialCoverage         `json:"differential,omitempty"`
}

type DifferentialCoverage struct {
	Baseline     float64            `json:"baseline"`
	Current      float64            `json:"current"`
	Increase     float64            `json:"increase"`
	NewPackages  []string           `json:"new_packages"`
	ImprovedPkgs map[string]float64 `json:"improved_packages"`
}

func main() {
	flag.Parse()

	if testFile == "" {
		flag.Usage()
		os.Exit(1)
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

	// Analyze test files
	files, err := getTestFiles(testFile)
	if err != nil {
		log.Fatalf("Error finding test files: %v", err)
	}

	stitcher := testcallgraph.NewEnhancedStitcher()
	results := make([]*TestCoverageResult, 0)

	for _, file := range files {
		if verbose {
			fmt.Fprintf(os.Stderr, "Analyzing %s...\n", file)
		}

		result, err := analyzeTestWithCoverage(stitcher, file)
		if err != nil {
			log.Printf("Error analyzing %s: %v", file, err)
			continue
		}

		// Apply threshold filter
		if differential && threshold > 0 {
			if result.Differential == nil || result.Differential.Increase < threshold {
				continue
			}
		}

		results = append(results, result)
	}

	// Output results
	switch format {
	case "json":
		outputJSON(out, results)
	default:
		outputText(out, results)
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

func analyzeTestWithCoverage(stitcher *testcallgraph.EnhancedStitcher, file string) (*TestCoverageResult, error) {
	// First, get the testcallgraph analysis
	if err := stitcher.AnalyzeScriptTest(file); err != nil {
		return nil, err
	}

	executions := stitcher.TestToProgramMap[file]
	edges := stitcher.CreateCallGraphConnections(file)

	// Extract unique programs
	programMap := make(map[string]bool)
	for _, exec := range executions {
		programMap[exec.Program] = true
	}

	programs := make([]string, 0, len(programMap))
	for prog := range programMap {
		programs = append(programs, prog)
	}
	sort.Strings(programs)

	result := &TestCoverageResult{
		TestFile: file,
		Programs: programs,
		Edges:    edges,
		Coverage: make(map[string]CoverageData),
	}

	// Get coverage data if available
	if coverageDir != "" {
		coverage, err := getCoverageData(coverageDir)
		if err != nil {
			return nil, fmt.Errorf("error getting coverage: %v", err)
		}
		result.Coverage = coverage
		result.TotalCoverage = calculateTotalCoverage(coverage)

		// Get differential coverage if requested
		if differential && baseline != "" {
			diff, err := getDifferentialCoverage(baseline, coverageDir)
			if err != nil {
				return nil, fmt.Errorf("error getting differential coverage: %v", err)
			}
			result.Differential = diff
		}
	}

	return result, nil
}

func getCoverageData(coverDir string) (map[string]CoverageData, error) {
	// Run go tool covdata to get coverage percentages
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i", coverDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	coverage := make(map[string]CoverageData)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.Contains(line, "\t") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pkg := parts[0]
		percentStr := strings.TrimSuffix(parts[1], "%")
		percent := 0.0
		fmt.Sscanf(percentStr, "%f", &percent)

		coverage[pkg] = CoverageData{
			Package:  pkg,
			Coverage: percent,
		}
	}

	return coverage, scanner.Err()
}

func calculateTotalCoverage(coverage map[string]CoverageData) float64 {
	if len(coverage) == 0 {
		return 0.0
	}

	total := 0.0
	for _, data := range coverage {
		total += data.Coverage
	}

	return total / float64(len(coverage))
}

func getDifferentialCoverage(baselineDir, currentDir string) (*DifferentialCoverage, error) {
	baselineCov, err := getCoverageData(baselineDir)
	if err != nil {
		return nil, err
	}

	currentCov, err := getCoverageData(currentDir)
	if err != nil {
		return nil, err
	}

	diff := &DifferentialCoverage{
		Baseline:     calculateTotalCoverage(baselineCov),
		Current:      calculateTotalCoverage(currentCov),
		NewPackages:  make([]string, 0),
		ImprovedPkgs: make(map[string]float64),
	}

	diff.Increase = diff.Current - diff.Baseline

	// Find new packages and improvements
	for pkg, current := range currentCov {
		if baseline, exists := baselineCov[pkg]; exists {
			improvement := current.Coverage - baseline.Coverage
			if improvement > 0 {
				diff.ImprovedPkgs[pkg] = improvement
			}
		} else {
			diff.NewPackages = append(diff.NewPackages, pkg)
		}
	}

	return diff, nil
}

func outputText(w io.Writer, results []*TestCoverageResult) {
	for _, result := range results {
		fmt.Fprintf(w, "=== %s ===\n", result.TestFile)

		fmt.Fprintf(w, "\nPrograms executed: %s\n", strings.Join(result.Programs, ", "))

		if len(result.Coverage) > 0 {
			fmt.Fprintf(w, "\nCoverage by package:\n")

			// Sort packages by coverage
			type pkgCov struct {
				pkg string
				cov float64
			}
			var sorted []pkgCov
			for pkg, data := range result.Coverage {
				sorted = append(sorted, pkgCov{pkg, data.Coverage})
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].cov > sorted[j].cov
			})

			for _, pc := range sorted {
				fmt.Fprintf(w, "  %s: %.1f%%\n", pc.pkg, pc.cov)
			}

			fmt.Fprintf(w, "\nTotal coverage: %.1f%%\n", result.TotalCoverage)
		}

		if result.Differential != nil {
			fmt.Fprintf(w, "\nDifferential coverage:\n")
			fmt.Fprintf(w, "  Baseline: %.1f%%\n", result.Differential.Baseline)
			fmt.Fprintf(w, "  Current:  %.1f%%\n", result.Differential.Current)
			fmt.Fprintf(w, "  Increase: %+.1f%%\n", result.Differential.Increase)

			if len(result.Differential.NewPackages) > 0 {
				fmt.Fprintf(w, "  New packages: %s\n", strings.Join(result.Differential.NewPackages, ", "))
			}

			if len(result.Differential.ImprovedPkgs) > 0 {
				fmt.Fprintf(w, "  Improved packages:\n")
				for pkg, improvement := range result.Differential.ImprovedPkgs {
					fmt.Fprintf(w, "    %s: %+.1f%%\n", pkg, improvement)
				}
			}
		}

		fmt.Fprintf(w, "\nCall graph connections: %d\n", len(result.Edges))
		fmt.Fprintf(w, "\n")
	}
}

func outputJSON(w io.Writer, results []*TestCoverageResult) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(results)
}
