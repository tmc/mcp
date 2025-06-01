package coverageviz

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CoverageIntegrator combines Go coverage data with MCP test results
type CoverageIntegrator struct {
	coverageData map[string]*FileCoverage
	sourceFiles  map[string]*FileData
}

// NewCoverageIntegrator creates a new coverage integrator
func NewCoverageIntegrator() *CoverageIntegrator {
	return &CoverageIntegrator{
		coverageData: make(map[string]*FileCoverage),
		sourceFiles:  make(map[string]*FileData),
	}
}

// ParseCoverageProfile parses a Go coverage profile
func (c *CoverageIntegrator) ParseCoverageProfile(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	
	// Skip mode line
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "mode:") {
			return fmt.Errorf("invalid coverage profile format")
		}
	}
	
	// Regular expression to parse coverage lines
	// Format: filename:startline.startcol,endline.endcol statements hitcount
	coverageRegex := regexp.MustCompile(`^(.+):(\d+)\.(\d+),(\d+)\.(\d+)\s+(\d+)\s+(\d+)$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		matches := coverageRegex.FindStringSubmatch(line)
		if len(matches) < 8 {
			continue
		}
		
		filename := matches[1]
		startLine, _ := strconv.Atoi(matches[2])
		endLine, _ := strconv.Atoi(matches[4])
		hitCount, _ := strconv.Atoi(matches[7])
		
		// Initialize file coverage if needed
		if _, ok := c.coverageData[filename]; !ok {
			c.coverageData[filename] = &FileCoverage{
				Path: filename,
			}
		}
		
		// Update coverage data
		coverage := c.coverageData[filename]
		for line := startLine; line <= endLine; line++ {
			coverage.CoveredLines++
			if hitCount == 0 {
				coverage.CoveredLines--
			}
		}
		coverage.TotalLines = endLine
	}
	
	// Calculate coverage percentages
	for _, coverage := range c.coverageData {
		if coverage.TotalLines > 0 {
			coverage.CoveragePercent = float64(coverage.CoveredLines) / float64(coverage.TotalLines) * 100
		}
	}
	
	return scanner.Err()
}

// ParseCoverageJSON parses Go coverage data in JSON format
func (c *CoverageIntegrator) ParseCoverageJSON(data []byte) error {
	// This would parse the newer JSON format from go tool cover
	// For now, we'll use the text format
	return fmt.Errorf("JSON coverage format not yet implemented")
}

// LoadSourceFile loads and parses a source file
func (c *CoverageIntegrator) LoadSourceFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fileData := &FileData{
		Path:  path,
		Lines: []LineData{},
	}
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		content := scanner.Text()
		
		lineData := LineData{
			Number:  lineNum,
			Content: content,
			Covered: false,
		}
		
		// Check if this line is covered
		if coverage, ok := c.coverageData[path]; ok {
			if lineNum <= coverage.TotalLines {
				lineData.Covered = coverage.CoveredLines > 0
			}
		}
		
		fileData.Lines = append(fileData.Lines, lineData)
	}
	
	// Extract package name and functions
	packageRegex := regexp.MustCompile(`^package\s+(\w+)`)
	functionRegex := regexp.MustCompile(`^func\s+(?:\(.*?\)\s*)?(\w+)\s*\(`)
	
	for i, line := range fileData.Lines {
		if matches := packageRegex.FindStringSubmatch(line.Content); len(matches) > 1 {
			fileData.Package = matches[1]
		}
		
		if matches := functionRegex.FindStringSubmatch(line.Content); len(matches) > 1 {
			function := FunctionData{
				Name:      matches[1],
				Package:   fileData.Package,
				StartLine: i + 1,
				Covered:   line.Covered,
			}
			
			// Find function end (simplified)
			braceCount := 1
			for j := i + 1; j < len(fileData.Lines); j++ {
				if strings.Contains(fileData.Lines[j].Content, "{") {
					braceCount++
				}
				if strings.Contains(fileData.Lines[j].Content, "}") {
					braceCount--
					if braceCount == 0 {
						function.EndLine = j + 1
						break
					}
				}
			}
			
			fileData.Functions = append(fileData.Functions, function)
		}
	}
	
	c.sourceFiles[path] = fileData
	return scanner.Err()
}

// IntegrateTestResults combines coverage data with test results
func (c *CoverageIntegrator) IntegrateTestResults(session TestSession) (*CoverageVisualization, error) {
	viz := &CoverageVisualization{
		Files:     make(map[string]*FileData),
		Generated: time.Now(),
	}
	
	// Copy source files
	for path, file := range c.sourceFiles {
		viz.Files[path] = file
		
		// Add coverage data
		if coverage, ok := c.coverageData[path]; ok {
			file.Coverage = *coverage
		}
	}
	
	// Add test session
	viz.Sessions = []TestSession{session}
	
	// Calculate summary
	viz.Summary = c.calculateSummary(viz)
	
	// Perform test impact analysis
	c.analyzeTestImpact(viz)
	
	return viz, nil
}

// calculateSummary computes overall coverage statistics
func (c *CoverageIntegrator) calculateSummary(viz *CoverageVisualization) Summary {
	summary := Summary{
		ByPackage: make(map[string]CoverageStats),
	}
	
	// Count tests
	for _, session := range viz.Sessions {
		summary.TotalTests += len(session.Tests)
		for _, test := range session.Tests {
			switch test.Result {
			case TestPassed:
				summary.PassedTests++
			case TestFailed:
				summary.FailedTests++
			}
		}
	}
	
	// Calculate file and line coverage
	packageStats := make(map[string]*struct {
		totalLines   int
		coveredLines int
	})
	
	for _, file := range viz.Files {
		summary.TotalFiles++
		if file.Coverage.CoveredLines > 0 {
			summary.CoveredFiles++
		}
		
		summary.TotalLines += file.Coverage.TotalLines
		summary.CoveredLines += file.Coverage.CoveredLines
		
		// Package stats
		if file.Package != "" {
			if _, ok := packageStats[file.Package]; !ok {
				packageStats[file.Package] = &struct {
					totalLines   int
					coveredLines int
				}{}
			}
			packageStats[file.Package].totalLines += file.Coverage.TotalLines
			packageStats[file.Package].coveredLines += file.Coverage.CoveredLines
		}
	}
	
	// Calculate percentages
	if summary.TotalLines > 0 {
		summary.Coverage.Line = float64(summary.CoveredLines) / float64(summary.TotalLines) * 100
	}
	
	// Package percentages
	for pkg, stats := range packageStats {
		if stats.totalLines > 0 {
			summary.ByPackage[pkg] = CoverageStats{
				Line: float64(stats.coveredLines) / float64(stats.totalLines) * 100,
			}
		}
	}
	
	return summary
}

// analyzeTestImpact determines which tests cover which code
func (c *CoverageIntegrator) analyzeTestImpact(viz *CoverageVisualization) {
	// This is a simplified version
	// In a real implementation, we would track coverage per test
	
	for _, file := range viz.Files {
		file.TestImpact = []TestImpact{}
		
		// For each test, determine its impact on this file
		for _, session := range viz.Sessions {
			for _, test := range session.Tests {
				impact := TestImpact{
					TestID:   test.TestName,
					TestName: test.TestName,
				}
				
				// Collect covered lines (simplified - would need per-test tracking)
				for _, line := range file.Lines {
					if line.Covered {
						impact.CoveredLines = append(impact.CoveredLines, line.Number)
					}
				}
				
				if len(impact.CoveredLines) > 0 {
					impact.Impact = float64(len(impact.CoveredLines)) / float64(file.Coverage.TotalLines) * 100
					file.TestImpact = append(file.TestImpact, impact)
				}
			}
		}
	}
}

// LoadFromDirectory loads coverage and source files from a directory
func (c *CoverageIntegrator) LoadFromDirectory(dir string) error {
	// Find coverage files
	coverageFiles, err := filepath.Glob(filepath.Join(dir, "*.out"))
	if err != nil {
		return err
	}
	
	for _, coverageFile := range coverageFiles {
		file, err := os.Open(coverageFile)
		if err != nil {
			return err
		}
		defer file.Close()
		
		if err := c.ParseCoverageProfile(file); err != nil {
			return err
		}
	}
	
	// Find source files
	sourceFiles, err := filepath.Glob(filepath.Join(dir, "**/*.go"))
	if err != nil {
		return err
	}
	
	for _, sourceFile := range sourceFiles {
		if err := c.LoadSourceFile(sourceFile); err != nil {
			// Skip files that can't be loaded
			continue
		}
	}
	
	return nil
}

// GetSourceFiles returns the loaded source files
func (c *CoverageIntegrator) GetSourceFiles() map[string]*FileData {
	return c.sourceFiles
}

// CalculateSummary calculates summary without a full visualization
func (c *CoverageIntegrator) CalculateSummary() Summary {
	summary := Summary{
		ByPackage: make(map[string]CoverageStats),
	}

	// Calculate file and line coverage
	packageStats := make(map[string]*struct {
		totalLines   int
		coveredLines int
	})

	for _, file := range c.sourceFiles {
		summary.TotalFiles++

		coverage, hasCoverage := c.coverageData[file.Path]
		if hasCoverage && coverage.CoveredLines > 0 {
			summary.CoveredFiles++
			summary.TotalLines += coverage.TotalLines
			summary.CoveredLines += coverage.CoveredLines

			// Package stats
			if file.Package != "" {
				if _, ok := packageStats[file.Package]; !ok {
					packageStats[file.Package] = &struct {
						totalLines   int
						coveredLines int
					}{}
				}
				packageStats[file.Package].totalLines += coverage.TotalLines
				packageStats[file.Package].coveredLines += coverage.CoveredLines
			}
		}
	}

	// Calculate percentages
	if summary.TotalLines > 0 {
		summary.Coverage.Line = float64(summary.CoveredLines) / float64(summary.TotalLines) * 100
	}

	// Package percentages
	for pkg, stats := range packageStats {
		if stats.totalLines > 0 {
			summary.ByPackage[pkg] = CoverageStats{
				Line: float64(stats.coveredLines) / float64(stats.totalLines) * 100,
			}
		}
	}

	return summary
}