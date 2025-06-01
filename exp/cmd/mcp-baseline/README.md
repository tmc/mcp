# mcp-baseline

A tool for creating and verifying baseline behavior of MCP servers.

## Overview

`mcp-baseline` helps you create a reliable baseline of expected behavior for an MCP server and verify that changes don't break that baseline. It's designed to be simple, pragmatic, and immediately useful for development and CI workflows.

## Key Features

- **Record baselines** from working MCP servers
- **Verify** servers against recorded baselines
- **Export** baselines as reusable test suites
- **Integrate** with CI/CD pipelines

## Usage

### Recording a Baseline

```bash
# Record a baseline from a working server
mcp-baseline record --server localhost:8080 --output filesystem-baseline.json

# Record specific methods only
mcp-baseline record --server localhost:8080 --methods initialize,read,write --output baseline-core.json

# Record with custom test data
mcp-baseline record --server localhost:8080 --test-data test_values.json --output custom-baseline.json
```

### Verifying Against a Baseline

```bash
# Verify a server against a recorded baseline
mcp-baseline verify --server localhost:8080 --baseline filesystem-baseline.json

# Run verification in CI mode (exit code indicates success/failure)
mcp-baseline verify --server localhost:8080 --baseline filesystem-baseline.json --ci

# Generate a detailed report
mcp-baseline verify --server localhost:8080 --baseline filesystem-baseline.json --report html --output report.html
```

### Managing Baselines

```bash
# List methods in a baseline
mcp-baseline list --baseline filesystem-baseline.json

# Extract a subset of methods
mcp-baseline extract --baseline full-baseline.json --methods initialize,read --output core-baseline.json

# Merge multiple baselines
mcp-baseline merge --baselines baseline1.json,baseline2.json --output combined.json

# Update specific methods in a baseline
mcp-baseline update --baseline filesystem-baseline.json --methods write --server localhost:8080
```

## How It Works

1. **Recording**: The tool sends a series of requests to the target server, storing the requests and responses as a baseline
2. **Verification**: When verifying, it runs the same requests against a server and compares responses to the baseline
3. **Reporting**: Differences are highlighted and reported in a structured format suitable for human or machine consumption

## Baseline Format

Baselines are stored in a structured JSON format:

```json
{
  "metadata": {
    "server_name": "mcp-filesystem-server",
    "server_version": "1.0.0",
    "recorded_at": "2024-11-26T12:34:56Z",
    "protocol_version": "2024-11-05"
  },
  "test_cases": [
    {
      "name": "initialize",
      "request": {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
          "protocolVersion": "2024-11-05",
          "capabilities": {},
          "clientInfo": {"name": "mcp-baseline", "version": "1.0.0"}
        }
      },
      "expected_response": {
        "jsonrpc": "2.0",
        "id": 1,
        "result": {
          "protocolVersion": "2024-11-05",
          "serverInfo": {"name": "mcp-filesystem-server", "version": "1.0.0"},
          "capabilities": {}
        }
      },
      "validation_rules": [
        {"path": "result.protocolVersion", "match": "exact"},
        {"path": "result.serverInfo.name", "match": "exact"},
        {"path": "result.capabilities", "match": "exists"}
      ]
    },
    {
      "name": "read_file",
      "request": {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "read",
        "params": {"path": "test.txt"}
      },
      "expected_response": {
        "jsonrpc": "2.0",
        "id": 2,
        "result": {
          "path": "test.txt",
          "content": "Test content",
          "encoding": "utf-8",
          "size": 12
        }
      },
      "validation_rules": [
        {"path": "result.path", "match": "exact"},
        {"path": "result.content", "match": "exact"},
        {"path": "result.encoding", "match": "exact"},
        {"path": "result.size", "match": "numeric_equal"}
      ],
      "setup": [
        {
          "method": "write",
          "params": {"path": "test.txt", "content": "Test content"}
        }
      ],
      "cleanup": [
        {
          "method": "delete",
          "params": {"path": "test.txt"}
        }
      ]
    }
  ]
}
```

## Use Cases

### Development Workflow

1. Build a working version of your MCP server
2. Record a baseline using `mcp-baseline record`
3. Make changes to your implementation
4. Verify your server still works using `mcp-baseline verify`

### CI/CD Integration

```yaml
# Example GitHub Action
name: MCP Server Verification
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build server
        run: go build -o server ./cmd/mcp-server
      
      - name: Start server
        run: ./server &
        
      - name: Verify against baseline
        run: mcp-baseline verify --server localhost:8080 --baseline ./test/baseline.json --ci
```

### Cross-Implementation Testing

1. Record a baseline from a reference implementation
2. Verify a new implementation against this baseline
3. Generate a compatibility report

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-baseline@latest
```

## Building from Source

```bash
git clone https://github.com/tmc/mcp.git
cd mcp/cmd/mcp-baseline
go build
```

## Coming Soon

- **Automatic test case generation** based on server capabilities
- **Performance benchmarks** alongside functional tests
- **Scenario-based testing** for complex workflows
- **Fault injection** for resilience testing