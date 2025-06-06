// Package main provides a tool to run individual Go tests with separate coverage reports
// and analyze how much coverage each test contributes, with support for Codecov JSON format.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TestResult holds the result of a single test run with coverage
type TestResult struct {
	TestName       string
	Package        string
	Passed         bool
	Duration       time.Duration
	CoverageDir    string
	CoverageReport map[string]float64       // package -> coverage percentage
	CoverageData   map[string][]interface{} // Codecov format: file -> coverage array
	Error          error
}

// CoverageAnalysis represents the coverage contribution analysis
type CoverageAnalysis struct {
	Baseline       map[string]float64
	TestResults    []TestResult
	CoverageDelta  map[string]map[string]float64 // test -> package -> delta
	UniquelyTested map[string][]string           // test -> packages only it tests
	TotalTime      time.Duration
}

// CodecovReport represents the Codecov JSON format
type CodecovReport struct {
	Coverage map[string][]interface{}     `json:"coverage"`
	Messages map[string]map[string]string `json:"messages,omitempty"`
}

func main() {
	var (
		pkgPath       = flag.String("pkg", ".", "Package path to test")
		outputDir     = flag.String("out", "coverage_analysis", "Output directory for results")
		verbose       = flag.Bool("v", false, "Verbose output")
		timeout       = flag.Duration("timeout", 10*time.Minute, "Timeout for each test")
		baseline      = flag.Bool("baseline", true, "Run baseline coverage (all tests)")
		pattern       = flag.String("run", "", "Regex pattern for test selection")
		jsonOutput    = flag.Bool("json", false, "Output results in JSON format")
		codecovOutput = flag.String("codecov", "", "Output Codecov JSON format to directory")
		perTestOutput = flag.Bool("per-test", false, "Output individual Codecov files per test")
	)

	flag.Parse()

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create codecov output directory if specified
	if *codecovOutput != "" {
		if err := os.MkdirAll(*codecovOutput, 0755); err != nil {
			log.Fatalf("Failed to create codecov output directory: %v", err)
		}
	}

	// List all tests in the package
	tests, err := listTests(*pkgPath, *pattern)
	if err != nil {
		log.Fatalf("Failed to list tests: %v", err)
	}

	if len(tests) == 0 {
		log.Fatal("No tests found")
	}

	fmt.Printf("Found %d tests in %s\n", len(tests), *pkgPath)

	analysis := &CoverageAnalysis{
		CoverageDelta:  make(map[string]map[string]float64),
		UniquelyTested: make(map[string][]string),
	}

	// Run baseline if requested
	if *baseline {
		fmt.Println("\nRunning baseline coverage (all tests)...")
		baselineResult := runTest(*pkgPath, "", *outputDir, "baseline", *timeout)
		if baselineResult.Error != nil {
			log.Printf("Warning: baseline failed: %v", baselineResult.Error)
		} else {
			analysis.Baseline = baselineResult.CoverageReport
			if *codecovOutput != "" && baselineResult.CoverageData != nil {
				codecovFile := filepath.Join(*codecovOutput, "baseline-coverage.json")
				if err := writeCodecovFile(baselineResult.CoverageData, codecovFile, "baseline"); err != nil {
					log.Printf("Failed to write baseline Codecov report: %v", err)
				}
			}
		}
	}

	// Run each test individually
	fmt.Println("\nRunning tests individually...")
	for i, test := range tests {
		if *verbose {
			fmt.Printf("[%d/%d] Running %s...\n", i+1, len(tests), test)
		}

		result := runTest(*pkgPath, test, *outputDir, test, *timeout)
		analysis.TestResults = append(analysis.TestResults, result)
		analysis.TotalTime += result.Duration

		if result.Error != nil {
			log.Printf("Test %s failed: %v", test, result.Error)
			continue
		}

		// Write per-test Codecov file if requested
		if *codecovOutput != "" && *perTestOutput && result.CoverageData != nil {
			testFile := filepath.Join(*codecovOutput, fmt.Sprintf("test-%s.json", test))
			if err := writeCodecovFile(result.CoverageData, testFile, test); err != nil {
				log.Printf("Failed to write Codecov file for %s: %v", test, err)
			}
		}

		// Calculate coverage delta
		analysis.CoverageDelta[test] = make(map[string]float64)
		for pkg, cov := range result.CoverageReport {
			baselineCov := 0.0
			if analysis.Baseline != nil {
				baselineCov = analysis.Baseline[pkg]
			}
			delta := cov - baselineCov
			if delta > 0 {
				analysis.CoverageDelta[test][pkg] = delta
			}
		}
	}

	// Generate combined Codecov report if requested
	if *codecovOutput != "" && !*perTestOutput {
		codecovFile := filepath.Join(*codecovOutput, "combined-coverage.json")
		if err := writeCombinedCodecov(analysis, codecovFile); err != nil {
			log.Printf("Failed to write combined Codecov report: %v", err)
		}

		// Generate test contribution report
		contributionFile := filepath.Join(*codecovOutput, "test-contributions.json")
		if err := writeTestContributions(analysis, contributionFile); err != nil {
			log.Printf("Failed to write test contribution report: %v", err)
		}
	}

	// Analyze unique coverage
	analyzeUniqueCoverage(analysis)

	// Output results
	if *jsonOutput {
		if err := outputJSON(analysis, filepath.Join(*outputDir, "analysis.json")); err != nil {
			log.Fatalf("Failed to write JSON output: %v", err)
		}
	} else {
		printAnalysis(analysis, *verbose)
	}

	// Generate summary report
	if err := generateSummaryReport(analysis, *outputDir); err != nil {
		log.Printf("Failed to generate summary report: %v", err)
	}
}

