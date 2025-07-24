# mcp-health

Health checking, service discovery, and cluster management for MCP services.

## Overview

`mcp-health` is a comprehensive health monitoring and service discovery tool designed for production MCP deployments. It provides:

- **Health Checking**: HTTP, TCP, and native MCP protocol health checks
- **Service Discovery**: Integration with Consul, etcd, and Kubernetes
- **Load Balancing**: Intelligent routing based on health status
- **Monitoring**: Prometheus metrics and alerting integration
- **Cluster Management**: Kubernetes operator for cloud-native deployments

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-health@latest
```

## Usage

### Basic Health Check

```bash
# Check a single service
mcp-health check --target localhost:8080

# Check with specific protocol
mcp-health check --target localhost:8080 --protocol mcp
```

### Continuous Monitoring

```bash
# Start monitoring with configuration file
mcp-health monitor --config health-config.yaml

# Start monitoring with basic settings
mcp-health monitor --target localhost:8080 --interval 30s
```

### Health Service API

```bash
# Start the health service API server
mcp-health serve --port 8080 --config health-config.yaml

# With default configuration
mcp-health serve --target localhost:8080
```

### Kubernetes Operator

```bash
# Run in Kubernetes operator mode
mcp-health operator --kubeconfig ~/.kube/config
```

## Configuration

### Health Check Configuration

```yaml
service_name: "mcp-health"
port: 8080
log_level: "info"

health_checks:
  - name: "api-server"
    target: "localhost:8080"
    protocol: "http"
    interval: "30s"
    timeout: "5s"
    failure_threshold: 3
    success_threshold: 1
    http_path: "/health"
    
  - name: "mcp-server"
    target: "localhost:9090"
    protocol: "mcp"
    interval: "60s"
    timeout: "10s"
    failure_threshold: 2
    success_threshold: 1
    mcp_method: "tools/list"

metrics:
  enabled: true
  path: "/metrics"
  prometheus:
    enabled: true
    address: "localhost:9090"
    job_name: "mcp-health"
```

### Service Discovery Configuration

```yaml
discovery:
  enabled: true
  backend: "consul"  # consul, etcd, k8s
  address: "localhost:8500"
  namespace: "mcp"

load_balancer:
  strategy: "round_robin"  # round_robin, least_conn, weighted
  weights:
    service1: 100
    service2: 50
```

### Kubernetes Integration

```yaml
kubernetes:
  enabled: true
  kubeconfig: "~/.kube/config"
  namespace: "mcp-system"
  service_name: "mcp-health"
  operator_mode: true
```

### Alerting Configuration

```yaml
alerting:
  enabled: true
  webhook: "https://hooks.slack.com/services/..."
  
  slack:
    enabled: true
    token: "xoxb-..."
    channel: "#alerts"
    
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    from: "alerts@company.com"
    to: ["admin@company.com"]
    
  prometheus:
    enabled: true
    address: "localhost:9090"
    job_name: "mcp-health"
```

## API Endpoints

### Health Status

```bash
# Get all service health status
curl http://localhost:8080/health

# Get specific service health
curl http://localhost:8080/health/api-server

# Readiness probe (returns 200 if all services are ready)
curl http://localhost:8080/readiness

# Liveness probe (always returns 200)
curl http://localhost:8080/liveness
```

### Service Discovery

```bash
# List all discovered services
curl http://localhost:8080/services

# Get service discovery info
curl http://localhost:8080/services/api-server
```

### Metrics

```bash
# Get Prometheus metrics
curl http://localhost:8080/metrics
```

## Health Check Protocols

### HTTP Health Checks

```yaml
- name: "http-service"
  target: "localhost:8080"
  protocol: "http"
  http_path: "/health"
  expected_status: 200
  headers:
    Authorization: "Bearer token123"
```

### TCP Health Checks

```yaml
- name: "tcp-service"
  target: "localhost:5432"
  protocol: "tcp"
  timeout: "5s"
```

### MCP Health Checks

```yaml
- name: "mcp-service"
  target: "localhost:9090"
  protocol: "mcp"
  mcp_method: "tools/list"  # tools/list, resources/list, ping
  timeout: "10s"
```

## Service Discovery

### Consul Integration

```yaml
discovery:
  enabled: true
  backend: "consul"
  address: "localhost:8500"
  namespace: "mcp"
```

### Kubernetes Integration

```yaml
discovery:
  enabled: true
  backend: "k8s"
  namespace: "mcp-system"
```

### etcd Integration

```yaml
discovery:
  enabled: true
  backend: "etcd"
  address: "localhost:2379"
  namespace: "/mcp/services"
```

## Load Balancing

### Round Robin

```yaml
load_balancer:
  strategy: "round_robin"
```

### Least Connections

```yaml
load_balancer:
  strategy: "least_conn"
```

### Weighted

```yaml
load_balancer:
  strategy: "weighted"
  weights:
    server1: 100
    server2: 50
    server3: 25
```

## Monitoring and Alerting

### Prometheus Metrics

The service exposes the following metrics:

- `mcp_health_check{service, check, status}` - Health check results
- `mcp_health_check_duration_seconds{service, check}` - Check duration
- `mcp_health_service_status{service, status}` - Overall service status
- `mcp_health_checks_total{service}` - Total number of checks

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: mcp-health
    rules:
      - alert: MCPServiceDown
        expr: mcp_health_check == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "MCP service {{ $labels.service }} is down"
          
      - alert: MCPServiceDegraded
        expr: mcp_health_service_status{status="degraded"} == 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "MCP service {{ $labels.service }} is degraded"
```

## Kubernetes Operator

### Installation

```bash
# Apply CRDs
kubectl apply -f deploy/crds/

# Deploy operator
kubectl apply -f deploy/operator.yaml
```

### Custom Resource Definition

```yaml
apiVersion: mcp.tmc.dev/v1alpha1
kind: MCPHealthCheck
metadata:
  name: api-server-health
  namespace: mcp-system
spec:
  target: "api-server:8080"
  protocol: "http"
  interval: "30s"
  timeout: "5s"
  httpPath: "/health"
  alerting:
    enabled: true
    webhook: "https://hooks.slack.com/services/..."
```

## Examples

### Basic Setup

```bash
# Start health checking for local MCP server
mcp-health serve --target localhost:8080 --port 9090
```

### Production Setup

```bash
# Start with comprehensive configuration
mcp-health serve --config production-health.yaml
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-health ./cmd/mcp-health

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/mcp-health /usr/local/bin/
ENTRYPOINT ["mcp-health"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-health
  namespace: mcp-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mcp-health
  template:
    metadata:
      labels:
        app: mcp-health
    spec:
      containers:
      - name: mcp-health
        image: mcp-health:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_FILE
          value: "/etc/mcp-health/config.yaml"
        volumeMounts:
        - name: config
          mountPath: /etc/mcp-health
      volumes:
      - name: config
        configMap:
          name: mcp-health-config
```

## Integration with Existing Systems

### Existing Middleware

`mcp-health` integrates with the existing MCP middleware system:

```go
// Custom health check middleware
func HealthCheckMiddleware(checker *HealthChecker) mcp.Middleware {
    return mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
        return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
            // Perform health check before handling request
            if !checker.IsHealthy(ctx) {
                return nil, errors.New("service is not healthy")
            }
            return next.Handle(ctx, req)
        })
    })
}
```

### Transport Layer

Works with all existing MCP transports:

- **stdio**: Direct process health checking
- **HTTP/SSE**: HTTP-based health endpoints
- **WebSocket**: WebSocket connection health

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.