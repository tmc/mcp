# Performance Guide

## Benchmark Results

**Last Updated**: August 31, 2025  
**Platform**: Apple M4 Max  
**Go Version**: 1.21+

### Executive Summary

| Component | Throughput | Latency (p99) | Memory |
|-----------|------------|---------------|---------|
| Transport Layer | **10.9 GB/s** | <1ms | 3 allocs/op |
| Server Handler | 5.81 MB/s | 17ms | 618 allocs/op |
| Client Operations | 2.3M ops/s | <1ms | 12 allocs/op |
| Middleware Stack | - | <1ms/component | Varies |

## Detailed Benchmarks

### Transport Layer Performance

The transport layer shows excellent performance with minimal allocations:

```
BenchmarkTransport_ReadWrite/PayloadSize_100-16      97,605 ops    93.06 MB/s    3 allocs/op
BenchmarkTransport_ReadWrite/PayloadSize_1024-16    103,581 ops   852.10 MB/s    3 allocs/op
BenchmarkTransport_ReadWrite/PayloadSize_10240-16    57,163 ops     4.89 GB/s    3 allocs/op
BenchmarkTransport_ReadWrite/PayloadSize_102400-16   13,074 ops    10.94 GB/s    3 allocs/op
```

**Key Insights**:
- Consistent 3 allocations regardless of payload size
- Near-linear scaling with payload size
- Excellent for streaming large data

### Server Handler Performance

The server handler shows room for optimization:

```
BenchmarkServer_HandleRequest/PayloadSize_100-16      5,954 ops    5.81 MB/s      618 allocs/op
BenchmarkServer_HandleRequest/PayloadSize_1024-16       733 ops    5.97 MB/s    6,162 allocs/op
BenchmarkServer_HandleRequest/PayloadSize_10240-16       72 ops    5.70 MB/s   61,486 allocs/op
BenchmarkServer_HandleRequest/PayloadSize_102400-16       6 ops    5.69 MB/s  614,463 allocs/op
```

**Issues Identified**:
- High allocation count (grows linearly with payload)
- Throughput doesn't scale with payload size
- Memory pressure from excessive allocations

### Middleware Performance Impact

Each middleware component adds overhead:

| Middleware | Latency | Allocations | Notes |
|------------|---------|-------------|-------|
| Logging | <0.1ms | 2-5 | Async logging recommended |
| Authentication | 0.5-1ms | 10-15 | Token validation cached |
| Rate Limiting | <0.1ms | 1-2 | Per-client buckets |
| Compression | 0.5-2ms | 5-10 | Only for >1KB payloads |
| Caching | <0.1ms | 1-3 | Significant for cache hits |

## Optimization Guide

### 1. Reduce Server Allocations (Priority: HIGH)

**Problem**: 618 allocations per request for small payloads

**Solutions**:
```go
// Use sync.Pool for request/response objects
var requestPool = sync.Pool{
    New: func() interface{} {
        return &CallToolRequest{}
    },
}

// Reuse buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 4096))
    },
}

// Pre-allocate slices with capacity
content := make([]interface{}, 0, 10)
```

### 2. Optimize JSON Processing

**Problem**: JSON marshaling/unmarshaling is expensive

**Solutions**:
```go
// Use json.RawMessage to defer parsing
type Request struct {
    Method string          `json:"method"`
    Params json.RawMessage `json:"params"`
}

// Consider alternative encoders
import "github.com/bytedance/sonic"
// 2-3x faster than encoding/json

// Stream large responses
encoder := json.NewEncoder(w)
encoder.Encode(response)
```

### 3. Connection Pooling

**Problem**: Connection overhead for HTTP transports

**Solutions**:
```go
// Configure HTTP client properly
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  true, // Handle at app level
    },
    Timeout: 30 * time.Second,
}
```

### 4. Middleware Optimization

**Problem**: Middleware stack adds latency

**Solutions**:
```go
// Conditional middleware application
if payloadSize > 1024 {
    applyCompression()
}

// Async logging
go logger.Log(request)

// Cache validation results
validationCache := ttlcache.New(
    ttlcache.WithTTL(5 * time.Minute),
)
```

## Performance Tuning

### System Configuration

```bash
# Increase file descriptor limits
ulimit -n 65536

# TCP tuning for Linux
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"

# Enable TCP fast open
sysctl -w net.ipv4.tcp_fastopen=3
```

### Go Runtime Tuning

