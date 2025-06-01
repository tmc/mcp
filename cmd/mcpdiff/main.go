// Command mcpdiff compares two MCP trace files and highlights differences.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ANSI color codes
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
	bgRed   = "\033[41m"
	bgGreen = "\033[42m"
)

var (
	ignoreTimestamps = flag.Bool("t", true, "ignore timestamps when comparing")
	ignoreIDs        = flag.Bool("i", false, "ignore IDs when comparing")
	semanticCompare  = flag.Bool("s", false, "compare JSON values semantically (ignoring formatting)")
	verbose          = flag.Bool("v", false, "verbose output (show all lines being considered)")
	ignoreOrder      = flag.Bool("o", false, "ignore order of messages (best effort matching)")
	noColor          = flag.Bool("no-color", false, "disable colorized output")
	contextLines     = flag.Int("c", 3, "number of context lines to show")
	onlyDiffs        = flag.Bool("d", false, "only show lines that differ")
	jsonOutput       = flag.Bool("json", false, "output differences in JSON format")
	wordDiff         = flag.Bool("word-diff", false, "show word-level differences (like git --word-diff)")
)

// MCPRecord represents a parsed line from an MCP trace file
type MCPRecord struct {
	Direction  string         // "send" or "recv"
	RawContent string         // Raw JSON content string
	JSON       map[string]any // Parsed JSON
	Timestamp  float64        // Unix timestamp with milliseconds as decimal
	LineNum    int            // Line number in original file
}

// Regular expression to parse MCP records:
// - Group 1: Direction (send/recv)
// - Group 2: JSON content
// - Group 3: Timestamp seconds
// - Group 4: Timestamp milliseconds (optional)
var recordRegex = regexp.MustCompile(`^mcp-(\w+)\s+(\{.*\})\s+#\s+(\d+)(?:\.(\d+))?$`)

// parseFile reads an MCP trace file and returns structured records
func parseFile(filename string) ([]MCPRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filename, err)
	}
	defer file.Close()

	var records []MCPRecord
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines or non-MCP lines
		if line == "" || !strings.HasPrefix(line, "mcp-") {
			continue
		}

		match := recordRegex.FindStringSubmatch(line)
		if match == nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse line %d in %s: %s\n", lineNum, filename, line)
			continue
		}

		direction := match[1]
		jsonContent := match[2]
		seconds, _ := strconv.ParseInt(match[3], 10, 64)

		var milliseconds int64
		if match[4] != "" {
			// Ensure we have exactly 3 digits for milliseconds
			ms := match[4]
			for len(ms) < 3 {
				ms += "0"
			}
			milliseconds, _ = strconv.ParseInt(ms[:3], 10, 64)
		}

		timestamp := float64(seconds) + float64(milliseconds)/1000.0

		// Parse JSON
		var jsonData map[string]any
		if err := json.Unmarshal([]byte(jsonContent), &jsonData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid JSON at line %d in %s: %s\n", lineNum, filename, err)
			// Store an empty map to avoid nil checks later
			jsonData = make(map[string]any)
		}

		records = append(records, MCPRecord{
			Direction:  direction,
			RawContent: jsonContent,
			JSON:       jsonData,
			Timestamp:  timestamp,
			LineNum:    lineNum,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filename, err)
	}

	return records, nil
}

// compareRecords determines if two records are equal based on flags
func compareRecords(a, b MCPRecord) bool {
	if a.Direction != b.Direction {
		return false
	}

	// If we're not ignoring timestamps and they differ, records are different
	if !*ignoreTimestamps && a.Timestamp != b.Timestamp {
		return false
	}

	// For simple comparison, just check raw content
	if !*semanticCompare {
		return a.RawContent == b.RawContent
	}

	// For semantic comparison, normalize and compare the JSON
	normalizedA := normalizeJSON(a.JSON)
	normalizedB := normalizeJSON(b.JSON)

	return compareJSON(normalizedA, normalizedB)
}