// listTests returns all test functions in the package
func listTests(pkgPath, pattern string) ([]string, error) {
	cmd := exec.Command("go", "test", "-list", ".", pkgPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tests: %w", err)
	}

	var tests []string
	var re *regexp.Regexp
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid test pattern: %w", err)
		}
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") && !strings.Contains(line, "/") {
			if re == nil || re.MatchString(line) {
				tests = append(tests, line)
			}
		}
	}

	return tests, nil
}

// runTest runs a single test with coverage collection
func runTest(pkgPath, testName, outputDir, coverName string, timeout time.Duration) TestResult {
	result := TestResult{
		TestName: testName,
		Package:  pkgPath,
	}

	// Create coverage directory for this test
	coverDir := filepath.Join(outputDir, "covdata", coverName)
	os.RemoveAll(coverDir) // Clean any existing data
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create coverage dir: %w", err)
		return result
	}
	result.CoverageDir = coverDir

	// Build the test command
	args := []string{"test", "-cover", pkgPath}
	if testName != "" {
		args = append(args, "-run", "^"+testName+"$")
	}
	args = append(args, "-covermode=atomic")

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverDir)

	// Run the test
	start := time.Now()
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("test failed: %w\nOutput: %s", err, string(output))
		return result
	}

	result.Passed = true

	// Parse coverage report - text format
	result.CoverageReport, err = parseCoverageReport(coverDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse coverage: %w", err)
		return result
	}

	// Parse coverage report - Codecov format
	coverReport := filepath.Join(coverDir, "coverage.out")
	covCmd := exec.Command("go", "tool", "covdata", "textfmt", "-i="+coverDir, "-o="+coverReport)
	if err := covCmd.Run(); err == nil {
		result.CoverageData, _ = parseToCodecov(coverReport)
	}

	return result
}

// parseCoverageReport extracts coverage percentages from the coverage data
func parseCoverageReport(coverDir string) (map[string]float64, error) {
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i="+coverDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to generate coverage report: %w", err)
	}

	coverage := make(map[string]float64)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like: "package/path coverage: 85.2% of statements"
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[1] == "coverage:" {
			pkg := parts[0]
			pctStr := strings.TrimSuffix(parts[2], "%")
			var pct float64
			if _, err := fmt.Sscanf(pctStr, "%f", &pct); err == nil {
				coverage[pkg] = pct
			}
		}
	}

	return coverage, nil
}

