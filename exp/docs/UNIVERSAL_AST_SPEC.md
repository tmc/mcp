# Universal AST Specification for MCP

This document defines a language-agnostic AST (Abstract Syntax Tree) representation for MCP trace analysis and code generation.

## Core Concepts

The Universal AST (UAST) for MCP provides a common intermediate representation that can be:
1. Generated from MCP traces
2. Transformed to language-specific code
3. Analyzed for patterns and optimizations
4. Validated for correctness

## UAST Node Types

### 1. Root Node
```json
{
  "type": "MCPImplementation",
  "kind": "server" | "client",
  "metadata": {
    "sourceTrace": "trace.jsonl",
    "generatedAt": "2023-10-15T10:30:00Z",
    "language": "universal"
  },
  "children": []
}
```

### 2. Tool Definition
```json
{
  "type": "ToolDefinition",
  "name": "calculate",
  "description": "Performs a calculation",
  "inputSchema": {
    "type": "Schema",
    "properties": {...}
  },
  "outputSchema": {
    "type": "Schema",
    "properties": {...}
  },
  "implementation": {
    "type": "FunctionBody",
    "children": [...]
  }
}
```

### 3. Resource Definition
```json
{
  "type": "ResourceDefinition",
  "name": "config",
  "description": "Configuration resource",
  "uri": "config://settings",
  "mimeType": "application/json",
  "operations": {
    "read": {
      "type": "FunctionBody",
      "children": [...]
    },
    "write": {
      "type": "FunctionBody",
      "children": [...]
    },
    "subscribe": {
      "type": "FunctionBody",
      "children": [...]
    }
  }
}
```

### 4. Type Definitions
```json
{
  "type": "TypeDefinition",
  "name": "CalculationRequest",
  "kind": "struct",
  "fields": [
    {
      "name": "x",
      "type": "number",
      "required": true
    },
    {
      "name": "y",
      "type": "number",
      "required": true
    }
  ]
}
```

### 5. Function Nodes
```json
{
  "type": "Function",
  "name": "handleCalculate",
  "async": true,
  "parameters": [
    {
      "name": "request",
      "type": "CalculationRequest"
    }
  ],
  "returnType": "CalculationResponse",
  "body": {
    "type": "Block",
    "children": [...]
  }
}
```

### 6. Control Flow
```json
{
  "type": "IfStatement",
  "condition": {
    "type": "BinaryExpression",
    "operator": "==",
    "left": {"type": "Identifier", "name": "x"},
    "right": {"type": "Literal", "value": 0}
  },
  "consequent": {
    "type": "Block",
    "children": [...]
  },
  "alternate": {
    "type": "Block",
    "children": [...]
  }
}
```

### 7. Error Handling
```json
{
  "type": "TryCatch",
  "body": {
    "type": "Block",
    "children": [...]
  },
  "handler": {
    "type": "CatchClause",
    "param": {"type": "Identifier", "name": "error"},
    "body": {
      "type": "Block",
      "children": [...]
    }
  }
}
```

## Type System

### Primitive Types
- `string`
- `number`
- `boolean`
- `null`
- `undefined`
- `any`

### Composite Types
- `array<T>`
- `map<K,V>`
- `union<T1,T2,...>`
- `intersection<T1,T2,...>`
- `optional<T>`

### MCP-Specific Types
- `ToolResult`
- `ResourceContent`
- `PromptMessage`
- `LogLevel`
- `Notification`

## Language Mapping

### Python Mapping
```yaml
number: float | int
string: str
boolean: bool
array<T>: List[T]
map<K,V>: Dict[K,V]
optional<T>: Optional[T]
union<T1,T2>: Union[T1,T2]
```

### TypeScript Mapping
```yaml
number: number
string: string
boolean: boolean
array<T>: T[]
map<K,V>: Record<K,V>
optional<T>: T | undefined
union<T1,T2>: T1 | T2
```

### Rust Mapping
```yaml
number: f64 | i64
string: String
boolean: bool
array<T>: Vec<T>
map<K,V>: HashMap<K,V>
optional<T>: Option<T>
union<T1,T2>: enum { T1(T1), T2(T2) }
```

