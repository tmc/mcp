// +build ignore

// Concept for per-line coverage analysis in scripttest
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LineCoverage represents coverage data for a single test line
type LineCoverage struct {
	LineNumber int
	Line       string
	Command    string
	Coverage   map[string]float64 // tool -> coverage percentage
}

// PerLineAnalyzer analyzes coverage on a per-line basis
type PerLineAnalyzer struct {
	baseDir    string
	testFile   string
	txtarStart int
	txtar      string
}

// ParseScriptFile parses a scripttest file into executable lines and txtar content
func (a *PerLineAnalyzer) ParseScriptFile() ([]string, error) {
	file, err := os.Open(a.testFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	var txtarLines []string
	inTxtar := false
	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Check for txtar section start
		if strings.HasPrefix(line, "-- ") && strings.HasSuffix(line, " --") {
			inTxtar = true
			a.txtarStart = lineNum
		}

		if inTxtar {
			txtarLines = append(txtarLines, line)
		} else if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			// Executable line (not empty, not comment)
			lines = append(lines, line)
		}
	}

	a.txtar = strings.Join(txtarLines, "\n")
	return lines, scanner.Err()
}

// ExecuteLineWithCoverage runs a single line with coverage collection
func (a *PerLineAnalyzer) ExecuteLineWithCoverage(lineNum int, line string) LineCoverage {
	// Create coverage directory for this line
	lineDir := filepath.Join(a.baseDir, fmt.Sprintf("line_%d", lineNum))
	os.MkdirAll(lineDir, 0755)
	
	// Set GOCOVERDIR for this line execution
	os.Setenv("GOCOVERDIR", lineDir)
	
	// Create temporary test file with just this line + txtar
	tmpFile := filepath.Join(lineDir, "test.txt")
	content := fmt.Sprintf("%s\n\n%s", line, a.txtar)
	os.WriteFile(tmpFile, []byte(content), 0644)
	
	// Execute the test (pseudo-code)
	// runScriptTest(tmpFile)
	
	// Analyze coverage
	coverage := a.analyzeCoverage(lineDir)
	
	return LineCoverage{
		LineNumber: lineNum,
		Line:       line,
		Command:    extractCommand(line),
		Coverage:   coverage,
	}
}

// analyzeCoverage examines coverage data for a directory
func (a *PerLineAnalyzer) analyzeCoverage(dir string) map[string]float64 {
	// Pseudo-code for coverage analysis
	// This would use go tool covdata to get percentages
	coverage := make(map[string]float64)
	
	// Example: parse covdata output
	// output := exec.Command("go", "tool", "covdata", "percent", "-i", dir).Output()
	// coverage["mcpdiff"] = parseCoveragePercent(output, "mcpdiff")
	
	return coverage
}

// GenerateAnnotatedTest creates a test file with coverage annotations
func (a *PerLineAnalyzer) GenerateAnnotatedTest(results []LineCoverage) error {
	outFile := strings.TrimSuffix(a.testFile, ".txt") + "_annotated.txt"
	
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read original file
	original, err := os.ReadFile(a.testFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(original), "\n")
	annotated := make([]string, 0, len(lines))
	
	resultMap := make(map[int]LineCoverage)
	for _, r := range results {
		resultMap[r.LineNumber] = r
	}

	for i, line := range lines {
		if result, ok := resultMap[i+1]; ok {
			// Add coverage annotation
			coverageStr := formatCoverage(result.Coverage)
			annotated = append(annotated, fmt.Sprintf("%s    # %s", line, coverageStr))
		} else {
			annotated = append(annotated, line)
		}
	}

	_, err = f.WriteString(strings.Join(annotated, "\n"))
	return err
}

func extractCommand(line string) string {
	parts := strings.Fields(line)
	if len(parts) > 1 && parts[0] == "exec" {
		return parts[1]
	}
	return ""
}

func formatCoverage(coverage map[string]float64) string {
	var parts []string
	for tool, percent := range coverage {
		parts = append(parts, fmt.Sprintf("%s: %.1f%%", tool, percent))
	}
	return strings.Join(parts, ", ")
}

// Example usage
func main() {
	analyzer := &PerLineAnalyzer{
		baseDir:  "/tmp/per-line-coverage",
		testFile: "testdata/example.txt",
	}

	// Parse the test file
	lines, err := analyzer.ParseScriptFile()
	if err != nil {
		panic(err)
	}

	// Analyze each line
	var results []LineCoverage
	for i, line := range lines {
		result := analyzer.ExecuteLineWithCoverage(i+1, line)
		results = append(results, result)
		
		fmt.Printf("Line %d: %s\n", result.LineNumber, result.Line)
		fmt.Printf("Coverage: %v\n\n", result.Coverage)
	}

	// Generate annotated test file
	err = analyzer.GenerateAnnotatedTest(results)
	if err != nil {
		panic(err)
	}

	fmt.Println("Generated annotated test file")
}