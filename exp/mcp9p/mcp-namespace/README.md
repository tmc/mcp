# MCP Namespace System

A Plan9-inspired namespace system for Model Context Protocol (MCP) services, providing hierarchical service discovery, mounting, and transparent access to local and remote MCP servers.

## Overview

The MCP namespace system consists of several components:

1. **mcp-namespace**: The namespace server that maintains the service registry
2. **mcp-ns**: Command-line client for namespace operations
3. **mcp-mount**: Tool for creating mounts and bindings (like Plan9 bind)
4. **mcp-fs**: FUSE filesystem that exposes the namespace as a filesystem
5. **mcp-tunnel**: Integration with namespace for automatic service discovery

## Architecture

```
                     ┌─────────────────┐
                     │  mcp-namespace  │
                     │    (server)     │
                     └────────┬────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼─────┐         ┌────▼─────┐         ┌────▼─────┐
   │  mcp-ns  │         │mcp-mount │         │  mcp-fs  │
   │ (client) │         │  (bind)  │         │  (FUSE)  │
   └──────────┘         └──────────┘         └──────────┘
                              │
                        ┌─────▼─────┐
                        │mcp-tunnel │
                        │(with ns)  │
                        └───────────┘
```

## Namespace Structure

The namespace follows a hierarchical structure similar to Plan9:

```
/
├── services/           # User services
│   ├── echo           # Echo service
│   ├── calculator     # Calculator service
│   └── ai/            # AI services
│       ├── gpt        # GPT service
│       └── claude     # Claude service
├── local/             # Local machine services
│   ├── fs             # Filesystem access
│   └── shell          # Shell access
└── remote/            # Remote services
    ├── server1/       # Services on server1
    └── cloud/         # Cloud services
```

## Usage Examples

### Starting the Namespace Server

```bash
# Start namespace server on default port 9000
mcp-namespace

# Start with custom port and persistence
mcp-namespace -addr :8080 -data /var/lib/mcp/namespace
```

### Registering Services

```bash
# Register a local stdio service
mcp-ns -c register /services/echo \
  -type local \
  -transport stdio \
  -command "npx" \
  -args "@modelcontextprotocol/server-echo,stdio"

# Register a remote HTTP service
mcp-ns -c register /remote/api/calculator \
  -type remote \
  -transport http \
  -address "https://api.example.com/mcp"

# Register with metadata
mcp-ns -c register /services/ai/gpt \
  -type remote \
  -transport http \
  -address "https://gpt.api/mcp" \
  -metadata "model=gpt-4,rate_limit=100"
```

### Looking Up Services

```bash
# Lookup a specific service
mcp-ns -c lookup /services/echo

# List all services in a directory
mcp-ns -c list /services

# List root namespace
mcp-ns -c list /
```

### Creating Mounts and Bindings

```bash
# Mount a service at another location
mcp-mount /services/echo /local/echo

# Create a binding (alias)
mcp-mount -type bind /services/ai/gpt /gpt

# Create a union mount
mcp-mount -type union /ai /services/ai/gpt /services/ai/claude

# Auto-mount with command
mcp-mount -type auto /services/time -- \
  npx @modelcontextprotocol/server-time stdio
```

### Using with mcp-tunnel

```bash
# Connect to a service via namespace
mcp-tunnel -- ns://localhost:9000/services/echo

# Create tunnel and register in namespace
mcp-tunnel \
  -namespace ns://localhost:9000/public/echo \
  -- npx @modelcontextprotocol/server-echo stdio

# Use custom namespace server
mcp-tunnel \
  -ns-server http://ns.example.com:9000 \
  -- ns://services/calculator
```

### Filesystem Access

```bash
# Mount namespace as filesystem
sudo mcp-fs -mount /mnt/mcp -ns http://localhost:9000

# Browse namespace
ls /mnt/mcp/services
cat /mnt/mcp/services/echo

# Use standard tools
find /mnt/mcp -name "*ai*"
grep -r "gpt" /mnt/mcp
```

## Advanced Features

### Namespace URIs

The system supports namespace URIs for easy service addressing:

```
ns://server/path
ns://localhost:9000/services/echo
ns://namespace.example.com/remote/api/calculator
```

### Service Types

- **local**: Services running on the local machine
- **remote**: Services accessible over network
- **namespace**: Nested namespace (directory)
- **mount**: Mount point referencing another location
- **bind**: Binding (alias) to another service
- **union**: Union of multiple services
- **tunneled**: Services exposed via mcp-tunnel

### Metadata

Services can have arbitrary metadata:

```json
{
  "name": "gpt",
  "type": "remote",
  "transport": "http",
  "address": "https://api.openai.com/mcp",
  "metadata": {
    "model": "gpt-4",
    "rate_limit": "100",
    "requires_auth": "true"
  }
}
```

### Persistence

The namespace server can persist its state:

```bash
# Start with persistence
mcp-namespace -data /var/lib/mcp/namespace

# State saved to namespace.json
cat /var/lib/mcp/namespace/namespace.json
```

## Integration with Existing MCP Tools

### mcp-serve

```bash
# Start a server and register it
mcp-serve -- command args &
mcp-ns -c register /services/myservice \
  -type local \
  -transport stdio \
  -metadata "pid=$!"
```

### mcp-proxy

```bash
# Proxy a namespace service
SERVICE=$(mcp-ns -c lookup /services/echo | jq -r .address)
mcp-proxy -v -- $SERVICE
```

### mcp-connect

```bash
# Connect using namespace resolution
ADDR=$(mcp-ns -c lookup /services/calculator | jq -r .address)
TRANSPORT=$(mcp-ns -c lookup /services/calculator | jq -r .transport)
mcp-connect -transport=$TRANSPORT -server=$ADDR
```

## Security Considerations

1. **Authentication**: The namespace server should implement authentication
2. **Authorization**: Access control for namespace modifications
3. **Encryption**: Use TLS for namespace server connections
4. **Isolation**: Separate namespaces for different users/projects

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mcp-namespace ./cmd/mcp-namespace

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/mcp-namespace /usr/local/bin/
EXPOSE 9000
CMD ["mcp-namespace"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-namespace
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mcp-namespace
  template:
    metadata:
      labels:
        app: mcp-namespace
    spec:
      containers:
      - name: mcp-namespace
        image: mcp-namespace:latest
        ports:
        - containerPort: 9000
        volumeMounts:
        - name: data
          mountPath: /data
        env:
        - name: MCP_NS_DATA
          value: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: mcp-namespace-data
```

## Future Enhancements

1. **9P Protocol**: Full Plan9 9P protocol support
2. **Distributed Namespace**: Federation of namespace servers
3. **Service Discovery**: Automatic service registration
4. **Health Checks**: Monitor service availability
5. **Load Balancing**: Distribute requests across instances
6. **Caching**: Client-side namespace caching
7. **Watch/Subscribe**: Real-time namespace updates
8. **ACLs**: Fine-grained access control

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - See LICENSE file for details