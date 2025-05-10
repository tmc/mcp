# Google MCP Integration Plan

This document outlines the plan for integrating Google's internal MCP implementation (`golang/tools/internal/mcp`) into our codebase while maintaining compatibility with our existing implementation.

## Goals

1. Leverage Google's high-quality MCP implementation design patterns
2. Maintain Git history for future reference and updates
3. Preserve compatibility with our existing codebase
4. Avoid dependency conflicts by standardizing on `golang.org/x/exp/jsonrpc2`

## Integration Strategy

### 1. Repository Setup

1. Create a `third_party/googlemcp` directory in our project
2. Use Git subtree to import Google's MCP implementation with its history:
   ```bash
   git subtree add --prefix=third_party/googlemcp --squash <golang-tools-repo-url> internal/mcp
   ```
3. This preserves the commit history while isolating the code within our repository

### 2. Dependency Adaptation

The primary challenge is adapting Google's implementation (which uses `internal/jsonrpc2_v2`) to work with `golang.org/x/exp/jsonrpc2`:

1. Create a shim layer in `third_party/googlemcp/shim/` that adapts interfaces between the two implementations
2. Replace direct import references to `internal/jsonrpc2_v2` with our shim layer
3. Create interface adapters where APIs differ significantly

### 3. Type Aliasing Strategy

To make our existing code interface with the Google implementation:

1. Define type aliases in our codebase to reference Google's implementation types
2. Create adapter functions to convert between our representations and Google's
3. Gradually migrate core functionality to use Google's implementations while maintaining our API

### 4. Component-by-Component Integration

#### JSON Schema Validation

1. Port Google's `jsonschema` package with minimal changes
2. Create adapter functions to integrate it with our existing code
3. Enhance our typed tool APIs to leverage the schema validation

#### Content Types

1. Adopt Google's content type system as a reference implementation
2. Create mappings between our content types and Google's
3. Expand our content type support based on their implementation

#### Transport Layer

1. Implement our `Transport` interface on top of Google's transport abstractions
2. Create adapter code to bridge between our and Google's transport systems
3. Add support for additional transport options (like SSE) based on Google's implementation

#### Client/Server Implementation

1. Create facade classes that expose our existing API but delegate to Google's implementation internally
2. Gradually refactor our code to more directly use Google's patterns
3. Maintain backward compatibility through the transition

### 5. Testing and Validation

1. Create comprehensive tests for the integration layer
2. Ensure existing tests continue to pass during the transition
3. Add new tests targeting Google-specific functionality

## Implementation Phases

### Phase 1: Initial Import and Adaptation

- Import Google's MCP code into `third_party/googlemcp`
- Create basic shim layers for jsonrpc2 compatibility
- Implement minimal adapters to make the code compile

### Phase 2: Component Integration

- Integrate the JSON schema validation first (it's most independent)
- Implement transport adapter layers
- Create content type mapping system

### Phase 3: API Refinement

- Refine our public API to better align with Google's design patterns
- Create comprehensive documentation for the integrated system
- Optimize performance of the adapter layers

### Phase 4: Migration

- Gradually replace our implementation with direct calls to the adapted Google implementation
- Ensure backward compatibility throughout the process
- Complete the migration component by component

## Long-term Maintenance

- Monitor changes to Google's implementation for potential updates
- Periodically pull in updates using Git subtree
- Contribute improvements back to the original codebase where possible