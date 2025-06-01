package fuzzing

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BinaryIntrospector analyzes binaries to understand their command-line interfaces
type BinaryIntrospector struct {
	cache map[string]*BinaryInfo
}

// BinaryInfo contains information about a binary's interface
type BinaryInfo struct {
	Path          string
	SupportsHelp  bool
	HelpOutput    string
	Flags         []FlagInfo
	ValidExamples []string
	AcceptsStdin  bool
	SupportsCooperativeFuzzing bool
}

// FlagInfo describes a command-line flag
type FlagInfo struct {
	Name        string
	ShortName   string
	Type        string // "bool", "string", "int", etc.
	Description string
	Default     string
}

// NewBinaryIntrospector creates a new binary introspector
func NewBinaryIntrospector() *BinaryIntrospector {
	return &BinaryIntrospector{
		cache: make(map[string]*BinaryInfo),
	}
}

// IntrospectBinary analyzes a binary to understand its interface
func (bi *BinaryIntrospector) IntrospectBinary(binaryPath string) (*BinaryInfo, error) {
	// Check cache first
	if info, exists := bi.cache[binaryPath]; exists {
		return info, nil
	}

	info := &BinaryInfo{
		Path: binaryPath,
	}

	// Try to get help output
	if helpOutput, err := bi.getHelpOutput(binaryPath); err == nil {
		info.SupportsHelp = true
		info.HelpOutput = helpOutput
		info.Flags = bi.parseFlags(helpOutput)
	}

	// Check if binary accepts stdin
	info.AcceptsStdin = bi.checkStdinAcceptance(binaryPath)

	// Check if binary supports cooperative fuzzing
	info.SupportsCooperativeFuzzing = bi.checkCooperativeFuzzing(binaryPath)

	// Generate some valid examples based on what we learned
	info.ValidExamples = bi.generateExamples(info)

	// Cache the result
	bi.cache[binaryPath] = info

	return info, nil
}

// getHelpOutput tries to get help information from a binary
func (bi *BinaryIntrospector) getHelpOutput(binaryPath string) (string, error) {
	// Try different help flags
	helpFlags := []string{"--help", "-h", "-help", "help", "-?"}
	
	for _, flag := range helpFlags {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		cmd := exec.CommandContext(ctx, binaryPath, flag)
		output, err := cmd.CombinedOutput()
		
		// Check if we got help text (even if exit code is non-zero)
		outputStr := string(output)
		if strings.Contains(outputStr, "Usage:") || 
		   strings.Contains(outputStr, "usage:") ||
		   strings.Contains(outputStr, "Options:") ||
		   strings.Contains(outputStr, "Flags:") {
			return outputStr, nil
		}
		
		// Some programs output help to stderr with exit code 0
		if err == nil && len(output) > 50 {
			return outputStr, nil
		}
	}
	
	return "", fmt.Errorf("could not get help output")
}

// parseFlags extracts flag information from help text
func (bi *BinaryIntrospector) parseFlags(helpText string) []FlagInfo {
	var flags []FlagInfo
	
	lines := strings.Split(helpText, "\n")
	inFlagsSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for flags section
		if strings.Contains(strings.ToLower(line), "flags:") ||
		   strings.Contains(strings.ToLower(line), "options:") {
			inFlagsSection = true
			continue
		}
		
		// Exit flags section on empty line or new section
		if inFlagsSection && (line == "" || strings.HasSuffix(line, ":")) {
			inFlagsSection = false
			continue
		}
		
		if inFlagsSection {
			// Parse flag line (common patterns)
			// -h, --help    Show help
			// --verbose     Enable verbose output
			// -o <file>     Output file
			
			flag := parseFlagLine(line)
			if flag.Name != "" {
				flags = append(flags, flag)
			}
		}
	}
	
	return flags
}

