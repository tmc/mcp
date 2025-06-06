// Package main provides a tool to convert Go's binary coverage data to Codecov-compatible format
// It handles merging unit and integration test coverage and outputs in text or JSON format.
// Supports both line and branch coverage information when available.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// CodecovReport represents the JSON format for Codecov
type CodecovReport struct {
	Coverage map[string][]interface{}     `json:"coverage"`
	Messages map[string]map[string]string `json:"messages,omitempty"`
}

func main() {
	var (
		unitCovDir    = flag.String("unit", "", "Unit test coverage directory (GOCOVERDIR)")
		integCovDir   = flag.String("integ", "", "Integration test coverage directory")
		inputDirs     = flag.String("input", "", "Comma-separated list of coverage directories")
		outputFile    = flag.String("output", "coverage.txt", "Output file for coverage")
		jsonFormat    = flag.Bool("json", false, "Output in Codecov JSON format instead of text")
		mergeDir      = flag.String("merge", "", "Directory to store merged binary coverage")
		packages      = flag.String("pkg", "", "Filter packages (comma-separated)")
		verbose       = flag.Bool("v", false, "Verbose output")
		skipMerge     = flag.Bool("skip-merge", false, "Skip merge step and convert directly")
		codecovUpload = flag.Bool("upload", false, "Upload to Codecov after conversion")
		codecovToken  = flag.String("token", "", "Codecov upload token")
		codecovFlags  = flag.String("flags", "", "Codecov flags (comma-separated)")
		testInfo      = flag.String("test-info", "", "Add test information to JSON output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Converts Go's binary coverage data to Codecov-compatible format.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Convert a single coverage directory to text\n")
		fmt.Fprintf(os.Stderr, "  %s -input coverage/integration -output coverage.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Convert to JSON format with test info\n")
		fmt.Fprintf(os.Stderr, "  %s -input coverage -output coverage.json -json -test-info 'TestFoo'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Merge unit and integration coverage\n")
		fmt.Fprintf(os.Stderr, "  %s -unit coverage/unit -integ coverage/integ -output coverage.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Multiple directories with package filter\n")
		fmt.Fprintf(os.Stderr, "  %s -input dir1,dir2,dir3 -pkg github.com/myproject -output coverage.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate inputs
	inputList := []string{}
	if *unitCovDir != "" {
		inputList = append(inputList, *unitCovDir)
	}
	if *integCovDir != "" {
		inputList = append(inputList, *integCovDir)
	}
	if *inputDirs != "" {
		inputList = append(inputList, strings.Split(*inputDirs, ",")...)
	}

	if len(inputList) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No input directories specified\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate directories exist
	for _, dir := range inputList {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Fatalf("Directory does not exist: %s", dir)
		}
	}

	// Merge if multiple directories and not skipping merge
	finalDir := inputList[0]
	if len(inputList) > 1 && !*skipMerge {
		if *mergeDir == "" {
			tempDir, err := os.MkdirTemp("", "coverage-merge-*")
			if err != nil {
				log.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)
			*mergeDir = tempDir
		}

		if err := mergeDirectories(inputList, *mergeDir, *verbose); err != nil {
			log.Fatalf("Failed to merge coverage: %v", err)
		}
		finalDir = *mergeDir
	}

	// Convert to requested format
	if *jsonFormat {
		if err := convertToJSON(finalDir, *outputFile, *packages, *testInfo, *verbose); err != nil {
			log.Fatalf("Failed to convert to JSON: %v", err)
		}
	} else {
		if err := convertToText(finalDir, *outputFile, *packages, *verbose); err != nil {
			log.Fatalf("Failed to convert to text: %v", err)
		}
	}

	fmt.Printf("Coverage data converted to: %s\n", *outputFile)

	// Show coverage summary
	if *verbose && !*jsonFormat {
		showSummary(*outputFile)
	}

	// Upload to Codecov if requested
	if *codecovUpload {
		if err := uploadToCodecov(*outputFile, *codecovToken, *codecovFlags, *verbose); err != nil {
			log.Fatalf("Failed to upload to Codecov: %v", err)
		}
	}
}

// mergeDirectories merges multiple coverage directories into one
func mergeDirectories(inputDirs []string, outputDir string, verbose bool) error {
	if verbose {
		fmt.Printf("Merging %d directories into %s\n", len(inputDirs), outputDir)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create merge directory: %w", err)
	}

	// Build merge command
	args := []string{"tool", "covdata", "merge"}
	args = append(args, "-i="+strings.Join(inputDirs, ","))
	args = append(args, "-o="+outputDir)

	cmd := exec.Command("go", args...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	return nil
}

// convertToText converts binary coverage data to text format
func convertToText(inputDir, outputFile string, packages string, verbose bool) error {
	if verbose {
		fmt.Printf("Converting %s to text format\n", inputDir)
	}

	// Build textfmt command
	args := []string{"tool", "covdata", "textfmt"}
	args = append(args, "-i="+inputDir)
	args = append(args, "-o="+outputFile)

	// Add package filter if specified
	if packages != "" {
		args = append(args, "-pkg="+packages)
	}

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("textfmt failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// showSummary displays a coverage summary from the text file
func showSummary(coverageFile string) {
	// Run go tool cover -func on the output file
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to show summary: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		// Show last line (total coverage)
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				fmt.Printf("\nTotal %s\n", lines[i])
				break
			}
		}
	}
}

// uploadToCodecov uploads the coverage file to Codecov
func uploadToCodecov(coverageFile, token, flags string, verbose bool) error {
	fmt.Println("Uploading to Codecov...")

	// First, check if codecov CLI is available
	codecovCmd := "codecov"
	if _, err := exec.LookPath(codecovCmd); err != nil {
		// Try the bash uploader as fallback
		fmt.Println("Codecov CLI not found, trying bash uploader...")
		return uploadUsingBashUploader(coverageFile, token, flags, verbose)
	}

	// Build codecov command
	args := []string{}
	args = append(args, "--file", coverageFile)

	if token != "" {
		args = append(args, "--token", token)
	}

	if flags != "" {
		args = append(args, "--flag", flags)
	}

	if verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.Command(codecovCmd, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codecov upload failed: %w", err)
	}

	fmt.Println("Coverage uploaded successfully!")
	return nil
}

// uploadUsingBashUploader uses the Codecov bash uploader as a fallback
func uploadUsingBashUploader(coverageFile, token, flags string, verbose bool) error {
	// Download and run the bash uploader
	curlCmd := exec.Command("curl", "-s", "https://codecov.io/bash")
	bashCmd := exec.Command("bash", "-s", "--", "-f", coverageFile)

	if token != "" {
		bashCmd.Args = append(bashCmd.Args, "-t", token)
	}

	if flags != "" {
		bashCmd.Args = append(bashCmd.Args, "-F", flags)
	}

	if verbose {
		bashCmd.Args = append(bashCmd.Args, "-v")
	}

	// Pipe curl output to bash
	pipe, err := curlCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	bashCmd.Stdin = pipe
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr

	if err := curlCmd.Start(); err != nil {
		return fmt.Errorf("failed to start curl: %w", err)
	}

	if err := bashCmd.Start(); err != nil {
		return fmt.Errorf("failed to start bash: %w", err)
	}

	if err := curlCmd.Wait(); err != nil {
		return fmt.Errorf("curl failed: %w", err)
	}

	if err := bashCmd.Wait(); err != nil {
		return fmt.Errorf("bash uploader failed: %w", err)
	}

	return nil
}

// getCoverageInfo extracts basic information about coverage directories
func getCoverageInfo(dir string) error {
	cmd := exec.Command("go", "tool", "covdata", "pkglist", "-i="+dir)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	packages := strings.Split(strings.TrimSpace(string(output)), "\n")
	fmt.Printf("Directory %s contains coverage for %d packages\n", dir, len(packages))

	// Show coverage percentages
	cmd = exec.Command("go", "tool", "covdata", "percent", "-i="+dir)
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get percentages: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i < 5 && line != "" { // Show first 5 packages
			fmt.Printf("  %s\n", line)
		}
	}
	if len(lines) > 5 {
		fmt.Printf("  ... and %d more packages\n", len(lines)-5)
	}

	return nil
}
