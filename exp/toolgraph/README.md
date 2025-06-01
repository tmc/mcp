# MCP Tool Graph Visualization

This tool analyzes MCP test files and generates interactive visualizations of the tool dependency graph.

## Overview

The `mcp-tool-graph` command analyzes test files (both Go tests and scripttest files) to:
- Identify tool invocations
- Map dependencies between tests, tools, and commands
- Generate interactive visualizations using React Flow
- Export to various formats (React Flow, Graphviz DOT, JSON)

## Installation

```bash
cd exp/toolgraph
go install ../cmd/mcp-tool-graph
```

## Usage

### Basic Usage

Analyze a single test file:
```bash
mcp-tool-graph -target my_test.go
```

Analyze all tests in a directory:
```bash
mcp-tool-graph -target ./tests
```

### Visualization Options

Generate React Flow visualization (default):
```bash
mcp-tool-graph -target tests/ -format react -output viz/
```

Start a web server to view the visualization:
```bash
mcp-tool-graph -target tests/ -server -port 8080
```

Generate Graphviz DOT file:
```bash
mcp-tool-graph -target tests/ -format dot -output graph.dot
```

Generate JSON representation:
```bash
mcp-tool-graph -target tests/ -format json -output graph.json
```

### Advanced Options

Limit traversal depth:
```bash
mcp-tool-graph -target tests/ -max-depth 5
```

Include standard library dependencies:
```bash
mcp-tool-graph -target tests/ -include-std
```

Enable verbose output:
```bash
mcp-tool-graph -target tests/ -verbose
```

## Graph Elements

The visualization includes different types of nodes:

- **Test Nodes** (🧪): Test files and test functions
- **Tool Nodes** (🔧): MCP tools and operations
- **Command Nodes** (⚡): System commands (exec calls)
- **File Nodes** (📄): Files accessed by tests
- **Package Nodes** (📦): Go packages

## Example Workflow

1. Analyze a scripttest file:
```bash
mcp-tool-graph -target auth_test.txt -output auth-viz/
```

2. Start the web server:
```bash
mcp-tool-graph -target auth_test.txt -server
```

3. Open http://localhost:8080 in your browser

4. Interact with the graph:
   - Click nodes to see details
   - Drag to reposition nodes
   - Use controls to zoom and pan
   - View the minimap for navigation

## Output Formats

### React Flow Format
Generates an interactive web-based visualization with:
- `index.html` - Main HTML file
- `styles.css` - Styling
- `app.js` - React application
- `graph-data.json` - Graph data

### DOT Format
Generates a Graphviz DOT file that can be rendered:
```bash
dot -Tpng graph.dot -o graph.png
dot -Tsvg graph.dot -o graph.svg
```

### JSON Format
Exports the raw graph data structure for custom processing.

## Node Types and Colors

- **Test** (Blue): Test files and functions
- **Tool** (Purple): MCP tools and operations  
- **Command** (Green): System commands
- **File** (Orange): File operations
- **Package** (Pink): Go packages

## Edge Types

- `executes`: Test executes a command
- `invokes`: Command invokes a tool
- `calls`: Tool calls another tool
- `imports`: Go import relationship
- `reads`/`writes`: File operations
- `expects`: Expected output

## Architecture

The tool consists of several components:

1. **Graph Builder**: Constructs the dependency graph
2. **AST Parser**: Analyzes Go source code
3. **Scripttest Parser**: Analyzes scripttest files
4. **Layout Engine**: Positions nodes for visualization
5. **Format Converters**: Exports to different formats

## Extending

To add support for new patterns:

1. Extend the parser in `scripttest.go` or `graph.go`
2. Add new node types in `graph.go`
3. Update the visualization in `convert.go`
4. Add styling in the React components

## Use Cases

- **Test Analysis**: Understand test dependencies
- **Tool Discovery**: Find all tools used in tests
- **Documentation**: Generate visual documentation
- **Refactoring**: Identify coupled components
- **Debugging**: Trace tool invocation paths

## Limitations

- Basic layout algorithm (may need manual adjustment)
- Limited pattern matching for complex cases
- No real-time updates (static analysis only)

## Future Enhancements

- Real-time test execution tracking
- Coverage overlay on the graph
- Test failure highlighting
- Performance metrics visualization
- Export to more formats (Mermaid, PlantUML)