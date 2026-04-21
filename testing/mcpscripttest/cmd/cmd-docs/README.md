# cmd-docs - MCPScriptTest Command Documentation Generator

`cmd-docs` is a comprehensive analysis and documentation tool for mcpscripttest custom commands. It parses Go source files to extract command definitions, automatically infer documentation, and generate improvement suggestions in multiple formats.

## Purpose

MCPScriptTest extends the rsc.io/script/scripttest framework with custom MCP-specific commands. As these commands evolve, maintaining comprehensive documentation becomes critical. The `cmd-docs` tool addresses this by:

- **Discovering** all custom commands in a codebase
- **Analyzing** command implementations for documentation gaps
- **Generating** documentation in multiple formats (text, JSON, structured edits)
- **Suggesting** improvements for better documentation coverage
- **Mapping** commands to their source locations for easy navigation

## Installation

```bash
go install ./testing/mcpscripttest/cmd/cmd-docs@latest
```

Or build locally:

```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp
go build -o cmd-docs ./testing/mcpscripttest/cmd/cmd-docs
```

## Quick Start

### Basic Usage

Analyze commands in the current directory:

```bash
cmd-docs
```

This outputs text-formatted documentation for all discovered commands to stdout.

### Generate JSON Output

Export structured data for programmatic processing:

```bash
cmd-docs -format json -output commands.json
```

### Focus on Specific Commands

Document a single command by name:

```bash
cmd-docs -cmd "mcp-server-start"
```

### Generate Documentation Edits

Create structured suggestions for improving documentation:

```bash
cmd-docs -format edits -output improvements.json
```

### Analyze Different Source Directory

```bash
cmd-docs -source /path/to/source -format text
```

## Output Formats

### Text Format (Default)

Human-readable documentation with suggestions:

```
=== MCPScriptTest Commands Documentation ===

## mcp-server-start
File: internal/server.go:45
Function: serverStartCmd

Description: Start a server process
Usage: mcp-server-start <server-name> [options]

Arguments:
  - server-name: string (required) - Name of the server

Examples:
  mcp-server-start myserver -- go run server.go
    # Start a Go server

Suggestions:
  - arguments: Commands typically have arguments that should be documented
    Current: none
    Suggested: TODO: Document command arguments

---

Total commands found: 23
```

### JSON Format

Structured output suitable for tooling and integration:

```json
[
  {
    "name": "mcp-server-start",
    "file": "internal/server.go",
    "line": 45,
    "function": "serverStartCmd",
    "description": "Start a server process",
    "usage": "mcp-server-start <server-name> [options]",
    "arguments": [
      {
        "name": "server-name",
        "type": "string",
        "required": true,
        "description": "Name of the server"
      }
    ],
    "examples": [
      {
        "command": "mcp-server-start myserver -- go run server.go",
        "description": "Start a Go server"
      }
    ],
    "registration": {
      "type": "Cmds",
      "location": "internal/server.go:45:1"
    },
    "suggestions": [
      {
        "type": "documentation",
        "field": "description",
        "current": "Start a server process",
        "suggested": "Start a server process with specified name and optional arguments",
        "reason": "Description could be more specific"
      }
    ]
  }
]
```

### Edits Format

Structured edit suggestions for automated documentation improvements:

```json
[
  {
    "file": "internal/server.go",
    "start_line": 44,
    "end_line": 44,
    "old_text": "",
    "new_text": "// mcp-server-start Start a server process\n// Usage: mcp-server-start <server-name> [options]\n// Arguments:\n//   server-name (string) - required: Name of the server\n// Examples:\n//   mcp-server-start myserver -- go run server.go",
    "description": "Add documentation for mcp-server-start command"
  }
]
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-source` | `.` | Source directory to analyze for Go files |
| `-output` | (stdout) | Output file path (if empty, writes to stdout) |
| `-format` | `text` | Output format: `text`, `json`, or `edits` |
| `-cmd` | (all) | Filter analysis to specific command by name |
| `-verbose` | false | Enable verbose output including parse errors |
| `-structured` | false | Enable structured edit suggestion output |
| `-help` | N/A | Display help information |

## Examples

### Analyze MCP Scripttest Commands

Analyze all commands in the mcpscripttest package:

```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp
cmd-docs -source ./testing/mcpscripttest -format text
```

### Create Documentation for Specific Command

