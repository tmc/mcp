# mcp-deploy

Deployment automation with multi-platform support, rolling deployments, rollback capabilities, and environment promotion.

## Overview

`mcp-deploy` is a comprehensive deployment automation tool designed for production MCP deployments. It provides:

- **Multi-Platform Support**: Docker, Kubernetes, and serverless platforms
- **Deployment Strategies**: Rolling, blue-green, and canary deployments
- **Health Checking**: Automated health validation during deployments
- **Rollback Capabilities**: Automatic and manual rollback support
- **Environment Promotion**: Seamless promotion between environments
- **CI/CD Integration**: Integration with popular CI/CD systems
- **Monitoring**: Comprehensive deployment monitoring and alerting

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-deploy@latest
```

## Usage

### Initialize Deployment

```bash
# Initialize deployment configuration
mcp-deploy init

# Initialize for specific platform
mcp-deploy init --platform kubernetes
```

### Deploy Application

```bash
# Deploy to environment
mcp-deploy deploy development v1.2.3

# Deploy with specific strategy
mcp-deploy deploy --environment production --version v1.2.3 --strategy canary
```

### Rollback Deployment

```bash
# Rollback to previous version
mcp-deploy rollback production

# Rollback to specific revision
mcp-deploy rollback production --revision 5
```

### Check Status

```bash
# Show all deployments
mcp-deploy status

# Show specific deployment
mcp-deploy status deploy-12345

# Show environment status
mcp-deploy status --environment production
```

### Promote Between Environments

```bash
# Promote from staging to production
mcp-deploy promote --from staging --to production

# Promote specific version
mcp-deploy promote --from staging --to production --version v1.2.3
```

### Validate Configuration

```bash
# Validate deployment configuration
mcp-deploy validate deploy/mcp-deploy.yaml

# Validate environment configuration
mcp-deploy validate --environment production
```

### Scale Deployment

```bash
# Scale deployment
mcp-deploy scale production 5

# Auto-scale configuration
mcp-deploy scale production --auto --min 3 --max 10
```

### Canary Management

```bash
# Start canary deployment
mcp-deploy canary start production v1.2.3 --traffic 10

# Promote canary
mcp-deploy canary promote production

# Rollback canary
mcp-deploy canary rollback production
```

### Start Deployment Service

```bash
# Start API server
mcp-deploy serve --port 8080

# Start with custom configuration
mcp-deploy serve --config deploy/mcp-deploy.yaml
```

## Configuration

### Main Configuration File

```yaml
service_name: "mcp-service"
port: 8080
log_level: "info"

# Platform configuration
platform:
  type: "kubernetes"  # docker, kubernetes, serverless
  
  # Docker configuration
  docker:
    registry: "docker.io"
    repository: "mycompany/mcp-service"
    tag: "latest"
    dockerfile: "Dockerfile"
    context: "."
    build_args:
      GO_VERSION: "1.21"
    labels:
      maintainer: "team@company.com"
    networks:
      - "mcp-network"
    volumes:
      - "/data:/app/data"
    environment:
      - "ENV=production"
    ports:
      - "8080:8080"
    health_check:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: "30s"
      timeout: "10s"
      retries: 3
      start_period: "60s"
  
  # Kubernetes configuration
  kubernetes:
    kubeconfig: "~/.kube/config"
    namespace: "mcp-system"
    context: "production"
    manifests:
      - "deploy/k8s/deployment.yaml"
      - "deploy/k8s/service.yaml"
      - "deploy/k8s/ingress.yaml"
    helm_chart:
      chart: "mcp-service"
      repository: "https://charts.company.com"
      version: "1.0.0"
      values_file: "deploy/helm/values.yaml"
      values:
        image:
          tag: "v1.2.3"
        replicas: 3
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
    kustomize:
      dir: "deploy/k8s/overlays/production"
      images:
        - "mcp-service=mycompany/mcp-service:v1.2.3"
    resources:
      cpu: "500m"
      memory: "512Mi"
      storage: "1Gi"
    replicas: 3
    security_context:
      run_as_user: 1000
      run_as_group: 1000
      read_only_root: true
  
  # Serverless configuration
  serverless:
    provider: "aws"  # aws, gcp, azure
    function:
      name: "mcp-service"
      description: "MCP Service Lambda Function"
      runtime: "go1.x"
      handler: "main"
    timeout: "30s"
    memory: 512
    package: "deployment.zip"
    environment:
      LOG_LEVEL: "info"
      ENVIRONMENT: "production"
    vpc:
      security_groups:
        - "sg-12345678"
      subnets:
        - "subnet-12345678"
        - "subnet-87654321"
    iam:
      role: "arn:aws:iam::123456789012:role/lambda-execution-role"
      policies:
        - "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"

