package testcallgraph

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// BashStitcher extends the EnhancedStitcher to support bash script analysis
type BashStitcher struct {
	*EnhancedStitcher
	
	// Map of bash scripts to their execution information
	BashScriptMap map[string][]BashExecution
	
	// Map of bash scripts to their coverage data  
	BashCoverageMap map[string]*BashCoverage
}

// BashExecution represents execution of a bash script from a test
type BashExecution struct {
	ScriptPath  string
	Command     string
	Line        int
	ExecutedBy  string   // "exec", "bash", "sh", custom command, etc
	Arguments   []string 
	WithCoverage bool    // Whether coverage collection is enabled
}

// BashCoverage represents coverage data for a bash script
type BashCoverage struct {
	ScriptPath     string
	TotalLines     int
	ExecutedLines  map[int]bool
	CoveragePercent float64
	Functions      map[string]*BashFunction
}

// BashFunction represents a function definition in a bash script
type BashFunction struct {
	Name       string
	StartLine  int
	EndLine    int
	Called     bool
	CallCount  int
}

// NewBashStitcher creates a new bash-aware stitcher
func NewBashStitcher() *BashStitcher {
	return &BashStitcher{
		EnhancedStitcher: NewEnhancedStitcher(),
		BashScriptMap:    make(map[string][]BashExecution),
		BashCoverageMap:  make(map[string]*BashCoverage),
	}
}

// AnalyzeScriptTest overrides to also detect bash script executions
func (bs *BashStitcher) AnalyzeScriptTest(testFile string) error {
	// First run the base analysis
	if err := bs.EnhancedStitcher.AnalyzeScriptTest(testFile); err != nil {
		return err
	}
	
	// Now analyze for bash scripts
	file, err := os.Open(testFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	executions := []BashExecution{}
	lineNum := 0
	
	// Regular expressions for detecting bash script executions
	bashExecRegex := regexp.MustCompile(`^\s*(exec|bash|sh)\s+(.+\.sh)(\s+(.*))?$`)
	coverageRegex := regexp.MustCompile(`^\s*(kcov|bashcov|coverage)\s+`)
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check for bash script execution
		if matches := bashExecRegex.FindStringSubmatch(line); matches != nil {
			cmd := matches[1]
			scriptPath := matches[2]
			args := strings.Fields(matches[4])
			
			// Check if this is a coverage-enabled execution
			withCoverage := coverageRegex.MatchString(line)
			
			executions = append(executions, BashExecution{
				ScriptPath:   scriptPath,
				Command:      line,
				Line:         lineNum,
				ExecutedBy:   cmd,
				Arguments:    args,
				WithCoverage: withCoverage,
			})
		}
		
		// Also check for custom commands that might execute bash scripts
		parts := strings.Fields(line)
		if len(parts) > 0 {
			cmdName := parts[0]
			if mapping, ok := bs.CustomCommandMap[cmdName]; ok {
				scriptPath := bs.extractBashScriptFromCommand(cmdName, line, mapping)
				if scriptPath != "" && strings.HasSuffix(scriptPath, ".sh") {
					executions = append(executions, BashExecution{
						ScriptPath: scriptPath,
						Command:    line,
						Line:       lineNum,
						ExecutedBy: cmdName,
						Arguments:  parts[1:],
					})
				}
			}
		}
	}
	
	bs.BashScriptMap[testFile] = executions

	// Analyze the bash scripts themselves for structure
	for _, exec := range executions {
		if _, err := bs.analyzeBashScript(exec.ScriptPath); err != nil {
			// Log error but don't fail - script might not exist yet
			fmt.Fprintf(os.Stderr, "Warning: Failed to analyze %s: %v\n", exec.ScriptPath, err)
		}
	}

	// Also look for bash coverage traces created by our bash command
	// These would be in files specified by LAST_BASH_TRACE environment variable
	if tracePath := os.Getenv("LAST_BASH_TRACE"); tracePath != "" {
		if err := bs.ProcessBashTraceData(tracePath, ""); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to process bash trace %s: %v\n", tracePath, err)
		}
	}

	return scanner.Err()
}

