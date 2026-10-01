# Integration Guide for mcptrace-to-otel

This guide explains how to integrate mcptrace-to-otel into your MCP observability stack.

## Overview

mcptrace-to-otel bridges the gap between MCP's native trace format and the OpenTelemetry ecosystem, enabling you to:

- Visualize MCP interactions in distributed tracing systems
- Correlate MCP traces with other application traces
- Analyze performance and behavior patterns
- Debug complex MCP interactions

## Architecture

```
[MCP Client] <-> [MCP Server]
     |               |
     v               v
  [mcpspy]       [mcpspy]
     |               |
     v               v
[MCPTrace Files (.mcp)]
         |
         v
  [mcptrace-to-otel]
         |
    +----|----+----+----+
    |    |    |    |    |
    v    v    v    v    v
[Jaeger][Zipkin][Tempo][OTLP Collector]
         |
         v
   [Visualization UI]
```

## Integration Patterns

### 1. Real-time Tracing Pipeline

```bash
# Capture MCP traffic with trace context and export in real-time
mcpspy -trace -f - your-mcp-server | \
  mcptrace-to-otel -f - -type otlp-grpc -endpoint localhost:4317
```

### 2. Batch Processing

```bash
# Process trace files in batch
for trace in traces/*.mcp; do
  mcptrace-to-otel -f "$trace" \
    -type jaeger \
    -endpoint http://jaeger:14268/api/traces \
    -service "mcp-${trace%.mcp}"
done
```

### 3. Shadow Traffic Analysis

```bash
# Capture shadow traffic and analyze differences
mcp-shadow \
  -primary "mcp-server-v1" \
  -shadow "mcp-server-v2" \
  -trace -o shadow.mcp < requests.txt

# Export to tracing backend
mcptrace-to-otel -f shadow.mcp \
  -type otlp-grpc \
  -endpoint tempo:4317 \
  -service mcp-shadow-test
```

### 4. Kubernetes Integration

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mcp-trace-export
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trace-exporter
            image: your-registry/mcptrace-to-otel:latest
            command:
            - sh
            - -c
            - |
              for trace in /traces/*.mcp; do
                mcptrace-to-otel -f "$trace" \
                  -type otlp-grpc \
                  -endpoint otel-collector:4317
              done
            volumeMounts:
            - name: traces
              mountPath: /traces
          volumes:
          - name: traces
            persistentVolumeClaim:
              claimName: mcp-traces
```

## Correlation with Application Traces

### Using Trace Context Propagation

```go
// In your application
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// When starting MCP server
ctx := context.Background()
tracer := otel.Tracer("mcp-app")
ctx, span := tracer.Start(ctx, "mcp-session")
defer span.End()

// Get trace context
carrier := propagation.HeaderCarrier{}
propagator := otel.GetTextMapPropagator()
propagator.Inject(ctx, carrier)

// Pass to mcpspy
traceParent := carrier.Get("traceparent")
cmd := exec.Command("mcpspy", 
    "-trace-parent", traceParent,
    "-f", "session.mcp",
    "mcp-server")
```

### Environment Variables

```bash
# Set trace context in environment
export TRACEPARENT="00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
export TRACESTATE="vendor1=value1"

# mcpspy will use these automatically
mcpspy -trace -f app.mcp mcp-server
```

## Advanced Configurations

### Custom Span Attributes

```bash
# Add custom service attributes
mcptrace-to-otel -f trace.mcp \
  -type otlp-grpc \
  -endpoint localhost:4317 \
  -service mcp-custom \
  -attr "deployment.environment=staging" \
  -attr "service.version=2.1.0"
```

### Sampling Strategies

```bash
# Only export traces with errors
mcptrace-to-otel -f trace.mcp \
  -type jaeger \
  -endpoint http://localhost:14268/api/traces \
  -filter "error=true"

# Sample 10% of traces
mcptrace-to-otel -f trace.mcp \
  -type otlp-grpc \
  -endpoint localhost:4317 \
  -sample-rate 0.1
```

### Multi-Backend Export

```bash
# Export to multiple backends
mcptrace-to-otel -f trace.mcp \
  -type multi \
  -endpoints "jaeger=http://localhost:14268/api/traces,zipkin=http://localhost:9411/api/v2/spans"
```

## Monitoring & Alerting

### Prometheus Metrics

mcptrace-to-otel can expose metrics:

```bash
mcptrace-to-otel -f trace.mcp \
  -type otlp-grpc \
  -endpoint localhost:4317 \
  -metrics-port 9090
```

Metrics available:
- `mcptrace_spans_exported_total`
- `mcptrace_export_errors_total`
- `mcptrace_export_duration_seconds`

### Grafana Dashboard

Example query for MCP performance:

```promql
# Average response time by method
avg by (method) (
  rate(mcp_request_duration_seconds_sum[5m]) /
  rate(mcp_request_duration_seconds_count[5m])
)

# Error rate
sum(rate(mcp_errors_total[5m])) by (method, error_code)
```

## Best Practices

1. **Trace Context**: Always generate trace context at the edge of your system
2. **Sampling**: Use appropriate sampling rates to balance observability and cost
3. **Retention**: Configure trace retention based on your debugging needs
4. **Attributes**: Include relevant attributes for filtering and analysis
5. **Security**: Don't include sensitive data in traces

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Check if collector is running
   curl http://localhost:4317/v1/health
   ```

2. **Invalid Trace Context**
   ```bash
   # Validate trace context format
   mcptrace-to-otel -f trace.mcp -validate-only
   ```

3. **Memory Issues**
   ```bash
   # Process large files in chunks
   mcptrace-to-otel -f large.mcp -chunk-size 1000
   ```

## Integration Examples

### with Datadog

```bash
export DD_TRACE_AGENT_URL="http://localhost:8126"
mcptrace-to-otel -f trace.mcp \
  -type otlp-http \
  -endpoint "$DD_TRACE_AGENT_URL/v0.4/traces"
```

### with New Relic

```bash
export NEW_RELIC_API_KEY="your-api-key"
mcptrace-to-otel -f trace.mcp \
  -type otlp-grpc \
  -endpoint otlp.nr-data.net:4317 \
  -headers "api-key=$NEW_RELIC_API_KEY"
```

### with AWS X-Ray

```bash
# Use OTEL collector with X-Ray exporter
mcptrace-to-otel -f trace.mcp \
  -type otlp-grpc \
  -endpoint localhost:4317
```

## Next Steps

1. Set up automated trace export pipelines
2. Create custom dashboards for MCP metrics
3. Implement alerting based on trace data
4. Correlate MCP traces with application logs
5. Use trace data for performance optimization