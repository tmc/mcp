# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o mcp-tunnel-server cmd/mcp-tunnel/server/main.go

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/mcp-tunnel-server /usr/local/bin/mcp-tunnel-server

# Switch to non-root user
USER appuser

# Set the PORT environment variable (Cloud Run requirement)
ENV PORT=8080

# Expose the port
EXPOSE 8080

# Run the application
CMD ["mcp-tunnel-server"]