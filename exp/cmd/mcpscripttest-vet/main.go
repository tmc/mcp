// Command mcpscripttest-vet is a linting tool for MCP scripttest files.
// It validates the syntax and structure of test scripts to catch common errors.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Command represents a scripttest command
type Command struct {
	Name       string
	LineNumber int
	Args       []string
	Raw        string
}

// ValidationIssue represents a validation problem found in a script file
type ValidationIssue struct {
	File     string
	Line     int
	Message  string
	Severity string // "error", "warning", "info"
	Context  string
	Command  *Command
}

var (
	// Command-line flags
	verbose          = flag.Bool("v", false, "Enable verbose output")
	exitOnError      = flag.Bool("e", true, "Exit with non-zero status if any errors are found")
	warningsAsErrors = flag.Bool("w", false, "Treat warnings as errors")
	includeInfos     = flag.Bool("i", false, "Include informational messages in the output")
	recursive        = flag.Bool("r", false, "Recursively process directories")
	// Help text
	helpText = `mcpscripttest-vet - Validate MCP scripttest files

Usage:
  mcpscripttest-vet [flags] file1.txt [file2.txt ...] [dir1] [dir2 ...]

Flags:
  -v    Enable verbose output
  -e    Exit with non-zero status if any errors are found (default: true)
  -w    Treat warnings as errors
  -i    Include informational messages in the output
  -r    Recursively process directories

Examples:
  # Check a single file
  mcpscripttest-vet testdata/test1.txt

  # Check all txt files in a directory
  mcpscripttest-vet testdata/

  # Check specific files and directories with verbose output
  mcpscripttest-vet -v test1.txt test2.txt testdir/

  # Recursively check directories
  mcpscripttest-vet -r testdata/ otherdata/
`
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", helpText)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Process all file and directory arguments
	var allIssues []ValidationIssue
	for _, arg := range args {
		// Check if the argument is a file or directory
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if info.IsDir() {
			// Process directory
			dirIssues := processDirectory(arg, *recursive)
			allIssues = append(allIssues, dirIssues...)
		} else {
			// Process single file
			if !strings.HasSuffix(arg, ".txt") {
				if *verbose {
					fmt.Printf("Skipping non-txt file: %s\n", arg)
				}
				continue
			}
			fileIssues := validateFile(arg)
			allIssues = append(allIssues, fileIssues...)
		}
	}

	// Report all issues
	errorCount, warningCount, infoCount := reportIssues(allIssues)

	// Exit with appropriate status code
	if *exitOnError && (errorCount > 0 || (*warningsAsErrors && warningCount > 0)) {
		fmt.Fprintf(os.Stderr, "Found %d errors and %d warnings\n", errorCount, warningCount)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Validation complete: %d errors, %d warnings, %d infos\n",
			errorCount, warningCount, infoCount)
	}
}

// processDirectory processes all .txt files in a directory
func processDirectory(dir string, recursive bool) []ValidationIssue {
	var allIssues []ValidationIssue

	walkFn := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", path, err)
			return nil // Continue despite the error
		}

		// Skip directories if not recursive
		if info.IsDir() {
			if path != dir && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .txt files
		if !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		if *verbose {
			fmt.Printf("Validating %s\n", path)
		}

		fileIssues := validateFile(path)
		allIssues = append(allIssues, fileIssues...)
		return nil
	}

	err := filepath.Walk(dir, walkFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", dir, err)
	}

	return allIssues
}

