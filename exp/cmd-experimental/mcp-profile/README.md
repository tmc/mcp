# mcp-profile: Runtime Performance Analysis Tool

`mcp-profile` is a comprehensive runtime performance analysis tool for MCP (Model Context Protocol) servers. It provides CPU profiling, memory profiling, goroutine analysis, I/O monitoring, and performance visualization capabilities.

## Features

### Profiling Types
- **CPU Profiling**: Detailed CPU usage analysis with call graphs
- **Memory Profiling**: Heap analysis and memory allocation tracking
- **Goroutine Profiling**: Concurrent execution analysis
- **Blocking Operations**: I/O and synchronization bottleneck detection
- **Mutex Contention**: Lock contention analysis
- **Execution Tracing**: Timeline-based execution visualization

### Analysis Capabilities
- **Hot Path Detection**: Identify performance-critical code paths
- **Memory Leak Detection**: Spot potential memory leaks
- **Bottleneck Analysis**: Pinpoint performance bottlenecks
- **Comparative Analysis**: Compare profiles between runs
- **Trend Analysis**: Track performance over time
- **Regression Detection**: Identify performance regressions

### Visualization
- **Call Graph Visualization**: Visual representation of function calls
- **Timeline Analysis**: Execution timeline with interactive charts
- **Memory Usage Graphs**: Memory allocation and usage patterns
- **Performance Dashboards**: Real-time performance monitoring

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-profile@latest
```

Or build from source:

```bash
cd cmd/mcp-profile
go build -o mcp-profile
```

## Usage

### Basic Profiling

```bash
# Basic CPU and memory profiling
mcp-profile -cpu -mem go run ./server

# Profile all aspects for 60 seconds
mcp-profile -all -duration 60s go run ./server

# Profile with load testing
mcp-profile -cpu -mem -load-test -concurrency 10 go run ./server
```

### Specific Profile Types

```bash
# CPU profiling only
mcp-profile -cpu -duration 30s go run ./server

# Memory profiling with custom sampling
mcp-profile -mem -mem-sampling-rate 1048576 go run ./server

# Goroutine profiling
mcp-profile -goroutine go run ./server

# Blocking operations profiling
mcp-profile -block go run ./server

# Mutex contention profiling
mcp-profile -mutex go run ./server

# Execution tracing
mcp-profile -trace -output trace.trace go run ./server
```

### Analysis and Comparison

```bash
# Analyze existing profiles
mcp-profile -analyze cpu.prof mem.prof

# Compare two profiles
mcp-profile -compare baseline.prof current.prof

# Generate detailed analysis report
mcp-profile -analyze -visualize -output analysis.json cpu.prof
```

### Continuous Profiling

```bash
# Continuous profiling with 5-minute intervals
mcp-profile -continuous -interval 5m go run ./server

# Continuous profiling with custom retention
mcp-profile -continuous -interval 1m -retention 24h go run ./server
```

### Advanced Options

```bash
# Profile with custom sampling rates
mcp-profile -cpu -sampling-rate 200 -mem-sampling-rate 512000 go run ./server

# Profile with load testing
mcp-profile -cpu -mem -load-test -concurrency 20 -requests 1000 go run ./server

