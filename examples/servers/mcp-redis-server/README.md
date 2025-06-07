# MCP Redis Server

A Model Context Protocol server that provides tools for interacting with Redis key-value stores.

## Features

- Get values from Redis keys
- Set values in Redis with optional TTL
- Delete Redis keys
- Check if keys exist
- List keys matching patterns

## Environment Variables

- `REDIS_HOST`: Redis server host (default: localhost)
- `REDIS_PORT`: Redis server port (default: 6379)
- `REDIS_DB`: Redis database number (default: 0)

## Tools

### redis_get
Get value from a Redis key.
- `key` (string, required): Redis key to retrieve

### redis_set
Set value in Redis.
- `key` (string, required): Redis key to set
- `value` (string, required): Value to store
- `ttl` (integer, optional): Time to live in seconds

### redis_delete
Delete a Redis key.
- `key` (string, required): Redis key to delete

### redis_exists
Check if a Redis key exists.
- `key` (string, required): Redis key to check

### redis_keys
List Redis keys matching pattern.
- `pattern` (string, optional): Pattern to match keys (default: *)

## Usage

```bash
go run main.go
```

## Building

```bash
go build -o mcp-redis-server .
```