# Environment configurations
environments:
  development:
    name: "development"
    description: "Development environment"
    strategy: "rolling"
    replicas: 1
    resources:
      cpu: "100m"
      memory: "128Mi"
    config:
      LOG_LEVEL: "debug"
      DATABASE_URL: "postgresql://localhost:5432/mcp_dev"
    secrets:
      database_password: "dev-db-secret"
    ingress:
      enabled: true
      host: "mcp-dev.company.com"
      path: "/"
      tls: false
    monitoring:
      enabled: true
      prometheus:
        enabled: true
        namespace: "mcp-dev"
    
  staging:
    name: "staging"
    description: "Staging environment"
    strategy: "blue_green"
    replicas: 2
    resources:
      cpu: "200m"
      memory: "256Mi"
    config:
      LOG_LEVEL: "info"
      DATABASE_URL: "postgresql://staging-db:5432/mcp_staging"
    secrets:
      database_password: "staging-db-secret"
    ingress:
      enabled: true
      host: "mcp-staging.company.com"
      path: "/"
      tls: true
      certificate: "staging-tls-cert"
    monitoring:
      enabled: true
      prometheus:
        enabled: true
        namespace: "mcp-staging"
      alerting:
        enabled: true
        slack:
          enabled: true
          channel: "#alerts-staging"
    
  production:
    name: "production"
    description: "Production environment"
    strategy: "canary"
    replicas: 5
    resources:
      cpu: "500m"
      memory: "512Mi"
    config:
      LOG_LEVEL: "warn"
      DATABASE_URL: "postgresql://prod-db:5432/mcp_prod"
    secrets:
      database_password: "prod-db-secret"
    ingress:
      enabled: true
      host: "mcp.company.com"
      path: "/"
      tls: true
      certificate: "prod-tls-cert"
      annotations:
        kubernetes.io/ingress.class: "nginx"
        cert-manager.io/cluster-issuer: "letsencrypt-prod"
    auto_scale:
      enabled: true
      min_replicas: 5
      max_replicas: 20
      cpu_threshold: 80
      memory_threshold: 80
      scale_up_stabilization: "5m"
      scale_down_stabilization: "15m"
    monitoring:
      enabled: true
      prometheus:
        enabled: true
        namespace: "mcp-prod"
        labels:
          environment: "production"
      grafana:
        enabled: true
        dashboard: "mcp-production-dashboard"
      alerting:
        enabled: true
        webhook: "https://hooks.slack.com/services/..."
        slack:
          enabled: true
          channel: "#alerts-production"
        email:
          enabled: true
          to: ["oncall@company.com"]
        rules:
          - name: "deployment_failed"
            expression: "mcp_deployment_status == 0"
            duration: "2m"
            severity: "critical"
            description: "Deployment has failed"
          - name: "high_error_rate"
            expression: "mcp_error_rate > 0.05"
            duration: "5m"
            severity: "warning"
            description: "High error rate detected"

