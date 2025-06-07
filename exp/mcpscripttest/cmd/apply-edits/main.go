// apply-edits applies structured edits to files in a sandboxed manner
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
)

var (
	editsFile string
	dryRun    bool
	backup    bool
	outputDir string
	verbose   bool
	force     bool
	deadlock  bool
)

func init() {
	flag.StringVar(&editsFile, "edits", "", "JSON file containing edits")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be changed without applying")
	flag.BoolVar(&backup, "backup", true, "Create backup files")
	flag.StringVar(&outputDir, "output", "", "Output directory for modified files (sandbox mode)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&force, "force", false, "Force apply even with conflicts")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "apply-edits - Apply structured edits to files\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  apply-edits [options] -edits <file>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Dry run to see changes\n")
		fmt.Fprintf(os.Stderr, "  apply-edits -edits suggestions.json -dry-run\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Apply to sandbox directory\n")
		fmt.Fprintf(os.Stderr, "  apply-edits -edits suggestions.json -output /tmp/sandbox\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Apply in place with backups\n")
		fmt.Fprintf(os.Stderr, "  apply-edits -edits suggestions.json -backup\n")
	}
}

type Edit struct {
	File        string `json:"file"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	OldText     string `json:"old_text"`
	NewText     string `json:"new_text"`
	Description string `json:"description"`
}

type ApplyResult struct {
	File      string    `json:"file"`
	Applied   bool      `json:"applied"`
	Error     string    `json:"error,omitempty"`
	Backup    string    `json:"backup,omitempty"`
	Changes   []Change  `json:"changes"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type Change struct {
	Line   int    `json:"line"`
	Before string `json:"before"`
	After  string `json:"after"`
}

func main() {
	flag.Parse()

	if editsFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Load edits
	edits, err := loadEdits(editsFile)
	if err != nil {
		log.Fatalf("Error loading edits: %v", err)
	}

	// Group edits by file
	fileEdits := groupEditsByFile(edits)

	// Apply edits
	results := []ApplyResult{}

	for file, edits := range fileEdits {
		result := applyFileEdits(file, edits)
		results = append(results, result)
	}

	// Output results
	if verbose || dryRun {
		outputResults(results)
	}

	// Summary
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Applied {
			successful++
		} else {
			failed++
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Files processed: %d\n", len(results))
	fmt.Printf("  Successful: %d\n", successful)
	fmt.Printf("  Failed: %d\n", failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func loadEdits(filename string) ([]Edit, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var edits []Edit
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&edits); err != nil {
		return nil, err
	}

	return edits, nil
}

func groupEditsByFile(edits []Edit) map[string][]Edit {
	groups := make(map[string][]Edit)

	for _, edit := range edits {
		groups[edit.File] = append(groups[edit.File], edit)
	}

	// Sort edits within each file by line number (reverse order for applying)
	for file, fileEdits := range groups {
		// Sort in reverse order so we can apply from bottom to top
		for i := 0; i < len(fileEdits)-1; i++ {
			for j := i + 1; j < len(fileEdits); j++ {
				if fileEdits[i].StartLine < fileEdits[j].StartLine {
					fileEdits[i], fileEdits[j] = fileEdits[j], fileEdits[i]
				}
			}
		}
		groups[file] = fileEdits
	}

	return groups
}

func applyFileEdits(filename string, edits []Edit) ApplyResult {
	result := ApplyResult{
		File:      filename,
		StartTime: time.Now(),
		Changes:   []Change{},
	}

	// Read original file
	content, err := readFile(filename)
	if err != nil {
		result.Error = fmt.Sprintf("Error reading file: %v", err)
		result.EndTime = time.Now()
		return result
	}

	lines := strings.Split(content, "\n")

	// Apply edits in reverse order (bottom to top)
	for _, edit := range edits {
		if verbose {
			log.Printf("Applying edit to %s:%d-%d: %s", filename, edit.StartLine, edit.EndLine, edit.Description)
		}

		// Validate line numbers
		if edit.StartLine < 1 || edit.StartLine > len(lines) {
			if !force {
				result.Error = fmt.Sprintf("Invalid start line %d (file has %d lines)", edit.StartLine, len(lines))
				result.EndTime = time.Now()
				return result
			}
			continue
		}

		// Record changes
		for i := edit.StartLine - 1; i <= edit.EndLine-1 && i < len(lines); i++ {
			result.Changes = append(result.Changes, Change{
				Line:   i + 1,
				Before: lines[i],
				After:  "", // Will be updated
			})
		}

		// Apply edit
		if edit.OldText != "" {
			// Verify old text matches
			actualText := extractLines(lines, edit.StartLine, edit.EndLine)
			if actualText != edit.OldText && !force {
				result.Error = fmt.Sprintf("Text mismatch at line %d: expected %q, got %q",
					edit.StartLine, edit.OldText, actualText)
				result.EndTime = time.Now()
				return result
			}
		}

		// Replace lines
		newLines := strings.Split(edit.NewText, "\n")
		lines = replaceLines(lines, edit.StartLine, edit.EndLine, newLines)

		// Update changes with new content
		for i, change := range result.Changes {
			if change.Line >= edit.StartLine && change.Line <= edit.EndLine {
				newIndex := change.Line - edit.StartLine
				if newIndex < len(newLines) {
					result.Changes[i].After = newLines[newIndex]
				}
			}
		}
	}

	// Determine output path
	outputPath := filename
	if outputDir != "" {
		// Sandbox mode
		relPath, err := filepath.Rel(".", filename)
		if err != nil {
			relPath = filepath.Base(filename)
		}
		outputPath = filepath.Join(outputDir, relPath)

		// Create directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			result.Error = fmt.Sprintf("Error creating directory: %v", err)
			result.EndTime = time.Now()
			return result
		}
	}

	// Create backup if requested
	if backup && outputDir == "" {
		backupPath := filename + ".bak"
		if err := copyFile(filename, backupPath); err != nil {
			result.Error = fmt.Sprintf("Error creating backup: %v", err)
			result.EndTime = time.Now()
			return result
		}
		result.Backup = backupPath
	}

	// Write modified content
	if !dryRun {
		modifiedContent := strings.Join(lines, "\n")
		if err := writeFile(outputPath, modifiedContent); err != nil {
			result.Error = fmt.Sprintf("Error writing file: %v", err)
			result.EndTime = time.Now()
			return result
		}
	}

	result.Applied = true
	result.EndTime = time.Now()
	return result
}

func readFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func extractLines(lines []string, start, end int) string {
	if start < 1 || start > len(lines) {
		return ""
	}

	if end > len(lines) {
		end = len(lines)
	}

	extracted := lines[start-1 : end]
	return strings.Join(extracted, "\n")
}

func replaceLines(lines []string, start, end int, newLines []string) []string {
	result := make([]string, 0, len(lines)-end+start-1+len(newLines))

	// Add lines before the edit
	result = append(result, lines[:start-1]...)

	// Add new lines
	result = append(result, newLines...)

	// Add lines after the edit
	if end < len(lines) {
		result = append(result, lines[end:]...)
	}

	return result
}

func outputResults(results []ApplyResult) {
	for _, result := range results {
		fmt.Printf("\n=== %s ===\n", result.File)
		fmt.Printf("Duration: %v\n", result.EndTime.Sub(result.StartTime))

		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
			continue
		}

		if result.Backup != "" {
			fmt.Printf("Backup: %s\n", result.Backup)
		}

		fmt.Printf("Changes:\n")
		for _, change := range result.Changes {
			fmt.Printf("  Line %d:\n", change.Line)
			if change.Before != "" {
				fmt.Printf("    - %s\n", change.Before)
			}
			if change.After != "" {
				fmt.Printf("    + %s\n", change.After)
			}
		}

		if dryRun {
			fmt.Printf("(DRY RUN - no changes applied)\n")
		}
	}
}
