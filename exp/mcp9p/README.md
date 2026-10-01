# MCP9P - Plan9-Inspired Namespace System for MCP

This directory contains experimental Plan9-inspired namespace tools for the Model Context Protocol (MCP). These tools provide a hierarchical namespace for service discovery, mounting, and management inspired by Plan9's elegant namespace concepts.

## Why Plan9?

Plan9's namespace system provides elegant abstractions for distributed systems:
- Everything is a file (or namespace entry)
- Union mounts for composing services
- Per-process namespaces for isolation
- Simple, powerful mounting and binding operations

## Components

### Core Services

- **mcp-namespace**: The namespace server that maintains the service registry
- **mcp-ns**: Command-line client for namespace operations
- **mcp-mount**: Tool for creating mounts and bindings (like Plan9 bind)
- **mcp-fs**: FUSE filesystem that exposes the namespace as a filesystem
- **mcp-tunnel-ns**: Namespace-aware version of mcp-tunnel

### Architecture

```
                     ┌─────────────────┐
                     │  mcp-namespace  │
                     │    (server)     │
                     └────────┬────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼─────┐         ┌────▼─────┐         ┌────▼─────┐
   │  mcp-ns  │         │mcp-mount │         │  mcp-fs  │
   │ (client) │         │  (bind)  │         │  (FUSE)  │
   └──────────┘         └──────────┘         └──────────┘
```

## Namespace Structure

The namespace follows a hierarchical structure similar to Plan9:

```
/
├── services/           # User services
│   ├── echo           # Echo service
│   ├── calculator     # Calculator service
│   └── ai/            # AI services
│       ├── gpt        # GPT service
│       └── claude     # Claude service
├── local/             # Local machine services
│   ├── fs             # Filesystem access
│   └── shell          # Shell access
└── remote/            # Remote services
    ├── server1/       # Services on server1
    └── cloud/         # Cloud services
```

## Quick Start

1. Start the namespace server:
```bash
cd exp/mcp9p/mcp-namespace
go run main.go
```

2. Register a service:
```bash
cd exp/mcp9p/mcp-ns
go run main.go -c register /services/echo \
  -type local \
  -transport stdio \
  -command npx \
  -args "@modelcontextprotocol/server-echo,stdio"
```

3. List services:
```bash
go run main.go -c list /services
```

4. Mount the namespace as a filesystem:
```bash
cd exp/mcp9p/mcp-fs
sudo go run main.go -mount /mnt/mcp -ns http://localhost:9000
```

## Plan9 Concepts Applied

### Union Mounts
Combine multiple services into a single namespace view:
```bash
mcp-mount -type union /ai /services/ai/gpt /services/ai/claude
```

### Bind Mounts
Create aliases for services:
```bash
mcp-mount -type bind /services/ai/gpt /gpt
```

### Per-Process Namespaces
Each client can have its own view of the namespace (future feature).

### Everything is a File
Access service information through the filesystem:
```bash
cat /mnt/mcp/services/echo
```

## Namespace URIs

The system supports Plan9-inspired namespace URIs:
```
ns://server/path
ns://localhost:9000/services/echo
ns://namespace.example.com/remote/api/calculator
```

## Current Status

This is an experimental implementation exploring how Plan9 concepts can enhance MCP service discovery and management. Key areas of development:

1. **Protocol Design**: Defining the namespace protocol for MCP
2. **Security Model**: Authentication and authorization for namespace operations
3. **Distribution**: Federation of namespace servers
4. **Performance**: Caching and optimization for production use
5. **9P Protocol**: Potential implementation of actual 9P protocol

## Examples

See the `examples/` directory for usage examples and patterns.

## Future Directions

- Full 9P protocol implementation
- Distributed namespace with federation
- Private namespaces per user/process
- Integration with container orchestration
- Native OS integration (mount as real filesystem)

## Contributing

This is experimental software. We welcome ideas and contributions to explore how Plan9's elegant namespace concepts can improve MCP service management.

## References

- [Plan 9 from Bell Labs](http://doc.cat-v.org/plan_9/)
- [The Use of Name Spaces in Plan 9](http://doc.cat-v.org/plan_9/4th_edition/papers/names)