# Deployment strategies
strategies:
  rolling:
    type: "rolling"
    rolling:
      max_unavailable: "25%"
      max_surge: "25%"
      batch_size: 1
      pause: "30s"
    health_check:
      enabled: true
      path: "/health"
      port: 8080
      interval: "10s"
      timeout: "5s"
      healthy_threshold: 2
      unhealthy_threshold: 3
      initial_delay: "60s"
    timeout: "10m"
    rollback:
      enabled: true
      auto_rollback: true
      failure_threshold: 3
      timeout: "5m"
    
  blue_green:
    type: "blue_green"
    blue_green:
      traffic_split: 50
      test_traffic: 10
      promotion_delay: "5m"
      auto_promotion: false
      health_threshold: 90
    health_check:
      enabled: true
      path: "/health"
      port: 8080
      interval: "10s"
      timeout: "5s"
      healthy_threshold: 3
      unhealthy_threshold: 2
    timeout: "15m"
    rollback:
      enabled: true
      auto_rollback: true
      failure_threshold: 2
      timeout: "10m"
    
  canary:
    type: "canary"
    canary:
      traffic_percent: 10
      step_percent: 10
      step_duration: "5m"
      success_threshold: 95
      failure_threshold: 5
      auto_promotion: true
    health_check:
      enabled: true
      path: "/health"
      port: 8080
      interval: "10s"
      timeout: "5s"
      healthy_threshold: 2
      unhealthy_threshold: 3
    timeout: "30m"
    rollback:
      enabled: true
      auto_rollback: true
      failure_threshold: 3
      timeout: "15m"

# Global health check configuration
health_check:
  enabled: true
  path: "/health"
  port: 8080
  interval: "30s"
  timeout: "10s"
  healthy_threshold: 2
  unhealthy_threshold: 3
  initial_delay: "30s"

# Global rollback configuration
rollback:
  enabled: true
  auto_rollback: true
  failure_threshold: 3
  timeout: "10m"
  max_history: 10

# Monitoring configuration
monitoring:
  enabled: true
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    namespace: "mcp"
    labels:
      service: "mcp-service"
  grafana:
    enabled: true
    endpoint: "http://grafana:3000"
    dashboard: "mcp-deployment-dashboard"
  alerting:
    enabled: true
    webhook: "https://hooks.slack.com/services/..."
    rules:
      - name: "deployment_failed"
        expression: "mcp_deployment_status == 0"
        duration: "2m"
        severity: "critical"
        description: "Deployment has failed"
      - name: "deployment_slow"
        expression: "mcp_deployment_duration > 600"
        duration: "1m"
        severity: "warning"
        description: "Deployment is taking too long"
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268"
    service: "mcp-deploy"
  logging:
    enabled: true
    level: "info"
    format: "json"

# CI/CD integration
cicd:
  enabled: true
  provider: "github"  # github, gitlab, jenkins
  repository: "company/mcp-service"
  branch: "main"
  webhook: "https://api.github.com/repos/company/mcp-service/hooks"
  triggers:
    - type: "push"
      branch: "main"
      environment: "staging"
    - type: "tag"
      pattern: "v*"
      environment: "production"
  variables:
    DOCKER_REGISTRY: "docker.io"
    KUBE_NAMESPACE: "mcp-system"
  secrets:
    DOCKER_PASSWORD: "docker-registry-secret"
    KUBE_CONFIG: "kubeconfig-secret"
```

## Deployment Strategies

### Rolling Deployment

Rolling deployments gradually replace old instances with new ones:

```yaml
strategies:
  rolling:
    type: "rolling"
    rolling:
      max_unavailable: "25%"  # Max pods that can be unavailable
      max_surge: "25%"        # Max pods that can be created above desired
      batch_size: 2           # Number of pods to update at once
      pause: "30s"            # Pause between batches
    health_check:
      enabled: true
      path: "/health"
      healthy_threshold: 2
      unhealthy_threshold: 3
    timeout: "10m"
```

### Blue-Green Deployment

Blue-green deployments maintain two identical environments:

```yaml
strategies:
  blue_green:
    type: "blue_green"
    blue_green:
      traffic_split: 50       # Percentage of traffic to new version
      test_traffic: 10        # Percentage for testing
      promotion_delay: "5m"   # Delay before automatic promotion
      auto_promotion: false   # Manual promotion required
      health_threshold: 90    # Health percentage required for promotion
    timeout: "15m"
