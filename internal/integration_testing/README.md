# Integration Testing

This directory contains integration tests that verify interoperability between different MCP implementations.

## Modules

### marklabs-interop
Tests interoperability between this MCP implementation and the Mark3labs MCP Go SDK.

### golang-sdk-interop  
Tests interoperability between this MCP implementation and the official Go MCP SDK.

### protocol-interop
Tests protocol-level compatibility and conformance across different MCP implementations.

### typescript-sdk-interop
Tests interoperability between this MCP implementation and the official
TypeScript MCP SDK over stdio.

## Running Tests

Each module contains its own test suite. To run all integration tests:

```bash
cd internal/integration_testing && make test
```

To run tests for a specific module:

```bash
(cd marklabs-interop && GOWORK=off go test ./...)
(cd golang-sdk-interop && GOWORK=off go test ./...)
(cd protocol-interop && GOWORK=off go test ./...)
(cd typescript-sdk-interop && GOWORK=off go test ./...)
```

## Test Coverage

The integration tests cover:
- Protocol message compatibility
- Transport layer interoperability
- Tool execution compatibility
- Resource access compatibility
- Client-server handshaking
- Error handling consistency
