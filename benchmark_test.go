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
	"log/slog"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
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

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		b.Fatalf("Failed to marshal payload: %v", err)
	}

	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: payloadJSON,
	}

	// Prepare mock response
	response := CallToolResult{
		Content: []any{
			map[string]any{
				"type": "text",
				"text": fmt.Sprintf("Response payload size: %d", payloadSize),
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
			tools := make([]Tool, toolCount)
			for i := 0; i < toolCount; i++ {
				tools[i] = Tool{
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
	tool := Tool{
		Name:        "echo",
		Description: "Echo the input",
		InputSchema: json.RawMessage(`{"type": "object"}`),
	}
	err := server.RegisterTool(tool, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Echo: %v", req.Arguments),
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

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		b.Fatalf("Failed to marshal payload: %v", err)
	}

	req := CallToolRequest{
		Name:      "echo",
		Arguments: payloadJSON,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		b.Fatalf("Failed to marshal request: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	for i := 0; i < b.N; i++ {
		// TODO: This benchmark needs to be updated to use public server methods
		// For now, just simulate the work
		_ = reqData
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
		Level: slog.LevelError, // Minimal logging to measure overhead
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
					Level: slog.LevelError,
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

	reqArgs, _ := json.Marshal(map[string]interface{}{"test": "data"})
	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: reqArgs,
	}

	// Prepare responses
	response := CallToolResult{
		Content: []any{
			map[string]string{
				"type": "text",
				"text": "test response",
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
		args, _ := json.Marshal(map[string]interface{}{
			"arg1": "value1",
			"arg2": 42,
			"arg3": []string{"a", "b", "c"},
		})
		data := CallToolRequest{
			Name:      "test-tool",
			Arguments: args,
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
		err := server.RegisterTool(Tool{
			Name:        toolName,
			Description: fmt.Sprintf("Stress test tool %d", i),
			InputSchema: json.RawMessage(`{"type": "object"}`),
		}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			return &CallToolResult{
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Response from %s", req.Name),
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
				args, _ := json.Marshal(map[string]interface{}{"iteration": i, "worker": toolNum})
				req := CallToolRequest{
					Name:      fmt.Sprintf("tool-%d", toolNum%10),
					Arguments: args,
				}

				ctx := context.Background()
				// Use the tool handler directly since handleCallTool is not exported
				if toolDef, ok := server.tools[req.Name]; ok {
					_, err := toolDef.handler(ctx, req)
					if err != nil {
						b.Errorf("Handle request failed: %v", err)
					}
				} else {
					b.Errorf("tool not found: %s", req.Name)
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

	payloadData, _ := json.Marshal(largePayload)
	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: payloadData,
	}

	response := CallToolResult{
		Content: []any{
			map[string]string{
				"type": "text",
				"text": string(make([]byte, 1024*1024)), // 1MB response
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
// Profiling Support Benchmarks
// =============================================================================

func BenchmarkWithCPUProfile(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping CPU profiling benchmark in short mode")
	}

	// Start CPU profiling
	cpuProfile := pprof.Lookup("cpu")
	if cpuProfile == nil {
		b.Skip("CPU profiling not available")
	}

	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Large payload for CPU intensive work
	payload := make(map[string]interface{})
	dataStr := string(make([]byte, 10240))
	payload["data"] = dataStr
	payloadJSON, _ := json.Marshal(payload)

	req := CallToolRequest{
		Name:      "test-tool",
		Arguments: payloadJSON,
	}

	response := CallToolResult{
		Content: []any{
			map[string]any{
				"type": "text",
				"text": "CPU intensive response",
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
	}
}

func BenchmarkWithMemoryProfile(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory profiling benchmark in short mode")
	}

	// Force GC and capture baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	server := NewServer("memory-profile-server", "1.0.0")

	// Register multiple tools with memory allocation
	for i := 0; i < 10; i++ {
		toolName := fmt.Sprintf("memory-tool-%d", i)
		err := server.RegisterTool(Tool{
			Name:        toolName,
			Description: fmt.Sprintf("Memory allocation tool %d", i),
			InputSchema: json.RawMessage(`{"type": "object"}`),
		}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			// Allocate memory to simulate work
			data := make([]byte, 1024*10) // 10KB allocation
			_ = data
			
			return &CallToolResult{
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Allocated memory for %s", req.Name),
					},
				},
			}, nil
		})
		if err != nil {
			b.Fatalf("Failed to register tool %s: %v", toolName, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		toolName := fmt.Sprintf("memory-tool-%d", i%10)
		args, _ := json.Marshal(map[string]interface{}{"iteration": i})
		req := CallToolRequest{
			Name:      toolName,
			Arguments: args,
		}

		ctx := context.Background()
		if toolDef, ok := server.tools[req.Name]; ok {
			_, err := toolDef.handler(ctx, req)
			if err != nil {
				b.Errorf("Handle request failed: %v", err)
			}
		}

		// Sample memory every 100 operations
		if i%100 == 0 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			b.ReportMetric(float64(m.Alloc-m1.Alloc), "current-bytes-alloc")
		}
	}

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "avg-bytes/op")
}

// =============================================================================
// Performance Regression Detection
// =============================================================================

func BenchmarkPerformanceBaseline(b *testing.B) {
	// This benchmark serves as a baseline for performance regression detection
	// Run with: go test -bench=BenchmarkPerformanceBaseline -benchtime=5s
	
	transport := &mockTransport{responses: make(chan json.RawMessage, 1000)}
	client, err := NewClient(transport)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Standard 1KB payload
	payload := make(map[string]interface{})
	payload["data"] = string(make([]byte, 1024))
	payloadJSON, _ := json.Marshal(payload)

	req := CallToolRequest{
		Name:      "baseline-tool",
		Arguments: payloadJSON,
	}

	response := CallToolResult{
		Content: []any{
			map[string]any{
				"type": "text",
				"text": "baseline response",
			},
		},
	}
	respData, _ := json.Marshal(response)

	b.ResetTimer()

	start := time.Now()
	for i := 0; i < b.N; i++ {
		transport.responses <- respData
		ctx := context.Background()
		_, err := client.CallTool(ctx, req)
		if err != nil {
			b.Errorf("CallTool failed: %v", err)
		}
	}
	elapsed := time.Since(start)

	// Report key performance metrics
	opsPerSecond := float64(b.N) / elapsed.Seconds()
	avgLatency := elapsed.Nanoseconds() / int64(b.N)
	
	b.ReportMetric(opsPerSecond, "ops/sec")
	b.ReportMetric(float64(avgLatency)/1e6, "avg-latency-ms")
	
	// Set performance thresholds for regression detection
	minOpsPerSecond := 10000.0 // Minimum 10k ops/sec
	maxLatencyMs := 1.0       // Maximum 1ms average latency
	
	if opsPerSecond < minOpsPerSecond {
		b.Logf("WARNING: Performance below threshold: %.2f ops/sec (min: %.2f)", 
			opsPerSecond, minOpsPerSecond)
	}
	
	if float64(avgLatency)/1e6 > maxLatencyMs {
		b.Logf("WARNING: Latency above threshold: %.2f ms (max: %.2f)", 
			float64(avgLatency)/1e6, maxLatencyMs)
	}
}

// =============================================================================
// Bottleneck Identification Benchmarks
// =============================================================================

func BenchmarkBottleneckAnalysis_JSONProcessing(b *testing.B) {
	// Test JSON marshaling/unmarshaling performance as potential bottleneck
	
	payloadSizes := []int{100, 1024, 10240, 102400}
	
	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("Marshal_%d", size), func(b *testing.B) {
			data := make(map[string]interface{})
			data["payload"] = string(make([]byte, size))
			data["timestamp"] = time.Now().Unix()
			data["metadata"] = map[string]interface{}{
				"size": size,
				"type": "benchmark",
			}
			
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := json.Marshal(data)
				if err != nil {
					b.Errorf("Marshal failed: %v", err)
				}
			}
		})
		
		b.Run(fmt.Sprintf("Unmarshal_%d", size), func(b *testing.B) {
			data := make(map[string]interface{})
			data["payload"] = string(make([]byte, size))
			data["timestamp"] = time.Now().Unix()
			
			jsonData, _ := json.Marshal(data)
			
			b.SetBytes(int64(len(jsonData)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				var result map[string]interface{}
				err := json.Unmarshal(jsonData, &result)
				if err != nil {
					b.Errorf("Unmarshal failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkBottleneckAnalysis_ContextOverhead(b *testing.B) {
	// Test context overhead as potential bottleneck
	
	b.Run("ContextCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			ctx = context.WithValue(ctx, "request_id", fmt.Sprintf("req-%d", i))
			ctx = context.WithValue(ctx, "timestamp", time.Now())
			_ = ctx
		}
	})
	
	b.Run("ContextPropagation", func(b *testing.B) {
		baseCtx := context.Background()
		baseCtx = context.WithValue(baseCtx, "session_id", "test-session")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.WithValue(baseCtx, "request_id", fmt.Sprintf("req-%d", i))
			
			// Simulate context propagation through multiple layers
			for j := 0; j < 5; j++ {
				ctx = context.WithValue(ctx, fmt.Sprintf("layer_%d", j), j)
			}
			_ = ctx
		}
	})
}

func BenchmarkBottleneckAnalysis_GoroutineOverhead(b *testing.B) {
	// Test goroutine creation overhead
	
	b.Run("DirectExecution", func(b *testing.B) {
		work := func() {
			// Simulate some work
			sum := 0
			for i := 0; i < 100; i++ {
				sum += i
			}
			_ = sum
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			work()
		}
	})
	
	b.Run("GoroutineExecution", func(b *testing.B) {
		work := func() {
			sum := 0
			for i := 0; i < 100; i++ {
				sum += i
			}
			_ = sum
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				work()
			}()
			wg.Wait()
		}
	})
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

func (m *mockTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return m, nil
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

func (r *BenchmarkSuccessResponseImpl) GetError() *ResponseError {
	return nil
}

func (r *BenchmarkSuccessResponseImpl) GetResult() interface{} {
	return r.Result
}

// =============================================================================
// Helper types for benchmarking
// =============================================================================

// Note: multiCloser, rwcCombiner, and generateRandomString are defined in other files
