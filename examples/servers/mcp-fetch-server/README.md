# MCP Fetch Server

A Model Context Protocol (MCP) server that provides web content fetching capabilities with security controls.

## Features

- **URL Fetching**: Download content from HTTP/HTTPS URLs
- **HTML to Text**: Convert HTML content to clean plain text
- **Header Inspection**: Get HTTP headers without downloading full content
- **Security Controls**: Domain allowlists, blocklists, and private IP protection
- **Content Limits**: Configurable response size limits (default: 10MB)

## Available Tools

### fetch
Fetch raw content from a URL
- `url` (string): The URL to fetch

Returns:
- `url`: The requested URL
- `content`: Raw content as received
- `contentType`: HTTP Content-Type header
- `size`: Content size in bytes

### fetch_text
Fetch content from a URL and convert HTML to plain text
- `url` (string): The URL to fetch

Returns:
- `url`: The requested URL
- `text`: Clean plain text content (HTML converted)
- `contentType`: HTTP Content-Type header
- `originalSize`: Original content size in bytes
- `textSize`: Text content size in bytes

### get_headers
Get HTTP headers for a URL without fetching the full content
- `url` (string): The URL to inspect

Returns:
- `url`: The requested URL
- `statusCode`: HTTP status code
- `status`: HTTP status text
- `headers`: All HTTP response headers

## Security Features

- **Scheme Validation**: Only HTTP and HTTPS URLs are allowed
- **Private IP Protection**: Blocks access to localhost and private IP ranges
- **Domain Controls**: Optional allowlist and blocklist for domains
- **Size Limits**: Prevents excessive memory usage with response size limits
- **Timeout Protection**: 30-second timeout on all requests

## Usage

```bash
# Basic usage
./mcp-fetch-server

# The server automatically blocks private IPs and localhost
```

## Example Requests

### Fetch Raw Content
```json
{
  "method": "call_tool",
  "params": {
    "name": "fetch",
    "arguments": {
      "url": "https://httpbin.org/json"
    }
  }
}
```

### Fetch and Convert HTML to Text
```json
{
  "method": "call_tool",
  "params": {
    "name": "fetch_text",
    "arguments": {
      "url": "https://example.com"
    }
  }
}
```

### Get Headers Only
```json
{
  "method": "call_tool",
  "params": {
    "name": "get_headers",
    "arguments": {
      "url": "https://httpbin.org/headers"
    }
  }
}
```

## Installation

```bash
go build -o mcp-fetch-server main.go
```

## Integration

This server can be integrated with MCP clients like Claude Desktop by adding it to your configuration:

```json
{
  "mcpServers": {
    "fetch": {
      "command": "/path/to/mcp-fetch-server"
    }
  }
}
```

## Blocked Domains

The server automatically blocks:
- localhost and 127.x.x.x addresses
- Private IP ranges (10.x.x.x, 192.168.x.x, 172.16-31.x.x)
- Any custom domains added to the blocklist

## Content Processing

- HTML content is automatically detected and can be converted to clean plain text
- Script and style tags are removed during HTML-to-text conversion
- Whitespace is normalized for better readability
- Binary content is returned as-is with appropriate content type information