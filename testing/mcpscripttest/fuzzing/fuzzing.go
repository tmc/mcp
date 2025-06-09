package fuzzing

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// FuzzGenerator generates valid scripttest content for fuzzing
type FuzzGenerator struct {
	rng    *rand.Rand
	schema *mcpscripttest.Schema
}

// NewFuzzGenerator creates a new fuzzing generator with the given seed
func NewFuzzGenerator(seed int64) *FuzzGenerator {
	return &FuzzGenerator{
		rng:    rand.New(rand.NewSource(seed)),
		schema: mcpscripttest.GetSchema(),
	}
}

// Generate creates a valid scripttest script for fuzzing
func (g *FuzzGenerator) Generate() string {
	var lines []string

	// Start with a comment
	lines = append(lines, "# Generated test script for fuzzing")
	lines = append(lines, "")

	// Generate random number of commands (between 3 and 15)
	numCommands := g.rng.Intn(13) + 3

	for i := 0; i < numCommands; i++ {
		line := g.generateCommand()
		if line != "" {
			lines = append(lines, line)
		}
	}

	// Ensure we have at least one exec command to make the test meaningful
	if !containsExec(lines) {
		lines = append(lines, "exec echo 'test complete'")
		lines = append(lines, "stdout 'test complete'")
	}

	return strings.Join(lines, "\n")
}

func containsExec(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "exec ") {
			return true
		}
	}
	return false
}

// generateCommand generates a single random command
func (g *FuzzGenerator) generateCommand() string {
	// Weighted selection of command types
	cmdType := g.rng.Intn(100)

	switch {
	case cmdType < 40: // 40% chance of core commands
		return g.generateCoreCommand()
	case cmdType < 70: // 30% chance of MCP commands
		return g.generateMCPCommand()
	case cmdType < 85: // 15% chance of directives
		return g.generateDirective()
	default: // 15% chance of comments/empty lines
		if g.rng.Intn(2) == 0 {
			return fmt.Sprintf("# %s", g.randomComment())
		}
		return ""
	}
}

func (g *FuzzGenerator) generateCoreCommand() string {
	cmd := g.schema.CoreCommands[g.rng.Intn(len(g.schema.CoreCommands))]

	switch cmd.Name {
	case "exec":
		return g.generateExecCommand()
	case "stdin":
		return fmt.Sprintf("stdin %s", g.randomInput())
	case "stdout":
		return fmt.Sprintf("stdout %s", g.randomExpectation())
	case "stderr":
		return fmt.Sprintf("stderr %s", g.randomExpectation())
	case "cat":
		return fmt.Sprintf("cat %s", g.randomFileName())
	case "cp":
		return fmt.Sprintf("cp %s %s", g.randomFileName(), g.randomFileName())
	case "mv":
		return fmt.Sprintf("mv %s %s", g.randomFileName(), g.randomFileName())
	case "rm":
		return fmt.Sprintf("rm %s", g.randomFileName())
	case "mkdir":
		return fmt.Sprintf("mkdir %s", g.randomDirName())
	case "cd":
		return fmt.Sprintf("cd %s", g.randomDirName())
	case "env":
		return fmt.Sprintf("env %s=%s", g.randomEnvName(), g.randomEnvValue())
	case "sleep":
		return fmt.Sprintf("sleep %dms", g.rng.Intn(1000)+100)
	case "wait":
		if g.rng.Intn(2) == 0 {
			return "wait"
		}
		return "wait $server"
	default:
		return ""
	}
}

func (g *FuzzGenerator) generateMCPCommand() string {
	cmd := g.schema.MCPCommands[g.rng.Intn(len(g.schema.MCPCommands))]

	switch cmd.Name {
	case "mcp-send":
		return fmt.Sprintf("mcp-send %s", g.randomJSONRPC())
	case "mcp-recv":
		return fmt.Sprintf("mcp-recv %s", g.randomExpectation())
	case "mcp-serve":
		return fmt.Sprintf("mcp-serve -- %s", g.randomServerCommand())
	case "mcp-trace":
		return fmt.Sprintf("mcp-trace %s", g.randomTraceFile())
	default:
		return ""
	}
}

