package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	mcpcoverageviz "github.com/tmc/mcp/exp/mcp-coverage-viz"
)

func main() {
	var (
		coverageFile = flag.String("coverage", "", "Go coverage profile file")
		traceFile    = flag.String("trace", "", "MCP trace file (.mcp)")
		traceDir     = flag.String("trace-dir", "", "Directory containing MCP trace files")
		outputFile   = flag.String("output", "", "Output file for visualization data (JSON)")
		serve        = flag.Bool("serve", false, "Start web server for visualization")
		port         = flag.String("port", ":8080", "Port for web server")
		sourceDir    = flag.String("source", ".", "Source code directory")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "MCP Coverage Visualization Tool\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Analyze coverage and traces, start web server\n")
		fmt.Fprintf(os.Stderr, "  %s -coverage coverage.out -trace trace.mcp -serve\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Process multiple trace files\n")
		fmt.Fprintf(os.Stderr, "  %s -coverage coverage.out -trace-dir ./traces -serve\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Export visualization data to JSON\n")
		fmt.Fprintf(os.Stderr, "  %s -coverage coverage.out -trace trace.mcp -output viz.json\n", os.Args[0])
	}
	
	flag.Parse()
	
	// Validate inputs
	if *coverageFile == "" && *traceFile == "" && *traceDir == "" {
		log.Fatal("At least one of -coverage, -trace, or -trace-dir must be specified")
	}
	
	// Create coverage integrator
	integrator := mcpcoverageviz.NewCoverageIntegrator()
	
	// Load coverage data if provided
	if *coverageFile != "" {
		file, err := os.Open(*coverageFile)
		if err != nil {
			log.Fatalf("Failed to open coverage file: %v", err)
		}
		defer file.Close()
		
		if err := integrator.ParseCoverageProfile(file); err != nil {
			log.Fatalf("Failed to parse coverage profile: %v", err)
		}
		
		log.Printf("Loaded coverage data from %s", *coverageFile)
	}
	
	// Load source files
	if err := loadSourceFiles(integrator, *sourceDir); err != nil {
		log.Fatalf("Failed to load source files: %v", err)
	}
	
	// Parse MCP traces
	var sessions []mcpcoverageviz.TestSession
	
	if *traceFile != "" {
		session, err := parseTraceFile(*traceFile)
		if err != nil {
			log.Fatalf("Failed to parse trace file: %v", err)
		}
		sessions = append(sessions, session)
	}
	
	if *traceDir != "" {
		dirSessions, err := parseTraceDirectory(*traceDir)
		if err != nil {
			log.Fatalf("Failed to parse trace directory: %v", err)
		}
		sessions = append(sessions, dirSessions...)
	}
	
	// Combine data
	var viz *mcpcoverageviz.CoverageVisualization
	if len(sessions) > 0 {
		// Use first session as primary, merge others
		viz, err := integrator.IntegrateTestResults(sessions[0])
		if err != nil {
			log.Fatalf("Failed to integrate test results: %v", err)
		}
		
		// Add remaining sessions
		if len(sessions) > 1 {
			viz.Sessions = append(viz.Sessions, sessions[1:]...)
		}
	} else {
		// Coverage only, no test sessions
		viz = &mcpcoverageviz.CoverageVisualization{
			Files:     integrator.GetSourceFiles(),
			Summary:   integrator.CalculateSummary(),
			Generated: time.Now(),
		}
	}
	
	// Output results
	if *outputFile != "" {
		if err := saveVisualizationData(viz, *outputFile); err != nil {
			log.Fatalf("Failed to save visualization data: %v", err)
		}
		log.Printf("Saved visualization data to %s", *outputFile)
	}
	
	// Start web server if requested
	if *serve {
		server, err := mcpcoverageviz.NewWebServer(viz, *port)
		if err != nil {
			log.Fatalf("Failed to create web server: %v", err)
		}
		
		log.Printf("Starting web server on %s", *port)
		log.Printf("Open http://localhost%s in your browser", *port)
		
		if err := server.Serve(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
	
	// If neither serve nor output specified, print summary
	if !*serve && *outputFile == "" {
		printSummary(viz)
	}
}

func loadSourceFiles(integrator *mcpcoverageviz.CoverageIntegrator, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if filepath.Ext(path) == ".go" && !strings.Contains(path, "vendor/") {
			if err := integrator.LoadSourceFile(path); err != nil {
				// Skip files that can't be loaded
				log.Printf("Warning: failed to load %s: %v", path, err)
			}
		}
		
		return nil
	})
}

func parseTraceFile(path string) (mcpcoverageviz.TestSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return mcpcoverageviz.TestSession{}, err
	}
	defer file.Close()
	
	parser := mcpcoverageviz.NewTraceParser(file)
	traces, err := parser.ParseMCPTrace()
	if err != nil {
		return mcpcoverageviz.TestSession{}, err
	}
	
	sessions := mcpcoverageviz.GroupTracesBySession(traces)
	if len(sessions) > 0 {
		return sessions[0], nil
	}
	
	// Create single session from all traces
	return mcpcoverageviz.TestSession{
		ID:        filepath.Base(path),
		Name:      filepath.Base(path),
		StartTime: traces[0].Timestamp,
		EndTime:   traces[len(traces)-1].Timestamp,
		Traces:    traces,
		Tests:     mcpcoverageviz.ExtractTestInfo(traces),
	}, nil
}

func parseTraceDirectory(dir string) ([]mcpcoverageviz.TestSession, error) {
	var sessions []mcpcoverageviz.TestSession
	
	files, err := filepath.Glob(filepath.Join(dir, "*.mcp"))
	if err != nil {
		return nil, err
	}
	
	for _, file := range files {
		session, err := parseTraceFile(file)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", file, err)
			continue
		}
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

func saveVisualizationData(viz *mcpcoverageviz.CoverageVisualization, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(viz)
}

func printSummary(viz *mcpcoverageviz.CoverageVisualization) {
	fmt.Printf("Coverage Summary\n")
	fmt.Printf("================\n\n")
	
	fmt.Printf("Line Coverage: %.1f%%\n", viz.Summary.Coverage.Line)
	fmt.Printf("Files: %d covered / %d total\n", viz.Summary.CoveredFiles, viz.Summary.TotalFiles)
	fmt.Printf("Lines: %d covered / %d total\n", viz.Summary.CoveredLines, viz.Summary.TotalLines)
	
	if len(viz.Sessions) > 0 {
		fmt.Printf("\nTest Sessions: %d\n", len(viz.Sessions))
		fmt.Printf("Total Tests: %d\n", viz.Summary.TotalTests)
		fmt.Printf("Passed: %d\n", viz.Summary.PassedTests)
		fmt.Printf("Failed: %d\n", viz.Summary.FailedTests)
	}
	
	fmt.Printf("\nPackage Coverage:\n")
	for pkg, stats := range viz.Summary.ByPackage {
		fmt.Printf("  %-30s %.1f%%\n", pkg, stats.Line)
	}
}