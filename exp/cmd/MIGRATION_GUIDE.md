# Migration Guide: MCP Tools to General Purpose Tools

This guide helps you migrate from MCP-specific commands to their general-purpose equivalents.

## Tool Migrations

### 1. mcp-jsonrpc2gostruct → json2go

**Old Usage:**
```bash
mcp-jsonrpc2gostruct -input request.json -output types.go
```

**New Usage:**
```bash
json2go -jsonrpc -input request.json -output types.go
```

**Changes:**
- Tool renamed to reflect broader JSON support
- Use `-jsonrpc` flag for JSON-RPC specific handling
- Also supports regular JSON and JSON Schema

---

### 2. mcp-ctx-go-src → gopackdump

**Old Usage:**
```bash
mcp-ctx-go-src github.com/tmc/mcp
```

**New Usage:**
```bash
gopackdump github.com/tmc/mcp
```

**Changes:**
- Name reflects general Go package dumping
- Supports multiple output formats beyond txtar
- Added recursive dependency support

---

### 3. mcpscripttest → scripttest

**Old Usage:**
```bash
mcpscripttest test_*.txt
```

**New Usage:**
```bash
scripttest test_*.txt
```

**Changes:**
- Removed MCP prefix for general use
- Works with any CLI tool testing
- Same scripttest format preserved

---

### 4. mcpcolor → logcolor

**Old Usage:**
```bash
mcpcolor < trace.jsonl
```

**New Usage:**
```bash
logcolor < trace.jsonl
```

**Changes:**
- Renamed for general log colorization
- Auto-detects more log formats
- Added theme support

---

### 5. mcp-tsnorm → tsnorm

**Old Usage:**
```bash
mcp-tsnorm -b "2024-01-01T00:00:00Z" < trace.log
```

**New Usage:**
```bash
tsnorm -base "2024-01-01T00:00:00Z" < trace.log
```

**Changes:**
- Simplified name for general use
- Expanded timestamp format support
- Better auto-detection

---

### 6. mcp2go → schema2go

**Old Usage:**
```bash
mcp2go -input tools.json -output types.go
```

**New Usage:**
```bash
schema2go -type mcp -input tools.json -output types.go
```

**Changes:**
- Name reflects support for multiple schema types
- Use `-type mcp` for MCP-specific schemas
- Also supports JSON Schema, OpenAPI, etc.

---

### 7. mcp-tool-graph → tool-graph

**Old Usage:**
```bash
mcp-tool-graph -target tests/
```

**New Usage:**
```bash
tool-graph -target tests/
```

**Changes:**
- Simplified name
- Works with any test dependency analysis
- Same visualization features

---

## Feature Mappings

### Enhanced Features in General Tools

1. **json2go** (formerly mcp-jsonrpc2gostruct)
   - Added: JSON Schema support
   - Added: Plain JSON inference
   - Added: Custom type prefixes
   - Added: Multiple struct tags

2. **scripttest** (formerly mcpscripttest)
   - Added: Update mode for changing tests
   - Added: Parallel execution
   - Added: Custom environment variables
   - Added: Better error reporting

3. **logcolor** (formerly mcpcolor)
   - Added: Multiple color themes
   - Added: Line numbering
   - Added: Pattern filtering
   - Added: More format detection

4. **schema2go** (formerly mcp2go)
   - Added: OpenAPI support
   - Added: Protocol Buffer support
   - Added: Schema validation
   - Added: Custom imports

## Command Line Changes

### Standardized Flags

All tools now follow consistent flag patterns:

- `-input` / `-output`: File I/O (with `-` for stdin/stdout)
- `-format`: Output format selection
- `-verbose`: Detailed output
- `-help`: Usage information

### Removed MCP-Specific Flags

Some MCP-specific flags have been generalized or removed:

```bash
# Old MCP-specific
mcp-tool --mcp-server localhost:8080

# New general form
tool --server localhost:8080  # or removed if too specific
```

## Configuration Files

The general-purpose tools use standard configuration approaches:

```bash
# Old: MCP-specific config
mcp-tool --config mcp.yaml

# New: Standard config
tool --config config.yaml
```

## Integration Changes

### Shell Scripts

Update shell scripts to use new names:

```bash
# Old
for file in *.json; do
  mcp-jsonrpc2gostruct -input "$file" -output "${file%.json}.go"
done

# New
for file in *.json; do
  json2go -input "$file" -output "${file%.json}.go"
done
```

### Makefiles

Update Makefile targets:

```makefile
# Old
generate:
	mcp2go -input schema.json -output types.go

# New
generate:
	schema2go -input schema.json -output types.go
```

### CI/CD Pipelines

Update pipeline configurations:

```yaml
# Old
steps:
  - name: Generate types
    run: mcp-jsonrpc2gostruct -input api.json

# New
steps:
  - name: Generate types
    run: json2go -input api.json
```

## Breaking Changes

### Minimal Breaking Changes

Most tools maintain backward compatibility with flags and behavior. The primary changes are:

1. **Binary names**: Remove `mcp-` prefix
2. **Import paths**: May change for Go libraries
3. **Default behaviors**: Some tools have smarter defaults

### API Compatibility

For tools used as libraries:

```go
// Old import
import "github.com/tmc/mcp/exp/mcpscripttest"

// New import
import "github.com/tmc/mcp/exp/scripttest"
```

## Migration Checklist

- [ ] Update binary names in scripts
- [ ] Update import paths in Go code
- [ ] Review flag changes
- [ ] Test with new defaults
- [ ] Update documentation
- [ ] Update CI/CD configurations

## Benefits of Migration

1. **Broader Applicability**: Tools work beyond MCP
2. **Better Maintenance**: More general tools get more attention
3. **Enhanced Features**: General versions often have more features
4. **Community**: Larger user base for general tools
5. **Simplicity**: Easier names and clearer purpose

## Support

During migration:

1. Both versions may coexist temporarily
2. File issues for missing features
3. Contribute improvements back
4. Share migration experiences

## Future Deprecation

The MCP-prefixed versions may be deprecated in future releases. Plan migration accordingly:

- **Phase 1**: Both versions available
- **Phase 2**: MCP versions marked deprecated
- **Phase 3**: MCP versions removed

Start migrating early to avoid disruption.