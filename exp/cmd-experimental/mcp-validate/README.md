# mcp-validate

Comprehensive protocol compliance and schema validation tool for MCP implementations.

## Overview

`mcp-validate` is a powerful validation tool that ensures MCP servers and clients adhere to protocol specifications, JSON schemas, and best practices. It provides detailed error reporting, compliance scoring, and actionable suggestions for fixing issues.

## Features

- **Schema Validation**: Validate request/response against JSON schemas
- **Protocol Compliance**: Check adherence to MCP specification
- **Capability Verification**: Validate server capability declarations vs actual behavior
- **Error Analysis**: Detailed error reporting with fix suggestions
- **Batch Processing**: Validate entire trace files or live sessions
- **Multiple Output Formats**: JSON, JUnit XML, HTML reports
- **Strict Mode**: Additional performance and compliance checks
- **Live Monitoring**: Continuous validation of running servers

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-validate@latest
```

## Usage

### Basic Server Validation

```bash
# Validate a server's protocol compliance
mcp-validate validate --server "python my_server.py"

# Enable strict validation mode
mcp-validate validate --server "python my_server.py" --strict

# Generate HTML report
mcp-validate validate --server "python my_server.py" --report compliance.html --output-format html
```

### Trace File Validation

```bash
# Validate existing trace file
mcp-validate validate --trace session.mcp

# Validate with custom schema directory
mcp-validate validate --trace session.mcp --schema-dir ./schemas/

# Output JSON report
mcp-validate validate --trace session.mcp --report validation.json
```

### Batch Validation

```bash
# Create a servers.json file:
# ["python server1.py", "node server2.js", "./server3"]

# Batch validate multiple implementations
mcp-validate validate --batch servers.json --output-format junit-xml --report results.xml
```

### Live Validation

```bash
# Monitor and validate a running server
mcp-validate validate --live --target localhost:8080 --verbose

# Generate continuous compliance reports
mcp-validate validate --live --target mcp://production.example.com --report live-compliance.html
```

## Output Formats

### JSON Format

```json
{
  "version": "0.1.0",
  "timestamp": "2025-01-17T10:30:00Z",
  "summary": {
    "totalChecks": 25,
    "passedChecks": 23,
    "failedChecks": 1,
    "warningCount": 1,
    "complianceRate": 92.0,
    "status": "passed_with_warnings"
  },
  "violations": [
    {
      "severity": "warning",
      "category": "protocol",
      "rule": "version_mismatch",
      "message": "Server uses protocol version 2024-01-01, expected 2025-03-26",
      "location": "",
      "suggestion": "Update server to use the latest protocol version"
    }
  ],
  "capabilities": {
    "declared": {
      "tools": {},
      "resources": {}
    },
    "actual": {
      "tools": true,
      "resources": true
    },
    "mismatches": []
  }
}
```

### JUnit XML Format

Compatible with CI/CD systems for test reporting:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="mcp-validate" tests="25" failures="1" errors="0" time="0">
  <testcase name="protocol.initialization" classname="protocol"/>
  <testcase name="protocol.version_mismatch" classname="protocol">
    <failure message="Server uses old protocol version">Update to latest version</failure>
  </testcase>
</testsuite>
```

### HTML Report

Interactive HTML reports with:
- Visual compliance summary
- Sortable violation table
- Severity color coding
- Detailed suggestions
- Capability comparison

## Validation Categories

### Protocol Compliance
- JSON-RPC 2.0 message format
- Protocol version compatibility
- Method name validation
- Request/response structure
- Error format compliance

### Capability Verification
- Tools declaration vs implementation
- Resources support validation
- Prompts functionality check
- Change notification support
- Subscription mechanism validation

### Error Handling
- Proper error responses for invalid requests
- Standard error code usage
- Meaningful error messages
- Graceful failure handling

### Performance (Strict Mode)
- Response time requirements
- Throughput expectations
- Resource usage limits
- Connection stability

## Exit Codes

- `0`: All validations passed
- `1`: Validation failures detected
- `2`: Configuration or usage error
- `3`: Internal error

## Configuration

### Environment Variables

- `MCP_VALIDATE_SCHEMA_DIR`: Default schema directory location
- `MCP_VALIDATE_STRICT`: Enable strict mode by default
- `MCP_VALIDATE_TIMEOUT`: Default timeout for operations

### Schema Directory Structure

```
schemas/
├── stable/
│   ├── initialize.json
│   ├── tools.json
│   └── resources.json
└── draft/
    └── experimental.json
```

## Examples

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Validate MCP Server
  run: |
    mcp-validate validate \
      --server "python server.py" \
      --output-format junit-xml \
      --report test-results.xml \
      --strict
```

### Docker Validation

```bash
# Validate containerized server
docker run -d --name mcp-server my-mcp-server:latest
mcp-validate validate --target localhost:8080 --report docker-compliance.json
```

### Development Workflow

```bash
# Quick validation during development
mcp-validate validate --server "go run ./server" --verbose

# Pre-commit validation
mcp-validate validate --trace latest.mcp --strict
```

## Best Practices

1. **Regular Validation**: Run validation as part of CI/CD pipeline
2. **Strict Mode**: Use strict mode for production readiness
3. **Schema Updates**: Keep schemas updated with protocol versions
4. **Batch Testing**: Test multiple server configurations
5. **Report Archival**: Store validation reports for compliance tracking

## Troubleshooting

### Common Issues

**Connection Failed**
- Ensure server is running and accessible
- Check firewall/network settings
- Verify server startup command

**Schema Validation Errors**
- Update to latest schema files
- Check JSON syntax in responses
- Validate against correct protocol version

**Capability Mismatches**
- Ensure all declared capabilities are implemented
- Update capability declarations to match implementation
- Test each capability endpoint

## See Also

- [mcp-schema](../mcp-schema/README.md) - Schema generation and analysis
- [mcp-contract](../mcp-contract/README.md) - API contract testing
- [MCP Specification](https://spec.modelcontextprotocol.io)