# MCP9P Development Guide

This guide covers development of the Plan9-inspired namespace system for MCP.

## Getting Started

1. **Build all tools**:
```bash
make build
```

2. **Run the demo**:
```bash
./demo.sh
```

3. **Start development server**:
```bash
make server
```

## Architecture

### Namespace Server (mcp-namespace)

The core server that maintains the namespace state:
- HTTP API for registration, lookup, and listing
- In-memory storage with optional persistence
- Hierarchical namespace structure
- Support for multiple entry types

### Client Tools

- **mcp-ns**: CLI for namespace operations
- **mcp-mount**: Creates mounts and bindings
- **mcp-fs**: FUSE filesystem interface
- **mcp-tunnel-ns**: Namespace-aware tunneling

## Adding New Features

### 1. New Entry Types

To add a new entry type (e.g., "proxy"):

```go
// In mcp-namespace/main.go
type Entry struct {
    Name      string            `json:"name"`
    Type      string            `json:"type"` // Add "proxy" here
    // ... other fields
}

// Add handling in register endpoint
switch req.Entry.Type {
case "proxy":
    // Handle proxy-specific logic
}
```

### 2. New Operations

To add new operations (e.g., "copy"):

```go
// In mcp-ns/main.go
case "copy":
    source := flag.Arg(0)
    target := flag.Arg(1)
    // Implement copy logic
```

### 3. New Protocol Features

For protocol enhancements:
1. Update message types in relevant files
2. Add handlers in namespace server
3. Update client libraries
4. Add tests

## Testing

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
# Start server
make server &

# Run integration tests
go test -tags integration ./...
```

### Manual Testing
Use the demo script for manual testing:
```bash
./demo.sh
```

## Debugging

### Enable Debug Logging
```bash
# Start server with debug logging
MCP9P_DEBUG=1 ./bin/mcp-namespace

# Use verbose flag in clients
./bin/mcp-ns -v -c list /
```

### Common Issues

1. **Port Already in Use**:
```bash
lsof -i :9000
kill <PID>
```

2. **FUSE Permission Issues**:
```bash
# Run as root or configure FUSE permissions
sudo ./bin/mcp-fs -mount /mnt/mcp
```

3. **Connection Refused**:
- Ensure namespace server is running
- Check firewall settings
- Verify correct port

## Code Style

- Follow standard Go conventions
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

## Future Development

### Phase 1: Core Features
- [x] Basic namespace operations
- [x] Mount and bind support
- [x] FUSE filesystem
- [ ] Persistence improvements
- [ ] Better error handling

### Phase 2: Advanced Features
- [ ] Authentication and authorization
- [ ] Namespace federation
- [ ] 9P protocol implementation
- [ ] Performance optimizations
- [ ] Distributed namespace

### Phase 3: Production Ready
- [ ] High availability
- [ ] Monitoring and metrics
- [ ] Admin tools
- [ ] Documentation
- [ ] Security audit

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## References

- [Plan 9 Namespaces](http://doc.cat-v.org/plan_9/4th_edition/papers/names)
- [9P Protocol](http://9p.cat-v.org/documentation/)
- [Go FUSE Library](https://github.com/hanwen/go-fuse)