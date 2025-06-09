//go:build !test
// +build !test

package internal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"rsc.io/script"
	"rsc.io/script/scripttest"

	"github.com/tmc/mcp/testing/mcpscripttest/conditions"
)

// NewStandaloneEngine creates a script engine for standalone execution (not in go test)
func NewStandaloneEngine(options *MCPScripttestOptions) *script.Engine {
	if options == nil {
		options = DefaultOptions()
	}

	e := script.NewEngine()

	// Add default commands from scripttest
	for k, v := range scripttest.DefaultCmds() {
		e.Cmds[k] = v
	}

	// Add minimal conditions without the problematic testing.Short()
	e.Conds["GOOS"] = script.OnceCondition(fmt.Sprintf("runtime.GOOS == %q", runtime.GOOS), func() (bool, error) {
		return true, nil
	})
	e.Conds["GOARCH"] = script.OnceCondition(fmt.Sprintf("runtime.GOARCH == %q", runtime.GOARCH), func() (bool, error) {
		return true, nil
	})
	e.Conds["unix"] = script.OnceCondition("unix", func() (bool, error) {
		return runtime.GOOS != "windows", nil
	})
	e.Conds["windows"] = script.OnceCondition("windows", func() (bool, error) {
		return runtime.GOOS == "windows", nil
	})

	// Add environment variable conditions
	for _, name := range []string{"GITHUB_ACTIONS", "CI"} {
		if val := os.Getenv(name); val != "" {
			e.Conds[name] = script.OnceCondition(name, func() (bool, error) {
				return true, nil
			})
		}
	}

	// Add default MCP commands if enabled
	if options.IncludeDefaultMCPCommands {
		addDefaultMCPCommands(e)
	}

	// Add default MCP conditions
	conditions.AddDefaultMCPConditions(e)

	// Add server command support
	addServerCommandSupport(e)

	// Register server management commands
	registerServerCommands(e)

	// Apply custom commands
	for name, cmd := range options.CustomCommands {
		e.Cmds[name] = cmd
	}

	// Apply custom conditions
	for name, cond := range options.CustomConditions {
		e.Conds[name] = cond
	}

	// Skip server command conditions for standalone mode

	return e
}

// RunTestsStandalone runs tests without the go test framework
func (r *TestRunner) RunTestsStandalone(pattern string) int {
	// Find files matching the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("Error finding test files: %v\n", err)
		return 1
	}

	if len(files) == 0 {
		fmt.Printf("No script files found matching pattern: %s\n", pattern)
		return 1
	}

	// Set up the script engine with MCP commands
	engine := NewStandaloneEngine(r.Options)

	// Get environment variables
	env := getTestEnvironment(r.Options.AdditionalEnvVars)

	// Create a context
	ctx := context.Background()
	if r.Options.RunDeadcodeCheck {
		// Add a timeout if deadcode check is enabled
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
	}

	failures := 0
	for _, file := range files {
		fmt.Printf("\nRunning %s:\n", file)

		// Run the test
		err := runStandaloneScriptFile(ctx, engine, env, file, r.Verbose)
		if err != nil {
			failures++
			fmt.Printf("FAIL: %s: %v\n", file, err)
		} else {
			fmt.Printf("PASS: %s\n", file)
		}
	}

	return failures
}

func runStandaloneScriptFile(ctx context.Context, engine *script.Engine, env []string, file string, verbose bool) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	s, err := script.NewState(ctx, filepath.Dir(file), env)
	if err != nil {
		return fmt.Errorf("failed to create script state: %w", err)
	}
	defer s.CloseAndWait(os.Stdout)

	// Execute the script
	r := strings.NewReader(string(content))
	err = engine.Execute(s, file, bufio.NewReader(r), os.Stdout)
	return err
}
