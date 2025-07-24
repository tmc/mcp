# mcp-bench: Comprehensive Performance Testing Tool

`mcp-bench` is a comprehensive performance testing tool for MCP (Model Context Protocol) servers. It provides load testing, stress testing, latency analysis, throughput measurement, and resource monitoring capabilities.

## Features

### Core Testing Capabilities
- **Load Testing**: Sustained load with configurable concurrency and duration
- **Stress Testing**: Gradual load increase to find breaking points
- **Spike Testing**: Sudden load bursts to test resilience
- **Endurance Testing**: Extended duration testing for stability analysis

### Performance Metrics
- Request latency statistics (min, max, mean, median, P90, P95, P99, P99.9)
- Throughput measurement (requests per second)
- Error rate tracking and categorization
- Resource utilization monitoring (CPU, memory, goroutines)
- Real-time performance visualization

### Advanced Features
- **Profiling Integration**: CPU and memory profiling with pprof
- **Export Capabilities**: Prometheus metrics, JMeter plans, k6 scripts
- **Distributed Testing**: Multi-node load generation (experimental)
- **Real-time Monitoring**: Live performance dashboard
- **Rate Limiting**: Configurable request rate control

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-bench@latest
```

Or build from source:

```bash
cd cmd/mcp-bench
go build -o mcp-bench
```

## Usage

### Basic Load Testing

```bash
# Basic load test with 10 concurrent clients for 30 seconds
mcp-bench -c 10 -d 30s go run ./examples/servers/mcp-time-server

# Test specific tool with custom arguments
mcp-bench -c 5 -tool "get_time" -tool-args '{"timezone":"UTC"}' go run ./server

# Run with verbose output and save results
mcp-bench -v -output results.json -c 20 -d 60s go run ./server
```

### Stress Testing

```bash
# Automatically scale from 1 to 100 concurrent clients
mcp-bench -stress-test -c 100 go run ./server

# Spike test with sudden load increases
mcp-bench -spike-test -c 50 go run ./server

# Long-running endurance test
mcp-bench -endurance-test -c 10 -d 24h go run ./server
```

### Performance Profiling

```bash
# Enable CPU and memory profiling
mcp-bench -profile -c 20 -d 60s go run ./server

# Specific profiling options
mcp-bench -cpu-profile -mem-profile -profile-dir ./profiles go run ./server

# Enable execution tracing
mcp-bench -trace execution.trace -c 10 -d 30s go run ./server
```

### Real-time Monitoring

```bash
# Enable real-time dashboard
mcp-bench -realtime -c 20 -d 300s go run ./server

# With rate limiting
mcp-bench -rate-limit 50 -c 10 -d 60s go run ./server
```

### Export Capabilities

```bash
# Export to Prometheus format
mcp-bench -export-prometheus -c 10 -d 60s go run ./server

# Export JMeter test plan
mcp-bench -export-jmeter -c 20 -d 120s go run ./server

# Export k6 script
mcp-bench -export-k6 -c 15 -d 90s go run ./server
```

### Transport Options

```bash
# HTTP transport
mcp-bench -transport http -http-url http://localhost:8080/mcp -c 10 -d 30s

# Server-Sent Events (SSE)
mcp-bench -transport sse -sse-url http://localhost:8080/sse -c 5 -d 60s

# Default stdio transport
mcp-bench -transport stdio -c 10 -d 30s go run ./server
```

## Command Line Options

### Core Options
- `-c, -concurrency`: Number of concurrent clients (default: 1)
- `-r, -requests`: Total requests per client (default: 100)
- `-d, -duration`: Test duration (default: 30s)
- `-warmup`: Warmup period (default: 5s)
- `-cooldown`: Cooldown period (default: 2s)

### Test Types
- `-load-test`: Run load test (default)
- `-stress-test`: Run stress test with scaling
- `-spike-test`: Run spike test
- `-endurance-test`: Run endurance test

### Tool Selection
- `-tool`: Specific tool to test (tests all if empty)
- `-tool-args`: JSON arguments for the tool (default: "{}")

### Output Options
- `-output`: Output file for results (JSON format)
- `-v, -verbose`: Verbose output
- `-q, -quiet`: Quiet mode
- `-realtime`: Real-time monitoring

### Profiling Options
- `-profile`: Enable CPU and memory profiling
- `-cpu-profile`: Enable CPU profiling only
- `-mem-profile`: Enable memory profiling only
- `-profile-dir`: Directory for profile outputs (default: "./profiles")
- `-trace`: Enable execution tracing to file

### Export Options
- `-export-prometheus`: Export Prometheus metrics
- `-export-jmeter`: Export JMeter test plan
- `-export-k6`: Export k6 script

### Rate Control
- `-rate-limit`: Rate limit in requests/second (0 = no limit)
- `-throttle`: Throttle delay between requests

### Transport Options
- `-transport`: Transport type (stdio, http, sse)
- `-http-url`: HTTP URL for HTTP transport
- `-sse-url`: SSE URL for SSE transport
- `-timeout`: Request timeout (default: 10s)

### Monitoring
- `-monitor-interval`: Monitoring interval (default: 1s)
- `-metrics-port`: Port for metrics HTTP server (default: 8080)

## Output Format

### Console Output
```
=== Benchmark Results ===
Test Type: load
Duration: 30.045s
Concurrency: 10

