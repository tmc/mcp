# MCP Server Examples

This guide showcases various MCP server implementations, from simple to complex.

## Available Server Examples

### 1. Echo Server

**Location**: `examples/servers/mcp-echo-server/`

The simplest possible MCP server that echoes back messages.

```go
func main() {
    server := mcp.NewServer()
    
    server.AddTool("echo", mcp.ToolFunc(func(args map[string]any) (any, error) {
        message, ok := args["message"].(string)
        if !ok {
            return nil, fmt.Errorf("message must be a string")
        }
        return message, nil
    }))
    
    server.Serve(mcp.NewStdioTransport())
}
```

**Features**:
- Single tool: `echo`
- Basic error handling
- stdio transport

**Usage**:
```bash
# Run server
go run ./examples/servers/mcp-echo-server

# Test with client
echo '{"jsonrpc":"2.0","id":1,"method":"tools/execute","params":{"name":"echo","arguments":{"message":"Hello!"}}}' | ./mcp-echo-server
```

### 2. Time Server

**Location**: `examples/servers/mcp-time-server/`

Provides current time and timezone information.

```go
type TimeServer struct {
    location *time.Location
}

func (s *TimeServer) GetCurrentTime(args map[string]any) (any, error) {
    format := "2006-01-02 15:04:05"
    if f, ok := args["format"].(string); ok {
        format = f
    }
    
    return map[string]string{
        "time":     time.Now().In(s.location).Format(format),
        "timezone": s.location.String(),
    }, nil
}
```

**Features**:
- Current time tool
- Timezone conversion
- Format customization
- List timezones resource

**Tools**:
- `getCurrentTime` - Get current time
- `convertTimezone` - Convert between timezones
- `listTimezones` - List available timezones

### 3. Calculator Server

**Location**: `examples/servers/mcp-calculator-server/`

Mathematical operations server with advanced functions.

```go
type Calculator struct {
    precision int
}

func (c *Calculator) Calculate(args map[string]any) (any, error) {
    op := args["operation"].(string)
    
    switch op {
    case "add", "subtract", "multiply", "divide":
        return c.basicOperation(op, args)
    case "sqrt", "pow", "log":
        return c.advancedOperation(op, args)
    default:
        return nil, fmt.Errorf("unknown operation: %s", op)
    }
}
```

**Features**:
- Basic arithmetic
- Advanced math functions
- Error handling for edge cases
- Configurable precision

**Operations**:
- Basic: `add`, `subtract`, `multiply`, `divide`
- Advanced: `sqrt`, `pow`, `log`, `sin`, `cos`, `tan`
- Constants: `pi`, `e`

### 4. File System Server

**Location**: `examples/servers/mcp-filesystem-server/`

Safe file system access with permissions control.

```go
type FileSystemServer struct {
    root        string
    permissions FilePermissions
}

func (fs *FileSystemServer) ReadFile(args map[string]any) (any, error) {
    path := args["path"].(string)
    
    // Security check
    if !fs.isPathAllowed(path) {
        return nil, fmt.Errorf("access denied: %s", path)
    }
    
    content, err := os.ReadFile(filepath.Join(fs.root, path))
    return string(content), err
}
```

**Features**:
- Read/write files
- Directory operations
- Path security
- File metadata
- Search functionality

**Tools**:
- `readFile` - Read file content
- `writeFile` - Write to file
- `listDirectory` - List directory contents  
- `createDirectory` - Create new directory
- `deleteFile` - Remove file
- `getFileInfo` - Get file metadata

### 5. SQLite Server

**Location**: `examples/servers/mcp-sqlite-server/`

Database operations with SQLite.

```go
type SQLiteServer struct {
    db *sql.DB
}

func (s *SQLiteServer) ExecuteQuery(args map[string]any) (any, error) {
    query := args["query"].(string)
    params := args["params"].([]any)
    
    // Security: Use prepared statements
    rows, err := s.db.Query(query, params...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return scanRows(rows)
}
```

**Features**:
- Execute queries
- Prepared statements
- Transaction support
- Schema inspection
- Backup/restore

**Tools**:
- `query` - Execute SELECT queries
- `execute` - Execute INSERT/UPDATE/DELETE
- `transaction` - Run multiple operations atomically
- `getSchema` - Inspect database schema

### 6. Hello World Server

**Location**: `examples/servers/mcp-helloworld-server/`

Beginner-friendly example with greetings and fortunes.