// normalizeJSON creates a copy of the JSON with IDs removed if needed
func normalizeJSON(data map[string]any) map[string]any {
	// Create a deep copy to avoid modifying the original
	result := make(map[string]any, len(data))
	for k, v := range data {
		// Skip ID field if flag is set
		if *ignoreIDs && k == "id" {
			continue
		}

		// Handle nested maps
		if nestedMap, ok := v.(map[string]any); ok {
			result[k] = normalizeJSON(nestedMap)
		} else if nestedSlice, ok := v.([]any); ok {
			// Handle nested slices
			normalizedSlice := make([]any, len(nestedSlice))
			for i, item := range nestedSlice {
				if nestedItemMap, ok := item.(map[string]any); ok {
					normalizedSlice[i] = normalizeJSON(nestedItemMap)
				} else {
					normalizedSlice[i] = item
				}
			}
			result[k] = normalizedSlice
		} else {
			// Copy primitive values directly
			result[k] = v
		}
	}
	return result
}

// compareJSON checks if two JSON objects are equivalent
func compareJSON(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for k, aVal := range a {
		bVal, exists := b[k]
		if !exists {
			return false
		}

		// Compare based on type
		switch av := aVal.(type) {
		case map[string]any:
			// For nested objects, recurse
			if bv, ok := bVal.(map[string]any); ok {
				if !compareJSON(av, bv) {
					return false
				}
			} else {
				return false
			}
		case []any:
			// For arrays, check if the elements match
			if bv, ok := bVal.([]any); ok {
				if !compareArrays(av, bv) {
					return false
				}
			} else {
				return false
			}
		default:
			// For primitive values, direct comparison
			if aVal != bVal {
				return false
			}
		}
	}
	return true
}

