# MCP Experimental Tools

This document explains the organization of experimental tools in the MCP project ecosystem.

## Overview

To maintain a clean separation between the core MCP protocol implementation and experimental/in-development tools, we've moved several components to a dedicated repository: `github.com/tmc/mcp-tools-experimental`.

## Rationale

The core MCP protocol implementation needs to remain focused and well-tested, while still allowing for experimental features and tools. By separating these components:

1. The core protocol remains stable and well-tested
2. Experimental tools can evolve more freely
3. Dependency management is simplified
4. Test failures in experimental components don't block core development

## Components Moved to Experimental Repository

The following components have been moved to the experimental repository:

1. **mcp-scripttest-server**: Test server implementation for scripttest-based testing
2. **mcp-send**: Tool for sending MCP messages
3. **mcp-start**: Utility to start MCP servers from specification files
4. **mcp-test**: Test MCP servers against specification
5. **mcp-verify**: Verify MCP server implementation compliance
6. **mcptrace2gostruct**: Convert MCP traces to Go struct definitions

Additionally, the mcpscripttest package was moved to the pkg/ directory in the experimental repository.

## Using Experimental Tools

To use experimental tools:

1. Clone the experimental repository:
   ```
   git clone https://github.com/tmc/mcp-tools-experimental.git
   ```

2. Build and use the tools:
   ```
   cd mcp-tools-experimental
   go build ./cmd/...
   ```

## Contributing to Experimental Tools

1. Make changes in the experimental repository
2. Ensure tests pass within the experimental repository context
3. Once a tool is mature and well-tested, it may be considered for promotion to the main repository

## Relationship with Core Repository

The experimental repository depends on the core MCP repository, not vice versa. This ensures that:

1. The core repository has no dependencies on experimental code
2. Changes to experimental tools don't affect core protocol stability
3. The experimental repository can track the latest developments in the core repository

## Future Plans

As experimental tools mature:

1. They may be promoted to the core repository if they become essential components
2. They may remain in the experimental repository if they serve specialized use cases
3. They may be moved to independent repositories if they become substantial projects on their own