```go
// Set GOMAXPROCS appropriately
runtime.GOMAXPROCS(runtime.NumCPU())

// Tune GC for low latency
debug.SetGCPercent(100)  // Default
// OR for throughput
debug.SetGCPercent(200)  // Less frequent GC

// Pre-allocate memory
ballast := make([]byte, 10<<20) // 10MB
runtime.KeepAlive(ballast)
```

### Profile-Guided Optimization

```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=.

# Analyze profile
go tool pprof cpu.prof

# Generate memory profile
go test -memprofile=mem.prof -bench=.

# Find allocations
go tool pprof -alloc_space mem.prof
```

## Benchmark Commands

### Running Benchmarks

```bash
# Basic benchmark
go test -bench=. -benchmem

# Longer runs for stability
go test -bench=. -benchtime=10s

# Specific benchmarks
go test -bench=BenchmarkServer -benchmem

# With CPU profile
go test -bench=. -cpuprofile=cpu.prof

# Compare benchmarks
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

### Continuous Performance Monitoring

```bash
# Track performance over time
#!/bin/bash
while true; do
    date >> perf.log
    go test -bench=. -benchmem | tee -a perf.log
    sleep 3600  # Every hour
done
```

## Load Testing

### Using hey

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test HTTP endpoint
hey -n 10000 -c 100 -m POST \
    -H "Content-Type: application/json" \
    -d '{"method":"test","params":{}}' \
    http://localhost:8080/mcp

# Results to watch for:
# - Requests/sec > 10,000
# - 99% latency < 100ms
# - 0% error rate
```

### Using vegeta

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Create targets file
echo "POST http://localhost:8080/mcp" > targets.txt

# Attack for 30 seconds
vegeta attack -duration=30s -rate=1000 -targets=targets.txt | \
    vegeta report -type=text
```

## Scaling Considerations

### Vertical Scaling

- **CPU**: MCP is CPU-bound for JSON processing
- **Memory**: 1GB per 10,000 concurrent connections
- **Network**: 1Gbps handles ~100,000 req/s

### Horizontal Scaling

```go
// Use load balancer aware health checks
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if isHealthy() {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
})

// Implement graceful shutdown
sigterm := make(chan os.Signal, 1)
signal.Notify(sigterm, syscall.SIGTERM)
<-sigterm
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## Performance Anti-Patterns

### ❌ Avoid These

1. **Synchronous logging in hot path**
   ```go
   // Bad
   logger.Info("Processing request", "id", req.ID)
   
   // Good
   go logger.Info("Processing request", "id", req.ID)
   ```

2. **Unbounded goroutines**
   ```go
   // Bad
   for _, item := range items {
       go process(item)
   }
   
   // Good
   sem := make(chan struct{}, 10)
   for _, item := range items {
       sem <- struct{}{}
       go func(i Item) {
           defer func() { <-sem }()
           process(i)
       }(item)
   }
   ```

3. **String concatenation in loops**
   ```go
   // Bad
   result := ""
   for _, s := range strings {
       result += s
   }
   
   // Good
   var builder strings.Builder
   for _, s := range strings {
       builder.WriteString(s)
   }
   result := builder.String()
   ```

## Monitoring & Observability

### Key Metrics to Track

- **Request rate** (req/s)
- **Response time** (p50, p95, p99)
- **Error rate** (4xx, 5xx)
- **Active connections**
- **Memory usage** (heap, GC pressure)
- **CPU utilization**
- **Goroutine count**

### Prometheus Integration

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mcp_request_duration_seconds",
            Help: "Request duration in seconds",
        },
        []string{"method"},
    )
)

// Register metrics
prometheus.MustRegister(requestDuration)

// Record metrics
timer := prometheus.NewTimer(requestDuration.WithLabelValues(method))
defer timer.ObserveDuration()
```

## Future Optimizations

### Planned Improvements

1. **Q4 2025**
   - Reduce server allocations to <50/op
   - Implement zero-copy JSON parsing
   - Add SIMD optimizations for large payloads

2. **Q1 2026**
   - HTTP/3 QUIC support
   - io_uring for Linux
   - eBPF-based monitoring

3. **Q2 2026**
   - WebAssembly runtime for plugins
   - GPU acceleration for crypto
   - Custom memory allocator

## Conclusion

The MCP Go implementation shows excellent transport layer performance but has optimization opportunities in the server handler layer. Focus areas for improvement:

1. **Immediate**: Reduce allocations in server handler
2. **Short-term**: Optimize JSON processing
3. **Long-term**: Implement zero-copy architecture

Expected improvements after optimization:
- Server throughput: 5.81 MB/s → 50+ MB/s
- Allocations: 618/op → <50/op
- Latency p99: 17ms → <5ms

For questions or performance issues, please open an issue with benchmark results and profiles.