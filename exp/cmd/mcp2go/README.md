# mcp2go

Generate Go source code from MCP tool descriptions and JSON schemas.

## Overview

`mcp2go` is the inverse of `sourcereflect` - it generates Go types and interfaces from:
- MCP tool descriptions
- JSON schemas

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/mcp2go@latest
```

## Usage

```bash
# Auto-detect format and generate code
mcp2go tool.json

# Explicitly specify MCP tool format
mcp2go -type mcp weather_tool.json

# Generate from JSON schema
mcp2go -type jsonschema user_schema.json

# Specify output directory and package name
mcp2go -output ./generated -package myapp tool.json

# Use custom filename
mcp2go -name custom_name tool.json
```

## Examples

### MCP Tool Description

Given an MCP tool description:

```json
{
  "name": "get_weather",
  "description": "Get the current weather",
  "inputSchema": {
    "type": "object",
    "properties": {
      "location": {
        "type": "string",
        "description": "City name"
      }
    },
    "required": ["location"]
  },
  "returnType": {
    "type": "object",
    "properties": {
      "temperature": {
        "type": "number"
      }
    }
  }
}
```

`mcp2go` generates:

```go
package generated

import (
    "context"
    "fmt"
)

// GetWeatherTool Get the current weather
type GetWeatherTool interface {
    Execute(ctx context.Context, input *GetWeatherInput) (*GetWeatherOutput, error)
}

// GetWeatherInput represents the input for GetWeather
type GetWeatherInput struct {
    // Location City name
    Location string `json:"location"`
}

// GetWeatherOutput represents the output for GetWeather
type GetWeatherOutput struct {
    Temperature *float64 `json:"temperature,omitempty"`
}

// GetWeatherImpl implements the GetWeatherTool interface
type GetWeatherImpl struct{}

// Execute implements GetWeatherTool
func (t *GetWeatherImpl) Execute(ctx context.Context, input *GetWeatherInput) (*GetWeatherOutput, error) {
    // TODO: Implement tool logic
    return nil, fmt.Errorf("not implemented")
}
```

### JSON Schema

Given a JSON schema:

```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "name": {"type": "string"},
    "age": {"type": "integer"}
  },
  "required": ["id", "name"]
}
```

`mcp2go` generates:

```go
package generated

// User represents user
type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Age  *int   `json:"age,omitempty"`
}
```

## Features

- Auto-detects input format (MCP tool or JSON schema)
- Generates idiomatic Go code with proper JSON tags
- Handles required vs optional fields correctly
- Supports enums with const generation
- Handles special formats like date-time
- Creates interface and implementation stubs for MCP tools
- Respects Go naming conventions

## Limitations

- Nested object schemas generate placeholder types (for now)
- References ($ref) are not yet supported
- Complex schema features (allOf, anyOf, oneOf) not yet implemented