// Package mcpscripttest provides testing utilities for MCP tools using script-based testing.
//
// This package provides both minimal and full-featured testing modes:
//   - Minimal mode: Essential functionality only (core scripttest + setstdin)
//   - Full mode: All commands and conditions (requires extensions)
//
// The minimal mode is suitable for basic testing without external dependencies.
// The full mode includes all MCP tools, server management, and advanced conditions.
package mcpscripttest

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/coverage"
	"github.com/tmc/mcp/testing/mcpscripttest/internal"
	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

// Test runs script tests using the default full-featured options.
// This maintains backward compatibility with existing tests.
func Test(t *testing.T, pattern string, opts ...*Options) {
	internal.Test(t, pattern, opts...)
}

// TestMinimal runs script tests using minimal options (essential commands only).
// This is recommended for new tests that don't need advanced MCP commands.
func TestMinimal(t *testing.T, pattern string, opts ...*MinimalOptions) {
	internal.TestMinimal(t, pattern, opts...)
}

// TestWithOptions runs script tests with the given full-featured options.
func TestWithOptions(t *testing.T, pattern string, opts *Options) {
	internal.TestWithOptions(t, pattern, opts)
}

// TestWithCoverageOptions runs script tests with coverage options.
func TestWithCoverageOptions(t *testing.T, pattern string, coverageOpts *coverage.CoverageOptions, opts ...*Options) {
	internal.TestWithCoverageOptions(t, pattern, coverageOpts, opts...)
}

// Options represents configuration for full-featured script testing.
type Options = internal.Options

// MinimalOptions represents configuration for minimal script testing.
type MinimalOptions = internal.MinimalOptions

// CoverageOptions represents configuration for coverage.
type CoverageOptions = coverage.CoverageOptions

// TestRunner is a runner for script tests.
type TestRunner = internal.TestRunner

// DefaultOptions returns the default options for full-featured script testing.
func DefaultOptions() *Options {
	return internal.DefaultOptions()
}

// DefaultMinimalOptions returns the default options for minimal script testing.
func DefaultMinimalOptions() *MinimalOptions {
	return internal.DefaultMinimalOptions()
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