// parseToCodecov converts Go coverage format to Codecov JSON format
func parseToCodecov(coverFile string) (map[string][]interface{}, error) {
	data, err := ioutil.ReadFile(coverFile)
	if err != nil {
		return nil, err
	}

	coverage := make(map[string][]interface{})
	fileData := make(map[string]map[int]int) // file -> line -> hit count
	fileLines := make(map[string]int)        // file -> max line number

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line: file:startLine.startCol,endLine.endCol count
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		loc := parts[0]
		count, _ := strconv.Atoi(parts[1])

		// Extract file and line info
		colonIdx := strings.LastIndex(loc, ":")
		if colonIdx == -1 {
			continue
		}

		file := loc[:colonIdx]
		rangeStr := loc[colonIdx+1:]

		// Parse line range
		rangeParts := strings.Split(rangeStr, ",")
		if len(rangeParts) != 2 {
			continue
		}

		startParts := strings.Split(rangeParts[0], ".")
		endParts := strings.Split(rangeParts[1], ".")
		if len(startParts) != 2 || len(endParts) != 2 {
			continue
		}

		startLine, _ := strconv.Atoi(startParts[0])
		endLine, _ := strconv.Atoi(endParts[0])

		// Initialize file data if needed
		if _, ok := fileData[file]; !ok {
			fileData[file] = make(map[int]int)
		}

		// Mark lines as covered
		for line := startLine; line <= endLine; line++ {
			if count > 0 {
				fileData[file][line] = count
			} else if _, exists := fileData[file][line]; !exists {
				fileData[file][line] = 0
			}
			if line > fileLines[file] {
				fileLines[file] = line
			}
		}
	}

	// Convert to Codecov format
	for file, lineMap := range fileData {
		maxLine := fileLines[file]
		covArray := make([]interface{}, maxLine+1)

		// First element is always null
		covArray[0] = nil

		// Fill in coverage data
		for i := 1; i <= maxLine; i++ {
			if count, exists := lineMap[i]; exists {
				covArray[i] = count
			} else {
				covArray[i] = nil
			}
		}

		coverage[file] = covArray
	}

	return coverage, nil
}

