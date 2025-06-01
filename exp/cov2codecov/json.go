package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// convertToJSON converts binary coverage data to Codecov JSON format
func convertToJSON(inputDir, outputFile string, packages string, testInfo string, verbose bool) error {
	if verbose {
		fmt.Printf("Converting %s to JSON format\n", inputDir)
	}

	// First convert to text format in a temp file
	tempFile, err := ioutil.TempFile("", "coverage-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	if err := convertToText(inputDir, tempFile.Name(), packages, verbose); err != nil {
		return err
	}

	// Parse text format to JSON
	return parseTextToJSON(tempFile.Name(), outputFile, testInfo)
}

// parseTextToJSON converts Go coverage text format to Codecov JSON format
func parseTextToJSON(textFile, jsonFile string, testInfo string) error {
	data, err := ioutil.ReadFile(textFile)
	if err != nil {
		return fmt.Errorf("failed to read text file: %w", err)
	}

	coverage := make(map[string][]interface{})
	fileData := make(map[string]map[int]int) // file -> line -> hit count
	fileLines := make(map[string]int)        // file -> max line number

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
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
				if existing, ok := fileData[file][line]; ok {
					fileData[file][line] = existing + count
				} else {
					fileData[file][line] = count
				}
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

	// Create Codecov report
	report := CodecovReport{
		Coverage: coverage,
		Messages: make(map[string]map[string]string),
	}

	// Add metadata
	report.Messages["_metadata"] = map[string]string{
		"generated":  time.Now().Format(time.RFC3339),
		"generator":  "cov2codecov",
		"format":     "json",
	}

	if testInfo != "" {
		report.Messages["_metadata"]["test_info"] = testInfo
	}

	// Write JSON output
	data, err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return ioutil.WriteFile(jsonFile, data, 0644)
}