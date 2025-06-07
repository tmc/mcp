package fuzzing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// CoverageFeedback provides feedback to the fuzzer based on coverage data
type CoverageFeedback struct {
	mu               sync.RWMutex
	baselineCoverage map[string]float64 // package -> coverage percent
	coverageDir      string
	testCounter      atomic.Int64
}

// NewCoverageFeedback creates a new coverage feedback system
func NewCoverageFeedback(coverageDir string) (*CoverageFeedback, error) {
	if coverageDir == "" {
		coverageDir = os.Getenv("GOCOVERDIR")
		if coverageDir == "" {
			var err error
			coverageDir, err = os.MkdirTemp("", "mcp-fuzz-coverage-*")
			if err != nil {
				return nil, fmt.Errorf("failed to create temp coverage dir: %w", err)
			}
		}
	}

	cf := &CoverageFeedback{
		baselineCoverage: make(map[string]float64),
		coverageDir:      coverageDir,
	}

	// Load initial baseline
	if err := cf.loadBaseline(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load baseline: %w", err)
	}

	return cf, nil
}

// loadBaseline loads the current coverage state
func (cf *CoverageFeedback) loadBaseline() error {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	// Use go tool covdata to get current coverage
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i", cf.coverageDir)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Parse output (format: package coverage%)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip the header line
		if strings.HasPrefix(line, "github.com/tmc/mcp") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkg := parts[0]
				coverage := strings.TrimSuffix(parts[1], "%")
				if percent, err := strconv.ParseFloat(coverage, 64); err == nil {
					cf.baselineCoverage[pkg] = percent
				}
			}
		}
	}

	return nil
}

// RunWithCoverage runs a test function and collects coverage data
func (cf *CoverageFeedback) RunWithCoverage(t *testing.T, testFunc func()) (*TestCoverageResult, error) {
	testID := cf.testCounter.Add(1)
	testName := fmt.Sprintf("fuzz_%d", testID)
	testDir := filepath.Join(cf.coverageDir, testName)

	// Create test-specific directory
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create test dir: %w", err)
	}

	// Save current GOCOVERDIR and set it to test-specific dir
	origCoverDir := os.Getenv("GOCOVERDIR")
	t.Setenv("GOCOVERDIR", testDir)

	// Run the test
	testFunc()

	// Restore original GOCOVERDIR
	if origCoverDir != "" {
		t.Setenv("GOCOVERDIR", origCoverDir)
	}

	// Analyze coverage for this test
	result := &TestCoverageResult{
		TestID:   testID,
		TestName: testName,
		Dir:      testDir,
	}

	// Get coverage data for this test
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i", testDir)
	output, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("failed to get coverage: %w", err)
	}

	// Parse and compare with baseline
	newCoverage := make(map[string]float64)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "github.com/tmc/mcp") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkg := parts[0]
				coverage := strings.TrimSuffix(parts[1], "%")
				if percent, err := strconv.ParseFloat(coverage, 64); err == nil {
					newCoverage[pkg] = percent

					// Check if coverage increased
					if baseline, exists := cf.baselineCoverage[pkg]; exists {
						increase := percent - baseline
						if increase > 0 {
							result.CoverageIncrease += increase
							result.NewPackages = append(result.NewPackages, pkg)
						}
					} else {
						// New package covered
						result.CoverageIncrease += percent
						result.NewPackages = append(result.NewPackages, pkg)
					}
				}
			}
		}
	}

	result.TotalCoverage = cf.calculateTotalCoverage(newCoverage)
	return result, nil
}

// calculateTotalCoverage calculates the average coverage across all packages
func (cf *CoverageFeedback) calculateTotalCoverage(coverage map[string]float64) float64 {
	if len(coverage) == 0 {
		return 0.0
	}

	total := 0.0
	for _, percent := range coverage {
		total += percent
	}
	return total / float64(len(coverage))
}