```

### Canary Deployment

Canary deployments gradually shift traffic to new versions:

```yaml
strategies:
  canary:
    type: "canary"
    canary:
      traffic_percent: 10     # Initial traffic percentage
      step_percent: 10        # Traffic increase per step
      step_duration: "5m"     # Duration of each step
      success_threshold: 95   # Success rate threshold
      failure_threshold: 5    # Failure rate threshold
      auto_promotion: true    # Automatic promotion on success
    timeout: "30m"
```

## Platform Support

### Docker Platform

```yaml
platform:
  type: "docker"
  docker:
    registry: "docker.io"
    repository: "mycompany/mcp-service"
    dockerfile: "Dockerfile"
    build_args:
      GO_VERSION: "1.21"
      CGO_ENABLED: "0"
    labels:
      maintainer: "team@company.com"
      version: "v1.2.3"
    health_check:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: "30s"
      timeout: "10s"
      retries: 3
```

**Deployment Process:**
1. Build Docker image with version tag
2. Push image to registry
3. Deploy containers with rolling update
4. Health check validation
5. Traffic routing

### Kubernetes Platform

```yaml
platform:
  type: "kubernetes"
  kubernetes:
    namespace: "mcp-system"
    
    # Raw manifests
    manifests:
      - "deploy/k8s/deployment.yaml"
      - "deploy/k8s/service.yaml"
      - "deploy/k8s/ingress.yaml"
    
    # Helm chart
    helm_chart:
      chart: "mcp-service"
      repository: "https://charts.company.com"
      values:
        image:
          tag: "v1.2.3"
        replicas: 3
    
    # Kustomize
    kustomize:
      dir: "deploy/k8s/overlays/production"
      images:
        - "mcp-service=mycompany/mcp-service:v1.2.3"
```

**Deployment Process:**
1. Apply Kubernetes manifests
2. Rolling update deployment
3. Service and ingress configuration
4. Health check validation
5. Traffic routing

### Serverless Platform

```yaml
platform:
  type: "serverless"
  serverless:
    provider: "aws"
    function:
      name: "mcp-service"
      runtime: "go1.x"
      handler: "main"
    timeout: "30s"
    memory: 512
    environment:
      LOG_LEVEL: "info"
    vpc:
      security_groups: ["sg-12345678"]
      subnets: ["subnet-12345678"]
```

**Deployment Process:**
1. Package function code
2. Deploy to serverless platform
3. Configure triggers and permissions
4. Health check validation
5. Traffic routing

## Environment Management

### Environment Configuration

```yaml
environments:
  development:
    name: "development"
    strategy: "rolling"
    replicas: 1
    resources:
      cpu: "100m"
      memory: "128Mi"
    config:
      LOG_LEVEL: "debug"
      DEBUG: "true"
    secrets:
      database_password: "dev-secret"
    
  production:
    name: "production"
    strategy: "canary"
    replicas: 5
    resources:
      cpu: "500m"
      memory: "512Mi"
    config:
      LOG_LEVEL: "warn"
      DEBUG: "false"
    secrets:
      database_password: "prod-secret"
    auto_scale:
      enabled: true
      min_replicas: 5
      max_replicas: 20
      cpu_threshold: 80
```

### Environment Promotion

```bash
# Promote from staging to production
mcp-deploy promote --from staging --to production

# Promote with validation
mcp-deploy promote --from staging --to production --validate

# Promote specific version
mcp-deploy promote --from staging --to production --version v1.2.3
```

## Health Checking

### Health Check Configuration

```yaml
health_check:
  enabled: true
  path: "/health"
  port: 8080
  interval: "30s"
  timeout: "10s"
  healthy_threshold: 2
  unhealthy_threshold: 3
  initial_delay: "30s"
```

### Custom Health Checks

```yaml
health_check:
  enabled: true
  custom_checks:
    - name: "database"
      type: "tcp"
      target: "database:5432"
      timeout: "5s"
    - name: "redis"
      type: "tcp"
      target: "redis:6379"
      timeout: "3s"
    - name: "api"
      type: "http"
      target: "http://api:8080/health"
      timeout: "10s"
      expected_status: 200
