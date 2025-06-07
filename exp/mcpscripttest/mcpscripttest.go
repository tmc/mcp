// Package mcpscripttest provides testing utilities for MCP tools using script-based testing.
//
// This package wraps the rsc.io/script/scripttest package to provide script-based testing for MCP
// commands and servers. It handles coverage instrumentation, tool installation, and other
// MCP-specific test requirements.
package mcpscripttest

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/coverage"
	"github.com/tmc/mcp/exp/mcpscripttest/internal"
	"github.com/tmc/mcp/exp/mcpscripttest/tools"
)

// Test runs script tests using the default options.
// This is a convenience wrapper around TestWithOptions.
func Test(t *testing.T, pattern string, opts ...*Options) {
	internal.Test(t, pattern, opts...)
}

// TestWithOptions runs script tests with the given options.
func TestWithOptions(t *testing.T, pattern string, opts *Options) {
	internal.TestWithOptions(t, pattern, opts)
}

// TestWithCoverageOptions runs script tests with coverage options.
func TestWithCoverageOptions(t *testing.T, pattern string, coverageOpts *coverage.CoverageOptions, opts ...*Options) {
	internal.TestWithCoverageOptions(t, pattern, coverageOpts, opts...)
}

// Options represents configuration for script testing.
type Options = internal.Options

// DefaultOptions returns the default options for script testing.
func DefaultOptions() *Options {
	return internal.DefaultOptions()
}

// DefaultCoverageOptions returns the default coverage options.
func DefaultCoverageOptions() *coverage.CoverageOptions {
	return coverage.DefaultCoverageOptions()
}

// CheckCoverageWarning checks and warns about coverage status.
func CheckCoverageWarning(t *testing.T) {
	coverage.CheckWarning(t)
}

// InstallMCPTools installs MCP tools with the given options.
func InstallMCPTools(t *testing.T, opts *tools.ToolsOptions) func() {
	return tools.InstallMCPTools(t, opts)
}

// TestMainOptions represents options for TestMain.
type TestMainOptions = internal.TestMainOptions

// DefaultTestMainOptions returns the default TestMain options.
func DefaultTestMainOptions() *TestMainOptions {
	return internal.DefaultTestMainOptions()
}

// RunTestMain runs the test main with options.
func RunTestMain(m *testing.M, opts *TestMainOptions) int {
	return internal.RunTestMain(m, opts)
}


// TestWithInProcessRegular runs tests with in-process servers without synctest.
// This provides faster tests than exec while maintaining compatibility with older Go versions.
// Disabled - part of experimental code that was cleaned up
/*
func TestWithInProcessRegular(t *testing.T, pattern string, serverRegistry map[string]func(), opts ...*Options) {
	// Convert options to interface{} slice for internal call
	var optsInterface []interface{}
	for _, opt := range opts {
		optsInterface = append(optsInterface, opt)
	}
	internal.TestWithInProcessRegular(t, pattern, serverRegistry, optsInterface...)
}
*/

