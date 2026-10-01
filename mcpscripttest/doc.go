// Package mcpscripttest runs script-based tests for command-line tools and MCP servers.
//
// Tests use txtar files with commands and assertions. For example:
//
//	func TestScripts(t *testing.T) {
//		mcpscripttest.TestMinimal(t, "testdata/*.txt")
//	}
//
// Use TestWithCoverageOptions to set GOCOVERDIR for a script test. Use
// InstallMCPTools to build toolkit commands and add them to PATH.
package mcpscripttest