# Profile with visualization
mcp-profile -all -visualize -output-dir ./profiles go run ./server
```

## Command Line Options

### Profiling Types
- `-cpu`: Enable CPU profiling
- `-mem`: Enable memory profiling
- `-goroutine`: Enable goroutine profiling
- `-block`: Enable blocking operations profiling
- `-mutex`: Enable mutex contention profiling
- `-trace`: Enable execution tracing
- `-all`: Enable all profiling types

### Analysis Options
- `-duration`: Profiling duration (default: 30s)
- `-sampling-rate`: CPU profiling sampling rate in Hz (default: 100)
- `-mem-sampling-rate`: Memory profiling sampling rate in bytes (default: 512KB)
- `-analyze`: Analyze existing profiles
- `-compare`: Compare two profiles
- `-visualize`: Generate visualization files

### Output Options
- `-output-dir`: Output directory for profile files (default: ./profiles)
- `-output`: Output file for specific profile
- `-format`: Output format (pprof, json, text)
- `-top`: Show top N functions in analysis (default: 10)

### Load Testing Integration
- `-load-test`: Run load test during profiling
- `-concurrency`: Concurrent clients for load test (default: 10)
- `-requests`: Total requests for load test (default: 1000)
- `-tool`: Specific tool to test during profiling
- `-tool-args`: JSON arguments for the tool

### Continuous Profiling
- `-continuous`: Enable continuous profiling mode
- `-interval`: Profiling interval (default: 30s)
- `-retention`: Profile retention period (default: 24h)

### Transport Options
- `-transport`: Transport type (stdio, http, sse)
- `-http-url`: HTTP URL for HTTP transport
- `-sse-url`: SSE URL for SSE transport
- `-timeout`: Request timeout (default: 10s)

## Output Formats

### Profile Files
- **CPU Profile**: `cpu.prof` - CPU usage profile in pprof format
- **Memory Profile**: `mem.prof` - Memory allocation profile
- **Goroutine Profile**: `goroutine.prof` - Goroutine state snapshot
- **Block Profile**: `block.prof` - Blocking operations profile
- **Mutex Profile**: `mutex.prof` - Mutex contention profile
- **Execution Trace**: `trace.trace` - Execution timeline trace

### Analysis Results
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "duration": "30s",
  "profileTypes": ["cpu", "mem"],
  "files": {
    "cpu": "./profiles/cpu20240115-103000.prof",
    "mem": "./profiles/mem20240115-103000.prof"
  },
  "metrics": {
    "cpuUsage": 65.5,
    "memoryUsage": 134217728,
    "goroutineCount": 25,
    "gcPauses": 12
  },
  "analysis": {
    "topFunctions": [...],
    "hotPaths": [...],
    "memoryLeaks": [...],
    "recommendations": [...]
  }
}
```

### Console Output
```
=== Profiling Results ===
Duration: 30.045s
Timestamp: 2024-01-15 10:30:00

Profile Files:
  cpu: ./profiles/cpu20240115-103000.prof
  mem: ./profiles/mem20240115-103000.prof

Metrics:
  Average Memory Usage: 134217728 bytes
  Average Goroutines: 25
  Heap Size: 128MB
  GC Pauses: 12

Load Test Results:
  Total Requests: 1000
  Successful: 985
  Failed: 15
  Requests/sec: 32.84
  Average Latency: 18.5ms
  P95 Latency: 45.2ms
  P99 Latency: 78.9ms
  Error Rate: 1.50%

Recommendations:
  1. [medium] main.worker: Optimize hot loop in request processing
     Impact: 20-30% CPU reduction
  2. [medium] encoding/json.Marshal: Reduce memory allocations
     Impact: 15-25% memory reduction
```

## Integration with pprof

The generated profile files are compatible with Go's pprof tool:

```bash
# Analyze CPU profile
go tool pprof cpu.prof

# Analyze memory profile
go tool pprof mem.prof

# Generate call graph
go tool pprof -png cpu.prof > callgraph.png

# Generate flame graph
go tool pprof -http=:8080 cpu.prof
```

## Analysis Examples

### CPU Hotspots
```bash
# Profile CPU usage and identify hotspots
mcp-profile -cpu -duration 60s go run ./server

# Analyze the generated profile
go tool pprof cpu.prof
(pprof) top10
(pprof) list main.worker
(pprof) web
```

### Memory Analysis
```bash
# Profile memory usage
mcp-profile -mem -duration 60s go run ./server

# Analyze memory allocations
go tool pprof mem.prof
(pprof) top10 -cum
(pprof) list encoding/json.Marshal
```

### Goroutine Analysis
```bash
# Profile goroutine usage
mcp-profile -goroutine go run ./server

# Analyze goroutine states
go tool pprof goroutine.prof
(pprof) top10
(pprof) traces
```

### Blocking Operations
```bash
# Profile blocking operations
mcp-profile -block -duration 60s go run ./server

# Analyze blocking patterns
go tool pprof block.prof
(pprof) top10 -cum
(pprof) list main.worker
```