// writeCodecovFile writes coverage data in Codecov JSON format
func writeCodecovFile(coverage map[string][]interface{}, filename, testName string) error {
	report := CodecovReport{
		Coverage: coverage,
		Messages: map[string]map[string]string{
			"_metadata": {
				"test_name": testName,
				"generated": time.Now().Format(time.RFC3339),
				"generator": "covtest",
			},
		},
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// writeCombinedCodecov writes combined coverage from all tests
func writeCombinedCodecov(analysis *CoverageAnalysis, filename string) error {
	combined := make(map[string][]interface{})
	testInfo := make(map[string]string)

	// Merge coverage from all tests
	for i, result := range analysis.TestResults {
		if result.CoverageData == nil {
			continue
		}

		testInfo[fmt.Sprintf("test_%d", i)] = result.TestName

		for file, covArray := range result.CoverageData {
			if existing, ok := combined[file]; ok {
				// Merge coverage arrays
				for j := 0; j < len(covArray) && j < len(existing); j++ {
					if covArray[j] == nil {
						continue
					}
					if existingVal, ok := existing[j].(int); ok {
						if newVal, ok := covArray[j].(int); ok {
							existing[j] = existingVal + newVal
						}
					} else {
						existing[j] = covArray[j]
					}
				}
				// Extend if new array is longer
				if len(covArray) > len(existing) {
					combined[file] = append(existing, covArray[len(existing):]...)
				}
			} else {
				combined[file] = covArray
			}
		}
	}

	report := CodecovReport{
		Coverage: combined,
		Messages: map[string]map[string]string{
			"_metadata": {
				"type":       "combined",
				"test_count": fmt.Sprintf("%d", len(analysis.TestResults)),
				"generated":  time.Now().Format(time.RFC3339),
				"generator":  "covtest",
			},
			"_tests": testInfo,
		},
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// writeTestContributions writes a report showing what each test contributes to coverage
func writeTestContributions(analysis *CoverageAnalysis, filename string) error {
	contributions := make(map[string]map[string]interface{})

	for _, result := range analysis.TestResults {
		if result.CoverageData == nil {
			continue
		}

		testContrib := make(map[string]interface{})
		testContrib["duration"] = result.Duration.String()
		testContrib["passed"] = result.Passed

		// Calculate unique lines covered by this test
		uniqueLines := 0
		totalLines := 0
		for file, covArray := range result.CoverageData {
			for i, val := range covArray {
				if val != nil && val != 0 {
					totalLines++
					// Check if this line is covered by any other test
					isUnique := true
					for _, otherResult := range analysis.TestResults {
						if otherResult.TestName == result.TestName {
							continue
						}
						if otherCov, ok := otherResult.CoverageData[file]; ok {
							if i < len(otherCov) && otherCov[i] != nil && otherCov[i] != 0 {
								isUnique = false
								break
							}
						}
					}
					if isUnique {
						uniqueLines++
					}
				}
			}
		}

		testContrib["total_lines"] = totalLines
		testContrib["unique_lines"] = uniqueLines
		testContrib["coverage_delta"] = analysis.CoverageDelta[result.TestName]

		contributions[result.TestName] = testContrib
	}

	data, err := json.MarshalIndent(contributions, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// analyzeUniqueCoverage finds which packages are uniquely tested by each test
func analyzeUniqueCoverage(analysis *CoverageAnalysis) {
	// For each package, track which tests cover it
	pkgToTests := make(map[string][]string)

	for _, result := range analysis.TestResults {
		if result.Error != nil {
			continue
		}
		for pkg, cov := range result.CoverageReport {
			if cov > 0 {
				pkgToTests[pkg] = append(pkgToTests[pkg], result.TestName)
			}
		}
	}

	// Find uniquely tested packages for each test
	for pkg, tests := range pkgToTests {
		if len(tests) == 1 {
			testName := tests[0]
			analysis.UniquelyTested[testName] = append(analysis.UniquelyTested[testName], pkg)
		}
	}
}

// printAnalysis prints the analysis results
func printAnalysis(analysis *CoverageAnalysis, verbose bool) {
	fmt.Println("\n=== Coverage Analysis ===")
	fmt.Printf("Total test time: %v\n", analysis.TotalTime)
	fmt.Printf("Tests run: %d\n", len(analysis.TestResults))

	// Sort tests by coverage contribution
	type testContribution struct {
		name     string
		totalNew float64
		packages int
	}

	var contributions []testContribution
	for test, pkgDeltas := range analysis.CoverageDelta {
		total := 0.0
		for _, delta := range pkgDeltas {
			total += delta
		}
		contributions = append(contributions, testContribution{
			name:     test,
			totalNew: total,
			packages: len(pkgDeltas),
		})
	}

	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].totalNew > contributions[j].totalNew
	})

	fmt.Println("\nTop Coverage Contributors:")
	for i, contrib := range contributions {
		if i >= 10 && !verbose {
			break
		}
		fmt.Printf("%d. %s: +%.1f%% across %d packages\n",
			i+1, contrib.name, contrib.totalNew, contrib.packages)
	}

	// Show tests with unique coverage
	fmt.Println("\nTests with Unique Coverage:")
	for test, packages := range analysis.UniquelyTested {
		fmt.Printf("- %s: %d unique packages\n", test, len(packages))
		if verbose {
			for _, pkg := range packages {
				fmt.Printf("  - %s\n", pkg)
			}
		}
	}
}

// outputJSON writes the analysis results to a JSON file
func outputJSON(analysis *CoverageAnalysis, filename string) error {
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// generateSummaryReport creates a detailed markdown report
func generateSummaryReport(analysis *CoverageAnalysis, outputDir string) error {
	reportPath := filepath.Join(outputDir, "coverage_report.md")
	f, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# Coverage Analysis Report\n\n")
	fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "## Summary\n\n")
	fmt.Fprintf(f, "- Total tests: %d\n", len(analysis.TestResults))
	fmt.Fprintf(f, "- Total time: %v\n", analysis.TotalTime)

	var passed, failed int
	for _, result := range analysis.TestResults {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Fprintf(f, "- Passed: %d\n", passed)
	fmt.Fprintf(f, "- Failed: %d\n\n", failed)

	// Write baseline coverage if available
	if analysis.Baseline != nil {
		fmt.Fprintf(f, "## Baseline Coverage\n\n")
		fmt.Fprintf(f, "| Package | Coverage |\n")
		fmt.Fprintf(f, "|---------|----------|\n")
		for pkg, cov := range analysis.Baseline {
			fmt.Fprintf(f, "| %s | %.1f%% |\n", pkg, cov)
		}
		fmt.Fprintf(f, "\n")
	}

	// Write individual test results
	fmt.Fprintf(f, "## Individual Test Results\n\n")
	for _, result := range analysis.TestResults {
		fmt.Fprintf(f, "### %s\n\n", result.TestName)
		fmt.Fprintf(f, "- Duration: %v\n", result.Duration)
		fmt.Fprintf(f, "- Status: %s\n", func() string {
			if result.Passed {
				return "PASSED"
			}
			return "FAILED"
		}())

		if result.Error != nil {
			fmt.Fprintf(f, "- Error: %v\n", result.Error)
		}

		if result.CoverageReport != nil && len(result.CoverageReport) > 0 {
			fmt.Fprintf(f, "\nCoverage:\n\n")
			fmt.Fprintf(f, "| Package | Coverage |\n")
			fmt.Fprintf(f, "|---------|----------|\n")
			for pkg, cov := range result.CoverageReport {
				fmt.Fprintf(f, "| %s | %.1f%% |\n", pkg, cov)
			}
		}
		fmt.Fprintf(f, "\n")
	}

	return nil
}
