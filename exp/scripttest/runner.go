package scripttest

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Options for the test runner
type Options struct {
	Verbose         bool
	ContinueOnError bool
	Timeout         time.Duration
	KeepWork        bool
	UpdateMode      bool
	Environment     map[string]string
	BailAfter       int
}

// Results from running tests
type Results struct {
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Failures []Failure
}

// Failure describes a test failure
type Failure struct {
	Test  string
	Error string
	Line  int
}

// Runner runs script-based tests
type Runner struct {
	options Options
}

// NewRunner creates a new test runner
func NewRunner(options Options) *Runner {
	return &Runner{
		options: options,
	}
}

// RunTests runs all the specified test files
func (r *Runner) RunTests(testFiles []string) *Results {
	start := time.Now()
	results := &Results{
		Total:    len(testFiles),
		Failures: []Failure{},
	}

	for _, testFile := range testFiles {
		if r.options.Verbose {
			fmt.Printf("Running %s...\n", testFile)
		}

		err := r.runTest(testFile)

		if err != nil {
			results.Failed++
			results.Failures = append(results.Failures, Failure{
				Test:  testFile,
				Error: err.Error(),
			})

			if !r.options.ContinueOnError {
				break
			}

			if r.options.BailAfter > 0 && results.Failed >= r.options.BailAfter {
				break
			}
		} else {
			results.Passed++
		}
	}

	results.Duration = time.Since(start)
	return results
}

func (r *Runner) runTest(testFile string) error {
	// Create work directory
	workDir, err := os.MkdirTemp("", "scripttest")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	if !r.options.KeepWork {
		defer os.RemoveAll(workDir)
	} else if r.options.Verbose {
		fmt.Printf("Work directory: %s\n", workDir)
	}

	// Read test file
	content, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}

	// Parse and execute test
	test := &Test{
		file:    testFile,
		workDir: workDir,
		env:     r.makeEnv(),
		timeout: r.options.Timeout,
		verbose: r.options.Verbose,
		update:  r.options.UpdateMode,
	}

	return test.Run(string(content))
}

func (r *Runner) makeEnv() []string {
	env := os.Environ()

	// Add custom environment variables
	for k, v := range r.options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// Test represents a single test execution
type Test struct {
	file    string
	workDir string
	env     []string
	timeout time.Duration
	verbose bool
	update  bool

	currentDir string
	lineNum    int
	updates    map[int]string
}

// Run executes the test
func (t *Test) Run(content string) error {
	t.currentDir = t.workDir
	t.updates = make(map[int]string)

	scanner := bufio.NewScanner(strings.NewReader(content))
	t.lineNum = 0

	for scanner.Scan() {
		t.lineNum++
		line := scanner.Text()

		// Strip comments and whitespace
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if err := t.executeLine(line); err != nil {
			return fmt.Errorf("line %d: %w", t.lineNum, err)
		}
	}

	if t.update && len(t.updates) > 0 {
		return t.updateFile()
	}

	return scanner.Err()
}

func (t *Test) executeLine(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "exec":
		return t.execCommand(args)
	case "!":
		return t.execCommandExpectFail(args)
	case "stdin":
		return t.sendStdin(strings.Join(args, " "))
	case "stdout":
		return t.expectStdout(strings.Join(args, " "))
	case "stderr":
		return t.expectStderr(strings.Join(args, " "))
	case "cd":
		return t.changeDir(strings.Join(args, " "))
	case "env":
		return t.setEnv(strings.Join(args, " "))
	case "skip":
		return t.skip(strings.Join(args, " "))
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func (t *Test) execCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("exec requires command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = t.currentDir
	cmd.Env = t.env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if t.timeout > 0 {
		timer := time.AfterFunc(t.timeout, func() {
			cmd.Process.Kill()
		})
		defer timer.Stop()
	}

	err := cmd.Run()

	t.lastStdout = stdout.String()
	t.lastStderr = stderr.String()

	if t.verbose {
		if t.lastStdout != "" {
			fmt.Printf("stdout: %s\n", t.lastStdout)
		}
		if t.lastStderr != "" {
			fmt.Printf("stderr: %s\n", t.lastStderr)
		}
	}

	return err
}

func (t *Test) execCommandExpectFail(args []string) error {
	err := t.execCommand(args)
	if err == nil {
		return fmt.Errorf("expected command to fail but it succeeded")
	}
	return nil
}

func (t *Test) expectStdout(expected string) error {
	expected = expandVars(expected, t.env)

	if !matchOutput(t.lastStdout, expected) {
		if t.update {
			t.updates[t.lineNum] = fmt.Sprintf("stdout %s", quote(t.lastStdout))
			return nil
		}
		return fmt.Errorf("stdout mismatch:\nexpected: %s\nactual: %s", expected, t.lastStdout)
	}

	return nil
}

func (t *Test) expectStderr(expected string) error {
	expected = expandVars(expected, t.env)

	if !matchOutput(t.lastStderr, expected) {
		if t.update {
			t.updates[t.lineNum] = fmt.Sprintf("stderr %s", quote(t.lastStderr))
			return nil
		}
		return fmt.Errorf("stderr mismatch:\nexpected: %s\nactual: %s", expected, t.lastStderr)
	}

	return nil
}

func (t *Test) changeDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("cd requires directory")
	}

	dir = expandVars(dir, t.env)
	newDir := filepath.Join(t.currentDir, dir)

	if _, err := os.Stat(newDir); err != nil {
		return fmt.Errorf("directory not found: %s", newDir)
	}

	t.currentDir = newDir
	return nil
}

func (t *Test) setEnv(envStr string) error {
	parts := strings.SplitN(envStr, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid env format, expected KEY=value")
	}

	key := parts[0]
	value := expandVars(parts[1], t.env)
	t.env = append(t.env, fmt.Sprintf("%s=%s", key, value))

	return nil
}

func (t *Test) skip(message string) error {
	return fmt.Errorf("SKIP: %s", message)
}

func (t *Test) sendStdin(data string) error {
	// Store for next exec command
	t.pendingStdin = expandVars(data, t.env)
	return nil
}

// Add these fields to Test struct
var _ = (*Test)(nil).lastStdout
var _ = (*Test)(nil).lastStderr
var _ = (*Test)(nil).pendingStdin

type TestWithOutput struct {
	*Test
	lastStdout   string
	lastStderr   string
	pendingStdin string
}

func (t *Test) updateFile() error {
	// Read original file
	content, err := os.ReadFile(t.file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	// Apply updates in reverse order to maintain line numbers
	lineNums := make([]int, 0, len(t.updates))
	for line := range t.updates {
		lineNums = append(lineNums, line)
	}

	for i := len(lineNums) - 1; i >= 0; i-- {
		lineNum := lineNums[i]
		if lineNum > 0 && lineNum <= len(lines) {
			lines[lineNum-1] = t.updates[lineNum]
		}
	}

	// Write back
	return os.WriteFile(t.file, []byte(strings.Join(lines, "\n")), 0644)
}

// Helper functions

func matchOutput(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	// Simple string contains for now
	// Could add regex support later
	return strings.Contains(actual, expected)
}

func expandVars(s string, env []string) string {
	// Simple variable expansion
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			old := "$" + parts[0]
			s = strings.ReplaceAll(s, old, parts[1])
		}
	}
	return s
}

func quote(s string) string {
	if strings.ContainsAny(s, " \t\n'\"") {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "\\'"))
	}
	return s
}
