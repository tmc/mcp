# mcptrace-to-otel

Converts MCPTrace files to OpenTelemetry format for visualization in distributed tracing systems like Jaeger, Zipkin, or Grafana Tempo.

## Features

- Converts MCPTrace format to OpenTelemetry spans
- Preserves trace context from MCPTrace headers
- Supports multiple export formats:
  - OTLP (gRPC and HTTP)
  - Jaeger
  - Zipkin
  - Stdout (for debugging)
- Handles shadow/linked spans as events
- Preserves baggage and custom attributes
- Maintains timing information and span relationships

## Installation

```bash
cd cmd/mcptrace-to-otel
go install .
```

## Usage

### Basic Usage

```bash
# Export to stdout (for debugging)
mcptrace-to-otel -f trace.mcp

# Export to Jaeger
mcptrace-to-otel -f trace.mcp -type jaeger -endpoint http://localhost:14268/api/traces

# Export to Zipkin
mcptrace-to-otel -f trace.mcp -type zipkin -endpoint http://localhost:9411/api/v2/spans

# Export to OTLP gRPC (e.g., for Grafana Tempo)
mcptrace-to-otel -f trace.mcp -type otlp-grpc -endpoint localhost:4317

# Export to OTLP HTTP
mcptrace-to-otel -f trace.mcp -type otlp-http -endpoint localhost:4318
```

### Options

- `-f`: Input MCPTrace file (required)
- `-type`: Output type: `stdout`, `otlp-grpc`, `otlp-http`, `jaeger`, `zipkin` (default: `stdout`)
- `-endpoint`: Endpoint for exporter (required for non-stdout types)
- `-service`: Service name for traces (default: `mcp-trace`)
- `-v`: Verbose output
- `-batch`: Batch size for exports (default: 100)
- `-timeout`: Export timeout (default: 10s)
- `-insecure`: Use insecure connection for OTLP

## OpenTelemetry Mapping

MCPTrace elements are mapped to OpenTelemetry as follows:

### Trace Context
- MCPTrace header `traceparent` → OpenTelemetry trace context
- MCPTrace header `baggage` → Span attributes with `mcp.baggage.*` prefix

### Spans
- Each MCP message becomes a span
- Span name: `mcp.recv.{method}` or `mcp.send.{method}`
- Span kind: 
  - `recv` → `SpanKindServer`
  - `send` → `SpanKindClient`

### Attributes
- `mcp.direction`: "recv" or "send"
- `mcp.timestamp`: Original timestamp
- `rpc.jsonrpc.version`: JSON-RPC version
- `rpc.method`: Method name
- `rpc.jsonrpc.request_id`: Request ID
- `mcp.span_id`: Original span ID from trace
- `mcp.links_to`: Link to another span (for shadow responses)
- `mcp.baggage.*`: Baggage key-value pairs

### Shadow/Linked Spans
Shadow responses (from mcp-shadow) are added as events on the primary span:
- Event name: `shadow_send` or `shadow_recv`
- Event attributes include shadow response details and baggage

## Examples

### 1. Simple Trace Export

```bash
# Create a trace
echo '{"jsonrpc":"2.0","method":"test","id":1}' | mcpspy -trace -f test.mcp cat

# Export to Jaeger
mcptrace-to-otel -f test.mcp -type jaeger -endpoint http://localhost:14268/api/traces
```

### 2. Shadow Trace Analysis

```bash
# Create shadow trace
echo '{"jsonrpc":"2.0","method":"test","id":1}' | \
  mcp-shadow -primary "cat" -shadow "jq '.shadow=true'" -trace -o shadow.mcp

# Export to OpenTelemetry
mcptrace-to-otel -f shadow.mcp -type otlp-grpc -endpoint localhost:4317

# View in Jaeger UI at http://localhost:16686
```

### 3. Pipeline Processing

```bash
# Record trace → Normalize timestamps → Export to OTEL
mcpspy -trace -f raw.mcp some-mcp-server
mcp-tsnorm -start 0s -o normalized.mcp raw.mcp
mcptrace-to-otel -f normalized.mcp -type jaeger -endpoint http://localhost:14268/api/traces
```

### 4. Debug Export

```bash
# Export to stdout to see the span structure
mcptrace-to-otel -f trace.mcp -type stdout -v
```

## Docker Compose Example

```yaml
version: '3'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14268:14268"  # HTTP collector
    environment:
      - COLLECTOR_ZIPKIN_HTTP_PORT=9411

  # Convert and export traces
  mcptrace-exporter:
    build: .
    volumes:
      - ./traces:/traces
    command: >
      sh -c "
      for trace in /traces/*.mcp; do
        mcptrace-to-otel -f $$trace -type jaeger -endpoint http://jaeger:14268/api/traces
      done
      "
```

## Trace Visualization

After exporting to a tracing backend, you can:

1. **View trace timeline**: See the sequence of MCP messages
2. **Analyze latencies**: Measure time between request and response
3. **Compare shadow responses**: View primary and shadow responses side-by-side
4. **Filter by attributes**: Search for specific methods, IDs, or baggage values
5. **Trace dependencies**: Understand the flow of MCP communications

## Integration with MCP Tools

Works seamlessly with other MCP tools:

- Use after `mcpspy` to visualize recorded traces
- Use after `mcp-shadow` to compare primary and shadow responses
- Use after `mcp-tsnorm` to work with normalized timestamps
- Combine with `mcp-replay` for trace analysis

## See Also

- `mcpspy`: Records MCP interactions
- `mcp-shadow`: Creates shadow traces
- `mcp-tsnorm`: Normalizes timestamps
- [OpenTelemetry](https://opentelemetry.io/)
- [Jaeger](https://www.jaegertracing.io/)
- [Zipkin](https://zipkin.io/)