// validateFile validates a single scripttest file
func validateFile(filePath string) []ValidationIssue {
	file, err := os.Open(filePath)
	if err != nil {
		return []ValidationIssue{{
			File:     filePath,
			Line:     0,
			Message:  fmt.Sprintf("Failed to open file: %v", err),
			Severity: "error",
		}}
	}
	defer file.Close()

	var issues []ValidationIssue
	var commands []*Command

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Parse command line
		cmd, err := parseCommand(line, lineNum)
		if err != nil {
			issues = append(issues, ValidationIssue{
				File:     filePath,
				Line:     lineNum,
				Message:  fmt.Sprintf("Invalid command syntax: %v", err),
				Severity: "error",
				Context:  line,
			})
			continue
		}

		if cmd != nil {
			commands = append(commands, cmd)

			// Validate individual command
			cmdIssues := validateCommand(filePath, cmd)
			issues = append(issues, cmdIssues...)
		}
	}

	if err := scanner.Err(); err != nil {
		issues = append(issues, ValidationIssue{
			File:     filePath,
			Line:     lineNum,
			Message:  fmt.Sprintf("Error reading file: %v", err),
			Severity: "error",
		})
	}

	// Validate command sequences and flow
	flowIssues := validateCommandFlow(filePath, commands)
	issues = append(issues, flowIssues...)

	return issues
}

// parseCommand parses a command line from a scripttest file
func parseCommand(line string, lineNum int) (*Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// Split the line into command and arguments
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, nil
	}

	cmd := &Command{
		Name:       parts[0],
		LineNumber: lineNum,
		Args:       parts[1:],
		Raw:        line,
	}

	return cmd, nil
}

// validateCommand validates a single command
func validateCommand(filePath string, cmd *Command) []ValidationIssue {
	var issues []ValidationIssue

	// Check for known commands
	switch cmd.Name {
	case "mcp-replay", "mcp-spy", "mcp-start", "mcp-test", "mcp-verify",
		"mcp-send", "mcp-recv", "mcpdiff", "mcpspy", "stdout":
		// These are valid MCP commands
		// Validate arguments for each command
		switch cmd.Name {
		case "mcp-replay", "mcp-verify":
			if len(cmd.Args) < 1 {
				issues = append(issues, ValidationIssue{
					File:     filePath,
					Line:     cmd.LineNumber,
					Message:  fmt.Sprintf("%s command requires a recording file argument", cmd.Name),
					Severity: "error",
					Context:  cmd.Raw,
					Command:  cmd,
				})
			}
		case "stdout":
			if len(cmd.Args) < 1 {
				issues = append(issues, ValidationIssue{
					File:     filePath,
					Line:     cmd.LineNumber,
					Message:  "stdout command requires a pattern argument",
					Severity: "error",
					Context:  cmd.Raw,
					Command:  cmd,
				})
			}
		case "mcpdiff":
			if len(cmd.Args) < 2 {
				issues = append(issues, ValidationIssue{
					File:     filePath,
					Line:     cmd.LineNumber,
					Message:  "mcpdiff command requires two file arguments",
					Severity: "error",
					Context:  cmd.Raw,
					Command:  cmd,
				})
			}
		}
	case "exec", "go", "cd", "rm", "cat", "ls", "mkdir", "echo", "grep":
		// These are common commands that might be valid in a scripttest
		// Just add warnings for potentially suspicious usage
		if cmd.Name == "rm" && contains(cmd.Args, "-rf") {
			issues = append(issues, ValidationIssue{
				File:     filePath,
				Line:     cmd.LineNumber,
				Message:  "Using 'rm -rf' in tests is dangerous and may cause data loss",
				Severity: "warning",
				Context:  cmd.Raw,
				Command:  cmd,
			})
		}
	default:
		// Unknown command, issue a warning
		issues = append(issues, ValidationIssue{
			File:     filePath,
			Line:     cmd.LineNumber,
			Message:  fmt.Sprintf("Unknown command: %s", cmd.Name),
			Severity: "warning",
			Context:  cmd.Raw,
			Command:  cmd,
		})
	}

	// Check for common paths/patterns in arguments
	for _, arg := range cmd.Args {
		// Check for absolute paths
		if strings.HasPrefix(arg, "/") {
			issues = append(issues, ValidationIssue{
				File:     filePath,
				Line:     cmd.LineNumber,
				Message:  fmt.Sprintf("Absolute path '%s' may cause test to be non-portable", arg),
				Severity: "warning",
				Context:  cmd.Raw,
				Command:  cmd,
			})
		}

		// Check for environment variables that might not be set
		if strings.HasPrefix(arg, "$") && !strings.HasPrefix(arg, "${") &&
			arg != "$HOME" && arg != "$PATH" && arg != "$USER" && arg != "$GOCOVERDIR" {
			issues = append(issues, ValidationIssue{
				File:     filePath,
				Line:     cmd.LineNumber,
				Message:  fmt.Sprintf("Environment variable '%s' might not be set in the test environment", arg),
				Severity: "info",
				Context:  cmd.Raw,
				Command:  cmd,
			})
		}
	}

	return issues
}

