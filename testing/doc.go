// Package testing provides testing utilities and infrastructure for MCP implementations.
//
// This package serves as the parent for various testing-related subpackages:
//
//   - mcptestutil: Comprehensive testing utilities including mocks, assertions, and helpers
//
// The testing package itself provides common testing infrastructure and coordinates
// testing functionality across the MCP codebase.
//
// # Testing Philosophy
//
// The MCP testing approach emphasizes:
//   - Protocol compliance verification
//   - Cross-implementation compatibility
//   - Type-safe assertions and helpers
//   - Comprehensive mock implementations
//   - Table-driven test patterns
//
// # Usage
//
// Import the specific subpackage you need:
//
//	import "github.com/tmc/mcprepos/mcp/testing/mcptestutil"
//
//	func TestMyFeature(t *testing.T) {
//	    server := mcptestutil.NewMockServer()
//	    // ... test implementation
//	}
package testing