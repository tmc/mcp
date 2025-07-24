# mcp-config

Configuration management with environment-specific configs, secret management, validation, templates, and hot reloading.

## Overview

`mcp-config` is a comprehensive configuration management tool designed for production MCP deployments. It provides:

- **Environment Management**: Environment-specific configuration with inheritance
- **Secret Management**: Secure secret storage with multiple backends (Vault, K8s, file, env)
- **Template System**: Dynamic configuration generation with powerful templating
- **Validation**: Schema-based configuration validation with custom rules
- **Hot Reloading**: Real-time configuration updates without service restart
- **Audit Logging**: Complete audit trail of configuration changes
- **API Server**: RESTful API for configuration management

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-config@latest
```

## Usage

### Initialize Configuration Management

```bash
# Initialize with default structure
mcp-config init

# Initialize for specific environment
mcp-config init --environment production
```

### Validate Configuration

```bash
# Validate configuration file
mcp-config validate config/app.yaml

# Validate with custom schema
mcp-config validate --schema config/schema.yaml config/app.yaml
```

### Process Templates

```bash
# Process single template
mcp-config template config/app.yaml.tmpl config/app.yaml

# Process template directory
mcp-config template --input-dir templates --output-dir config
```

### Start Configuration Service

```bash
# Start API server
mcp-config serve --port 8080

# Start with custom configuration
mcp-config serve --config mcp-config.yaml
```

### Watch for Changes

```bash
# Watch configuration files
mcp-config watch --config mcp-config.yaml

# Watch specific paths
mcp-config watch --paths config/app.yaml,config/env --reload-command "systemctl reload mcp-server"
```

### Secret Management

```bash
# Get secret
mcp-config secret get database_password

# Set secret
mcp-config secret set database_password "new-password"

# List secrets
mcp-config secret list
```

### Environment Management

```bash
# List environments
mcp-config env list

# Show current environment
mcp-config env current

# Switch environment
mcp-config env switch production
```

## Configuration

### Main Configuration File

```yaml
service_name: "mcp-config"
port: 8080
log_level: "info"
environment: "development"

# Environment-specific configurations
environments:
  development:
    name: "development"
    description: "Development environment"
    config_paths:
      - "config/dev.yaml"
      - "config/common.yaml"
    secret_paths:
      - "secrets/dev"
    variables:
      log_level: "debug"
      database_host: "localhost"
      redis_host: "localhost"
      
  staging:
    name: "staging"
    description: "Staging environment"
    config_paths:
      - "config/staging.yaml"
      - "config/common.yaml"
    secret_paths:
      - "secrets/staging"
    variables:
      log_level: "info"
      database_host: "staging-db.internal"
      redis_host: "staging-redis.internal"
      
  production:
    name: "production"
    description: "Production environment"
    config_paths:
      - "config/prod.yaml"
      - "config/common.yaml"
    secret_paths:
      - "secrets/prod"
    variables:
      log_level: "warn"
      database_host: "prod-db.internal"
      redis_host: "prod-redis.internal"
    overrides:
      mcp:
        middleware:
          rate_limit:
            enabled: true
            requests_per_second: 1000

# Secret management configuration
secrets:
  backend: "vault"  # vault, k8s, file, env
  address: "https://vault.company.com"
  token: "${VAULT_TOKEN}"
  namespace: "mcp"
  paths:
    database_password: "secret/data/mcp/database"
    api_key: "secret/data/mcp/api"
    jwt_secret: "secret/data/mcp/jwt"
  encryption:
    enabled: true
    algorithm: "aes-256-gcm"
    key_env: "CONFIG_ENCRYPTION_KEY"

# Template system configuration
templates:
  enabled: true
  input_dir: "config/templates"
  output_dir: "config/generated"
  delimiters:
    left: "{{"
    right: "}}"
  variables:
    app_name: "mcp-service"
    version: "1.0.0"
    region: "us-west-2"
  functions:
    random_string: "github.com/tmc/mcp/internal/utils.RandomString"

# File watching configuration
watch:
  enabled: true
  paths:
    - "config/app.yaml"
    - "config/environments"
    - "config/templates"
  reload_command: "systemctl reload mcp-service"
  reload_delay: "5s"
  ignore_patterns:
    - "*.tmp"
    - "*.swp"
    - ".git/*"

# Validation configuration
validation:
  enabled: true
  schema_file: "config/schemas/app.schema.yaml"
  strict_mode: true
  rules:
    - name: "service_name_required"
      path: "service_name"
      type: "string"
      required: true
      min_length: 1
      max_length: 50
      pattern: "^[a-z0-9-]+$"
      
    - name: "port_range"
      path: "port"
      type: "number"
      required: true
      min_value: 1024
      max_value: 65535
      
    - name: "log_level_valid"
      path: "log_level"
      type: "string"
      required: true
      allowed_values: ["debug", "info", "warn", "error"]

# Audit logging configuration
audit:
  enabled: true
  log_file: "logs/config-audit.log"
  log_level: "info"
  format: "json"
