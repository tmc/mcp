# Multi-stage build for MCP Go tools
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /src

# Copy go mod files first for better caching
COPY go.mod go.sum ./
COPY exp/go.mod exp/go.sum ./exp/
RUN go mod download
RUN cd exp && GOWORK=off go mod download

# Copy source code
COPY . .

# Build all core tools
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp-probe ./cmd/mcp-probe
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp-serve ./cmd/mcp-serve
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp-proxy ./cmd/mcp-proxy
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp-shadow ./cmd/mcp-shadow
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp-replay ./cmd/mcp-replay
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcpdiff ./cmd/mcpdiff
RUN cd exp && GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcpspy ./cmd/mcpspy

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -s /bin/sh mcp

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy built binaries
COPY --from=builder /bin/mcp-* /usr/local/bin/
COPY --from=builder /bin/mcp* /usr/local/bin/

# Create non-root user
USER mcp

# Set up working directory
WORKDIR /home/mcp

# Default command shows available tools
CMD ["sh", "-c", "echo 'MCP Go Tools Available:' && ls -1 /usr/local/bin/mcp* | sed 's|/usr/local/bin/||' && echo '' && echo 'Usage: docker run --rm ghcr.io/tmc/mcp <tool> [args...]'"]

# Labels for metadata
LABEL org.opencontainers.image.title="MCP Go Tools"
LABEL org.opencontainers.image.description="Production-ready Go implementation of the Model Context Protocol"
LABEL org.opencontainers.image.source="https://github.com/tmc/mcp"
LABEL org.opencontainers.image.vendor="tmc"
LABEL org.opencontainers.image.licenses="Apache-2.0"