### Execution Tracing
```bash
# Generate execution trace
mcp-profile -trace -output trace.trace go run ./server

# View trace in browser
go tool trace trace.trace
```

## Performance Optimization Workflow

1. **Profile First**: Start with CPU and memory profiling
   ```bash
   mcp-profile -cpu -mem -duration 60s go run ./server
   ```

2. **Identify Hotspots**: Use pprof to find performance bottlenecks
   ```bash
   go tool pprof cpu.prof
   ```

3. **Optimize Code**: Fix identified issues

4. **Validate Changes**: Compare before/after profiles
   ```bash
   mcp-profile -compare baseline.prof optimized.prof
   ```

5. **Load Test**: Validate under realistic load
   ```bash
   mcp-profile -cpu -mem -load-test -concurrency 20 go run ./server
   ```

6. **Monitor**: Set up continuous profiling
   ```bash
   mcp-profile -continuous -interval 5m go run ./server
   ```

## Best Practices

### Profiling Guidelines
1. **Profile in Production-like Environment**: Use realistic data and load
2. **Profile for Sufficient Duration**: 30-60 seconds for meaningful results
3. **Profile Under Load**: Use `-load-test` for realistic conditions
4. **Compare Profiles**: Use `-compare` to validate optimizations
5. **Continuous Monitoring**: Set up continuous profiling for production

### Memory Profiling
1. **Adjust Sampling Rate**: Lower rates for more detailed analysis
2. **Force GC**: Memory profiles include GC to get accurate heap state
3. **Look for Leaks**: Check for continuously growing allocations
4. **Analyze Allocation Patterns**: Use `go tool pprof -alloc_objects`

### CPU Profiling
1. **Sufficient Sampling**: Use at least 100Hz sampling rate
2. **Focus on Hot Paths**: Optimize functions with highest CPU usage
3. **Check Call Depth**: Deep call stacks may indicate inefficiency
4. **Measure Wall Clock Time**: Consider `-wall` flag for I/O heavy workloads

### Goroutine Profiling
1. **Check for Leaks**: Look for continuously growing goroutine count
2. **Analyze Blocking**: Use `-block` profiling for synchronization issues
3. **Monitor Patterns**: Look for goroutine creation/destruction patterns

## Troubleshooting

### Common Issues

**No Profile Data Generated**:
- Check if profiling duration is sufficient
- Verify server is receiving requests during profiling
- Ensure profiling types are enabled

**High Memory Usage**:
- Reduce memory sampling rate
- Use shorter profiling durations
- Check for memory leaks in the server

**Inaccurate Results**:
- Increase profiling duration
- Use load testing for realistic conditions
- Check for profiling overhead impact

### Performance Impact

**CPU Profiling**: 5-10% overhead
**Memory Profiling**: 1-2% overhead
**Goroutine Profiling**: Minimal overhead
**Blocking Profiling**: Minimal overhead
**Execution Tracing**: 10-20% overhead

### File Size Considerations

**CPU Profiles**: 1-10MB depending on duration and complexity
**Memory Profiles**: 1-50MB depending on allocation patterns
**Execution Traces**: 10-100MB+ depending on duration

## Integration with Monitoring

### Prometheus Integration
```bash
# Export profiling metrics to Prometheus format
mcp-profile -cpu -mem -export-prometheus go run ./server
```

### Grafana Dashboards
Use the generated metrics to create performance dashboards showing:
- CPU usage trends
- Memory allocation patterns
- Goroutine count over time
- GC pause frequency

### Alerting
Set up alerts for:
- High CPU usage (>80% for extended periods)
- Memory growth (indicating potential leaks)
- Goroutine count spikes
- Frequent GC pauses

## Development and Contributing

### Building from Source
```bash
cd cmd/mcp-profile
go build -o mcp-profile
```

### Running Tests
```bash
go test ./...
```

### Adding New Profile Types
1. Add flag for new profile type
2. Implement profile collection logic
3. Add analysis functions
4. Update documentation

## License

This tool is part of the MCP Go implementation and follows the same license terms.