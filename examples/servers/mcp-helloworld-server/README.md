# MCP HelloWorld Server

A beginner-friendly Model Context Protocol (MCP) server that demonstrates basic MCP concepts with greetings and fortunes.

## Features

- Multi-language greetings (12 languages supported)
- Random inspirational fortunes
- Combined greetings with fortunes
- Language listing functionality

## Tools

### greeting
Generate a greeting in various languages.

**Parameters:**
- `name` (string, optional): Name to greet (defaults to "World")
- `language` (string, optional): Language for the greeting (defaults to "english")

**Supported Languages:**
- english, spanish, french, german, italian, portuguese
- russian, japanese, chinese, korean, arabic, hindi

### fortune
Get a random inspirational fortune.

**Parameters:** None

### list_languages
List all supported languages for greetings.

**Parameters:** None

### greeting_with_fortune
Generate a greeting with a bonus fortune.

**Parameters:**
- `name` (string, optional): Name to greet (defaults to "World")
- `language` (string, optional): Language for the greeting (defaults to "english")

## Usage

```bash
# Start the server
go run .
```

## Examples

### Simple greeting
```json
{
  "tool": "greeting",
  "arguments": {
    "name": "Alice"
  }
}
```

### Greeting in Spanish
```json
{
  "tool": "greeting",
  "arguments": {
    "name": "Carlos",
    "language": "spanish"
  }
}
```

### Get a fortune
```json
{
  "tool": "fortune",
  "arguments": {}
}
```

### List supported languages
```json
{
  "tool": "list_languages",
  "arguments": {}
}
```

### Greeting with fortune
```json
{
  "tool": "greeting_with_fortune",
  "arguments": {
    "name": "Bob",
    "language": "french"
  }
}
```