Document the "bash" command implementation:

```bash
cmd-docs -source ./testing/mcpscripttest -cmd bash -format text
```

### Generate Improvement Suggestions

Export actionable documentation improvements:

```bash
cmd-docs -format edits -output /tmp/suggestions.json
cat /tmp/suggestions.json | jq '.[] | "\(.file):\(.start_line) - \(.description)"'
```

### Verbose Analysis with Error Reporting

Identify parse issues in source files:

```bash
cmd-docs -verbose -source ./testing/mcpscripttest
```

### Export All Commands as Structured Data

Create a database of all commands for automation:

```bash
cmd-docs -format json > commands_inventory.json
jq '.[] | "\(.name): \(.description)"' commands_inventory.json
```

## Command Discovery

The tool identifies custom commands by finding command registration patterns:

```go
// Pattern 1: Direct assignment
e.Cmds["command-name"] = handlerFunc

// Pattern 2: Commands field assignment
e.Commands["command-name"] = handlerFunc
```

Commands are extracted from:
- Variable assignments using selector expressions
- Index expressions with string literals
- Handler function references

## Documentation Inference

For commands without explicit documentation comments, the tool infers information:

### Description Inference

Converts function names to natural language descriptions:

- `serverStartCmd` → "Start a server process"
- `sendDataCmd` → "Send data to a process"
- `verifyOutputCmd` → "Verify output or behavior"

### Usage Pattern Generation

Generates appropriate usage patterns based on command names:

- `mcp-server-*` → `mcp-server-<name> <server-name> [options]`
- `mcp-*` → `mcp-<name> [options] <args>`
- Other commands → `<name> <args>`

### Argument Inference

Common patterns are recognized for typical argument types:

- Commands containing "server" get `server-name` argument
- Commands containing "send" get `data` argument
- Others require manual documentation

### Example Generation

Basic examples are auto-generated for:

- `mcp-server-start` → "Start a Go server"
- MCP commands → "--help" usage example
- Others → Generic example structure

## Suggestion Generation

The tool analyzes documentation completeness and generates suggestions:

### Documentation Gaps

- **Missing descriptions**: Generic or inferred descriptions trigger improvement suggestions
- **Missing arguments**: Commands typically have arguments that should be documented
- **Missing examples**: Examples help users understand usage patterns
- **Incomplete usage**: Usage strings that could be more specific

### Suggestion Types

| Type | Field | Example Reason |
|------|-------|--------|
| `documentation` | `description` | Missing or generic description |
| `documentation` | `arguments` | Commands typically have arguments |
| `documentation` | `examples` | Examples help users understand usage |
| `documentation` | `usage` | Usage could be more specific |

## Automation Workflows

### Update Documentation Automatically

Use the edits format with tools like `apply-edits`:

```bash
cmd-docs -format edits | apply-edits
```

### Generate Markdown Documentation

Convert JSON output to markdown:

```bash
cmd-docs -format json | jq -r '.[] |
  "## \(.name)\n\n\(.description)\n\nUsage: \(.usage)\n"' > commands.md
```

### Create Command Inventory

Build a searchable command database:

```bash
cmd-docs -format json > inventory.json
# Use with external tools or dashboards
```

### CI/CD Integration

Check for undocumented commands in CI:

```bash
cmd-docs -format json | jq '
  .[] | select(.suggestions | length > 0) |
  "\(.name): \(.suggestions[0].reason)"
' | tee /tmp/doc-issues.txt && [ ! -s /tmp/doc-issues.txt ]
```

## Architecture

### Code Structure

```
cmd/cmd-docs/
├── main.go          # Entry point and CLI
├── types.go         # Data structures (Command, Argument, etc.)
├── analysis.go      # Documentation analysis and suggestion generation
├── discovery.go     # Command discovery from source files
├── inference.go     # Documentation inference (descriptions, usage, etc.)
├── output.go        # Output formatting (text, JSON, edits)
└── README.md        # This file
```

### Key Components

**Command Discovery** (`discovery.go`):
- Walks source tree for Go files
- Parses AST to find command registrations
- Extracts command names, functions, and locations

**Analysis** (`analysis.go`):
- Infers documentation from command implementations
- Analyzes completeness of existing documentation
- Generates improvement suggestions

**Output Formatting** (`output.go`):
- Renders text-based documentation
- Serializes to JSON for programmatic use
- Generates structured edit suggestions

