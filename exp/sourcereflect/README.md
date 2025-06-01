# sourcereflect

Package sourcereflect provides functionality to generate JSON schemas from Go types using reflection.

## Features

- Generate JSON Schema from Go types and values
- Support for struct tags (json tags)
- Automatic detection of required/optional fields
- Support for arrays, slices, and maps
- Caller context information for debugging
- Fluent schema builder API

## Installation

```bash
go get github.com/tmc/mcp/exp/sourcereflect
```

## Usage

### Basic Type Reflection

```go
type User struct {
    ID       int      `json:"id"`
    Name     string   `json:"name"`
    Email    string   `json:"email"`
    Age      int      `json:"age,omitempty"`
    Tags     []string `json:"tags"`
}

// Generate schema from type
schema, err := sourcereflect.FromType(reflect.TypeOf(User{}))
if err != nil {
    log.Fatal(err)
}

// Convert to JSON
jsonStr, _ := schema.ToPrettyJSON()
fmt.Println(jsonStr)
```

Output:
```json
{
  "type": "object",
  "title": "User",
  "properties": {
    "id": {
      "type": "integer"
    },
    "name": {
      "type": "string"
    },
    "email": {
      "type": "string"
    },
    "age": {
      "type": "integer"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  },
  "required": ["id", "name", "email", "tags"]
}
```

### Using Caller Context

```go
// Generate schema with source location information
schema, err := sourcereflect.SchemaFromCaller(User{})
```

This will include metadata about where the schema was generated from:

```json
{
  "type": "object",
  "$sourceLocation": {
    "file": "/path/to/file.go",
    "line": 42,
    "function": "main.generateSchema",
    "package": "main"
  }
}
```

### Schema Builder

```go
schema := sourcereflect.NewSchemaBuilder().
    WithType("object").
    WithTitle("Config").
    WithProperty("host", &sourcereflect.Schema{Type: "string"}).
    WithProperty("port", &sourcereflect.Schema{Type: "integer"}).
    WithRequired("host", "port").
    Build()
```

## Supported Go Types

- Basic types: `string`, `int`, `float`, `bool`
- Structs (converted to objects)
- Slices and arrays (converted to arrays)
- Maps with string keys (converted to objects with additionalProperties)
- Pointers (automatically dereferenced)
- Interfaces (generic object type)

## JSON Tags

The package respects JSON struct tags:

- `json:"name"` - Use custom field name
- `json:"name,omitempty"` - Mark field as optional
- `json:"-"` - Skip field

## Command-line Tool

The package includes a command-line tool `sourcereflect` for generating JSON schemas from Go files:

```bash
# Install the command
go install github.com/tmc/mcp/exp/sourcereflect/cmd/sourcereflect@latest

# Generate a schema
sourcereflect -type MyStruct myfile.go

# Generate pretty-printed JSON
sourcereflect -pretty -type MyStruct myfile.go

# Include source location metadata
sourcereflect -pretty -type MyStruct -caller myfile.go
```

## MCP Tools Support

The package can generate Model Context Protocol (MCP) tool descriptions and analyze source code to determine tool behavior hints:

### Generating MCP Tool Descriptions

```go
// From a function type
funcType := reflect.TypeOf(MyFunction)
tool, err := sourcereflect.ToMCPTool("MyFunction", funcType)

// From command line
sourcereflect -func ProcessData myfile.go
sourcereflect -func ProcessData -analyze-hints myfile.go
```

### Source Code Analysis

The package can analyze Go source code to determine MCP tool hints:

- `readOnlyHint`: Whether the function modifies state or has side effects
- `destructiveHint`: Whether the function performs destructive operations (file writes, deletions)
- `idempotentHint`: Whether repeated calls have no additional effect
- `openWorldHint`: Whether the function interacts with external systems (network)

```bash
# Analyze a function for MCP hints
sourcereflect -func FetchData -analyze-hints data.go

# Pretty print the result
sourcereflect -func WriteFile -analyze-hints -pretty file.go
```

The analysis examines function calls to detect:
- Disk operations (os.WriteFile, os.Remove, etc.)
- Network operations (http.Get, net.Dial, etc.)
- State changes (assignments, mutations)

## Testing

The package includes comprehensive tests using both standard Go tests and `rsc.io/script/scripttest` for testing the command-line tool:

```bash
# Run all tests
go test ./...

# Run only the scripttest tests
go test -v -run TestScriptSimple

# Add new scripttest cases in testdata/*.txt
```

Scripttest files use the txtar format and can test various aspects of the command-line tool's behavior.

## License

This package is part of the MCP project and follows the project's license.