Requests:
  Total: 15420
  Successful: 15389
  Failed: 31
  Requests/sec: 512.84
  Error Rate: 0.20%

Latency:
  Min: 1.234ms
  Max: 89.567ms
  Mean: 18.456ms
  Median: 16.234ms
  P90: 28.567ms
  P95: 34.123ms
  P99: 45.678ms
  P99.9: 67.891ms

Resource Usage:
  Memory: 45123456 bytes
  Goroutines: 25
  GC Pauses: 12
```

### JSON Output
```json
{
  "config": {
    "concurrency": 10,
    "duration": "30s",
    "tool": "get_time",
    "transport": "stdio",
    "testType": "load"
  },
  "startTime": "2024-01-15T10:30:00Z",
  "endTime": "2024-01-15T10:30:30Z",
  "duration": "30.045s",
  "totalRequests": 15420,
  "successfulRequests": 15389,
  "failedRequests": 31,
  "requestsPerSecond": 512.84,
  "latencyStats": {
    "min": "1.234ms",
    "max": "89.567ms",
    "mean": "18.456ms",
    "median": "16.234ms",
    "p90": "28.567ms",
    "p95": "34.123ms",
    "p99": "45.678ms",
    "p999": "67.891ms"
  },
  "errors": [
    {
      "error": "context deadline exceeded",
      "count": 31,
      "firstSeen": "2024-01-15T10:30:05Z",
      "lastSeen": "2024-01-15T10:30:28Z"
    }
  ],
  "timeline": [...],
  "resourceStats": {...}
}
```

## Performance Analysis

### Interpreting Results

**Request Metrics:**
- **Requests/sec**: Higher is better, indicates throughput capacity
- **Error Rate**: Lower is better, should be <1% for healthy systems
- **Success Rate**: Should be >99% for production systems

**Latency Metrics:**
- **P95 Latency**: 95% of requests complete within this time
- **P99 Latency**: Critical for user experience, should be <100ms for interactive systems
- **Mean vs Median**: Large differences indicate latency variance

**Resource Usage:**
- **Memory**: Monitor for memory leaks during long tests
- **Goroutines**: Should remain stable, growth indicates leaks
- **GC Pauses**: High count may indicate memory pressure

### Performance Recommendations

1. **Optimize for P95/P99 latency** rather than just mean latency
2. **Monitor error rates** during load increases
3. **Use stress testing** to find system limits
4. **Profile during peak load** to identify bottlenecks
5. **Test with realistic request patterns** using appropriate tool arguments

## Integration with Other Tools

### Prometheus Integration
```bash
# Export metrics and scrape with Prometheus
mcp-bench -export-prometheus -c 20 -d 300s go run ./server
# Configure Prometheus to scrape metrics from file or HTTP endpoint
```

### JMeter Integration
```bash
# Generate JMeter test plan
mcp-bench -export-jmeter -c 50 -d 600s go run ./server
# Import generated .jmx file into JMeter GUI
```

### k6 Integration
```bash
# Generate k6 script
mcp-bench -export-k6 -c 30 -d 180s go run ./server
# Run with k6: k6 run k6_script.js
```

## Troubleshooting

### Common Issues

**High Error Rates:**
- Check server logs for specific error messages
- Increase timeout values with `-timeout`
- Reduce concurrency to identify capacity limits

**Memory Issues:**
- Enable memory profiling with `-mem-profile`
- Use `-endurance-test` to identify memory leaks
- Monitor GC statistics in output

**Inconsistent Results:**
- Use longer warmup periods with `-warmup`
- Run multiple tests and average results
- Check for system resource contention

### Performance Optimization

**For Better Throughput:**
- Increase concurrency gradually
- Use connection pooling in your server
- Optimize serialization/deserialization

**For Better Latency:**
- Reduce request processing time
- Use async processing where possible
- Minimize memory allocations

## Development

### Building from Source
```bash
cd cmd/mcp-bench
go build -o mcp-bench
```

### Running Tests
```bash
go test ./...
```

### Contributing
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

This tool is part of the MCP Go implementation and follows the same license terms.