```

## Rollback Support

### Automatic Rollback

```yaml
rollback:
  enabled: true
  auto_rollback: true
  failure_threshold: 3     # Failed health checks to trigger rollback
  timeout: "10m"           # Timeout for rollback operation
  max_history: 10          # Maximum versions to keep for rollback
```

### Manual Rollback

```bash
# Rollback to previous version
mcp-deploy rollback production

# Rollback to specific revision
mcp-deploy rollback production --revision 5

# Rollback with validation
mcp-deploy rollback production --validate
```

## Monitoring and Alerting

### Prometheus Integration

```yaml
monitoring:
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    namespace: "mcp"
    labels:
      service: "mcp-service"
      environment: "production"
```

### Grafana Dashboards

```yaml
monitoring:
  grafana:
    enabled: true
    endpoint: "http://grafana:3000"
    dashboard: "mcp-deployment-dashboard"
```

### Alerting Rules

```yaml
monitoring:
  alerting:
    enabled: true
    rules:
      - name: "deployment_failed"
        expression: "mcp_deployment_status == 0"
        duration: "2m"
        severity: "critical"
        description: "Deployment has failed"
        
      - name: "deployment_slow"
        expression: "mcp_deployment_duration > 600"
        duration: "1m"
        severity: "warning"
        description: "Deployment is taking too long"
        
      - name: "high_error_rate"
        expression: "mcp_error_rate > 0.05"
        duration: "5m"
        severity: "warning"
        description: "High error rate detected during deployment"
```

## API Endpoints

### Deployment Management

```bash
# List all deployments
curl http://localhost:8080/deployments

# Get specific deployment
curl http://localhost:8080/deployments/deploy-12345

# Create new deployment
curl -X POST http://localhost:8080/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "environment": "production",
    "version": "v1.2.3"
  }'

# Rollback deployment
curl -X POST http://localhost:8080/rollback \
  -H "Content-Type: application/json" \
  -d '{
    "environment": "production",
    "revision": "5"
  }'
```

### Environment Management

```bash
# List environments
curl http://localhost:8080/environments

# Get environment configuration
curl http://localhost:8080/environments/production

# Update environment
curl -X PUT http://localhost:8080/environments/production \
  -H "Content-Type: application/json" \
  -d '{
    "replicas": 10,
    "resources": {
      "cpu": "1000m",
      "memory": "1Gi"
    }
  }'
```

### Status and Health

```bash
# Get deployment status
curl http://localhost:8080/status

# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Deploy MCP Service

on:
  push:
    branches: [main]
  release:
    types: [published]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.21
          
      - name: Install mcp-deploy
        run: go install github.com/tmc/mcp/cmd/mcp-deploy@latest
        
      - name: Deploy to staging
        if: github.ref == 'refs/heads/main'
        run: |
          mcp-deploy deploy staging ${{ github.sha }}
          
      - name: Deploy to production
        if: github.event_name == 'release'
        run: |
          mcp-deploy deploy production ${{ github.event.release.tag_name }}
```

### GitLab CI

```yaml
stages:
  - build
  - deploy

variables:
  DOCKER_REGISTRY: "registry.gitlab.com"
  DOCKER_IMAGE: "${DOCKER_REGISTRY}/${CI_PROJECT_PATH}"

deploy_staging:
  stage: deploy
  script:
    - mcp-deploy deploy staging ${CI_COMMIT_SHA}
  only:
    - main

deploy_production:
  stage: deploy
  script:
    - mcp-deploy deploy production ${CI_COMMIT_TAG}
  only:
    - tags
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    stages {
        stage('Build') {
            steps {
                sh 'go build -o mcp-service .'
            }
        }
        
        stage('Deploy to Staging') {
            when {
                branch 'main'
            }
            steps {
                sh "mcp-deploy deploy staging ${env.BUILD_NUMBER}"
            }
        }
        
        stage('Deploy to Production') {
            when {
                tag pattern: "v\\d+\\.\\d+\\.\\d+", comparator: "REGEXP"
            }
            steps {
                sh "mcp-deploy deploy production ${env.TAG_NAME}"
            }
        }
    }
}
```

## Examples

### Basic Kubernetes Deployment

```bash
# Initialize deployment
mcp-deploy init --platform kubernetes