// UpdateBaseline updates the baseline with new coverage data
func (cf *CoverageFeedback) UpdateBaseline(result *TestCoverageResult) error {
	if result.CoverageIncrease <= 0 {
		return nil // No improvement, don't update
	}

	// Merge the test coverage data into the main coverage directory
	cmd := exec.Command("go", "tool", "covdata", "merge", "-i", result.Dir, "-o", cf.coverageDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge coverage: %w", err)
	}

	// Reload baseline
	return cf.loadBaseline()
}

// TestCoverageResult holds the results of a coverage-guided test run
type TestCoverageResult struct {
	TestID           int64
	TestName         string
	Dir              string
	TotalCoverage    float64
	CoverageIncrease float64
	NewPackages      []string
	Duration         time.Duration
}

// Score returns a score for this test result (higher is better)
func (r *TestCoverageResult) Score() float64 {
	// Weight new coverage heavily
	score := r.CoverageIncrease * 100.0

	// Bonus for covering new packages
	score += float64(len(r.NewPackages)) * 10.0

	// Small penalty for time (prefer faster tests with same coverage)
	if r.Duration > 0 {
		score -= r.Duration.Seconds() * 0.1
	}

	return score
}

// CoverageGuidedFuzzer integrates coverage feedback with fuzzing
type CoverageGuidedFuzzer struct {
	generator     *FuzzGenerator
	feedback      *CoverageFeedback
	goodInputs    []string
	inputScores   map[string]float64
	mu            sync.RWMutex
	maxGoodInputs int
	minScore      float64
}

// NewCoverageGuidedFuzzer creates a new coverage-guided fuzzer
func NewCoverageGuidedFuzzer(generator *FuzzGenerator, feedback *CoverageFeedback) *CoverageGuidedFuzzer {
	return &CoverageGuidedFuzzer{
		generator:     generator,
		feedback:      feedback,
		goodInputs:    make([]string, 0),
		inputScores:   make(map[string]float64),
		maxGoodInputs: 100, // Keep best 100 inputs
		minScore:      1.0, // Minimum score to keep an input
	}
}

// GenerateInput generates a new input, potentially mutating a good input
func (cgf *CoverageGuidedFuzzer) GenerateInput() string {
	cgf.mu.RLock()
	hasGoodInputs := len(cgf.goodInputs) > 0
	cgf.mu.RUnlock()

	// 50% chance to mutate an existing good input if we have any
	if hasGoodInputs && cgf.generator.rng.Float64() < 0.5 {
		cgf.mu.RLock()
		baseInput := cgf.goodInputs[cgf.generator.rng.Intn(len(cgf.goodInputs))]
		cgf.mu.RUnlock()

		return cgf.mutateInput(baseInput)
	}

	// Otherwise generate a new random input
	return cgf.generator.Generate()
}