// parseFlagLine parses a single flag line from help text
func parseFlagLine(line string) FlagInfo {
	// Simple parser for common flag formats
	flag := FlagInfo{}
	
	// Remove leading whitespace
	line = strings.TrimSpace(line)
	
	// Common patterns:
	// -h, --help           Show help
	// --verbose            Enable verbose
	// -o, --output <file>  Output file
	
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return flag
	}
	
	// Extract flag names
	for i, part := range parts {
		if strings.HasPrefix(part, "--") {
			flag.Name = strings.TrimSuffix(part, ",")
			if flag.Type == "" {
				flag.Type = "bool" // Default to bool
			}
		} else if strings.HasPrefix(part, "-") && len(part) == 2 {
			flag.ShortName = strings.TrimSuffix(part, ",")
		} else if strings.HasPrefix(part, "<") && strings.HasSuffix(part, ">") {
			// Argument type indicator
			flag.Type = "string"
		} else if i > 0 {
			// Rest is description
			flag.Description = strings.Join(parts[i:], " ")
			break
		}
	}
	
	return flag
}

// checkStdinAcceptance checks if a binary accepts stdin
func (bi *BinaryIntrospector) checkStdinAcceptance(binaryPath string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return false
	}
	
	// Try to write to stdin
	go func() {
		stdin.Write([]byte("test\n"))
		stdin.Close()
	}()
	
	// Wait for completion
	cmd.Wait()
	
	// If it didn't immediately fail, it probably accepts stdin
	return true
}

// generateExamples creates example command lines based on introspection
func (bi *BinaryIntrospector) generateExamples(info *BinaryInfo) []string {
	var examples []string
	
	// Basic execution
	examples = append(examples, info.Path)
	
	// With common flags
	for _, flag := range info.Flags {
		if flag.ShortName != "" && flag.Type == "bool" {
			examples = append(examples, fmt.Sprintf("%s %s", info.Path, flag.ShortName))
		}
		if flag.Name != "" && flag.Type == "string" {
			examples = append(examples, fmt.Sprintf("%s %s value", info.Path, flag.Name))
		}
	}
	
	return examples
}

// ValidateCommand checks if a command line seems valid for a binary
func (bi *BinaryIntrospector) ValidateCommand(command string) (bool, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, fmt.Errorf("empty command")
	}
	
	binaryPath := parts[0]
	args := parts[1:]
	
	// Quick validation by running with --help or similar
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Try to run with the given args but in a safe way
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	
	err := cmd.Run()
	
	// If it runs without error or exits cleanly, it's probably valid
	if err == nil {
		return true, nil
	}
	
	// Check for specific error types
	if exitErr, ok := err.(*exec.ExitError); ok {
		// Exit code 0 or 1 is usually OK (many tools use 1 for --help)
		if exitErr.ExitCode() <= 1 {
			return true, nil
		}
		
		// Check stderr for "unknown flag" or similar
		stderr := cmd.Stderr.(*bytes.Buffer).String()
		if strings.Contains(stderr, "unknown flag") ||
		   strings.Contains(stderr, "invalid") ||
		   strings.Contains(stderr, "unrecognized") {
			return false, fmt.Errorf("invalid arguments")
		}
	}

	return false, err
}

// checkCooperativeFuzzing checks if a binary supports cooperative fuzzing
func (bi *BinaryIntrospector) checkCooperativeFuzzing(binaryPath string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set environment variable to trigger introspection mode
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(cmd.Env, "MCP_SCRIPTTEST_INTROSPECT=1")

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if the output looks like JSON with binary_name field
	outputStr := string(output)
	return strings.Contains(outputStr, "\"binary_name\"") &&
	       strings.Contains(outputStr, "\"flags\"")
}

// GenerateCommand uses a binary's cooperative fuzzing support to generate a command
func (bi *BinaryIntrospector) GenerateCommand(binaryPath string, seed int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set environment variable to trigger generation mode
	cmd := exec.CommandContext(ctx, binaryPath, fmt.Sprintf("%d", seed))
	cmd.Env = append(cmd.Env, "MCP_SCRIPTTEST_GENERATE=1")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate command: %w", err)
	}

	// The output should be a valid command line
	command := strings.TrimSpace(string(output))
	if command == "" {
		return "", fmt.Errorf("generated empty command")
	}

	return command, nil
}