// extractBashScriptFromCommand extracts bash script path from custom commands
func (bs *BashStitcher) extractBashScriptFromCommand(cmdName, fullCmd string, mapping CommandMapping) string {
	switch mapping.ExecutedProgram {
	case "dynamic":
		// For dynamic commands, parse the script path from arguments
		parts := strings.Split(fullCmd, " -- ")
		if len(parts) >= 2 {
			cmdParts := strings.Fields(parts[1])
			for _, part := range cmdParts {
				if strings.HasSuffix(part, ".sh") {
					return part
				}
			}
		}
	default:
		// Check if the mapped program is itself a bash script
		if strings.HasSuffix(mapping.ExecutedProgram, ".sh") {
			return mapping.ExecutedProgram
		}
	}
	return ""
}

// analyzeBashScript analyzes the structure of a bash script
func (bs *BashStitcher) analyzeBashScript(scriptPath string) (*BashCoverage, error) {
	if coverage, exists := bs.BashCoverageMap[scriptPath]; exists {
		return coverage, nil
	}
	
	file, err := os.Open(scriptPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	coverage := &BashCoverage{
		ScriptPath:    scriptPath,
		ExecutedLines: make(map[int]bool),
		Functions:     make(map[string]*BashFunction),
	}
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	currentFunction := ""
	
	// Regex patterns for bash constructs
	functionDefRegex := regexp.MustCompile(`^\s*function\s+(\w+)|^\s*(\w+)\s*\(\s*\)\s*\{`)
	functionEndRegex := regexp.MustCompile(`^\s*}`)
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		coverage.TotalLines = lineNum
		
		// Check for function definitions
		if matches := functionDefRegex.FindStringSubmatch(line); matches != nil {
			funcName := matches[1]
			if funcName == "" {
				funcName = matches[2]
			}
			currentFunction = funcName
			coverage.Functions[funcName] = &BashFunction{
				Name:      funcName,
				StartLine: lineNum,
				Called:    false,
			}
		}
		
		// Check for function end
		if currentFunction != "" && functionEndRegex.MatchString(line) {
			if fn, ok := coverage.Functions[currentFunction]; ok {
				fn.EndLine = lineNum
			}
			currentFunction = ""
		}
	}
	
	bs.BashCoverageMap[scriptPath] = coverage
	return coverage, scanner.Err()
}

// ProcessCoverageData processes coverage data from tools like kcov or bashcov
func (bs *BashStitcher) ProcessCoverageData(coverageDir string) error {
	// This would integrate with actual coverage tools
	// For now, it's a placeholder for the interface
	return nil
}