```

### Environment-Specific Configuration

```yaml
# config/dev.yaml
mcp:
  server:
    name: "mcp-dev-server"
    port: 8080
    log_level: "debug"
  
  database:
    host: "{{.Env.database_host}}"
    port: 5432
    name: "mcp_dev"
    password: "{{secret \"database_password\"}}"
  
  redis:
    host: "{{.Env.redis_host}}"
    port: 6379
    
  middleware:
    rate_limit:
      enabled: false
    logging:
      enabled: true
      level: "debug"
```

```yaml
# config/prod.yaml
mcp:
  server:
    name: "mcp-prod-server"
    port: 8080
    log_level: "warn"
  
  database:
    host: "{{.Env.database_host}}"
    port: 5432
    name: "mcp_prod"
    password: "{{secret \"database_password\"}}"
    pool_size: 20
    max_connections: 100
  
  redis:
    host: "{{.Env.redis_host}}"
    port: 6379
    cluster: true
    
  middleware:
    rate_limit:
      enabled: true
      requests_per_second: 1000
      burst_size: 100
    logging:
      enabled: true
      level: "warn"
    security:
      enabled: true
      auth_required: true
```

## Template System

### Template Functions

The template system supports a rich set of functions:

```yaml
# config/app.yaml.tmpl
service_name: "{{.app_name}}"
version: "{{.version}}"
environment: "{{.environment}}"

# Environment variables
database_host: "{{env \"DATABASE_HOST\"}}"
redis_host: "{{env \"REDIS_HOST\" | default \"localhost\"}}"

# Secrets
database_password: "{{secret \"database_password\"}}"
api_key: "{{secret \"api_key\"}}"

# String manipulation
service_id: "{{.service_name | upper}}-{{.environment | upper}}"
config_file: "{{.service_name}}-{{.environment}}.yaml"

# Conditional logic
{{if eq .environment "production"}}
replicas: 3
{{else}}
replicas: 1
{{end}}

# Arrays and objects
allowed_hosts:
{{range .allowed_hosts}}
  - "{{.}}"
{{end}}

# Custom functions
session_secret: "{{random_string 32}}"
config_hash: "{{sha256 .config_content}}"
timestamp: "{{now | formatTime \"2006-01-02T15:04:05Z\"}}"
```

### Built-in Functions

- **env**: Get environment variable with optional default
- **secret**: Get secret value from secret backend
- **default**: Provide default value if empty
- **upper/lower/title**: String case transformation
- **replace**: String replacement
- **split/join**: String splitting and joining
- **trim**: Trim whitespace
- **base64**: Base64 encoding
- **sha256**: SHA256 hash
- **now**: Current timestamp
- **formatTime**: Format timestamp
- **random_string**: Generate random string

## Secret Management

### Vault Backend

```yaml
secrets:
  backend: "vault"
  address: "https://vault.company.com"
  token: "${VAULT_TOKEN}"
  namespace: "mcp"
  paths:
    database_password: "secret/data/mcp/database"
    api_key: "secret/data/mcp/api"
```

### Kubernetes Secrets Backend

```yaml
secrets:
  backend: "k8s"
  namespace: "mcp-system"
  paths:
    database_password: "mcp-secrets/database-password"
    api_key: "mcp-secrets/api-key"
```

### File Backend

```yaml
secrets:
  backend: "file"
  paths:
    database_password: "/etc/mcp/secrets/database_password"
    api_key: "/etc/mcp/secrets/api_key"
```

### Environment Variables Backend

```yaml
secrets:
  backend: "env"
  paths:
    database_password: "DATABASE_PASSWORD"
    api_key: "API_KEY"
```

## Configuration Validation

### Schema-Based Validation

```yaml
# config/schemas/app.schema.yaml
type: object
properties:
  service_name:
    type: string
    minLength: 1
    maxLength: 50
    pattern: "^[a-z0-9-]+$"
    
  port:
    type: integer
    minimum: 1024
    maximum: 65535
    
  log_level:
    type: string
    enum: ["debug", "info", "warn", "error"]
    
  mcp:
    type: object
    properties:
      server:
        type: object
        properties:
          name:
            type: string
          port:
            type: integer
        required: ["name", "port"]
        
required: ["service_name", "port", "log_level"]
```

### Custom Validation Rules

```yaml
validation:
  rules:
    - name: "port_not_reserved"
      path: "port"
      type: "number"
      custom_validator: "not_in_range"
      custom_params:
        ranges: [[1, 1023], [8080, 8080]]
        
    - name: "service_name_unique"
      path: "service_name"
      type: "string"
      custom_validator: "unique_service_name"
      custom_params:
        registry: "consul"
```

## API Endpoints

### Configuration Management

```bash
# Get current configuration
curl http://localhost:8080/config

# Validate configuration
curl -X POST http://localhost:8080/config/validate \
  -H "Content-Type: application/json" \
  -d '{"config_file": "config/app.yaml"}'

# Process template
curl -X POST http://localhost:8080/config/template \
  -H "Content-Type: application/json" \
  -d '{
    "input_file": "config/app.yaml.tmpl",
    "output_file": "config/app.yaml",
    "variables": {"version": "1.2.0"}
  }'

