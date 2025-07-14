// Package mcp - Comprehensive Benchmarking Suite
//
// This file contains comprehensive performance benchmarks for the MCP Go implementation.
// It covers core operations, transport mechanisms, middleware overhead, and stress scenarios.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// Benchmark Configuration
var (
	benchmarkPayloadSizes = []int{100, 1024, 10240, 102400} // bytes
	benchmarkConcurrency  = []int{1, 10, 100}
	benchmarkIterations   = []int{1000, 10000}
)

// =============================================================================
// Core Operation Benchmarks
// =============================================================================

func BenchmarkClient_Initialize(b *testing.B) {
	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	req := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "benchmark-client", Version: "1.0.0"},
	}

	// Prepare mock response
	response := InitializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ServerInfo:      Implementation{Name: "benchmark-server", Version: "1.0.0"},
		Capabilities:    ServerCapabilities{},
	}
	respData, _ := json.Marshal(response)
	transport.responses <- respData

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := client.Initialize(ctx, req)
		if err != nil {
			b.Errorf("Initialize failed: %v", err)
		}
	}
}

func BenchmarkClient_CallTool(b *testing.B) {
	for _, payloadSize := range benchmarkPayloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", payloadSize), func(b *testing.B) {
			benchmarkCallTool(b, payloadSize)
		})
	}
}

func benchmarkCallTool(b *testing.B, payloadSize int) {
	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Generate payload of specified size
	payload := make(map[string]interface{})
	dataStr := string(make([]byte, payloadSize))
	payload["data"] = dataStr

	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: payload,
	}

	// Prepare mock response
	response := CallToolResult{
		Content: []modelcontextprotocol.Content{
			&modelcontextprotocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Response payload size: %d", payloadSize),
			},
		},
	}
	respData, _ := json.Marshal(response)

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	for i := 0; i < b.N; i++ {
		transport.responses <- respData
		ctx := context.Background()
		_, err := client.CallTool(ctx, req)
		if err != nil {
			b.Errorf("CallTool failed: %v", err)
		}
	}
}

func BenchmarkClient_ListTools(b *testing.B) {
	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	req := ListToolsRequest{}

	// Prepare mock response with varying tool counts
	for _, toolCount := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("ToolCount_%d", toolCount), func(b *testing.B) {
			tools := make([]modelcontextprotocol.Tool, toolCount)
			for i := 0; i < toolCount; i++ {
				tools[i] = modelcontextprotocol.Tool{
					Name:        fmt.Sprintf("tool-%d", i),
					Description: fmt.Sprintf("Tool number %d for benchmarking", i),
					InputSchema: json.RawMessage(`{"type": "object"}`),
				}
			}

			response := ListToolsResult{Tools: tools}
			respData, _ := json.Marshal(response)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				transport.responses <- respData
				ctx := context.Background()
				_, err := client.ListTools(ctx, req)
				if err != nil {
					b.Errorf("ListTools failed: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Server Operation Benchmarks
// =============================================================================

func BenchmarkServer_HandleRequest(b *testing.B) {
	server := NewServer("benchmark-server", "1.0.0")

	// Register a simple tool
	err := server.RegisterTool("echo", modelcontextprotocol.Tool{
		Name:        "echo",
		Description: "Echo the input",
		InputSchema: json.RawMessage(`{"type": "object"}`),
	}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []modelcontextprotocol.Content{
				&modelcontextprotocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Echo: %v", req.Arguments),
				},
			},
		}, nil
	})
	if err != nil {
		b.Fatalf("Failed to register tool: %v", err)
	}

	for _, payloadSize := range benchmarkPayloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", payloadSize), func(b *testing.B) {
			benchmarkServerHandle(b, server, payloadSize)
		})
	}
}

func benchmarkServerHandle(b *testing.B, server *Server, payloadSize int) {
	// Generate payload of specified size
	payload := make(map[string]interface{})
	dataStr := string(make([]byte, payloadSize))
	payload["data"] = dataStr

	req := CallToolRequest{
		Name:      "echo",
		Arguments: payload,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		b.Fatalf("Failed to marshal request: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := server.handleCallTool(ctx, reqData)
		if err != nil {
			b.Errorf("Handle request failed: %v", err)
		}
	}
}

// =============================================================================
// Transport Benchmarks
// =============================================================================

func BenchmarkTransport_ReadWrite(b *testing.B) {
	for _, payloadSize := range benchmarkPayloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", payloadSize), func(b *testing.B) {
			benchmarkTransportReadWrite(b, payloadSize)
		})
	}
}

