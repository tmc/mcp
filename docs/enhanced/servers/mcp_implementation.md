# MCP Server Implementation Guide

## Core MCP Components

### Entry System
```go
// Entry represents a recorded MCP message
type Entry struct {
    Dir  string    `json:"dir"`            // "in" or "out"
    Data []byte    `json:"data"`           // Raw message data
    Time time.Time `json:"time,omitempty"` // Message timestamp
}
```

### Message Handling
```go
// WriteTo writes the entry to the given writer
func (e *Entry) WriteTo(w io.Writer) (int64, error) {
    if e.Time.IsZero() {
        e.Time = time.Now()
    }
    data, err := json.Marshal(e)
    if err != nil {
        return 0, err
    }
    n, err := w.Write(append(data, '
'))
    return int64(n), err
}
```

## Server Types

### Filesystem Server
Provides file system access for Claude desktop app.

Configuration:
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "/path/to/workspace",
        "/path/to/logs"
      ]
    }
  }
}
```

### MCP-Exec Server
Provides command execution capabilities.

Configuration:
```json
{
  "mcp": {
    "servers": {
      "mcp-exec": {
        "command": "/path/to/debug-wrapper.sh",
        "args": []
      }
    }
  }
}
```

## Implementation Guidelines

### Server Structure
1. Message handling
2. Configuration management
3. Error handling
4. Logging

### Message Flow
1. Receive request (Dir: "in")
2. Process request
3. Send response (Dir: "out")
4. Log transaction

### Error Handling
1. Input validation
2. Permission checks
3. Resource validation
4. Error reporting

### Configuration
1. Server-specific config
2. Environment variables
3. Command-line args
4. Default values

## Testing

### Unit Tests
```go
func TestEntry(t *testing.T) {
    entry := &Entry{
        Dir:  "in",
        Data: []byte("test"),
        Time: time.Now(),
    }
    
    var buf bytes.Buffer
    n, err := entry.WriteTo(&buf)
    assert.NoError(t, err)
    assert.True(t, n > 0)
}
```

### Integration Tests
```go
func TestServer(t *testing.T) {
    // Start server
    server := NewServer(config)
    defer server.Close()
    
    // Send test message
    entry := createTestEntry()
    response, err := server.Process(entry)
    assert.NoError(t, err)
    assert.NotNil(t, response)
}
```

## Deployment

### Requirements
1. Go runtime
2. Node.js (for filesystem server)
3. Required permissions
4. Configuration files

### Configuration Files
1. Claude desktop config
2. Server-specific config
3. Environment config
4. Logging config

### Directory Structure
```
/cmd
  /mcp-exec        # MCP execution server
/examples
  /filesystem      # Filesystem server
  /mcp-exec        # Exec server
/internal
  /mcp             # Core MCP package
```

## Security

### File Access
1. Path validation
2. Permission checks
3. Resource limits
4. Audit logging

### Command Execution
1. Command validation
2. Argument sanitization
3. Resource limits
4. Output handling

## Monitoring

### Logging
1. Request/response logging
2. Error logging
3. Performance metrics
4. Audit trails

### Metrics
1. Request count
2. Response times
3. Error rates
4. Resource usage

## Best Practices

### Code Organization
1. Clean package structure
2. Clear interfaces
3. Error handling
4. Documentation

### Testing
1. Comprehensive tests
2. Edge cases
3. Error scenarios
4. Performance tests

### Security
1. Input validation
2. Access control
3. Resource limits
4. Secure defaults

### Performance
1. Efficient processing
2. Resource management
3. Caching strategy
4. Error handling

## Future Improvements

### Planned Features
1. Enhanced security
2. Better monitoring
3. More server types
4. Advanced logging

### Integration
1. Additional protocols
2. External services
3. UI enhancements
4. Analytics