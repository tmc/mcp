// Package scripttest defines an Analyzer for validating scripttest files
package scripttest

import (
	"bufio"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the analyzer for scripttest files.
var Analyzer = &analysis.Analyzer{
	Name: "scripttest",
	Doc:  "check for common errors in scripttest files",
	Run:  run,
}

// Command represents a scripttest command
type Command struct {
	Name       string
	LineNumber int
	Args       []string
	Raw        string
}

func run(pass *analysis.Pass) (interface{}, error) {
	// We're looking for test files with .txt extension
	txtFiles := findScripttestFiles(pass)

	for _, file := range txtFiles {
		issues := validateFile(file)
		for _, issue := range issues {
			// Report the issue using the analysis package's reporting mechanism
			pass.Reportf(issue.Pos, "%s (%s): %s", issue.Category, issue.Severity, issue.Message)
		}
	}

	return nil, nil
}

// findScripttestFiles finds all scripttest files (.txt) in the analyzed package
func findScripttestFiles(pass *analysis.Pass) []string {
	var files []string

	// Check the package directory
	if pass.Pkg != nil && len(pass.Files) > 0 {
		// Determine the package directory from the first file
		pkgDir := filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)

		// Look for scripttest files in a "testdata" subdirectory
		testdataDir := filepath.Join(pkgDir, "testdata")
		if _, err := os.Stat(testdataDir); err == nil {
			// Find all .txt files in testdata directory
			entries, err := os.ReadDir(testdataDir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
						files = append(files, filepath.Join(testdataDir, entry.Name()))
					}
				}
			}
		}

		// Also check scripttest files directly in the package directory
		entries, err := os.ReadDir(pkgDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
					filePath := filepath.Join(pkgDir, entry.Name())
					// Skip the file if it's already in the list
					if !contains(files, filePath) {
						files = append(files, filePath)
					}
				}
			}
		}
	}

	return files
}

// Issue represents a validation issue found in a scripttest file
type Issue struct {
	Pos      token.Pos // Set to 0 for file-level issues
	Message  string
	Severity string // "error", "warning", "info"
	Category string
}

// validateFile validates a single scripttest file
func validateFile(filePath string) []Issue {
	// Since these are not Go source files, we don't have position information
	// from the analysis package. We'll use 0 as the position and include
	// line numbers in the messages.

	file, err := os.Open(filePath)
	if err != nil {
		return []Issue{
			{
				Pos:      0,
				Message:  fmt.Sprintf("Failed to open file %s: %v", filePath, err),
				Severity: "error",
				Category: "file-access",
			},
		}
	}
	defer file.Close()

	var issues []Issue
	var commands []*Command

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Parse command line
		cmd, err := parseCommand(line, lineNum)
		if err != nil {
			issues = append(issues, Issue{
				Pos:      0,
				Message:  fmt.Sprintf("%s:%d: Invalid command syntax: %v", filePath, lineNum, err),
				Severity: "error",
				Category: "syntax",
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
		issues = append(issues, Issue{
			Pos:      0,
			Message:  fmt.Sprintf("%s: Error reading file: %v", filePath, err),
			Severity: "error",
			Category: "file-read",
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
func validateCommand(filePath string, cmd *Command) []Issue {
	var issues []Issue

	// Check for known commands
	switch cmd.Name {
	case "mcp-replay", "mcp-spy", "mcp-start", "mcp-test", "mcp-verify",
		"mcp-send", "mcp-recv", "mcpdiff", "mcpspy", "stdout":
		// These are valid MCP commands
		// Validate arguments for each command
		switch cmd.Name {
		case "mcp-replay", "mcp-verify":
			if len(cmd.Args) < 1 {
				issues = append(issues, Issue{
					Pos:      0,
					Message:  fmt.Sprintf("%s:%d: %s command requires a recording file argument", filePath, cmd.LineNumber, cmd.Name),
					Severity: "error",
					Category: "args",
				})
			}
		case "stdout":
			if len(cmd.Args) < 1 {
				issues = append(issues, Issue{
					Pos:      0,
					Message:  fmt.Sprintf("%s:%d: stdout command requires a pattern argument", filePath, cmd.LineNumber),
					Severity: "error",
					Category: "args",
				})
			}
		case "mcpdiff":
			if len(cmd.Args) < 2 {
				issues = append(issues, Issue{
					Pos:      0,
					Message:  fmt.Sprintf("%s:%d: mcpdiff command requires two file arguments", filePath, cmd.LineNumber),
					Severity: "error",
					Category: "args",
				})
			}
		}
	case "exec", "go", "cd", "rm", "cat", "ls", "mkdir", "echo", "grep":
		// These are common commands that might be valid in a scripttest
		// Just add warnings for potentially suspicious usage
		if cmd.Name == "rm" && contains(cmd.Args, "-rf") {
			issues = append(issues, Issue{
				Pos:      0,
				Message:  fmt.Sprintf("%s:%d: Using 'rm -rf' in tests is dangerous and may cause data loss", filePath, cmd.LineNumber),
				Severity: "warning",
				Category: "dangerous",
			})
		}
	default:
		// Unknown command, issue a warning
		issues = append(issues, Issue{
			Pos:      0,
			Message:  fmt.Sprintf("%s:%d: Unknown command: %s", filePath, cmd.LineNumber, cmd.Name),
			Severity: "warning",
			Category: "unknown-cmd",
		})
	}

	// Check for common paths/patterns in arguments
	for _, arg := range cmd.Args {
		// Check for absolute paths
		if strings.HasPrefix(arg, "/") {
			issues = append(issues, Issue{
				Pos:      0,
				Message:  fmt.Sprintf("%s:%d: Absolute path '%s' may cause test to be non-portable", filePath, cmd.LineNumber, arg),
				Severity: "warning",
				Category: "portability",
			})
		}

		// Check for environment variables that might not be set
		if strings.HasPrefix(arg, "$") && !strings.HasPrefix(arg, "${") &&
			arg != "$HOME" && arg != "$PATH" && arg != "$USER" && arg != "$GOCOVERDIR" {
			issues = append(issues, Issue{
				Pos:      0,
				Message:  fmt.Sprintf("%s:%d: Environment variable '%s' might not be set in the test environment", filePath, cmd.LineNumber, arg),
				Severity: "info",
				Category: "env-var",
			})
		}
	}

	return issues
}

// validateCommandFlow validates the sequence and flow of commands
func validateCommandFlow(filePath string, commands []*Command) []Issue {
	var issues []Issue

	if len(commands) == 0 {
		issues = append(issues, Issue{
			Pos:      0,
			Message:  fmt.Sprintf("%s: File contains no commands", filePath),
			Severity: "warning",
			Category: "empty",
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
				issues = append(issues, Issue{
					Pos:      0,
					Message:  fmt.Sprintf("%s:%d: stdin command found, but no active async command detected", filePath, cmd.LineNumber),
					Severity: "warning",
					Category: "async",
				})
			}
		}

		// Check for wait/stop
		if cmd.Name == "stop" || cmd.Name == "wait" {
			if len(asyncCommands) == 0 {
				issues = append(issues, Issue{
					Pos:      0,
					Message:  fmt.Sprintf("%s:%d: %s command found, but no active async command detected", filePath, cmd.LineNumber, cmd.Name),
					Severity: "error",
					Category: "async",
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
			issues = append(issues, Issue{
				Pos:      0,
				Message:  fmt.Sprintf("%s:%d: Async command started but never stopped or waited for", filePath, cmd.LineNumber),
				Severity: "error",
				Category: "async",
			})
		}
	}

	return issues
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
