package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tmc/mcp/exp/coverage_viz"
)

func main() {
	var (
		serve     = flag.Bool("serve", false, "Start web server for visualization")
		port      = flag.Int("port", 8080, "Port for web server")
		report    = flag.Bool("report", false, "Generate text report")
		coverFile = flag.String("coverprofile", "", "Coverage profile file")
	)

	flag.Parse()

	// Mock coverage data for demonstration
	coverage := createMockCoverage()

	visualizer := coverage_viz.NewVisualizer(coverage)

	if *serve {
		log.Fatal(visualizer.Serve(*port))
	} else if *report {
		visualizer.GenerateReport(os.Stdout)
	} else {
		flag.Usage()
	}
}

// createMockCoverage creates sample coverage data for demonstration
func createMockCoverage() *coverage_viz.CoverageData {
	return &coverage_viz.CoverageData{
		Files: map[string]*coverage_viz.FileCoverage{
			"example_test.go": {
				Path:         "example_test.go",
				TotalLines:   30,
				CoveredLines: 25,
				Lines: map[int]*coverage_viz.LineCoverage{
					10: {Number: 10, Covered: true, HitCount: 7, Tests: []string{"TestParseMessage_Info", "TestParseMessage_Warn", "TestParseMessage_Empty", "TestParseMessage_InvalidFormat", "TestParseMessage_PartialCoverage"}},
					11: {Number: 11, Covered: true, HitCount: 1, Tests: []string{"TestParseMessage_Empty"}},
					12: {Number: 12, Covered: true, HitCount: 1, Tests: []string{"TestParseMessage_Empty"}},
					15: {Number: 15, Covered: true, HitCount: 6, Tests: []string{"TestParseMessage_Info", "TestParseMessage_Warn", "TestParseMessage_InvalidFormat", "TestParseMessage_PartialCoverage"}},
					16: {Number: 16, Covered: true, HitCount: 1, Tests: []string{"TestParseMessage_InvalidFormat"}},
					17: {Number: 17, Covered: true, HitCount: 1, Tests: []string{"TestParseMessage_InvalidFormat"}},
					20: {Number: 20, Covered: true, HitCount: 5, Tests: []string{"TestParseMessage_Info", "TestParseMessage_Warn", "TestParseMessage_PartialCoverage"}},
					21: {Number: 21, Covered: true, HitCount: 3, Tests: []string{"TestParseMessage_Info", "TestParseMessage_PartialCoverage"}},
					22: {Number: 22, Covered: true, HitCount: 3, Tests: []string{"TestParseMessage_Info", "TestParseMessage_PartialCoverage"}},
					23: {Number: 23, Covered: true, HitCount: 2, Tests: []string{"TestParseMessage_Warn", "TestParseMessage_PartialCoverage"}},
					24: {Number: 24, Covered: true, HitCount: 2, Tests: []string{"TestParseMessage_Warn", "TestParseMessage_PartialCoverage"}},
					25: {Number: 25, Covered: false, HitCount: 0, Tests: []string{}},
					26: {Number: 26, Covered: false, HitCount: 0, Tests: []string{}},
					27: {Number: 27, Covered: false, HitCount: 0, Tests: []string{}},
					28: {Number: 28, Covered: false, HitCount: 0, Tests: []string{}},
				},
			},
		},
		Tests: map[string]*coverage_viz.TestImpact{
			"TestParseMessage_Info": {
				Name:           "TestParseMessage_Info",
				CoveredLines:   9,
				UniqueCoverage: 0,
				Files:          []string{"example_test.go"},
			},
			"TestParseMessage_Warn": {
				Name:           "TestParseMessage_Warn",
				CoveredLines:   9,
				UniqueCoverage: 0,
				Files:          []string{"example_test.go"},
			},
			"TestParseMessage_Empty": {
				Name:           "TestParseMessage_Empty",
				CoveredLines:   3,
				UniqueCoverage: 3,
				Files:          []string{"example_test.go"},
			},
			"TestParseMessage_InvalidFormat": {
				Name:           "TestParseMessage_InvalidFormat",
				CoveredLines:   6,
				UniqueCoverage: 3,
				Files:          []string{"example_test.go"},
			},
			"TestParseMessage_PartialCoverage": {
				Name:           "TestParseMessage_PartialCoverage",
				CoveredLines:   11,
				UniqueCoverage: 0,
				Files:          []string{"example_test.go"},
			},
		},
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate text report\n")
		fmt.Fprintf(os.Stderr, "  %s -report\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n  # Start web server\n")
		fmt.Fprintf(os.Stderr, "  %s -serve -port 8080\n", os.Args[0])
	}
}
