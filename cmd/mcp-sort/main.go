// Command mcp-sort reads MCP trace files and sorts the entries by timestamp.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	// Format: [2024-05-10 14:32:15.123]
	timestampRegex = regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]`)
	stripTimestamp = flag.Bool("strip", false, "Strip timestamps instead of sorting")
	outputFile     = flag.String("output", "", "Output file (default: stdout)")
	inPlace        = flag.Bool("in-place", false, "Edit files in place")
)

// MCPEntry represents a parsed MCP log entry
type MCPEntry struct {
	Raw       string    // The raw log entry text
	Timestamp time.Time // The parsed timestamp
	Line      int       // Original line number for stable sort
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [files...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Reads MCP trace files and sorts the entries by timestamp.\n")
		fmt.Fprintf(os.Stderr, "If no files are specified, reads from stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// If no files specified, process stdin
	if flag.NArg() == 0 {
		if *inPlace {
			fmt.Fprintln(os.Stderr, "Error: -in-place requires file arguments")
			os.Exit(1)
		}
		processReader(os.Stdin, os.Stdout)
		return
	}

	// Process each file
	for _, filename := range flag.Args() {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", filename, err)
			continue
		}

		var output io.Writer
		var buf bytes.Buffer

		if *inPlace {
			output = &buf
		} else if *outputFile != "" {
			outFile, err := os.Create(*outputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", *outputFile, err)
				file.Close()
				continue
			}
			defer outFile.Close()
			output = outFile
		} else {
			output = os.Stdout
		}

		processReader(file, output)
		file.Close()

		// If in-place editing, write back to the original file
		if *inPlace {
			err := os.WriteFile(filename, buf.Bytes(), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", filename, err)
			}
		}
	}
}

func processReader(input io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(input)
	var entries []MCPEntry
	lineNum := 0

	// Read and parse all entries
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Only process lines that match our expected format
		if !strings.HasPrefix(line, "mcp-send") && !strings.HasPrefix(line, "mcp-recv") {
			// Keep non-matching lines with zero timestamp for stable ordering
			entries = append(entries, MCPEntry{
				Raw:       line,
				Timestamp: time.Time{},
				Line:      lineNum,
			})
			continue
		}

		// Extract timestamp
		matches := timestampRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			// Invalid format, keep with zero timestamp
			entries = append(entries, MCPEntry{
				Raw:       line,
				Timestamp: time.Time{},
				Line:      lineNum,
			})
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05.000", matches[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing timestamp on line %d: %v\n", lineNum, err)
			// Add with zero timestamp
			entries = append(entries, MCPEntry{
				Raw:       line,
				Timestamp: time.Time{},
				Line:      lineNum,
			})
			continue
		}

		// Add to entries
		entries = append(entries, MCPEntry{
			Raw:       line,
			Timestamp: timestamp,
			Line:      lineNum,
		})
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return
	}

	if *stripTimestamp {
		// Just strip timestamps, no sorting
		for _, entry := range entries {
			if strings.HasPrefix(entry.Raw, "mcp-send") || strings.HasPrefix(entry.Raw, "mcp-recv") {
				// Replace timestamp with placeholder
				replaced := timestampRegex.ReplaceAllString(entry.Raw, "[TIMESTAMP]")
				fmt.Fprintln(output, replaced)
			} else {
				fmt.Fprintln(output, entry.Raw)
			}
		}
		return
	}

	// Sort entries by timestamp, then by original line number for stable sort
	sort.Slice(entries, func(i, j int) bool {
		// If timestamps are equal, preserve original order
		if entries[i].Timestamp.Equal(entries[j].Timestamp) {
			return entries[i].Line < entries[j].Line
		}
		// Otherwise sort by timestamp
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	// Output sorted entries
	for _, entry := range entries {
		fmt.Fprintln(output, entry.Raw)
	}
}