### Go Mapping
```yaml
number: float64 | int64
string: string
boolean: bool
array<T>: []T
map<K,V>: map[K]V
optional<T>: *T
union<T1,T2>: interface{} with type assertion
```

## Pattern Library

### 1. Async Handler Pattern
```json
{
  "type": "Pattern",
  "name": "AsyncHandler",
  "template": {
    "type": "Function",
    "async": true,
    "parameters": [
      {"name": "request", "type": "$RequestType"}
    ],
    "returnType": "Promise<$ResponseType>",
    "body": {
      "type": "TryCatch",
      "body": {"$slot": "implementation"},
      "handler": {"$slot": "errorHandler"}
    }
  }
}
```

### 2. Resource Subscription Pattern
```json
{
  "type": "Pattern",
  "name": "ResourceSubscription",
  "template": {
    "type": "Class",
    "methods": [
      {
        "name": "subscribe",
        "parameters": [{"name": "uri", "type": "string"}],
        "body": {"$slot": "subscribeImpl"}
      },
      {
        "name": "unsubscribe",
        "parameters": [{"name": "subscriptionId", "type": "string"}],
        "body": {"$slot": "unsubscribeImpl"}
      }
    ]
  }
}
```

## Transformation Rules

### 1. Trace to UAST
```yaml
Initialize Request:
  - Create MCPImplementation node
  - Extract capabilities
  - Set metadata

Tool List Response:
  - Create ToolDefinition nodes
  - Parse schemas
  - Generate type definitions

Resource List Response:
  - Create ResourceDefinition nodes
  - Map URI patterns
  - Define operations
```

### 2. UAST to Language
```yaml
Python:
  - Map types using type hints
  - Convert async to async/await
  - Use dataclasses for structs

TypeScript:
  - Generate interfaces from types
  - Use proper async syntax
  - Add JSDoc comments

Rust:
  - Generate structs with serde
  - Use Result for error handling
  - Apply ownership rules

Go:
  - Generate structs with tags
  - Use error returns
  - Apply interface patterns
```

## Optimization Passes

### 1. Type Inference
- Infer types from usage when not explicit
- Propagate type information
- Detect type conflicts

### 2. Dead Code Elimination
- Remove unused tool definitions
- Eliminate unreachable code
- Prune unused types

### 3. Pattern Matching
- Identify common patterns
- Apply optimizations
- Suggest refactoring

## Validation Rules

### 1. Type Checking
- Ensure type consistency
- Validate schema compliance
- Check null safety

### 2. MCP Protocol Compliance
- Verify required fields
- Check method signatures
- Validate response formats

### 3. Language Constraints
- Apply language-specific rules
- Check naming conventions
- Ensure syntax validity

## Example UAST

```json
{
  "type": "MCPImplementation",
  "kind": "server",
  "metadata": {
    "name": "ExampleServer",
    "version": "1.0.0"
  },
  "children": [
    {
      "type": "ToolDefinition",
      "name": "add",
      "description": "Adds two numbers",
      "inputSchema": {
        "type": "Schema",
        "properties": {
          "a": {"type": "number"},
          "b": {"type": "number"}
        }
      },
      "implementation": {
        "type": "FunctionBody",
        "children": [
          {
            "type": "ReturnStatement",
            "expression": {
              "type": "BinaryExpression",
              "operator": "+",
              "left": {"type": "Identifier", "name": "a"},
              "right": {"type": "Identifier", "name": "b"}
            }
          }
        ]
      }
    }
  ]
}
```

## Implementation Notes

1. **Extensibility**: Design for easy addition of new languages
2. **Performance**: Optimize for large trace files
3. **Accuracy**: Maintain semantic correctness across transformations
4. **Tooling**: Provide utilities for UAST manipulation
5. **Testing**: Comprehensive test suite for all transformations

This UAST specification enables reliable cross-language MCP development while maintaining the protocol's semantics and best practices.