# Configure environment
cat > deploy/mcp-deploy.yaml << EOF
service_name: "mcp-service"
platform:
  type: "kubernetes"
  kubernetes:
    namespace: "default"
    manifests:
      - "deploy/k8s/deployment.yaml"
      - "deploy/k8s/service.yaml"
environments:
  production:
    name: "production"
    strategy: "rolling"
    replicas: 3
EOF

# Create Kubernetes manifests
mkdir -p deploy/k8s

# Deploy
mcp-deploy deploy production v1.0.0
```

### Docker Compose Deployment

```bash
# Initialize for Docker
mcp-deploy init --platform docker

# Configure Docker deployment
cat > deploy/mcp-deploy.yaml << EOF
service_name: "mcp-service"
platform:
  type: "docker"
  docker:
    registry: "docker.io"
    repository: "mycompany/mcp-service"
    dockerfile: "Dockerfile"
environments:
  production:
    name: "production"
    strategy: "rolling"
    replicas: 2
EOF

# Deploy
mcp-deploy deploy production v1.0.0
```

### Serverless Deployment

```bash
# Initialize for AWS Lambda
mcp-deploy init --platform serverless

# Configure serverless deployment
cat > deploy/mcp-deploy.yaml << EOF
service_name: "mcp-service"
platform:
  type: "serverless"
  serverless:
    provider: "aws"
    function:
      name: "mcp-service"
      runtime: "go1.x"
      handler: "main"
    timeout: "30s"
    memory: 512
environments:
  production:
    name: "production"
    strategy: "rolling"
EOF

# Deploy
mcp-deploy deploy production v1.0.0
```

## Integration with MCP Ecosystem

### Health Integration

```go
// Integration with mcp-health
func DeploymentHealthMiddleware(deployer *Deployer) mcp.Middleware {
    return mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
        return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
            // Check deployment health before processing
            if !deployer.IsHealthy(ctx) {
                return nil, errors.New("deployment is not healthy")
            }
            return next.Handle(ctx, req)
        })
    })
}
```

### Configuration Integration

```go
// Integration with mcp-config
func DeploymentConfigMiddleware(configManager *ConfigManager) mcp.Middleware {
    return mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
        return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
            // Inject deployment configuration
            config := configManager.GetDeploymentConfig()
            ctx = context.WithValue(ctx, "deployment_config", config)
            
            return next.Handle(ctx, req)
        })
    })
}
```

## Best Practices

### Security

1. **Secret Management**: Use proper secret management systems
2. **RBAC**: Implement role-based access control
3. **Network Security**: Use network policies and security groups
4. **Image Security**: Scan container images for vulnerabilities
5. **Audit Logging**: Enable comprehensive audit logging

### Performance

1. **Resource Limits**: Set appropriate resource limits
2. **Auto-scaling**: Configure auto-scaling based on metrics
3. **Health Checks**: Implement comprehensive health checks
4. **Monitoring**: Set up comprehensive monitoring and alerting
5. **Optimization**: Optimize container images and configurations

### Reliability

1. **Rollback Strategy**: Always have a rollback strategy
2. **Testing**: Test deployments in staging environments
3. **Gradual Rollout**: Use canary or blue-green deployments
4. **Monitoring**: Monitor deployments closely
5. **Documentation**: Document deployment procedures

## Troubleshooting

### Common Issues

1. **Deployment Failures**: Check logs and health checks
2. **Resource Constraints**: Monitor resource usage
3. **Network Issues**: Check network policies and connectivity
4. **Configuration Errors**: Validate configuration files
5. **Permission Issues**: Check RBAC and service accounts

### Debugging Commands

```bash
# Check deployment status
mcp-deploy status

# View deployment logs
mcp-deploy logs deploy-12345

# Validate configuration
mcp-deploy validate deploy/mcp-deploy.yaml

# Check health
mcp-deploy health production

# Debug mode
mcp-deploy --log-level debug deploy production v1.0.0
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.