func (g *FuzzGenerator) generateDirective() string {
	directives := []string{"!", "?", "[linux]", "[!windows]", "skip"}
	directive := directives[g.rng.Intn(len(directives))]

	switch directive {
	case "!":
		return fmt.Sprintf("! exec %s", g.randomCommand())
	case "?":
		return fmt.Sprintf("? exec %s", g.randomCommand())
	case "[linux]":
		return fmt.Sprintf("[linux] exec %s", g.randomCommand())
	case "[!windows]":
		return fmt.Sprintf("[!windows] exec %s", g.randomCommand())
	case "skip":
		if g.rng.Intn(2) == 0 {
			return "skip windows"
		}
		return "skip !linux"
	default:
		return ""
	}
}

func (g *FuzzGenerator) generateExecCommand() string {
	commands := []string{
		"echo test",
		"echo 'hello world'",
		"true",
		"false",
		"ls",
		"pwd",
		"date",
		"cat test.txt",
		"grep pattern file.txt",
		"wc -l file.txt",
		"sort data.txt",
	}
	return fmt.Sprintf("exec %s", commands[g.rng.Intn(len(commands))])
}

func (g *FuzzGenerator) randomInput() string {
	inputs := []string{
		"test input",
		"hello world",
		`{"jsonrpc":"2.0","method":"test","id":1}`,
		"line1\nline2\nline3",
		"special chars: !@#$%",
		"",
	}
	return inputs[g.rng.Intn(len(inputs))]
}

func (g *FuzzGenerator) randomExpectation() string {
	expectations := []string{
		"'success'",
		"result",
		"error",
		"'test complete'",
		"jsonrpc",
		"[0-9]+",
	}
	return expectations[g.rng.Intn(len(expectations))]
}

func (g *FuzzGenerator) randomFileName() string {
	files := []string{
		"test.txt",
		"output.json",
		"data.log",
		"config.yaml",
		"temp.tmp",
		"file with spaces.txt",
	}
	return files[g.rng.Intn(len(files))]
}

func (g *FuzzGenerator) randomDirName() string {
	dirs := []string{
		"test",
		"output",
		"logs",
		"temp",
		"subdir/nested",
		"..",
		".",
	}
	return dirs[g.rng.Intn(len(dirs))]
}

func (g *FuzzGenerator) randomEnvName() string {
	names := []string{
		"TEST_VAR",
		"DEBUG",
		"MCP_TEST",
		"GOCOVERDIR",
		"PATH_ADDON",
		"CONFIG",
	}
	return names[g.rng.Intn(len(names))]
}

func (g *FuzzGenerator) randomEnvValue() string {
	values := []string{
		"1",
		"true",
		"/tmp/test",
		"value with spaces",
		"special!chars@here",
		"",
	}
	return values[g.rng.Intn(len(values))]
}

func (g *FuzzGenerator) randomJSONRPC() string {
	methods := []string{"initialize", "tools/list", "tools/call", "resources/list"}
	id := g.rng.Intn(100) + 1
	method := methods[g.rng.Intn(len(methods))]

	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","id":%d}`, method, id)
}

func (g *FuzzGenerator) randomServerCommand() string {
	commands := []string{
		"go run server.go",
		"node server.js",
		"python server.py",
		"./server",
		"/usr/local/bin/mcp-server",
	}
	return commands[g.rng.Intn(len(commands))]
}

func (g *FuzzGenerator) randomTraceFile() string {
	return fmt.Sprintf("trace_%d.mcp", g.rng.Intn(1000))
}

func (g *FuzzGenerator) randomCommand() string {
	commands := []string{
		"true",
		"false",
		"echo test",
		"ls nonexistent",
		"cat missing.txt",
		"grep pattern file",
	}
	return commands[g.rng.Intn(len(commands))]
}