# Reload configuration
curl -X POST http://localhost:8080/config/reload
```

### Secret Management

```bash
# List secret keys
curl http://localhost:8080/secrets

# Get secret metadata (not value)
curl http://localhost:8080/secrets/database_password

# Set secret
curl -X POST http://localhost:8080/secrets/database_password \
  -H "Content-Type: application/json" \
  -d '{"value": "new-password"}'
```

### Environment Management

```bash
# List environments
curl http://localhost:8080/environments

# Get environment configuration
curl http://localhost:8080/environments/production

# Switch environment
curl -X POST http://localhost:8080/environments/production/activate
```

## Hot Reloading

### Configuration Watching

```yaml
watch:
  enabled: true
  paths:
    - "config/app.yaml"
    - "config/environments"
  reload_command: "systemctl reload mcp-service"
  reload_delay: "5s"
```

### Reload Strategies

1. **Signal-based**: Send SIGUSR1 to application
2. **Command-based**: Execute custom reload command
3. **API-based**: Call application reload endpoint
4. **File-based**: Touch reload trigger file

### Reload Hooks

```yaml
reload:
  pre_hooks:
    - "backup-config.sh"
    - "validate-config.sh"
  post_hooks:
    - "notify-team.sh"
    - "update-monitoring.sh"
```

## Docker Integration

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-config ./cmd/mcp-config

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/mcp-config /usr/local/bin/
COPY config/ /etc/mcp/
ENTRYPOINT ["mcp-config"]
CMD ["serve", "--config", "/etc/mcp/mcp-config.yaml"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  mcp-config:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config:/etc/mcp
      - ./secrets:/etc/mcp/secrets
    environment:
      - VAULT_TOKEN=${VAULT_TOKEN}
      - CONFIG_ENCRYPTION_KEY=${CONFIG_ENCRYPTION_KEY}
    depends_on:
      - vault
      - consul
      
  vault:
    image: vault:latest
    ports:
      - "8200:8200"
    environment:
      - VAULT_DEV_ROOT_TOKEN_ID=root
      
  consul:
    image: consul:latest
    ports:
      - "8500:8500"
```

## Kubernetes Integration

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-config
  namespace: mcp-system
data:
  mcp-config.yaml: |
    service_name: "mcp-config"
    port: 8080
    secrets:
      backend: "k8s"
      namespace: "mcp-system"
    watch:
      enabled: true
      paths:
        - "/etc/mcp/app.yaml"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-config
  namespace: mcp-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mcp-config
  template:
    metadata:
      labels:
        app: mcp-config
    spec:
      containers:
      - name: mcp-config
        image: mcp-config:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: mcp-config-secrets
              key: encryption-key
        volumeMounts:
        - name: config
          mountPath: /etc/mcp
        - name: secrets
          mountPath: /etc/mcp/secrets
      volumes:
      - name: config
        configMap:
          name: mcp-config
      - name: secrets
        secret:
          secretName: mcp-config-secrets
```

## Security

### Encryption

```yaml
secrets:
  encryption:
    enabled: true
    algorithm: "aes-256-gcm"
    key_file: "/etc/mcp/encryption.key"
    key_env: "CONFIG_ENCRYPTION_KEY"
```

### Access Control

```yaml
security:
  enabled: true
  auth_required: true
  api_keys:
    - name: "admin"
      key: "${ADMIN_API_KEY}"
      permissions: ["read", "write", "admin"]
    - name: "readonly"
      key: "${READONLY_API_KEY}"
      permissions: ["read"]
```

### Audit Logging

```yaml
audit:
  enabled: true
  log_file: "/var/log/mcp-config-audit.log"
  format: "json"
  events:
    - "config_read"
    - "config_write"
    - "secret_access"
    - "template_process"
```

## Integration with MCP Middleware

```go
// Custom configuration middleware
func ConfigMiddleware(configManager *ConfigManager) mcp.Middleware {
    return mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
        return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
            // Inject configuration into context
            config := configManager.GetConfig()
            ctx = context.WithValue(ctx, "config", config)
            
            return next.Handle(ctx, req)
        })
    })
}
```

## Examples

### Basic Setup

```bash
# Initialize configuration
mcp-config init

# Create environment-specific config
cat > config/dev.yaml << EOF
mcp:
  server:
    name: "mcp-dev-server"
    port: 8080
  database:
    host: "localhost"
    port: 5432
EOF

# Validate configuration
mcp-config validate config/dev.yaml

# Start service
mcp-config serve
```

### Template Processing

```bash
# Create template
cat > config/app.yaml.tmpl << EOF
service_name: "{{.app_name}}"
environment: "{{.environment}}"
database:
  host: "{{env "DATABASE_HOST"}}"
  password: "{{secret "database_password"}}"
EOF

# Process template
mcp-config template config/app.yaml.tmpl config/app.yaml

# Verify output
cat config/app.yaml
```

### Secret Management

```bash
# Set up Vault backend
export VAULT_TOKEN="your-vault-token"

# Set secret
mcp-config secret set database_password "secure-password"

# Get secret
mcp-config secret get database_password
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.