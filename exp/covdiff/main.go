// Package main provides a tool to analyze coverage differences between test runs
// and identify which tests provide the most unique coverage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CoverageData represents coverage information for a test run
type CoverageData struct {
	TestName  string
	Packages  map[string]PackageCoverage
	TotalCov  float64
	UniqueCov float64
}

// PackageCoverage represents coverage for a single package
type PackageCoverage struct {
	Package    string
	Coverage   float64
	Statements int
	Covered    int
}

// CoverageDiff represents the difference analysis
type CoverageDiff struct {
	BaseDir       string
	CompareDir    string
	Differences   map[string]float64 // package -> delta coverage
	UniquelyAdded map[string]bool    // packages covered only in compare
}

func main() {
	var (
		baseDir    = flag.String("base", "", "Base coverage directory")
		compareDir = flag.String("compare", "", "Directory to compare against base")
		mode       = flag.String("mode", "diff", "Mode: diff, merge, intersect, subtract")
		outputDir  = flag.String("out", "coverage_diff", "Output directory")
		verbose    = flag.Bool("v", false, "Verbose output")
		jsonOutput = flag.Bool("json", false, "Output in JSON format")
	)

	flag.Parse()

	if *baseDir == "" && *compareDir == "" {
		flag.Usage()
		log.Fatal("Both -base and -compare are required")
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	switch *mode {
	case "diff":
		if err := runDiff(*baseDir, *compareDir, *outputDir, *verbose, *jsonOutput); err != nil {
			log.Fatalf("Diff failed: %v", err)
		}
	case "merge":
		if err := runMerge(*baseDir, *compareDir, *outputDir); err != nil {
			log.Fatalf("Merge failed: %v", err)
		}
	case "intersect":
		if err := runIntersect(*baseDir, *compareDir, *outputDir); err != nil {
			log.Fatalf("Intersect failed: %v", err)
		}
	case "subtract":
		if err := runSubtract(*baseDir, *compareDir, *outputDir); err != nil {
			log.Fatalf("Subtract failed: %v", err)
		}
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

// runDiff performs a coverage difference analysis
func runDiff(baseDir, compareDir, outputDir string, verbose, jsonOutput bool) error {
	fmt.Println("Analyzing coverage differences...")

	// Get coverage data from both directories
	baseCov, err := getCoverageData(baseDir)
	if err != nil {
		return fmt.Errorf("failed to get base coverage: %w", err)
	}

	compareCov, err := getCoverageData(compareDir)
	if err != nil {
		return fmt.Errorf("failed to get compare coverage: %w", err)
	}

	// Calculate differences
	diff := &CoverageDiff{
		BaseDir:       baseDir,
		CompareDir:    compareDir,
		Differences:   make(map[string]float64),
		UniquelyAdded: make(map[string]bool),
	}

	// Compare package by package
	for pkg, cmpPkg := range compareCov {
		basePkg, exists := baseCov[pkg]
		if !exists {
			diff.UniquelyAdded[pkg] = true
			diff.Differences[pkg] = cmpPkg.Coverage
		} else {
			delta := cmpPkg.Coverage - basePkg.Coverage
			if delta != 0 {
				diff.Differences[pkg] = delta
			}
		}
	}

	// Check for packages that were removed
	for pkg, basePkg := range baseCov {
		if _, exists := compareCov[pkg]; !exists {
			diff.Differences[pkg] = -basePkg.Coverage
		}
	}

	// Output results
	if jsonOutput {
		return outputDiffJSON(diff, filepath.Join(outputDir, "diff.json"))
	}

	printDiff(diff, verbose)
	return generateDiffReport(diff, outputDir)
}

// getCoverageData reads coverage data from a directory
func getCoverageData(dir string) (map[string]PackageCoverage, error) {
	cmd := exec.Command("go", "tool", "covdata", "percent", "-i="+dir)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage data: %w", err)
	}

	packages := make(map[string]PackageCoverage)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like: "package/path coverage: 85.2% of statements"
		parts := strings.Fields(line)
		if len(parts) >= 5 && parts[1] == "coverage:" {
			pkg := parts[0]
			var cov float64
			fmt.Sscanf(strings.TrimSuffix(parts[2], "%"), "%f", &cov)
			
			packages[pkg] = PackageCoverage{
				Package:  pkg,
				Coverage: cov,
			}
		}
	}

	return packages, nil
}

// runMerge merges coverage data from two directories
func runMerge(baseDir, compareDir, outputDir string) error {
	fmt.Printf("Merging coverage data from %s and %s...\n", baseDir, compareDir)
	
	mergeDir := filepath.Join(outputDir, "merged")
	if err := os.MkdirAll(mergeDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("go", "tool", "covdata", "merge",
		"-i="+baseDir+","+compareDir,
		"-o="+mergeDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("merge failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Merged coverage data written to: %s\n", mergeDir)
	
	// Show merged coverage
	showCmd := exec.Command("go", "tool", "covdata", "percent", "-i="+mergeDir)
	showOutput, _ := showCmd.Output()
	fmt.Printf("\nMerged Coverage:\n%s", string(showOutput))
	
	return nil
}

// runIntersect finds the intersection of coverage between two directories
func runIntersect(baseDir, compareDir, outputDir string) error {
	fmt.Printf("Finding intersection of coverage data...\n")
	
	intersectDir := filepath.Join(outputDir, "intersect")
	if err := os.MkdirAll(intersectDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("go", "tool", "covdata", "intersect",
		"-i="+baseDir+","+compareDir,
		"-o="+intersectDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("intersect failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Intersection written to: %s\n", intersectDir)
	
	// Show intersection coverage
	showCmd := exec.Command("go", "tool", "covdata", "percent", "-i="+intersectDir)
	showOutput, _ := showCmd.Output()
	fmt.Printf("\nIntersection Coverage:\n%s", string(showOutput))
	
	return nil
}

// runSubtract subtracts compare coverage from base coverage
func runSubtract(baseDir, compareDir, outputDir string) error {
	fmt.Printf("Subtracting %s from %s...\n", compareDir, baseDir)
	
	subtractDir := filepath.Join(outputDir, "subtract")
	if err := os.MkdirAll(subtractDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("go", "tool", "covdata", "subtract",
		"-i="+baseDir+","+compareDir,
		"-o="+subtractDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("subtract failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Subtraction result written to: %s\n", subtractDir)
	
	// Show subtraction coverage
	showCmd := exec.Command("go", "tool", "covdata", "percent", "-i="+subtractDir)
	showOutput, _ := showCmd.Output()
	fmt.Printf("\nSubtraction Coverage:\n%s", string(showOutput))
	
	return nil
}

// printDiff prints the difference analysis
func printDiff(diff *CoverageDiff, verbose bool) {
	fmt.Println("\n=== Coverage Difference Analysis ===")
	fmt.Printf("Base: %s\n", diff.BaseDir)
	fmt.Printf("Compare: %s\n\n", diff.CompareDir)

	// Sort packages by absolute difference
	type pkgDiff struct {
		pkg   string
		delta float64
	}
	
	var diffs []pkgDiff
	for pkg, delta := range diff.Differences {
		diffs = append(diffs, pkgDiff{pkg, delta})
	}
	
	sort.Slice(diffs, func(i, j int) bool {
		return abs(diffs[i].delta) > abs(diffs[j].delta)
	})

	// Show improvements
	fmt.Println("Coverage Improvements:")
	count := 0
	for _, d := range diffs {
		if d.delta > 0 {
			fmt.Printf("  %s: +%.1f%%\n", d.pkg, d.delta)
			count++
			if !verbose && count >= 10 {
				fmt.Printf("  ... and %d more\n", len(diffs)-count)
				break
			}
		}
	}
	if count == 0 {
		fmt.Println("  None")
	}

	// Show regressions
	fmt.Println("\nCoverage Regressions:")
	count = 0
	for _, d := range diffs {
		if d.delta < 0 {
			fmt.Printf("  %s: %.1f%%\n", d.pkg, d.delta)
			count++
			if !verbose && count >= 10 {
				fmt.Printf("  ... and %d more\n", len(diffs)-count)
				break
			}
		}
	}
	if count == 0 {
		fmt.Println("  None")
	}

	// Show uniquely added packages
	if len(diff.UniquelyAdded) > 0 {
		fmt.Println("\nNewly Covered Packages:")
		for pkg := range diff.UniquelyAdded {
			fmt.Printf("  %s: %.1f%%\n", pkg, diff.Differences[pkg])
		}
	}
}

// outputDiffJSON writes the diff analysis to JSON
func outputDiffJSON(diff *CoverageDiff, filename string) error {
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// generateDiffReport creates a markdown report
func generateDiffReport(diff *CoverageDiff, outputDir string) error {
	reportPath := filepath.Join(outputDir, "diff_report.md")
	f, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# Coverage Difference Report\n\n")
	fmt.Fprintf(f, "Base: `%s`\n", diff.BaseDir)
	fmt.Fprintf(f, "Compare: `%s`\n\n", diff.CompareDir)

	// Calculate summary statistics
	var improvements, regressions, unchanged int
	var totalImprovement, totalRegression float64
	
	for _, delta := range diff.Differences {
		if delta > 0 {
			improvements++
			totalImprovement += delta
		} else if delta < 0 {
			regressions++
			totalRegression += delta
		} else {
			unchanged++
		}
	}

	fmt.Fprintf(f, "## Summary\n\n")
	fmt.Fprintf(f, "- Packages with improvements: %d\n", improvements)
	fmt.Fprintf(f, "- Packages with regressions: %d\n", regressions)
	fmt.Fprintf(f, "- Packages unchanged: %d\n", unchanged)
	fmt.Fprintf(f, "- Newly covered packages: %d\n", len(diff.UniquelyAdded))
	fmt.Fprintf(f, "- Total coverage change: %+.1f%%\n\n", totalImprovement+totalRegression)

	// Write detailed changes
	fmt.Fprintf(f, "## Detailed Changes\n\n")
	fmt.Fprintf(f, "| Package | Change | Type |\n")
	fmt.Fprintf(f, "|---------|--------|------|\n")
	
	// Sort by absolute change
	type change struct {
		pkg   string
		delta float64
	}
	var changes []change
	for pkg, delta := range diff.Differences {
		changes = append(changes, change{pkg, delta})
	}
	sort.Slice(changes, func(i, j int) bool {
		return abs(changes[i].delta) > abs(changes[j].delta)
	})

	for _, c := range changes {
		changeType := "Changed"
		if diff.UniquelyAdded[c.pkg] {
			changeType = "New"
		} else if c.delta > 0 {
			changeType = "Improved"
		} else if c.delta < 0 {
			changeType = "Regressed"
		}
		fmt.Fprintf(f, "| %s | %+.1f%% | %s |\n", c.pkg, c.delta, changeType)
	}

	return nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}