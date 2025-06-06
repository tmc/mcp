package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp/exp/changemanagement"
)

func main() {
	var (
		description = flag.String("description", "", "Natural language change description")
		codebase    = flag.String("codebase", ".", "Root directory of codebase")
		outputDir   = flag.String("output", "change-output", "Output directory for all artifacts")
		dryRun      = flag.Bool("dry-run", false, "Perform analysis only, don't execute changes")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *description == "" {
		log.Fatal("Please provide a change description via -description")
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Initialize orchestrator
	orchestrator := NewChangeOrchestrator(*outputDir, *verbose)

	// Execute change workflow
	result, err := orchestrator.ExecuteChange(*description, *codebase, *dryRun)
	if err != nil {
		log.Fatalf("Change execution failed: %v", err)
	}

	// Save final report
	reportPath := filepath.Join(*outputDir, "change_report.json")
	reportData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	fmt.Printf("\nChange execution completed!\n")
	fmt.Printf("Report saved to: %s\n", reportPath)
	fmt.Printf("All artifacts in: %s\n", *outputDir)
}

// ChangeOrchestrator orchestrates the entire change workflow
type ChangeOrchestrator struct {
	outputDir string
	verbose   bool
}

// NewChangeOrchestrator creates a new orchestrator
func NewChangeOrchestrator(outputDir string, verbose bool) *ChangeOrchestrator {
	return &ChangeOrchestrator{
		outputDir: outputDir,
		verbose:   verbose,
	}
}

// ExecuteChange executes the complete change workflow
func (o *ChangeOrchestrator) ExecuteChange(description, codebase string, dryRun bool) (*ChangeResult, error) {
	result := &ChangeResult{
		StartTime: time.Now(),
		Phases:    []PhaseResult{},
	}

	// Phase 1: Analyze the change
	analysisResult, err := o.analyzeChange(description)
	if err != nil {
		return result, fmt.Errorf("analysis phase failed: %w", err)
	}
	result.Analysis = analysisResult
	result.Phases = append(result.Phases, PhaseResult{
		Name:      "Analysis",
		Status:    "completed",
		Duration:  time.Since(result.StartTime),
		Artifacts: []string{filepath.Join(o.outputDir, "analysis.json")},
	})

	// Phase 2: Find affected tests
	testResult, err := o.findTests(analysisResult, codebase)
	if err != nil {
		return result, fmt.Errorf("test finding phase failed: %w", err)
	}
	result.TestResults = testResult
	result.Phases = append(result.Phases, PhaseResult{
		Name:      "Test Finding",
		Status:    "completed",
		Duration:  time.Since(result.Phases[len(result.Phases)-1].EndTime),
		Artifacts: []string{filepath.Join(o.outputDir, "affected_tests.json")},
	})

	// Phase 3: Generate documentation
	docResult, err := o.generateDocumentation(analysisResult)
	if err != nil {
		return result, fmt.Errorf("documentation phase failed: %w", err)
	}
	result.Documentation = docResult
	result.Phases = append(result.Phases, PhaseResult{
		Name:      "Documentation",
		Status:    "completed",
		Duration:  time.Since(result.Phases[len(result.Phases)-1].EndTime),
		Artifacts: docResult,
	})

	// Phase 4: Generate test mutations (only if not dry-run)
	if !dryRun && len(testResult.DefinitelyAffected) > 0 {
		mutationResult, err := o.generateMutations(testResult.DefinitelyAffected[0])
		if err != nil {
			log.Printf("Mutation phase failed: %v", err)
		} else {
			result.Mutations = mutationResult
			result.Phases = append(result.Phases, PhaseResult{
				Name:      "Test Mutations",
				Status:    "completed",
				Duration:  time.Since(result.Phases[len(result.Phases)-1].EndTime),
				Artifacts: mutationResult,
			})
		}
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	return result, nil
}

func (o *ChangeOrchestrator) analyzeChange(description string) (*changemanagement.AnalysisResult, error) {
	o.log("Analyzing change description...")

	// Run mcp-change-analyze
	outputPath := filepath.Join(o.outputDir, "analysis.json")
	cmd := exec.Command("mcp-change-analyze",
		"-description", description,
		"-output", outputPath,
		"-format", "json")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("analysis command failed: %w", err)
	}

	// Load analysis result
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	var analysis changemanagement.AnalysisResult
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, err
	}

	o.log("Analysis completed: Type=%s, Risk=%s", analysis.Type, analysis.RiskLevel)
	return &analysis, nil
}

func (o *ChangeOrchestrator) findTests(analysis *changemanagement.AnalysisResult, codebase string) (*changemanagement.TestFinderResult, error) {
	o.log("Finding affected tests...")

	// Save analysis for test finder
	analysisPath := filepath.Join(o.outputDir, "analysis.json")
	outputPath := filepath.Join(o.outputDir, "affected_tests.json")

	cmd := exec.Command("mcp-test-find",
		"-change", analysisPath,
		"-codebase", codebase,
		"-output", outputPath,
		"-format", "json")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("test finder command failed: %w", err)
	}

	// Load test results
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	var testResult changemanagement.TestFinderResult
	if err := json.Unmarshal(data, &testResult); err != nil {
		return nil, err
	}

	o.log("Found %d definitely affected tests", len(testResult.DefinitelyAffected))
	return &testResult, nil
}

func (o *ChangeOrchestrator) generateDocumentation(analysis *changemanagement.AnalysisResult) ([]string, error) {
	o.log("Generating documentation...")

	analysisPath := filepath.Join(o.outputDir, "analysis.json")
	docsDir := filepath.Join(o.outputDir, "docs")

	cmd := exec.Command("mcp-doc-gen",
		"-change", analysisPath,
		"-output", docsDir,
		"-format", "markdown")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("documentation command failed: %w", err)
	}

	// List generated files
	var docFiles []string
	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			docFiles = append(docFiles, path)
		}
		return nil
	})

	o.log("Generated %d documentation files", len(docFiles))
	return docFiles, err
}

func (o *ChangeOrchestrator) generateMutations(testFile string) ([]string, error) {
	o.log("Generating test mutations for %s...", testFile)

	mutationsDir := filepath.Join(o.outputDir, "mutations")

	cmd := exec.Command("mcp-test-mutate",
		"-test", testFile,
		"-output", mutationsDir,
		"-count", "5",
		"-strategies", "all")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mutation command failed: %w", err)
	}

	// List generated mutations
	var mutationFiles []string
	err := filepath.Walk(mutationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".txt") {
			mutationFiles = append(mutationFiles, path)
		}
		return nil
	})

	o.log("Generated %d test mutations", len(mutationFiles))
	return mutationFiles, err
}

func (o *ChangeOrchestrator) log(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	}
}

// Result structures
type ChangeResult struct {
	StartTime     time.Time                          `json:"start_time"`
	EndTime       time.Time                          `json:"end_time"`
	TotalDuration time.Duration                      `json:"total_duration"`
	Success       bool                               `json:"success"`
	Analysis      *changemanagement.AnalysisResult   `json:"analysis"`
	TestResults   *changemanagement.TestFinderResult `json:"test_results"`
	Documentation []string                           `json:"documentation"`
	Mutations     []string                           `json:"mutations"`
	Phases        []PhaseResult                      `json:"phases"`
}

type PhaseResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Artifacts []string      `json:"artifacts"`
	Error     string        `json:"error,omitempty"`
}