// compareArrays checks if two arrays have equivalent items
// (potentially in different order if ignoreOrder is set)
func compareArrays(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}

	if *ignoreOrder {
		// Check each element exists in both arrays
		// Note: this is O(n²) so could be slow for large arrays
		for _, aItem := range a {
			found := false
			for _, bItem := range b {
				if deepEqual(aItem, bItem) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	} else {
		// Check elements at same positions
		for i := range a {
			if !deepEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	}
}

// deepEqual compares any two values recursively
func deepEqual(a, b any) bool {
	// Handle nil values
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch av := a.(type) {
	case map[string]any:
		if bv, ok := b.(map[string]any); ok {
			return compareJSON(av, bv)
		}
	case []any:
		if bv, ok := b.([]any); ok {
			return compareArrays(av, bv)
		}
	default:
		return a == b
	}

	return false
}

// formatJSONDiff highlights differences between two JSON strings
func formatJSONDiff(a, b string) (string, string) {
	// Parse the JSON and re-format with indentation for better readability
	var aData, bData any
	if err := json.Unmarshal([]byte(a), &aData); err != nil {
		return a, b
	}
	if err := json.Unmarshal([]byte(b), &bData); err != nil {
		return a, b
	}

	aFormatted, _ := json.MarshalIndent(aData, "", "  ")
	bFormatted, _ := json.MarshalIndent(bData, "", "  ")

	if *noColor {
		return string(aFormatted), string(bFormatted)
	}

	// Basic line-by-line diff highlighting
	aLines := strings.Split(string(aFormatted), "\n")
	bLines := strings.Split(string(bFormatted), "\n")

	// Compare lines and highlight differences
	minLen := min(len(aLines), len(bLines))
	for i := 0; i < minLen; i++ {
		if aLines[i] != bLines[i] {
			aLines[i] = red + aLines[i] + reset
			bLines[i] = green + bLines[i] + reset
		}
	}

	// Highlight extra lines in each file
	for i := minLen; i < len(aLines); i++ {
		aLines[i] = red + aLines[i] + reset
	}
	for i := minLen; i < len(bLines); i++ {
		bLines[i] = green + bLines[i] + reset
	}

	return strings.Join(aLines, "\n"), strings.Join(bLines, "\n")
}

// JSONDiffResult represents a result for JSON output
type JSONDiffResult struct {
	File1      string         `json:"file1"`
	File2      string         `json:"file2"`
	Matches    bool           `json:"matches"`
	TotalDiffs int            `json:"total_diffs"`
	Diffs      []JSONDiffHunk `json:"diffs,omitempty"`
}

// JSONDiffHunk represents a hunk of differences for JSON output
type JSONDiffHunk struct {
	StartLine1 int         `json:"start_line1"`
	LineCount1 int         `json:"line_count1"`
	StartLine2 int         `json:"start_line2"`
	LineCount2 int         `json:"line_count2"`
	Removed    []MCPRecord `json:"removed,omitempty"`
	Added      []MCPRecord `json:"added,omitempty"`
}

// diffFiles compares two MCP trace files and prints differences in unified diff format
func diffFiles(file1, file2 string) {
	records1, err := parseFile(file1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file1, err)
		os.Exit(1)
	}

	records2, err := parseFile(file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file2, err)
		os.Exit(1)
	}

	// Print all records if verbose mode is enabled
	if *verbose {
		fmt.Println("File 1 records:")
		for i, rec := range records1 {
			fmt.Printf("  [%d] %s %s\n", i, rec.Direction, rec.RawContent)
		}

		fmt.Println("\nFile 2 records:")
		for i, rec := range records2 {
			fmt.Printf("  [%d] %s %s\n", i, rec.Direction, rec.RawContent)
		}

		fmt.Println("\nBeginning comparison...")
	}

	// For JSON output, we'll collect the differences in a structured format
	var jsonResult *JSONDiffResult
	if *jsonOutput {
		jsonResult = &JSONDiffResult{
			File1: file1,
			File2: file2,
			Diffs: []JSONDiffHunk{},
		}
	} else {
		// Print the unified diff header
		if !*noColor {
			fmt.Printf("%s--- %s%s\n", brightCyan, file1, reset)
			fmt.Printf("%s+++ %s%s\n", brightCyan, file2, reset)
		} else {
			fmt.Printf("--- %s\n", file1)
			fmt.Printf("+++ %s\n", file2)
		}
	}

	// Find the diffs
	diffs := []struct {
		pos1, pos2   int    // position in each file
		type1, type2 string // type of line: "+", "-", " "
	}{}

	i, j := 0, 0
	for i < len(records1) || j < len(records2) {
		// Both files have records at current position
		if i < len(records1) && j < len(records2) {
			if compareRecords(records1[i], records2[j]) {
				// Records match
				diffs = append(diffs, struct {
					pos1, pos2   int
					type1, type2 string
				}{i, j, " ", " "})
				i++
				j++
			} else {
				// Records don't match - try to find the next match
				matchFound := false

				// Look ahead in file 2 for a match to the current record in file 1
				for k := j + 1; k < min(j+10, len(records2)); k++ {
					if compareRecords(records1[i], records2[k]) {
						// Found a match ahead in file 2
						// Mark all intervening records in file 2 as additions
						for l := j; l < k; l++ {
							diffs = append(diffs, struct {
								pos1, pos2   int
								type1, type2 string
							}{-1, l, "", "+"})
						}
						j = k
						matchFound = true
						break
					}
				}

				if !matchFound {
					// Look ahead in file 1 for a match to the current record in file 2
					for k := i + 1; k < min(i+10, len(records1)); k++ {
						if compareRecords(records1[k], records2[j]) {
							// Found a match ahead in file 1
							// Mark all intervening records in file 1 as deletions
							for l := i; l < k; l++ {
								diffs = append(diffs, struct {
									pos1, pos2   int
									type1, type2 string
								}{l, -1, "-", ""})
							}
							i = k
							matchFound = true
							break
						}
					}
				}

				if !matchFound {
					// No match found in lookahead - mark as a direct change
					diffs = append(diffs, struct {
						pos1, pos2   int
						type1, type2 string
					}{i, j, "-", "+"})
					i++
					j++
				}
			}
		} else if i < len(records1) {
			// Only file 1 has records left - mark as deletions
			diffs = append(diffs, struct {
				pos1, pos2   int
				type1, type2 string
			}{i, -1, "-", ""})
			i++
		} else if j < len(records2) {
			// Only file 2 has records left - mark as additions
			diffs = append(diffs, struct {
				pos1, pos2   int
				type1, type2 string
			}{-1, j, "", "+"})
			j++
		}
	}

	// Print the unified diff output
	hasDiff := false
	inHunk := false
	hunkStart1, hunkStart2 := 0, 0
	hunkLines1, hunkLines2 := 0, 0
	var hunkBuffer []string

	for i := 0; i < len(diffs); i++ {
		diff := diffs[i]

		// Check if this is a difference or context line
		isDiff := diff.type1 == "-" || diff.type2 == "+"

		// Check if we need to start a new hunk
		if isDiff && !inHunk {
			// Calculate the starting point with context
			hunkStart := i - *contextLines
			if hunkStart < 0 {
				hunkStart = 0
			}

			// Count lines from each file in this hunk
			if hunkStart < i {
				// Count context lines before the diff
				for j := hunkStart; j < i; j++ {
					if diffs[j].pos1 >= 0 {
						hunkLines1++
					}
					if diffs[j].pos2 >= 0 {
						hunkLines2++
					}
				}
			}

			// Adjust the starting positions for the hunk header
			if hunkStart < len(diffs) {
				if diffs[hunkStart].pos1 >= 0 {
					hunkStart1 = diffs[hunkStart].pos1 + 1 // 1-based line numbers
				} else if hunkStart > 0 {
					hunkStart1 = diffs[hunkStart-1].pos1 + 2
				} else {
					hunkStart1 = 1
				}

				if diffs[hunkStart].pos2 >= 0 {
					hunkStart2 = diffs[hunkStart].pos2 + 1 // 1-based line numbers
				} else if hunkStart > 0 {
					hunkStart2 = diffs[hunkStart-1].pos2 + 2
				} else {
					hunkStart2 = 1
				}
			}

			// Start the hunk
			inHunk = true
			hunkBuffer = []string{}

			// Add context lines before the diff
			for j := hunkStart; j < i; j++ {
				if diffs[j].pos1 >= 0 {
					addContextLine(&hunkBuffer, records1[diffs[j].pos1])
				}
			}

			hasDiff = true
		}

		// Add the current line to the hunk
		if inHunk {
			if diff.type1 == "-" && diff.pos1 >= 0 {
				addDiffLine(&hunkBuffer, records1[diff.pos1], "-")
				hunkLines1++
			}
			if diff.type2 == "+" && diff.pos2 >= 0 {
				addDiffLine(&hunkBuffer, records2[diff.pos2], "+")
				hunkLines2++
			}
			if diff.type1 == " " && diff.type2 == " " {
				if diff.pos1 >= 0 {
					addContextLine(&hunkBuffer, records1[diff.pos1])
					hunkLines1++
					hunkLines2++
				}
			}
		}

		// Check if we need to close the current hunk
		endHunk := false

		if inHunk {
			// If we've reached the end of the diff sequence, close the hunk
			if i == len(diffs)-1 {
				endHunk = true
			} else {
				// Check if there are more diffs ahead within context range
				nextDiffFound := false
				for j := i + 1; j <= min(i+*contextLines, len(diffs)-1); j++ {
					if diffs[j].type1 == "-" || diffs[j].type2 == "+" {
						nextDiffFound = true
						break
					}
				}

				if !nextDiffFound && i < len(diffs)-*contextLines {
					// No more diffs within context range, close the hunk
					endHunk = true

					// Add context lines after the diff
					for j := i + 1; j <= min(i+*contextLines, len(diffs)-1); j++ {
						if diffs[j].pos1 >= 0 {
							addContextLine(&hunkBuffer, records1[diffs[j].pos1])
							hunkLines1++
							hunkLines2++
						}
					}
				}
			}

			if endHunk {
				// If JSON output is enabled, collect the data for the hunk
				if *jsonOutput {
					// Collect records for this hunk
					hunk := JSONDiffHunk{
						StartLine1: hunkStart1,
						LineCount1: hunkLines1,
						StartLine2: hunkStart2,
						LineCount2: hunkLines2,
						Removed:    []MCPRecord{},
						Added:      []MCPRecord{},
					}

					// Process each line to extract records
					for _, line := range hunkBuffer {
						if len(line) > 0 && line[0] == '-' && len(line) > 1 {
							// Extract the record data for removal
							parts := strings.SplitN(line[1:], " ", 2)
							if len(parts) == 2 {
								direction := parts[0]
								content := parts[1]
								// Handle timestamp if present
								timestamp := 0.0
								if strings.Contains(content, " # ") {
									contentParts := strings.SplitN(content, " # ", 2)
									content = contentParts[0]
									if len(contentParts) > 1 {
										timestamp, _ = strconv.ParseFloat(contentParts[1], 64)
									}
								}

								// Parse JSON content
								var jsonData map[string]any
								if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
									hunk.Removed = append(hunk.Removed, MCPRecord{
										Direction:  direction,
										RawContent: content,
										JSON:       jsonData,
										Timestamp:  timestamp,
									})
								}
							}
						} else if len(line) > 0 && line[0] == '+' && len(line) > 1 {
							// Extract the record data for addition
							parts := strings.SplitN(line[1:], " ", 2)
							if len(parts) == 2 {
								direction := parts[0]
								content := parts[1]
								// Handle timestamp if present
								timestamp := 0.0
								if strings.Contains(content, " # ") {
									contentParts := strings.SplitN(content, " # ", 2)
									content = contentParts[0]
									if len(contentParts) > 1 {
										timestamp, _ = strconv.ParseFloat(contentParts[1], 64)
									}
								}

								// Parse JSON content
								var jsonData map[string]any
								if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
									hunk.Added = append(hunk.Added, MCPRecord{
										Direction:  direction,
										RawContent: content,
										JSON:       jsonData,
										Timestamp:  timestamp,
									})
								}
							}
						}
					}

					// Add the hunk to the JSON result
					jsonResult.Diffs = append(jsonResult.Diffs, hunk)
				} else {
					// Process word diff if needed
					if *wordDiff {
						// Process word diff placeholders in the hunk buffer
						processedBuffer := processWordDiffs(hunkBuffer)
						hunkBuffer = processedBuffer
					}

					// Print the standard hunk header and buffer
					hunktText := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunkStart1, hunkLines1, hunkStart2, hunkLines2)
					if !*noColor {
						fmt.Printf("%s%s%s\n", cyan, hunktText, reset)
					} else {
						fmt.Println(hunktText)
					}

					// Print the hunk content
					for _, line := range hunkBuffer {
						fmt.Println(line)
					}
				}

				// Reset hunk state
				inHunk = false
				hunkLines1 = 0
				hunkLines2 = 0
				hunkBuffer = nil
			}
		}
	}

	if !hasDiff {
		if *jsonOutput {
			jsonResult.Matches = true
			jsonBytes, _ := json.MarshalIndent(jsonResult, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("Files match exactly!")
		}
	} else if *jsonOutput {
		jsonResult.Matches = false
		jsonResult.TotalDiffs = len(jsonResult.Diffs)
		jsonBytes, _ := json.MarshalIndent(jsonResult, "", "  ")
		fmt.Println(string(jsonBytes))
	}
}