## Data Structures

### Command

The core data structure representing a custom command:

```go
type Command struct {
    Name         string           // Command name (e.g., "mcp-server-start")
    File         string           // Source file path
    Line         int              // Line number in source
    Function     string           // Handler function name
    Description  string           // Documentation description
    Usage        string           // Usage string
    Arguments    []Argument       // List of arguments
    Examples     []Example        // Usage examples
    Registration Registration    // Registration type and location
    Suggestions  []Suggestion     // Improvement suggestions
}
```

### Argument

Represents a single command argument:

```go
type Argument struct {
    Name        string // Argument name
    Type        string // Type (string, int, bool, etc.)
    Required    bool   // Whether argument is required
    Description string // Argument documentation
}
```

### Example

Represents a command usage example:

```go
type Example struct {
    Command     string // Example command line
    Description string // Explanation of the example
}
```

### Suggestion

Represents a documentation improvement suggestion:

```go
type Suggestion struct {
    Type      string // "documentation", etc.
    Field     string // Which field needs improvement
    Current   string // Current value
    Suggested string // Suggested value
    Reason    string // Why this suggestion is needed
}
```

## Development

### Building from Source

```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp
go build -o cmd-docs ./testing/mcpscripttest/cmd/cmd-docs
```

### Running Tests

```bash
go test ./testing/mcpscripttest/cmd/cmd-docs/...
```

### Extending the Tool

To add new analysis capabilities:

1. Add new suggestion types to the `Suggestion` struct if needed
2. Implement analysis logic in `analysis.go`
3. Update output formatters in `output.go` as needed
4. Add tests for new analysis logic

## Common Tasks

### Export Commands for Integration

```bash
# Create JSON for database import
cmd-docs -format json > commands.json

# Use in external tools
sqlite3 commands.db < import_commands.sql
```

### Generate Documentation Skeleton

```bash
# Create structured suggestions for missing docs
cmd-docs -format edits -structured | \
  jq '.[] | .description' | sort | uniq
```

### Audit Command Coverage

```bash
# Find commands with no examples
cmd-docs -format json | \
  jq '.[] | select(.examples | length == 0) | .name'
```

### Compare Documentation Between Versions

```bash
# Export baseline
git checkout main
cmd-docs -format json > baseline.json

# Compare with current
cmd-docs -format json > current.json
diff baseline.json current.json
```

## Related Tools

- **apply-edits**: Apply structured edit suggestions
- **testcallgraph**: Analyze command test coverage
- **mcpscripttest**: The main testing framework
- **scripttest**: Underlying script test framework (rsc.io/script)

## Troubleshooting

### Commands Not Discovered

**Problem**: Expected commands don't appear in output

**Solutions**:
- Verify source directory with `-source` flag
- Check command registration pattern matches `e.Cmds["name"]` or `e.Commands["name"]`
- Use `-verbose` to see parse errors
- Ensure Go files use standard AST syntax

### Inference Results Seem Wrong

**Problem**: Inferred descriptions or arguments don't match implementation

**Solutions**:
- Add explicit documentation comments in source code
- The tool uses heuristics; explicit documentation always wins
- Consider filing an issue if inference patterns are consistently wrong

### Output File Not Created

**Problem**: `-output` flag doesn't create file

**Solutions**:
- Check directory exists and is writable
- Ensure proper permissions on parent directory
- Try writing to `/tmp` as test

## Performance Considerations

- Single-threaded analysis suitable for most projects
- Scales well to 1000+ commands
- Large source trees may take several seconds to parse
- Consider filtering with `-cmd` for large analyses

## Future Enhancements

Potential improvements for future versions:

- [ ] Extract documentation from code comments automatically
- [ ] Cross-reference commands with test coverage
- [ ] Generate HTML documentation output
- [ ] Validate documentation against actual implementation
- [ ] Integration with CI/CD for documentation coverage reporting
- [ ] Support for additional command registration patterns
- [ ] Interactive mode for documentation editing
- [ ] Multi-language documentation support

## Contributing

To improve cmd-docs:

1. Identify documentation gaps or analysis limitations
2. Create test cases demonstrating the issue
3. Implement analysis improvements
4. Add tests for new functionality
5. Update documentation

## License

cmd-docs is part of the MCP project and follows the same license.