func benchmarkTransportReadWrite(b *testing.B, payloadSize int) {
	// Create in-memory transport pair
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	serverRWC := &rwcCombiner{
		Reader: serverReader,
		Writer: serverWriter,
		Closer: &multiCloser{closers: []io.Closer{serverReader, serverWriter}},
	}

	clientRWC := &rwcCombiner{
		Reader: clientReader,
		Writer: clientWriter,
		Closer: &multiCloser{closers: []io.Closer{clientReader, clientWriter}},
	}

	serverTransport := &ReadWriteCloserTransport{serverRWC}
	clientTransport := &ReadWriteCloserTransport{clientRWC}

	// Generate test data
	testData := make([]byte, payloadSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(2)

		// Writer goroutine
		go func() {
			defer wg.Done()
			_, err := clientTransport.Write(testData)
			if err != nil {
				b.Errorf("Write failed: %v", err)
			}
		}()

		// Reader goroutine
		go func() {
			defer wg.Done()
			buffer := make([]byte, payloadSize)
			_, err := io.ReadFull(serverTransport, buffer)
			if err != nil {
				b.Errorf("Read failed: %v", err)
			}
		}()

		wg.Wait()
	}
}

// =============================================================================
// JSON Processing Benchmarks
// =============================================================================

func BenchmarkJSON_Marshal(b *testing.B) {
	for _, payloadSize := range benchmarkPayloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", payloadSize), func(b *testing.B) {
			benchmarkJSONMarshal(b, payloadSize)
		})
	}
}

func benchmarkJSONMarshal(b *testing.B, payloadSize int) {
	// Create test data structure
	data := make(map[string]interface{})
	data["type"] = "call_tool"
	data["name"] = "test-tool"
	data["arguments"] = map[string]interface{}{
		"data": string(make([]byte, payloadSize)),
		"size": payloadSize,
	}

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Errorf("Marshal failed: %v", err)
		}
	}
}

func BenchmarkJSON_Unmarshal(b *testing.B) {
	for _, payloadSize := range benchmarkPayloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", payloadSize), func(b *testing.B) {
			benchmarkJSONUnmarshal(b, payloadSize)
		})
	}
}

func benchmarkJSONUnmarshal(b *testing.B, payloadSize int) {
	// Pre-generate JSON data
	data := map[string]interface{}{
		"type": "call_tool",
		"name": "test-tool",
		"arguments": map[string]interface{}{
			"data": string(make([]byte, payloadSize)),
			"size": payloadSize,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		b.Fatalf("Failed to pre-marshal data: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(jsonData)))

	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := json.Unmarshal(jsonData, &result)
		if err != nil {
			b.Errorf("Unmarshal failed: %v", err)
		}
	}
}

// =============================================================================
// Middleware Benchmarks
// =============================================================================

func BenchmarkMiddleware_LoggingOverhead(b *testing.B) {
	// Create base handler
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &BenchmarkSuccessResponseImpl{Result: "success"}, nil
	})

	// Create logging middleware
	loggingMiddleware := NewLoggingMiddleware(LoggingConfig{
		Level: LogLevel("ERROR"), // Minimal logging to measure overhead
	})

	wrappedHandler := loggingMiddleware.Apply(baseHandler)

	req := &BenchmarkMockRequest{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := wrappedHandler.Handle(context.Background(), req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}
	}
}

func BenchmarkMiddleware_ChainOverhead(b *testing.B) {
	// Create base handler
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &BenchmarkSuccessResponseImpl{Result: "success"}, nil
	})

	// Test with different chain lengths
	for _, chainLength := range []int{1, 5, 10, 20} {
		b.Run(fmt.Sprintf("ChainLength_%d", chainLength), func(b *testing.B) {
			handler := MCPHandler(baseHandler)

			// Apply middleware chain
			for i := 0; i < chainLength; i++ {
				middleware := NewLoggingMiddleware(LoggingConfig{
					Level: LogLevel("ERROR"),
				})
				handler = middleware.Apply(handler)
			}

			req := &BenchmarkMockRequest{
				method: "test/method",
				id:     "test-id",
				params: json.RawMessage(`{"test": "data"}`),
				ctx:    context.Background(),
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := handler.Handle(context.Background(), req)
				if err != nil {
					b.Errorf("Handler failed: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Concurrency Benchmarks
// =============================================================================

func BenchmarkConcurrency_ClientOperations(b *testing.B) {
	for _, concurrency := range benchmarkConcurrency {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			benchmarkConcurrentClientOps(b, concurrency)
		})
	}
}

func benchmarkConcurrentClientOps(b *testing.B, concurrency int) {
	transport := &mockTransport{responses: make(chan json.RawMessage, 10000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: map[string]interface{}{"test": "data"},
	}

	// Prepare responses
	response := CallToolResult{
		Content: []modelcontextprotocol.Content{
			&modelcontextprotocol.TextContent{
				Type: "text",
				Text: "test response",
			},
		},
	}
	respData, _ := json.Marshal(response)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				transport.responses <- respData
				ctx := context.Background()
				_, err := client.CallTool(ctx, req)
				if err != nil {
					b.Errorf("CallTool failed: %v", err)
				}
			}()
		}
		wg.Wait()
	}
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkMemory_AllocationPatterns(b *testing.B) {
	b.Run("CreateClient", func(b *testing.B) {
		transport := &mockTransport{}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			client, err := NewClient(transport)
			if err != nil {
				b.Errorf("Failed to create client: %v", err)
			}
			client.Close()
		}
	})

	b.Run("CreateServer", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			server := NewServer("test-server", "1.0.0")
			_ = server
		}
	})

	b.Run("JSONMarshaling", func(b *testing.B) {
		data := CallToolRequest{
			Name: "test-tool",
			Arguments: map[string]interface{}{
				"arg1": "value1",
				"arg2": 42,
				"arg3": []string{"a", "b", "c"},
			},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(data)
			if err != nil {
				b.Errorf("Marshal failed: %v", err)
			}
		}
	})
}

