# MCP Puppeteer Server

A Go implementation of a Model Context Protocol (MCP) server that provides browser automation capabilities using Chrome DevTools Protocol.

## Installation

```bash
go get github.com/chromedp/chromedp
go build -o mcp-puppeteer-server
```

## Usage

The server can be run as a standalone executable that communicates over stdio:

```bash
./mcp-puppeteer-server
```

### Tools

The server exposes the following tools:

- **puppeteer_navigate**: Navigate to a URL
  - Required: `url` (string)
  - Optional: `launchOptions` (object), `allowDangerous` (boolean)

- **puppeteer_screenshot**: Take a screenshot of the current page or specific element
  - Required: `name` (string)
  - Optional: `selector` (string), `width` (number), `height` (number), `encoded` (boolean)

- **puppeteer_click**: Click an element on the page
  - Required: `selector` (string)

- **puppeteer_fill**: Fill out an input field
  - Required: `selector` (string), `value` (string)

- **puppeteer_select**: Select an option in a select element
  - Required: `selector` (string), `value` (string)

- **puppeteer_hover**: Hover over an element
  - Required: `selector` (string)

- **puppeteer_evaluate**: Execute JavaScript in the browser console
  - Required: `script` (string)

### Resources

The server provides access to:

- **console://logs**: Browser console logs
- **screenshot://{name}**: Saved screenshots

## Example Configuration

For Claude Desktop:

```json
{
  "servers": {
    "puppeteer": {
      "command": "/path/to/mcp-puppeteer-server"
    }
  }
}
```

## Dependencies

- [chromedp](https://github.com/chromedp/chromedp) - Chrome DevTools Protocol client
- MCP Go SDK

## Security Note

This server has access to browser automation capabilities. Be careful when granting permissions to execute arbitrary JavaScript or navigate to untrusted URLs.