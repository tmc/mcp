# Implementation Plan: mcp-goast

## Overview
`mcp-goast` is the foundational tool that enables intelligent code generation and analysis for the MCP ecosystem. It parses Go source code and provides structured access to AST information.

## Core Features

### Phase 1: Basic AST Parsing (Week 1)
1. **Package Analysis**
   - Parse single Go files
   - Extract package information
   - Handle imports correctly
   - Build symbol table

2. **Type Extraction**
   - Find all type definitions
   - Extract struct fields with tags
   - Identify interfaces
   - Map type relationships

3. **Function Analysis**
   - Extract function signatures
   - Identify receiver types
   - Parse parameter and return types
   - Handle generic constraints

### Phase 2: MCP Integration (Week 2)
1. **Handler Detection**
   - Identify MCP handler patterns
   - Extract tool definitions from code
   - Map handlers to tools
   - Validate handler signatures

2. **Schema Generation**
   - Convert Go types to JSON Schema
   - Handle nested structures
   - Support custom tags
   - Generate MCP tool descriptions

3. **Interface Implementation**
   - Find all implementations of an interface
   - Track embedding relationships
   - Validate interface satisfaction
   - Generate implementation stubs

### Phase 3: Advanced Analysis (Week 3)
1. **Dependency Graph**
   - Build package dependency tree
   - Identify circular dependencies
   - Track type usage across packages
   - Generate import optimization suggestions

2. **Code Generation Templates**
   - Extract patterns from existing code
   - Generate handler boilerplate
   - Create test templates
   - Produce documentation templates

## Technical Architecture

```go
package main

import (
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
)

type PackageAnalyzer struct {
    fset    *token.FileSet
    pkg     *types.Package
    info    *types.Info
    files   []*ast.File
}

type MCPHandler struct {
    Name       string
    Receiver   string
    Parameters []Parameter
    Returns    []Return
    ToolDef    *ToolDefinition
}

type ToolDefinition struct {
    Name        string
    Description string
    InputSchema  *JSONSchema
    OutputSchema *JSONSchema
}
```

## CLI Interface

```bash
# Analyze a package
mcp-goast analyze ./pkg/server

# Extract interfaces
mcp-goast interfaces ./pkg/server

# Find implementations
mcp-goast impl io.Reader ./...

# Generate MCP tool from function
mcp-goast gen-tool ./pkg/handlers.HandleTime

# Create handler from tool definition
mcp-goast gen-handler tool.json

# Visualize dependency graph
mcp-goast deps --graph ./...
```

## Integration Points

1. **With mcp-codegen**:
   - Provides AST information for code generation
   - Supplies type relationships for accurate generation
   - Enables pattern-based code creation

2. **With mcp-lint**:
   - Supplies AST for linting rules
   - Provides type information for validation
   - Enables custom rule creation

3. **With mcp-watch**:
   - Monitors AST changes
   - Triggers regeneration on structure changes
   - Provides change impact analysis

## Testing Strategy

1. **Unit Tests**:
   - AST parsing accuracy
   - Type extraction correctness
   - Schema generation validation

2. **Integration Tests**:
   - Real codebase analysis
   - Complex type relationships
   - Cross-package dependencies

3. **Benchmarks**:
   - Parse performance
   - Memory usage
   - Cache effectiveness

## Success Metrics

1. **Accuracy**: 99%+ correct type extraction
2. **Performance**: <100ms for average package
3. **Coverage**: Handle all Go language features
4. **Integration**: Seamless with other MCP tools

## Implementation Timeline

- **Week 1**: Core AST parsing and type extraction
- **Week 2**: MCP-specific analysis and generation
- **Week 3**: Advanced features and optimizations
- **Week 4**: Testing, documentation, and integration

## Dependencies

- Go standard library (go/ast, go/parser, go/types)
- No external dependencies initially
- Optional: golang.org/x/tools for advanced analysis

## Future Enhancements

1. **Incremental Analysis**: Only reparse changed files
2. **Caching Layer**: Store analysis results
3. **Plugin System**: Custom analyzers
4. **IDE Integration**: LSP support
5. **Cross-repository Analysis**: Analyze dependencies