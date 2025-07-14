# Unit Testing Guide

Unit testing ensures individual components of your MCP implementation work correctly in isolation.

## Overview

Unit tests should:
- Test single functions or methods
- Be fast and isolated
- Not depend on external systems
- Cover edge cases and error paths

## Basic Unit Tests

### Testing a Tool

```go
package tools

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCalculatorTool(t *testing.T) {
    calc := &CalculatorTool{}
    
    t.Run("addition", func(t *testing.T) {
        result, err := calc.Execute(map[string]any{
            "operation": "add",
            "a": 5.0,
            "b": 3.0,
        })
        
        assert.NoError(t, err)
        assert.Equal(t, 8.0, result)
    })
    
    t.Run("invalid operation", func(t *testing.T) {
        _, err := calc.Execute(map[string]any{
            "operation": "invalid",
            "a": 1.0,
            "b": 2.0,
        })
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "unknown operation")
    })
}
```

### Testing a Resource Handler

```go
func TestFileResource(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)
    
    resource := &FileResource{Root: tmpDir}
    
    t.Run("list files", func(t *testing.T) {
        files, err := resource.List()
        assert.NoError(t, err)
        assert.Len(t, files, 1)
        assert.Equal(t, "test.txt", files[0].Name)
    })
    
    t.Run("get file content", func(t *testing.T) {
        content, err := resource.Get("test.txt")
        assert.NoError(t, err)
        assert.Equal(t, "content", string(content))
    })
    
    t.Run("file not found", func(t *testing.T) {
        _, err := resource.Get("nonexistent.txt")
        assert.Error(t, err)
    })
}
```

## Testing Transports

### Mock Transport

```go
type MockTransport struct {
    SendFunc    func(ctx context.Context, msg json.RawMessage) error
    ReceiveFunc func(ctx context.Context) (json.RawMessage, error)
    CloseFunc   func() error
}

func (m *MockTransport) Send(ctx context.Context, msg json.RawMessage) error {
    if m.SendFunc != nil {
        return m.SendFunc(ctx, msg)
    }
    return nil
}

func TestClientWithMockTransport(t *testing.T) {
    transport := &MockTransport{
        ReceiveFunc: func(ctx context.Context) (json.RawMessage, error) {
            return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`), nil
        },
    }
    
    client := mcp.NewClient(transport)
    resp, err := client.Call(ctx, "test", nil)
    
    assert.NoError(t, err)
    assert.Equal(t, "ok", resp.Result)
}
```

## Testing Message Handling

### JSON-RPC Message Tests

```go
func TestMessageParsing(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "valid request",
            input:   `{"jsonrpc":"2.0","id":1,"method":"test"}`,
            wantErr: false,
        },
        {
            name:    "missing jsonrpc",
            input:   `{"id":1,"method":"test"}`,
            wantErr: true,
        },
        {
            name:    "invalid json",
            input:   `{invalid}`,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var msg mcp.Request
            err := json.Unmarshal([]byte(tt.input), &msg)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, "2.0", msg.JSONRPC)
            }
        })
    }
}
```

## Testing Error Handling

### Error Response Tests

```go
func TestErrorResponses(t *testing.T) {
    server := mcp.NewServer()
    
    testCases := []struct {
        name         string
        request      string
        expectedCode int
        expectedMsg  string
    }{
        {
            name:         "method not found",
            request:      `{"jsonrpc":"2.0","id":1,"method":"unknown"}`,
            expectedCode: -32601,
            expectedMsg:  "Method not found",
        },
        {
            name:         "invalid params",
            request:      `{"jsonrpc":"2.0","id":1,"method":"test","params":"invalid"}`,
            expectedCode: -32602,
            expectedMsg:  "Invalid params",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            resp := server.HandleRequest([]byte(tc.request))
            
            var response mcp.Response
            err := json.Unmarshal(resp, &response)
            assert.NoError(t, err)
            
            assert.NotNil(t, response.Error)
            assert.Equal(t, tc.expectedCode, response.Error.Code)
            assert.Contains(t, response.Error.Message, tc.expectedMsg)
        })
    }
}
```

## Testing Concurrency

### Concurrent Access Tests

```go
func TestConcurrentRequests(t *testing.T) {
    server := mcp.NewServer()
    
    // Add a tool that takes time
    server.AddTool("slow", mcp.ToolFunc(func(args map[string]any) (any, error) {
        time.Sleep(10 * time.Millisecond)
        return "done", nil
    }))
    
    // Send multiple concurrent requests
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            req := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"slow"}}`, id)
            resp := server.HandleRequest([]byte(req))
            
            var response mcp.Response
            if err := json.Unmarshal(resp, &response); err != nil {
                errors <- err
                return
            }
            
            if response.Error != nil {
                errors <- fmt.Errorf("request %d failed: %v", id, response.Error)
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent request failed: %v", err)
    }
}
```

