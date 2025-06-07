# MCP Conformance Testing with mcpscripttest

This document describes the MCP protocol conformance testing system, built using the `mcpscripttest` framework.

## Overview

The Model Context Protocol (MCP) conformance test suite is a comprehensive set of tests designed to validate that implementations correctly adhere to the protocol specification. These tests cover all key aspects of the protocol, including messaging format, lifecycle management, transport mechanisms, capabilities, tools, error handling, version negotiation, and security.

## Components

The conformance testing system consists of the following components:

1. **Test Scripts**: A set of script files located in `/testdata/mcp_conformance/` that define the test cases
2. **mcpscripttest Package**: A Go package in `exp/mcpscripttest/` that provides the framework for running the tests
3. **mcpscripttest Binary**: A standalone command-line tool in `cmd/mcpscripttest/` for running the tests without the Go testing framework

## Test Categories

The tests are organized into the following categories:

1. **Base Messaging** (`01_base_messaging.txt`) - Tests for correct JSON-RPC 2.0 message format and handling
2. **Lifecycle** (`02_lifecycle.txt`) - Tests for initialization, operation, and shutdown phases
3. **Transports** (`03_transports.txt`) - Tests for stdio and HTTP transport mechanisms
4. **Capability Negotiation** (`04_capability_negotiation.txt`) - Tests for client and server capability negotiation
5. **Tools** (`05_tools.txt`) - Tests for tool listing, invocation, and handling
6. **Error Handling** (`06_error_handling.txt`) - Tests for protocol and application error handling
7. **Version Negotiation** (`07_version_negotiation.txt`) - Tests for protocol version compatibility
8. **Security** (`08_security.txt`) - Tests for authentication, authorization, and validation

## Running Tests

### Using Go Test

You can run the tests using the Go testing framework:

```bash
# Run all conformance tests
go test -v ./exp/mcpscripttest -run TestMCPConformance

# Run a specific test category
go test -v ./exp/mcpscripttest -run TestMCPConformanceIndividual/01_base_messaging.txt

# Run with coverage
go test -coverprofile=coverage.out -v ./exp/mcpscripttest -run TestMCPConformance
```

### Using the Standalone Binary

For easier testing without the Go testing framework, use the `mcpscripttest` binary:

```bash
# Build the binary
cd cmd/mcpscripttest
go build

# Run all tests
./mcpscripttest -all

# Run a specific test
./mcpscripttest -test 01_base_messaging

# Run with coverage
./mcpscripttest -all -coverage
```

## Extending the Test Suite

To add new tests to the conformance suite:

1. Create a new script file in the `testdata/mcp_conformance/` directory
2. Follow the existing test structure and naming conventions
3. Use the scripting commands provided by the `mcpscripttest` framework
4. Run the tests to verify they work correctly

## Test Script Format

Test scripts use the `scripttest` DSL format, which provides commands for executing programs, comparing outputs, and verifying behaviors. Each test script typically follows this pattern:

1. Start a test server: `exec mcp-scripttest-server --stdio`
2. Initialize the connection: `setstdin {"jsonrpc":"2.0","id":1,"method":"initialize",...}`
3. Run specific tests for protocol features
4. Verify expected responses using assertions like `stdout` and `stderr`

## Coverage Analysis

The tests can be run with coverage instrumentation to analyze which parts of the protocol implementation are being tested:

```bash
# Using Go test
go test -coverprofile=coverage.out -v ./exp/mcpscripttest -run TestMCPConformance
go tool cover -html=coverage.out

# Using the standalone binary
./mcpscripttest -all -coverage
go tool covdata textfmt -i /tmp/mcp-coverage-* -o coverage.txt
go tool cover -html=coverage.txt -o coverage.html
```

## Conformance Certification

Implementations can use these tests to verify their conformance with the MCP specification. A successful run of all tests indicates that the implementation correctly handles all aspects of the protocol.

## Further Reading

- [MCP Specification](https://github.com/modelcontextprotocol/specification)
- [scripttest Documentation](https://pkg.go.dev/rsc.io/script/scripttest)
- [Test-Driven Development for Protocol Implementations](docs/development/test-driven-protocol-dev.md)