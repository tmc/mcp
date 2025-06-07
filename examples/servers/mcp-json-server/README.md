# MCP JSON Server

A Model Context Protocol server that provides JSON manipulation and validation tools.

## Features

- **validate_json**: Validates if a string is valid JSON
- **format_json**: Formats JSON with proper indentation
- **minify_json**: Removes whitespace from JSON to minimize size
- **extract_json_path**: Extracts value at specified JSON path using dot notation

## Usage

```bash
go run main.go
```

## Tools

### validate_json

Validates if a string is valid JSON.

**Parameters:**
- `json_string` (string): The JSON string to validate

**Example:**
```json
{
  "name": "validate_json",
  "arguments": {
    "json_string": "{\"key\": \"value\"}"
  }
}
```

### format_json

Formats JSON with proper indentation.

**Parameters:**
- `json_string` (string): The JSON string to format
- `indent` (integer, optional): Number of spaces for indentation (default: 2)

**Example:**
```json
{
  "name": "format_json", 
  "arguments": {
    "json_string": "{\"key\":\"value\",\"nested\":{\"item\":123}}",
    "indent": 4
  }
}
```

### minify_json

Removes whitespace from JSON to minimize size.

**Parameters:**
- `json_string` (string): The JSON string to minify

**Example:**
```json
{
  "name": "minify_json",
  "arguments": {
    "json_string": "{\n  \"key\": \"value\",\n  \"number\": 123\n}"
  }
}
```

### extract_json_path

Extracts value at specified JSON path using dot notation.

**Parameters:**
- `json_string` (string): The JSON string to extract from
- `path` (string): JSON path (e.g., 'data.items[0].name')

**Example:**
```json
{
  "name": "extract_json_path",
  "arguments": {
    "json_string": "{\"data\":{\"items\":[{\"name\":\"item1\"},{\"name\":\"item2\"}]}}",
    "path": "data.items[0].name"
  }
}
```

## JSON Path Syntax

The `extract_json_path` tool supports:
- Dot notation for object fields: `data.user.name`
- Array indexing: `items[0]`, `data.items[2]`
- Combined paths: `data.items[0].properties.name`