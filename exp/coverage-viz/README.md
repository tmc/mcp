# MCP Coverage Visualization

A tool for visualizing test coverage and MCP trace data in a unified web interface.

## Features

- **Unified Coverage View**: Combines Go test coverage with MCP trace data
- **Interactive Web UI**: Browse code with coverage highlights
- **Test Timeline**: Visualize test execution flow with MCP traces
- **Test Impact Analysis**: See which tests cover which code
- **Multiple Input Formats**: Supports standard Go coverage profiles and MCP trace files
- **Export Options**: Generate JSON, HTML, or CSV reports

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/mcp-coverage-viz@latest
```

## Usage

### Basic Usage

```bash
# Analyze coverage and start web server
mcp-coverage-viz -coverage coverage.out -trace trace.mcp -serve

# Process directory of trace files
mcp-coverage-viz -coverage coverage.out -trace-dir ./traces -serve

# Export to JSON
mcp-coverage-viz -coverage coverage.out -trace trace.mcp -output viz.json
```

### Command-line Options

- `-coverage`: Go coverage profile file
- `-trace`: Single MCP trace file (.mcp)
- `-trace-dir`: Directory containing MCP trace files
- `-output`: Output file for visualization data (JSON)
- `-serve`: Start web server for visualization
- `-port`: Port for web server (default: :8080)
- `-source`: Source code directory (default: .)

## Web Interface

The web interface provides:

1. **Dashboard**: Overview of coverage statistics and test results
2. **File Browser**: Interactive code view with coverage highlighting
3. **Test Timeline**: Visual representation of test execution
4. **Test Details**: Individual test results and traces
5. **Package View**: Coverage breakdown by package

## Data Model

The tool uses a unified data model that combines:

- Go coverage profiles (line and branch coverage)
- MCP trace data (requests, responses, notifications)
- Test execution metadata (timing, results, output)
- Source code analysis (functions, packages)

## API

The web server exposes REST endpoints:

- `GET /api/coverage`: Complete coverage data
- `GET /api/files/{path}`: File-specific coverage
- `GET /api/tests`: Test execution data
- `GET /api/sessions`: Test session information

## Examples

### Generate Combined Report

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Capture MCP traces
MCP_TRACE=trace.mcp go test ./...

# Visualize results
mcp-coverage-viz -coverage coverage.out -trace trace.mcp -serve
```

### Analyze Multiple Test Runs

```bash
# Run tests multiple times
for i in {1..5}; do
    MCP_TRACE=traces/run-$i.mcp go test ./...
done

# Visualize all runs
mcp-coverage-viz -coverage coverage.out -trace-dir traces -serve
```

### Export Static Report

```bash
# Generate JSON data
mcp-coverage-viz -coverage coverage.out -trace trace.mcp -output report.json

# Generate HTML report (via API)
curl http://localhost:8080/api/coverage | \
    mcp-coverage-viz -template html > report.html
```

## Architecture

The tool consists of several components:

1. **Parser**: Parses MCP trace files and extracts test information
2. **Coverage Integrator**: Combines Go coverage data with test results
3. **Web Server**: Serves interactive visualization
4. **Static Assets**: CSS, JavaScript, and templates

## Contributing

To add new features:

1. Update the data model in `types.go`
2. Extend parser/integrator functionality
3. Add web UI components
4. Update documentation

## Future Enhancements

- Real-time coverage tracking during test runs
- Integration with CI/CD pipelines
- Comparison between test runs
- Support for other coverage formats
- Advanced filtering and search