// validateCommandFlow validates the sequence and flow of commands
func validateCommandFlow(filePath string, commands []*Command) []ValidationIssue {
	var issues []ValidationIssue

	if len(commands) == 0 {
		issues = append(issues, ValidationIssue{
			File:     filePath,
			Line:     0,
			Message:  "File contains no commands",
			Severity: "warning",
		})
		return issues
	}

	// Check for balanced async commands
	var asyncCommands []*Command
	for _, cmd := range commands {
		// Check if it's an async command start
		if cmd.Name == "mcp-start" {
			asyncCommands = append(asyncCommands, cmd)
		}

		// Check for potentially missing stop/wait
		if cmd.Name == "stdin" {
			// stdin command often indicates interaction with an async process
			// Check if we have open async commands
			if len(asyncCommands) == 0 {
				issues = append(issues, ValidationIssue{
					File:     filePath,
					Line:     cmd.LineNumber,
					Message:  "stdin command found, but no active async command detected",
					Severity: "warning",
					Context:  cmd.Raw,
					Command:  cmd,
				})
			}
		}

		// Check for wait/stop
		if cmd.Name == "stop" || cmd.Name == "wait" {
			if len(asyncCommands) == 0 {
				issues = append(issues, ValidationIssue{
					File:     filePath,
					Line:     cmd.LineNumber,
					Message:  fmt.Sprintf("%s command found, but no active async command detected", cmd.Name),
					Severity: "error",
					Context:  cmd.Raw,
					Command:  cmd,
				})
			} else {
				// Remove the last async command
				asyncCommands = asyncCommands[:len(asyncCommands)-1]
			}
		}
	}

	// Check if we have outstanding async commands
	if len(asyncCommands) > 0 {
		for _, cmd := range asyncCommands {
			issues = append(issues, ValidationIssue{
				File:     filePath,
				Line:     cmd.LineNumber,
				Message:  "Async command started but never stopped or waited for",
				Severity: "error",
				Context:  cmd.Raw,
				Command:  cmd,
			})
		}
	}

	return issues
}

// reportIssues reports all validation issues
func reportIssues(issues []ValidationIssue) (int, int, int) {
	var errorCount, warningCount, infoCount int

	// Sort issues by file and line number
	// (not implemented for simplicity, but would be nice to have)

	// Report issues
	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			errorCount++
			fmt.Printf("%s:%d: error: %s\n", issue.File, issue.Line, issue.Message)
			if issue.Context != "" {
				fmt.Printf("  %s\n", issue.Context)
			}
		case "warning":
			warningCount++
			if *warningsAsErrors {
				fmt.Printf("%s:%d: error (promoted from warning): %s\n", issue.File, issue.Line, issue.Message)
			} else {
				fmt.Printf("%s:%d: warning: %s\n", issue.File, issue.Line, issue.Message)
			}
			if issue.Context != "" {
				fmt.Printf("  %s\n", issue.Context)
			}
		case "info":
			infoCount++
			if *includeInfos {
				fmt.Printf("%s:%d: info: %s\n", issue.File, issue.Line, issue.Message)
				if issue.Context != "" {
					fmt.Printf("  %s\n", issue.Context)
				}
			}
		}
	}

	return errorCount, warningCount, infoCount
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