// ProcessBashTraceData processes trace data from BASH_XTRACEFD output
func (bs *BashStitcher) ProcessBashTraceData(traceFile string, scriptPath string) error {
	coverage, ok := bs.BashCoverageMap[scriptPath]
	if !ok {
		// Create new coverage data
		coverage = &BashCoverage{
			ScriptPath:    scriptPath,
			ExecutedLines: make(map[int]bool),
			Functions:     make(map[string]*BashFunction),
		}
		bs.BashCoverageMap[scriptPath] = coverage
	}

	file, err := os.Open(traceFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Parse trace formats:
	// Old format: + command
	// New format: +(scriptname:line): command
	// Our format: +(bash:1): command or +(./script.sh:5): command
	traceRegex := regexp.MustCompile(`^\+\s+(.*)`)
	lineNumRegex := regexp.MustCompile(`^\+\(([^)]+):(\d+)\):\s+(.*)`)

	// Also track command executions for building call graph edges
	cmdExecRegex := regexp.MustCompile(`^\+[^:]*:\s*(exec|bash|sh)\s+([^\s]+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for line number format
		if matches := lineNumRegex.FindStringSubmatch(line); matches != nil {
			_ = matches[1] // scriptName - Could be used for validation
			lineNum := matches[2]
			cmd := matches[3]

			// Convert line number and mark as executed
			if num, err := strconv.Atoi(lineNum); err == nil {
				coverage.ExecutedLines[num] = true

				// Check if this line executes another command/script
				if cmdMatches := cmdExecRegex.FindStringSubmatch(line); cmdMatches != nil {
					execType := cmdMatches[1]
					target := cmdMatches[2]

					// Add to bash executions
					exec := BashExecution{
						ScriptPath:  target,
						Command:     cmd,
						Line:        num,
						ExecutedBy:  execType,
						WithCoverage: true,
					}
					bs.BashScriptMap[scriptPath] = append(bs.BashScriptMap[scriptPath], exec)
				}
			}
		} else if matches := traceRegex.FindStringSubmatch(line); matches != nil {
			// Simple trace format without line numbers
			cmd := matches[1]

			// Try to extract command executions
			if cmdMatches := cmdExecRegex.FindStringSubmatch("+"+line); cmdMatches != nil {
				execType := cmdMatches[1]
				target := cmdMatches[2]

				// Add without line number
				exec := BashExecution{
					ScriptPath:  target,
					Command:     cmd,
					Line:        0, // Unknown line
					ExecutedBy:  execType,
					WithCoverage: true,
				}
				bs.BashScriptMap[scriptPath] = append(bs.BashScriptMap[scriptPath], exec)
			}
		}
	}

	// Calculate coverage percentage
	if coverage.TotalLines > 0 {
		coverage.CoveragePercent = float64(len(coverage.ExecutedLines)) / float64(coverage.TotalLines) * 100
	}

	return scanner.Err()
}

// CreateBashCallGraph creates call graph edges including bash scripts
func (bs *BashStitcher) CreateBashCallGraph(testFile string) []CallGraphEdge {
	// Get regular call graph edges
	edges := bs.CreateCallGraphConnections(testFile)
	
	// Add bash-specific edges
	bashExecutions := bs.BashScriptMap[testFile]
	for _, exec := range bashExecutions {
		edge := CallGraphEdge{
			From:     fmt.Sprintf("%s:%d", testFile, exec.Line),
			To:       fmt.Sprintf("%s:1", exec.ScriptPath), // Entry point of bash script
			EdgeType: fmt.Sprintf("bash:%s", exec.ExecutedBy),
			IsServer: false, // Bash scripts are typically not long-running servers
			Command:  exec.Command,
		}
		edges = append(edges, edge)
		
		// If we have coverage data, add edges for function calls within the script
		if coverage, ok := bs.BashCoverageMap[exec.ScriptPath]; ok {
			for funcName, fn := range coverage.Functions {
				if fn.Called {
					funcEdge := CallGraphEdge{
						From:     fmt.Sprintf("%s:1", exec.ScriptPath),
						To:       fmt.Sprintf("%s:%d:%s", exec.ScriptPath, fn.StartLine, funcName),
						EdgeType: "bash:function",
						IsServer: false,
						Command:  fmt.Sprintf("function %s", funcName),
					}
					edges = append(edges, funcEdge)
				}
			}
		}
	}
	
	return edges
}

// GetBashCoverageReport generates a coverage report for bash scripts
func (bs *BashStitcher) GetBashCoverageReport() string {
	var report strings.Builder
	
	report.WriteString("=== Bash Script Coverage Report ===\n\n")
	
	for scriptPath, coverage := range bs.BashCoverageMap {
		executedCount := len(coverage.ExecutedLines)
		percentage := float64(executedCount) / float64(coverage.TotalLines) * 100
		
		report.WriteString(fmt.Sprintf("%s:\n", scriptPath))
		report.WriteString(fmt.Sprintf("  Lines: %d/%d (%.1f%%)\n", 
			executedCount, coverage.TotalLines, percentage))
		
		// Report on functions
		report.WriteString("  Functions:\n")
		for funcName, fn := range coverage.Functions {
			status := "not called"
			if fn.Called {
				status = fmt.Sprintf("called %d times", fn.CallCount)
			}
			report.WriteString(fmt.Sprintf("    %s (lines %d-%d): %s\n", 
				funcName, fn.StartLine, fn.EndLine, status))
		}
		report.WriteString("\n")
	}
	
	return report.String()
}