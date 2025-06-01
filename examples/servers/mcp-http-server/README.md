# MCP HTTP Server

A Model Context Protocol (MCP) server that provides HTTP client capabilities for making REST API calls and web requests.

## Features

- **HTTP Methods**: Support for GET, POST, PUT, DELETE, and custom HTTP methods
- **Custom Headers**: Add custom HTTP headers to requests
- **Request Bodies**: Send JSON, form data, or plain text in request bodies
- **Response Details**: Get complete response information including headers, status codes, and timing
- **Security**: Built-in protection against private IP access and domain filtering
- **Size Limits**: Configurable response size limits to prevent memory issues

## Setup

1. **Build the Server**:
   ```bash
   go build -o mcp-http-server
   ```

2. **Run the Server**:
   ```bash
   ./mcp-http-server
   ```

## Tools

### `http_get`
Make an HTTP GET request.

**Parameters:**
- `url` (required): The URL to send the GET request to
- `headers` (optional): Object containing HTTP headers as key-value pairs

**Example:**
```json
{
  "url": "https://api.github.com/users/octocat",
  "headers": {
    "User-Agent": "MyApp/1.0",
    "Accept": "application/json"
  }
}
```

**Response:**
```json
{
  "url": "https://api.github.com/users/octocat",
  "method": "GET",
  "status_code": 200,
  "status": "200 OK",
  "headers": {
    "Content-Type": ["application/json; charset=utf-8"],
    "Server": ["GitHub.com"]
  },
  "body": "{\"login\":\"octocat\",\"id\":1,...}",
  "size": 1234,
  "duration": "500ms"
}
```

### `http_post`
Make an HTTP POST request.

**Parameters:**
- `url` (required): The URL to send the POST request to
- `body` (optional): The request body (JSON, form data, or plain text)
- `headers` (optional): Object containing HTTP headers as key-value pairs

**Example:**
```json
{
  "url": "https://httpbin.org/post",
  "body": "{\"name\":\"John\",\"age\":30}",
  "headers": {
    "Content-Type": "application/json",
    "Authorization": "Bearer token123"
  }
}
```

### `http_put`
Make an HTTP PUT request.

**Parameters:**
- `url` (required): The URL to send the PUT request to
- `body` (optional): The request body
- `headers` (optional): Object containing HTTP headers as key-value pairs

**Example:**
```json
{
  "url": "https://httpbin.org/put",
  "body": "{\"updated_field\":\"new_value\"}",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

### `http_delete`
Make an HTTP DELETE request.

**Parameters:**
- `url` (required): The URL to send the DELETE request to
- `headers` (optional): Object containing HTTP headers as key-value pairs

**Example:**
```json
{
  "url": "https://httpbin.org/delete",
  "headers": {
    "Authorization": "Bearer token123"
  }
}
```

### `http_request`
Make a custom HTTP request with any method.

**Parameters:**
- `method` (required): HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, etc.)
- `url` (required): The URL to send the request to
- `body` (optional): The request body (for methods that support it)
- `headers` (optional): Object containing HTTP headers as key-value pairs

**Example:**
```json
{
  "method": "PATCH",
  "url": "https://httpbin.org/patch",
  "body": "{\"field_to_update\":\"new_value\"}",
  "headers": {
    "Content-Type": "application/json",
    "X-Custom-Header": "custom-value"
  }
}
```

## Security Features

### Domain Protection
- Private IP addresses (localhost, 127.x.x.x, 192.168.x.x, 10.x.x.x, 172.16-31.x.x) are blocked by default
- Support for allowed and blocked domain lists (configurable)

### Response Limits
- Maximum response size: 10MB (configurable)
- Request timeout: 30 seconds (configurable)

### Default Headers
- Automatic User-Agent header if not provided: "MCP-HTTP-Server/1.0"
- Automatic Content-Type header for POST/PUT requests with body: "application/json"

## Configuration for Claude Desktop

Add this to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "http": {
      "command": "/path/to/mcp-http-server"
    }
  }
}
```

## Example Usage

Once connected, you can ask Claude:
- "Make a GET request to https://api.github.com/users/octocat"
- "POST this JSON data to the API: {\"name\":\"John\"}"
- "Check the headers for https://httpbin.org/headers"
- "Make a DELETE request to remove this resource"
- "Send a PATCH request to update this API endpoint"

## Common Use Cases

### API Testing
```json
{
  "method": "GET",
  "url": "https://jsonplaceholder.typicode.com/posts/1"
}
```

### Authentication
```json
{
  "method": "POST",
  "url": "https://api.example.com/login",
  "body": "{\"username\":\"user\",\"password\":\"pass\"}",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

### REST API Operations
```json
{
  "method": "PUT",
  "url": "https://api.example.com/users/123",
  "body": "{\"name\":\"Updated Name\"}",
  "headers": {
    "Authorization": "Bearer your-token",
    "Content-Type": "application/json"
  }
}
```

## Error Handling

The server includes comprehensive error handling for:
- Invalid URLs
- Network connectivity issues
- Timeout errors
- Response size limits
- Domain restrictions
- Invalid JSON parsing
- HTTP error status codes

All errors are returned in a structured format with descriptive messages.

## Rate Limiting

The server doesn't implement built-in rate limiting, but it respects server-side rate limits and will return appropriate error messages when limits are exceeded.