func (g *FuzzGenerator) randomComment() string {
	comments := []string{
		"Test setup",
		"Verify output",
		"Clean up",
		"This should fail",
		"Platform-specific test",
		"Edge case",
	}
	return comments[g.rng.Intn(len(comments))]
}

// newRand creates a new rand.Rand with the given seed
func newRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// FuzzScriptTest is a coverage-guided fuzz test for scripttest generation
func FuzzScriptTest(f *testing.F) {
	// Set up coverage feedback if available
	coverageDir := os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		coverageDir = f.TempDir()
	}

	feedback, err := NewCoverageFeedback(coverageDir)
	if err != nil {
		f.Logf("Warning: Failed to initialize coverage feedback: %v", err)
		// Continue without coverage guidance
		feedback = nil
	}

	// Create coverage-guided fuzzer if feedback is available
	var fuzzer *CoverageGuidedFuzzer
	generator := NewFuzzGenerator(0) // Will set seed per iteration

	if feedback != nil {
		fuzzer = NewCoverageGuidedFuzzer(generator, feedback)
	}

	// Set up visualizer if requested
	var viz *Visualizer
	if os.Getenv("MCP_FUZZ_VISUALIZE") == "1" || testing.Verbose() {
		vizOpts := DefaultVisualizerOptions()
		vizOpts.Enabled = true
		vizOpts.ShowRejected = os.Getenv("MCP_FUZZ_SHOW_REJECTED") == "1"
		vizOpts.ClearScreen = os.Getenv("MCP_FUZZ_CLEAR_SCREEN") == "1"
		viz = NewVisualizer(vizOpts)
		defer viz.Close()
	}

	// Add seed corpus
	f.Add(int64(42))
	f.Add(int64(123))
	f.Add(int64(999))

	// Fuzz function
	f.Fuzz(func(t *testing.T, seed int64) {
		// Update generator seed
		generator.rng = newRand(seed)

		// Generate script (potentially based on good inputs)
		var script string
		if fuzzer != nil {
			script = fuzzer.GenerateInput()
		} else {
			script = generator.Generate()
		}

		// Notify visualizer of test start
		if viz != nil {
			viz.StartTest(script)
		}

		// Create a temporary file with the generated script
		tmpfile, err := os.CreateTemp("", "fuzz-*.txt")
		if err != nil {
			t.Skip("Could not create temp file")
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.WriteString(script); err != nil {
			t.Skip("Could not write to temp file")
		}
		if err := tmpfile.Close(); err != nil {
			t.Skip("Could not close temp file")
		}

		// Track test result for visualization
		var testPassed bool
		defer func() {
			if r := recover(); r != nil {
				// Test failed/panicked
				if viz != nil {
					viz.RejectScript(script, fmt.Errorf("panic: %v", r))
				}
				panic(r) // Re-panic for fuzzer
			}
		}()

		// Run the test with coverage collection if available
		if feedback != nil && fuzzer != nil {
			result, err := feedback.RunWithCoverage(t, func() {
				// Run the scripttest
				// This will panic if the parser has bugs, which is what we want for fuzzing
				mcpscripttest.Test(t, tmpfile.Name())
				testPassed = true
			})

			if err == nil && testPassed {
				// Test succeeded - notify visualizer
				if viz != nil {
					viz.AcceptScript(script)
				}

				// Record the result for future guidance
				fuzzer.RecordResult(script, result)

				// Update baseline if coverage improved
				if result.CoverageIncrease > 0 {
					if viz != nil {
						viz.UpdateCoverage(result.TotalCoverage, len(result.NewPackages))
					}
					if err := feedback.UpdateBaseline(result); err != nil {
						t.Logf("Failed to update baseline: %v", err)
					}
				}
			} else if err != nil && viz != nil {
				viz.RejectScript(script, err)
			}
		} else {
			// Run without coverage collection
			mcpscripttest.Test(t, tmpfile.Name())
			testPassed = true
			if viz != nil {
				viz.AcceptScript(script)
			}
		}
	})
}