```go
func main() {
    server := mcp.NewServer(
        mcp.WithName("HelloWorld Server"),
        mcp.WithVersion("1.0.0"),
    )
    
    // Simple greeting tool
    server.AddTool("greet", mcp.ToolFunc(func(args map[string]any) (any, error) {
        name := args["name"].(string)
        language := args["language"].(string)
        
        greeting := getGreeting(language)
        return fmt.Sprintf("%s, %s!", greeting, name), nil
    }))
    
    // Fortune cookie tool
    server.AddTool("fortune", mcp.ToolFunc(func(args map[string]any) (any, error) {
        return getRandomFortune(), nil
    }))
}
```

**Features**:
- Multi-language greetings
- Random fortunes
- Simple implementation
- Great starting point

### 7. Everything Server

**Location**: `examples/servers/mcp-everything-server/`

Comprehensive server implementing all MCP features.

```go
type EverythingServer struct {
    tools     map[string]Tool
    resources map[string]Resource
    prompts   map[string]Prompt
}

func NewEverythingServer() *EverythingServer {
    server := &EverythingServer{
        tools:     make(map[string]Tool),
        resources: make(map[string]Resource),
        prompts:   make(map[string]Prompt),
    }
    
    // Register all capabilities
    server.registerTools()
    server.registerResources()
    server.registerPrompts()
    
    return server
}
```

**Features**:
- All MCP capabilities
- Complex interactions
- State management
- Event notifications
- Error handling examples
- Performance optimizations

**Includes**:
- 20+ tools
- 10+ resources
- 5+ prompts
- Streaming support
- Batch operations

## Running Server Examples

### Basic Execution

```bash
# Run directly
go run ./examples/servers/mcp-echo-server/main.go

# Build and run
go build -o echo-server ./examples/servers/mcp-echo-server
./echo-server
```

### With Monitoring

```bash
# Monitor with mcp-spy
mcp-spy -v -pretty -- go run ./examples/servers/mcp-time-server

# Record interactions
mcp-spy -f trace.mcp -- ./calculator-server
```

### Testing Servers

```bash
# Test with mcp-connect
mcp-connect -cmd="go run ./examples/servers/mcp-echo-server"

# Test with mock client
mcp-replay -mock-client test-requests.mcp | ./server
```

## Common Patterns in Examples

### 1. Server Initialization

```go
func main() {
    server := mcp.NewServer(
        mcp.WithName("My Server"),
        mcp.WithVersion("1.0.0"),
        mcp.WithDescription("Server description"),
    )
    
    // Add capabilities
    setupTools(server)
    setupResources(server)
    
    // Start serving
    if err := server.Serve(mcp.NewStdioTransport()); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Tool Implementation

```go
type MyTool struct {
    config Config
}

func (t *MyTool) Execute(args map[string]any) (any, error) {
    // Validate inputs
    if err := t.validate(args); err != nil {
        return nil, err
    }
    
    // Process request
    result, err := t.process(args)
    if err != nil {
        return nil, fmt.Errorf("processing failed: %w", err)
    }
    
    return result, nil
}
```

### 3. Resource Handling

```go
func (s *Server) setupResources() {
    s.AddResource("config", mcp.ResourceFunc(func() (any, error) {
        return s.config, nil
    }))
    
    s.AddResource("status", mcp.ResourceFunc(func() (any, error) {
        return map[string]any{
            "uptime": time.Since(s.startTime),
            "requests": s.requestCount,
        }, nil
    }))
}
```

### 4. Error Handling

```go
func (t *Tool) Execute(args map[string]any) (any, error) {
    // Input validation
    value, ok := args["required"]
    if !ok {
        return nil, mcp.NewError(mcp.ErrorInvalidParams, 
            "missing required parameter")
    }
    
    // Business logic errors
    result, err := process(value)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, mcp.NewError(mcp.ErrorNotFound, 
                "resource not found")
        }
        return nil, mcp.WrapError(err, "processing failed")
    }
    
    return result, nil
}
```

## Testing Server Examples

Each server example includes tests:

```bash
# Run tests for a specific server
cd examples/servers/mcp-echo-server
go test ./...

# Run integration tests with scripttest
cd examples/servers/mcp-time-server
scripttest testdata/*.txt
```

## Creating Your Own Server

1. **Start with Echo Server** - Copy and modify
2. **Add Tools Incrementally** - One feature at a time
3. **Test Continuously** - Use mcp-spy and tests
4. **Handle Errors** - Return proper MCP errors
5. **Document Everything** - Clear README and comments

## Next Steps

- Try [Client Examples](./clients.md)
- Learn [Common Patterns](./patterns.md)
- Explore [Transport Examples](./transports.md)
- Read [Testing Guide](../testing/README.md)

## See Also

- [Server API Reference](../api/server.md)
- [Tool Implementation Guide](../api/tools.md)
- [Resource Implementation Guide](../api/resources.md)