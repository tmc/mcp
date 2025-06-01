# General-Purpose Tools Extracted from MCP

This document lists the general-purpose CLI tools that have been extracted from the MCP experimental commands and can be used independently of MCP.

## Tools Overview

### 1. json2go - JSON to Go Struct Converter
Converts JSON data, JSON Schema, or JSON-RPC to Go structs.

```bash
# Convert JSON API response to Go
curl https://api.example.com/data | json2go -type APIResponse

# Convert JSON Schema to Go types
json2go -schema -input schema.json -package models

# Convert JSON-RPC format
json2go -jsonrpc -input request.json -type RPCRequest
```

**Features:**
- Auto-detects JSON format
- Supports JSON Schema and JSON-RPC
- Customizable struct tags
- Package and type name configuration
- Handles nested structures

---

### 2. scripttest - Script-Based Testing Framework
Run script-based tests for CLI tools and command-line interfaces.

```bash
# Run all tests in current directory
scripttest

# Run specific test files
scripttest "test_*.txt"

# Update tests with actual output
scripttest -update failing.txt

# Verbose output with failure details
scripttest -verbose -bail 1
```

**Features:**
- Simple text-based test format
- Command execution and output verification
- Environment variable support
- Test updates for changing outputs
- Parallel test execution

---

### 3. gopackdump - Go Package Source Dumper
Dump Go package sources into various archive formats.

```bash
# Dump package to txtar format
gopackdump fmt

# Create tar archive with dependencies
gopackdump -recursive -format tar github.com/pkg/errors

# Extract to directory
gopackdump -format dir -o ./dump mypackage
```

**Features:**
- Multiple output formats (txtar, tar, directory)
- Recursive dependency inclusion
- Test file inclusion
- Custom path prefixes
- Vendor directory support

---

### 4. logcolor - Log Colorizer
Colorize log output with syntax highlighting for various formats.

```bash
# Colorize streaming logs
tail -f app.log | logcolor

# Colorize JSON logs
logcolor -format json < server.log

# Filter and colorize with line numbers
logcolor -filter ERROR -line-numbers < debug.log

# Dark theme for terminal
logcolor -theme dark < trace.log
```

**Features:**
- Auto-detects log formats
- Multiple color themes
- Pattern filtering
- Line numbering
- JSON and JSON-RPC support

---

### 5. tsnorm - Timestamp Normalizer
Normalize timestamps in text files to consistent formats.

```bash
# Normalize to RFC3339
tsnorm < app.log > normalized.log

# Convert to relative timestamps
tsnorm -relative < trace.log

# Convert to Unix timestamps
tsnorm -format unix < events.log

# Custom timestamp patterns
tsnorm -pattern '\d{8}T\d{6}' -parse '20060102T150405'
```

**Features:**
- Auto-detects common timestamp formats
- Multiple output formats
- Relative timestamp conversion
- Custom pattern support
- Preserves original text structure

---

### 6. schema2go - Schema to Go Code Generator
Generate Go code from various schema formats.

```bash
# JSON Schema to Go
schema2go -type json < schema.json

# OpenAPI to Go models
schema2go -type openapi api.yaml

# Auto-detect schema type
schema2go < unknown-schema.json > models.go
```

**Features:**
- Supports JSON Schema, OpenAPI, Protobuf
- Auto-detects schema format
- Customizable type prefixes
- Struct tag configuration
- Documentation preservation

---

### 7. tool-graph - Tool Dependency Visualizer
Analyze and visualize tool dependencies from test files.

```bash
# Visualize test dependencies
tool-graph -target tests/

# Generate interactive web view
tool-graph -target test.go -server

# Export to GraphViz
tool-graph -target tests/ -format dot
```

**Features:**
- Analyzes Go tests and scripttests
- Interactive React Flow visualization
- Multiple output formats
- Dependency graph generation
- Web server for viewing

---

## Installation

All tools can be installed using standard Go tooling:

```bash
# Install individual tools
go install github.com/tmc/mcp/exp/cmd/json2go@latest
go install github.com/tmc/mcp/exp/cmd/scripttest@latest
go install github.com/tmc/mcp/exp/cmd/gopackdump@latest
go install github.com/tmc/mcp/exp/cmd/logcolor@latest
go install github.com/tmc/mcp/exp/cmd/tsnorm@latest
go install github.com/tmc/mcp/exp/cmd/schema2go@latest
go install github.com/tmc/mcp/exp/cmd/tool-graph@latest

# Or build from source
cd exp
go build ./cmd/json2go
go build ./cmd/scripttest
# ... etc
```

## Use Cases

### Development Workflow
1. **API Development**: Use `json2go` to generate types from API responses
2. **Testing**: Use `scripttest` for CLI testing
3. **Documentation**: Use `gopackdump` to create documentation archives
4. **Debugging**: Use `logcolor` for readable log analysis
5. **Data Processing**: Use `tsnorm` to normalize timestamps across logs
6. **Code Generation**: Use `schema2go` for model generation
7. **Architecture**: Use `tool-graph` to understand dependencies

### DevOps and Operations
1. **Log Analysis**: Combine `logcolor` and `tsnorm` for log processing
2. **Test Automation**: Use `scripttest` in CI/CD pipelines
3. **Documentation**: Use `gopackdump` for code archival
4. **Monitoring**: Use `logcolor` for real-time log monitoring

### Data Engineering
1. **Schema Management**: Use `schema2go` for data model generation
2. **Log Processing**: Use `tsnorm` for timestamp standardization
3. **Testing**: Use `scripttest` for data pipeline testing

## Why These Tools?

These tools were extracted because they solve common development problems that extend well beyond the MCP ecosystem:

1. **json2go**: JSON is ubiquitous in modern APIs
2. **scripttest**: Testing CLIs is universally needed
3. **gopackdump**: Go package inspection is broadly useful
4. **logcolor**: Log readability affects all developers
5. **tsnorm**: Timestamp inconsistency is a common problem
6. **schema2go**: Schema-driven development is widespread
7. **tool-graph**: Understanding dependencies helps any project

## Contributing

These tools are part of the MCP project but are designed to be useful standalone. Contributions that maintain their general-purpose nature are welcome.

## License

These tools are part of the MCP project and follow the same license terms.