// addContextLine adds a context line to the hunk buffer
func addContextLine(buffer *[]string, rec MCPRecord) {
	// Format the record as a context line
	timestamp := ""
	if !*ignoreTimestamps {
		timestamp = fmt.Sprintf(" # %.3f", rec.Timestamp)
	}

	line := fmt.Sprintf(" %s%s %s", rec.Direction, timestamp, rec.RawContent)
	*buffer = append(*buffer, line)
}

// addDiffLine adds a diff line (addition or deletion) to the hunk buffer
func addDiffLine(buffer *[]string, rec MCPRecord, diffType string) {
	// Format the record as a diff line
	timestamp := ""
	if !*ignoreTimestamps {
		timestamp = fmt.Sprintf(" # %.3f", rec.Timestamp)
	}

	// For word diff mode, we'll handle this differently
	if *wordDiff && *semanticCompare {
		// In word diff mode with semantic comparison, we'll create special placeholder lines
		// which will be processed later in a complete diff hunk
		if diffType == "-" {
			*buffer = append(*buffer, fmt.Sprintf("WORDDIFF_REMOVE:%s%s %s", rec.Direction, timestamp, rec.RawContent))
		} else {
			*buffer = append(*buffer, fmt.Sprintf("WORDDIFF_ADD:%s%s %s", rec.Direction, timestamp, rec.RawContent))
		}
		return
	}

	line := ""
	if diffType == "-" {
		if !*noColor {
			line = fmt.Sprintf("%s-%s%s %s%s %s%s", red, reset, red, rec.Direction, timestamp, rec.RawContent, reset)
		} else {
			line = fmt.Sprintf("-%s%s %s", rec.Direction, timestamp, rec.RawContent)
		}
	} else {
		if !*noColor {
			line = fmt.Sprintf("%s+%s%s %s%s %s%s", green, reset, green, rec.Direction, timestamp, rec.RawContent, reset)
		} else {
			line = fmt.Sprintf("+%s%s %s", rec.Direction, timestamp, rec.RawContent)
		}
	}

	*buffer = append(*buffer, line)
}

