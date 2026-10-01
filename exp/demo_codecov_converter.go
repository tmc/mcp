package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CodecovReport represents the JSON format for Codecov
type CodecovReport struct {
	Coverage map[string][]interface{}     `json:"coverage"`
	Messages map[string]map[string]string `json:"messages,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run demo_codecov_converter.go <input.out> <output.json>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read the coverage file
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	coverage := make(map[string][]interface{})
	fileData := make(map[string]map[int]int)
	fileLines := make(map[string]int)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		// Extract file and range
		filepath := parts[0]
		countStr := parts[2]

		count, _ := strconv.Atoi(countStr)

		// Extract line info
		rangePos := strings.Index(filepath, ":")
		if rangePos != -1 {
			actualFile := filepath[:rangePos]
			lineRange := filepath[rangePos+1:]

			// Parse line range
			rangeParts := strings.Split(lineRange, ",")
			if len(rangeParts) == 2 {
				startParts := strings.Split(rangeParts[0], ".")
				endParts := strings.Split(rangeParts[1], ".")

				if len(startParts) >= 2 && len(endParts) >= 2 {
					startLine, _ := strconv.Atoi(startParts[0])
					endLine, _ := strconv.Atoi(endParts[0])

					// Initialize file data
					if _, ok := fileData[actualFile]; !ok {
						fileData[actualFile] = make(map[int]int)
					}

					// Mark lines
					for line := startLine; line <= endLine; line++ {
						fileData[actualFile][line] = count
						if line > fileLines[actualFile] {
							fileLines[actualFile] = line
						}
					}
				}
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

	// Create report
	report := CodecovReport{
		Coverage: coverage,
		Messages: map[string]map[string]string{
			"_metadata": {
				"format": "codecov-json",
				"source": inputFile,
			},
		},
	}

	// Write JSON
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputFile, output, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Converted %s to Codecov JSON format: %s\n", inputFile, outputFile)
}
