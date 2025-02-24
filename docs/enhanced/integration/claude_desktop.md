# Claude Desktop Integration Guide

## Overview
Detailed guide for integrating MCP servers with the Claude desktop application.

## Configuration

### Claude Desktop Config
```json
{
  "darkMode": "dark",
  "scale": 0,
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

### Server Configuration
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
    },
    "mcp-exec": {
      "command": "/path/to/mcp-exec",
      "args": []
    }
  }
}
```

## Message Protocol

### Entry Structure
```go
type Entry struct {
    Dir  string    // Direction: "in" or "out"
    Data []byte    // Raw message data
    Time time.Time // Message timestamp
}
```

### Message Flow
1. Client Request (Dir: "in")
   - Command/query
   - Arguments
   - Metadata

2. Server Response (Dir: "out")
   - Results
   - Status
   - Error info

## Server Integration

### Filesystem Server
1. File access
2. Directory operations
3. Path validation
4. Permission management

### MCP-Exec Server
1. Command execution
2. Process management
3. Output handling
4. Error handling

## Implementation

### Message Handling
```go
// Write message to output
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

### Server Implementation
```go
// Server interface
type Server interface {
    Handle(entry *Entry) (*Entry, error)
    Close() error
}

// Implementation example
func (s *Server) Handle(entry *Entry) (*Entry, error) {
    // Process incoming message
    result, err := s.process(entry)
    if err != nil {
        return nil, err
    }
    
    // Create response
    return &Entry{
        Dir:  "out",
        Data: result,
        Time: time.Now(),
    }, nil
}
```

## Security

### File Access Security
1. Path validation
2. Permission checks
3. Resource limits
4. Audit logging

### Command Execution Security
1. Command validation
2. Argument sanitization
3. Resource limits
4. Output handling

## Error Handling

### Common Errors
1. Invalid requests
2. Permission denied
3. Resource not found
4. Execution errors

### Error Responses
```go
// Error response
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

## Testing

### Unit Tests
```go
// Test message handling
func TestMessageHandling(t *testing.T) {
    entry := &Entry{
        Dir:  "in",
        Data: []byte("test"),
    }
    
    response, err := server.Handle(entry)
    assert.NoError(t, err)
    assert.Equal(t, "out", response.Dir)
}
```

### Integration Tests
```go
// Test server integration
func TestServerIntegration(t *testing.T) {
    // Start server
    server := NewServer(config)
    defer server.Close()
    
    // Test operations
    testFileOperations(t, server)
    testCommandExecution(t, server)
}
```

## Deployment

### Prerequisites
1. Go runtime
2. Node.js (for filesystem server)
3. Required permissions
4. Configuration files

### Configuration
1. Server paths
2. Command paths
3. Workspace paths
4. Log paths

## Monitoring

### Logging
1. Request/response logs
2. Error logs
3. Performance metrics
4. Audit logs

### Metrics
1. Request count
2. Response times
3. Error rates
4. Resource usage

## Best Practices

### Implementation
1. Clean code structure
2. Error handling
3. Security checks
4. Performance optimization

### Testing
1. Comprehensive tests
2. Edge cases
3. Error scenarios
4. Load testing

### Security
1. Input validation
2. Access control
3. Resource limits
4. Secure defaults

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