// processWordDiffs converts word diff placeholders into actual word diffs
func processWordDiffs(hunkBuffer []string) []string {
	result := make([]string, 0, len(hunkBuffer))

	// First pass - clean up any record markers
	cleanBuffer := make([]string, 0, len(hunkBuffer))
	for i := 0; i < len(hunkBuffer); i++ {
		line := hunkBuffer[i]

		// Skip internal record markers
		if strings.HasPrefix(line, "WORDDIFF_REMOVE:") ||
			strings.HasPrefix(line, "WORDDIFF_ADD:") {
			// Process these in second pass
			cleanBuffer = append(cleanBuffer, line)
		} else if line == "placeholder for detailed diff" {
			// Skip placeholders
			continue
		} else {
			// Keep normal lines
			cleanBuffer = append(cleanBuffer, line)
		}
	}

	// Process buffer to find paired word diffs
	i := 0
	for i < len(cleanBuffer) {
		line := cleanBuffer[i]

		if strings.HasPrefix(line, "WORDDIFF_REMOVE:") {
			removeLine := strings.TrimPrefix(line, "WORDDIFF_REMOVE:")

			// Look for matching add line
			var addLine string
			found := false

			for j := i + 1; j < len(cleanBuffer); j++ {
				if strings.HasPrefix(cleanBuffer[j], "WORDDIFF_ADD:") {
					addLine = strings.TrimPrefix(cleanBuffer[j], "WORDDIFF_ADD:")
					found = true

					// Create word diff between these lines
					wordDiffLine := createWordDiff(removeLine, addLine)
					result = append(result, wordDiffLine)

					// Skip the add line since we've processed it
					i = j + 1
					break
				} else if !strings.HasPrefix(cleanBuffer[j], "WORDDIFF") {
					// If we hit a non-worddiff line before finding a matching add,
					// just add the remove line as a regular line
					result = append(result, "-"+removeLine)
					i++
					found = true
					break
				}
			}

			if !found {
				// No matching add found, just add as regular line
				result = append(result, "-"+removeLine)
				i++
			}
		} else if strings.HasPrefix(line, "WORDDIFF_ADD:") {
			// This is an add without a matching remove (probably already processed)
			addLine := strings.TrimPrefix(line, "WORDDIFF_ADD:")
			result = append(result, "+"+addLine)
			i++
		} else {
			// Regular lines pass through
			result = append(result, line)
			i++
		}
	}

	return result
}