// =============================================================================
// Stress Test Benchmarks
// =============================================================================

func BenchmarkStress_HighThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}

	server := NewServer("stress-test-server", "1.0.0")

	// Register multiple tools
	for i := 0; i < 10; i++ {
		toolName := fmt.Sprintf("tool-%d", i)
		err := server.RegisterTool(toolName, modelcontextprotocol.Tool{
			Name:        toolName,
			Description: fmt.Sprintf("Stress test tool %d", i),
			InputSchema: json.RawMessage(`{"type": "object"}`),
		}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			return &CallToolResult{
				Content: []modelcontextprotocol.Content{
					&modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Response from %s", req.Name),
					},
				},
			}, nil
		})
		if err != nil {
			b.Fatalf("Failed to register tool %s: %v", toolName, err)
		}
	}

	b.ResetTimer()

	// High throughput test
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func(toolNum int) {
				defer wg.Done()
				req := CallToolRequest{
					Name:      fmt.Sprintf("tool-%d", toolNum%10),
					Arguments: map[string]interface{}{"iteration": i, "worker": toolNum},
				}

				reqData, _ := json.Marshal(req)
				ctx := context.Background()
				_, err := server.handleCallTool(ctx, reqData)
				if err != nil {
					b.Errorf("Handle request failed: %v", err)
				}
			}(j)
		}
		wg.Wait()
	}
}

func BenchmarkStress_MemoryPressure(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory pressure test in short mode")
	}

	// Force GC to establish baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Large payload request
	largePayload := make(map[string]interface{})
	largePayload["data"] = string(make([]byte, 1024*1024)) // 1MB

	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: largePayload,
	}

	response := CallToolResult{
		Content: []modelcontextprotocol.Content{
			&modelcontextprotocol.TextContent{
				Type: "text",
				Text: string(make([]byte, 1024*1024)), // 1MB response
			},
		},
	}
	respData, _ := json.Marshal(response)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		transport.responses <- respData
		ctx := context.Background()
		_, err := client.CallTool(ctx, req)
		if err != nil {
			b.Errorf("CallTool failed: %v", err)
		}

		// Periodic GC to measure pressure
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Measure final memory usage
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// =============================================================================
// Mock Types for Benchmarking
// =============================================================================

type mockTransport struct {
	responses chan json.RawMessage
	mu        sync.Mutex
	closed    bool
}

func (m *mockTransport) Read(p []byte) (n int, err error) {
	select {
	case response := <-m.responses:
		copy(p, response)
		return len(response), nil
	default:
		return 0, io.EOF
	}
}

func (m *mockTransport) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		close(m.responses)
		m.closed = true
	}
	return nil
}

// MockRequest for middleware benchmarks
type BenchmarkMockRequest struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func (r *BenchmarkMockRequest) GetMethod() string {
	return r.method
}

func (r *BenchmarkMockRequest) GetID() interface{} {
	return r.id
}

func (r *BenchmarkMockRequest) GetParams() json.RawMessage {
	return r.params
}

func (r *BenchmarkMockRequest) GetContext() context.Context {
	return r.ctx
}

func (r *BenchmarkMockRequest) WithContext(ctx context.Context) MCPRequest {
	return &BenchmarkMockRequest{
		method: r.method,
		id:     r.id,
		params: r.params,
		ctx:    ctx,
	}
}

// SuccessResponseImpl for testing
type BenchmarkSuccessResponseImpl struct {
	Result interface{}
}

func (r *BenchmarkSuccessResponseImpl) IsError() bool {
	return false
}

func (r *BenchmarkSuccessResponseImpl) GetError() error {
	return nil
}

func (r *BenchmarkSuccessResponseImpl) GetResult() interface{} {
	return r.Result
}

// Utility types for transport benchmarks (already exist in typed_test.go)
// Included here for completeness if they don't exist elsewhere

type multiCloserBench struct {
	closers []io.Closer
}

func (mc *multiCloserBench) Close() error {
	var lastErr error
	for _, c := range mc.closers {
		if err := c.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

type rwcCombinerBench struct {
	io.Reader
	io.Writer
	io.Closer
}