// mutateInput applies mutations to an existing input
func (cgf *CoverageGuidedFuzzer) mutateInput(input string) string {
	lines := strings.Split(input, "\n")

	// Apply random mutations
	mutationType := cgf.generator.rng.Intn(5)
	switch mutationType {
	case 0: // Add a line
		newLine := cgf.generateSingleCommand()
		position := cgf.generator.rng.Intn(len(lines) + 1)
		lines = append(lines[:position], append([]string{newLine}, lines[position:]...)...)

	case 1: // Remove a line
		if len(lines) > 1 {
			position := cgf.generator.rng.Intn(len(lines))
			lines = append(lines[:position], lines[position+1:]...)
		}

	case 2: // Modify a line
		if len(lines) > 0 {
			position := cgf.generator.rng.Intn(len(lines))
			lines[position] = cgf.generateSingleCommand()
		}

	case 3: // Duplicate a line
		if len(lines) > 0 {
			position := cgf.generator.rng.Intn(len(lines))
			lines = append(lines[:position], append([]string{lines[position]}, lines[position:]...)...)
		}

	case 4: // Swap two lines
		if len(lines) > 1 {
			i := cgf.generator.rng.Intn(len(lines))
			j := cgf.generator.rng.Intn(len(lines))
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	return strings.Join(lines, "\n")
}

// generateSingleCommand generates a single command
func (cgf *CoverageGuidedFuzzer) generateSingleCommand() string {
	// Reuse generator logic but get just one command
	script := cgf.generator.Generate()
	lines := strings.Split(script, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return "echo test"
}

// RecordResult records the result of a test run
func (cgf *CoverageGuidedFuzzer) RecordResult(input string, result *TestCoverageResult) {
	score := result.Score()

	cgf.mu.Lock()
	defer cgf.mu.Unlock()

	cgf.inputScores[input] = score

	// Keep the input if it's good enough
	if score >= cgf.minScore {
		cgf.goodInputs = append(cgf.goodInputs, input)

		// Trim to max size, keeping best inputs
		if len(cgf.goodInputs) > cgf.maxGoodInputs {
			// Simple strategy: keep most recent
			// Better strategy would sort by score
			cgf.goodInputs = cgf.goodInputs[len(cgf.goodInputs)-cgf.maxGoodInputs:]
		}
	}
}

// Run executes a scripttest directly with fuzzing support
func Run(testFunc func(script string) error, opts RunOptions) error {
	// Initialize coverage feedback
	feedback, err := NewCoverageFeedback(opts.CoverageDir)
	if err != nil {
		return fmt.Errorf("failed to initialize coverage feedback: %w", err)
	}

	// Initialize fuzzer
	generator := NewFuzzGenerator(int64(time.Now().UnixNano()))
	fuzzer := NewCoverageGuidedFuzzer(generator, feedback)

	// Set up visualizer if provided
	viz := opts.Visualizer
	if viz != nil {
		defer viz.Close()
	}

	// Run fuzzing iterations
	for i := 0; i < opts.Iterations; i++ {
		input := fuzzer.GenerateInput()

		// Notify visualizer of test start
		if viz != nil {
			viz.StartTest(input)
		}

		// Create a test context for coverage collection
		t := &testing.T{}

		// Run the test
		var testErr error
		result, err := feedback.RunWithCoverage(t, func() {
			testErr = testFunc(input)
		})

		// Update visualizer based on result
		if viz != nil {
			if testErr == nil {
				viz.AcceptScript(input)
			} else {
				viz.RejectScript(input, testErr)
			}
		}

		if err == nil && testErr == nil {
			fuzzer.RecordResult(input, result)

			// Update baseline if coverage improved
			if result.CoverageIncrease > 0 {
				feedback.UpdateBaseline(result)

				if viz != nil {
					viz.UpdateCoverage(result.TotalCoverage, len(result.NewPackages))
				}

				if opts.Verbose && viz == nil {
					fmt.Printf("Iteration %d: Coverage increased by %.2f%% (total: %.2f%%)\n",
						i, result.CoverageIncrease, result.TotalCoverage)
				}
			}
		}

		// Check if we should stop early
		if opts.MinCoverage > 0 && result.TotalCoverage >= opts.MinCoverage {
			if opts.Verbose && viz == nil {
				fmt.Printf("Reached target coverage %.2f%% after %d iterations\n",
					result.TotalCoverage, i+1)
			}
			break
		}
	}

	return nil
}

// RunOptions configures the Run function
type RunOptions struct {
	Iterations  int     // Number of fuzzing iterations
	CoverageDir string  // Directory for coverage data
	MinCoverage float64 // Stop when this coverage is reached
	Verbose     bool    // Enable verbose output
	Timeout     time.Duration
	Visualizer  *Visualizer // Optional visualizer for live display
}

// DefaultRunOptions returns default options
func DefaultRunOptions() RunOptions {
	return RunOptions{
		Iterations:  1000,
		CoverageDir: "",
		MinCoverage: 0,
		Verbose:     false,
		Timeout:     5 * time.Minute,
	}
}