// createWordDiff creates a git-style word diff between two strings
func createWordDiff(removeLine, addLine string) string {
	// Extract the JSON content
	removePrefix := ""
	removeContent := removeLine
	removeDirection := ""
	addContent := addLine

	// Try to extract direction and content
	if parts := strings.SplitN(removeLine, " ", 2); len(parts) == 2 {
		removeDirection = parts[0]
		removePrefix = removeDirection
		removeContent = parts[1]
	}

	if parts := strings.SplitN(addLine, " ", 2); len(parts) == 2 {
		// We'll use the remove prefix for consistency
		addContent = parts[1]
	}

	// Parse JSON for semantic comparison
	var removeJSON, addJSON map[string]interface{}
	if err := json.Unmarshal([]byte(removeContent), &removeJSON); err != nil {
		// Fallback to simple text diff if we can't parse JSON
		return fmt.Sprintf("-" + removeLine + "\n+" + addLine)
	}

	if err := json.Unmarshal([]byte(addContent), &addJSON); err != nil {
		return fmt.Sprintf("-" + removeLine + "\n+" + addLine)
	}

	// Find common fields and differences
	// For now, a simple approach - a more sophisticated version would do a deeper comparison
	line := fmt.Sprintf(" %s {", removePrefix)

	// Format JSON with indentation for readability
	prettyRemove, _ := json.MarshalIndent(removeJSON, "", "  ")
	prettyAdd, _ := json.MarshalIndent(addJSON, "", "  ")

	// Split into lines
	removeLines := strings.Split(string(prettyRemove), "\n")
	addLines := strings.Split(string(prettyAdd), "\n")

	// Simple line by line comparison
	minLines := min(len(removeLines), len(addLines))

	// Format as word diff
	for i := 0; i < minLines; i++ {
		if removeLines[i] == addLines[i] {
			// Unchanged line
			line += "\n  " + removeLines[i]
		} else {
			if !*noColor {
				line += fmt.Sprintf("\n  %s[--%s%s%s--]%s", red, reset, removeLines[i], red, reset)
				line += fmt.Sprintf("\n  %s{++%s%s%s++}%s", green, reset, addLines[i], green, reset)
			} else {
				line += fmt.Sprintf("\n  [--%s--]", removeLines[i])
				line += fmt.Sprintf("\n  {++%s++}", addLines[i])
			}
		}
	}

	// Add any extra lines from remove
	for i := minLines; i < len(removeLines); i++ {
		if !*noColor {
			line += fmt.Sprintf("\n  %s[--%s%s%s--]%s", red, reset, removeLines[i], red, reset)
		} else {
			line += fmt.Sprintf("\n  [--%s--]", removeLines[i])
		}
	}

	// Add any extra lines from add
	for i := minLines; i < len(addLines); i++ {
		if !*noColor {
			line += fmt.Sprintf("\n  %s{++%s%s%s++}%s", green, reset, addLines[i], green, reset)
		} else {
			line += fmt.Sprintf("\n  {++%s++}", addLines[i])
		}
	}

	line += "\n }"

	return line
}

// setupOutput configures stdout for immediate flushing
// This ensures output goes to pipes immediately without buffering
func setupOutput() {
	// Disable output buffering
	os.Stdout.Sync()
}

func main() {
	// Set up line-buffered output for piping
	setupOutput()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] file1.mcp file2.mcp\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	file1 := flag.Arg(0)
	file2 := flag.Arg(1)

	diffFiles(file1, file2)
}