## Table-Driven Tests

### Comprehensive Test Cases

```go
func TestToolValidation(t *testing.T) {
    tool := &CalculatorTool{}
    
    tests := []struct {
        name    string
        args    map[string]any
        want    float64
        wantErr string
    }{
        {
            name: "valid addition",
            args: map[string]any{"operation": "add", "a": 1.0, "b": 2.0},
            want: 3.0,
        },
        {
            name:    "missing operation",
            args:    map[string]any{"a": 1.0, "b": 2.0},
            wantErr: "operation required",
        },
        {
            name:    "invalid type",
            args:    map[string]any{"operation": "add", "a": "not-a-number", "b": 2.0},
            wantErr: "invalid type",
        },
        {
            name:    "division by zero",
            args:    map[string]any{"operation": "divide", "a": 1.0, "b": 0.0},
            wantErr: "division by zero",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := tool.Execute(tt.args)
            
            if tt.wantErr != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.wantErr)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

## Test Helpers

### Common Test Utilities

```go
// Test fixture creation
func createTestServer(t *testing.T) *mcp.Server {
    t.Helper()
    
    server := mcp.NewServer()
    server.AddTool("echo", mcp.ToolFunc(func(args map[string]any) (any, error) {
        return args["message"], nil
    }))
    
    return server
}

// Request builder
func buildRequest(t *testing.T, method string, params any) []byte {
    t.Helper()
    
    req := mcp.Request{
        JSONRPC: "2.0",
        ID:      1,
        Method:  method,
        Params:  params,
    }
    
    data, err := json.Marshal(req)
    assert.NoError(t, err)
    
    return data
}

// Response parser
func parseResponse(t *testing.T, data []byte) *mcp.Response {
    t.Helper()
    
    var resp mcp.Response
    err := json.Unmarshal(data, &resp)
    assert.NoError(t, err)
    
    return &resp
}
```

## Test Coverage

### Measuring Coverage

```bash
# Run with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Coverage by function
go tool cover -func=coverage.out
```

### Coverage Goals

Aim for:
- 80%+ overall coverage
- 100% coverage of critical paths
- All error cases covered
- Edge cases tested

## Best Practices

1. **Keep tests focused** - One concept per test
2. **Use descriptive names** - Test names should explain what they test
3. **Test edge cases** - Empty inputs, nil values, extreme values
4. **Test error paths** - Ensure errors are handled correctly
5. **Use test helpers** - DRY principle applies to tests too
6. **Mock external dependencies** - Keep tests isolated
7. **Run tests frequently** - Catch issues early

## Common Pitfalls

### Avoid These Mistakes

1. **Testing implementation instead of behavior**
   ```go
   // Bad - tests internal state
   assert.Equal(t, 5, calculator.internalCounter)
   
   // Good - tests behavior
   result := calculator.Calculate()
   assert.Equal(t, 5, result)
   ```

2. **Not testing error cases**
   ```go
   // Bad - only happy path
   result, _ := doSomething()
   assert.Equal(t, expected, result)
   
   // Good - test errors too
   result, err := doSomething()
   if err != nil {
       assert.Contains(t, err.Error(), "expected error")
   } else {
       assert.Equal(t, expected, result)
   }
   ```

3. **Overly complex tests**
   ```go
   // Bad - too much in one test
   func TestEverything(t *testing.T) {
       // 100 lines of test code
   }
   
   // Good - focused tests
   func TestAddition(t *testing.T) { }
   func TestSubtraction(t *testing.T) { }
   ```

## Next Steps

- Learn [Integration Testing](./integration-testing.md)
- Explore [Mock Testing](./mocking.md)
- Read [Testing Best Practices](./best-practices.md)

## See Also

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Assertion Library](https://github.com/stretchr/testify)
- [Coverage Guide](../development/COVERAGE.md)