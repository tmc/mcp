# Git Integration Tools for MCP

This document describes tools for integrating MCP code generation with Git version control, filesystem snapshots, and test evolution.

## Git Notes Tools

### 1. mcp-git-annotate
```bash
mcp-git-annotate --trace=trace.jsonl --commit=HEAD generated_server.go
```
- Attaches generation metadata to git notes
- Records trace file, timestamp, tool version
- Links generated code to source traces
- Supports custom note namespaces

Example git note:
```yaml
mcp-generation:
  trace: trace.jsonl
  timestamp: 2024-01-15T10:30:00Z
  tool: mcp-trace-codegen v1.2.0
  checksum: sha256:abc123...
  options:
    language: go
    template: default
  source_events: 
    - initialize: request#1
    - tools.list: response#3
    - tool.call: request#5
```

### 2. mcp-git-trace
```bash
mcp-git-trace log --since="1 week ago" --tool=mcp-trace-codegen
```
- Queries git notes for MCP metadata
- Shows generation history
- Tracks tool usage patterns
- Identifies trace sources

### 3. mcp-generation-diff
```bash
mcp-generation-diff HEAD~1 HEAD
```
- Compares generation metadata between commits
- Shows trace differences
- Highlights tool version changes
- Suggests regeneration when needed

## Git Worktree Management

### 4. mcp-worktree
```bash
mcp-worktree create feature/new-tool --trace=new_tool.jsonl
```
- Creates git worktrees for generated code
- Isolates generation experiments
- Manages multiple versions simultaneously
- Automates worktree lifecycle

Features:
- Automatic branch creation
- Parallel generation support
- Worktree templates
- Cleanup on merge

### 5. mcp-tree-sync
```bash
mcp-tree-sync --source=main --target=feature/generated
```
- Synchronizes generated code across worktrees
- Handles merge conflicts
- Preserves generation metadata
- Updates trace references

### 6. mcp-tree-test
```bash
mcp-tree-test --all-worktrees
```
- Runs tests across all worktrees
- Compares results
- Generates compatibility matrix
- Identifies breaking changes

## Filesystem Snapshots

### 7. mcp-snapshot
```bash
mcp-snapshot create before-generation
mcp-trace-codegen trace.jsonl
mcp-snapshot diff before-generation after-generation
```
- Creates filesystem snapshots
- Tracks generation changes
- Supports incremental updates
- Enables rollback

Snapshot format:
```json
{
  "id": "before-generation",
  "timestamp": "2024-01-15T10:30:00Z",
  "files": {
    "server.go": {
      "hash": "sha256:...",
      "size": 1024,
      "mode": "0644"
    }
  },
  "metadata": {
    "mcp_tools": ["mcp-trace-codegen"],
    "trace_files": ["trace.jsonl"]
  }
}
```

### 8. mcp-fs-branch
```bash
mcp-fs-branch create experiment/async-handlers
# Work in branched filesystem
mcp-fs-branch merge experiment/async-handlers
```
- Filesystem-level branching
- Copy-on-write optimization
- Merge capabilities
- Conflict resolution

### 9. mcp-fs-diff
```bash
mcp-fs-diff snapshot-1 snapshot-2 --format=patch
```
- Compares filesystem states
- Generates patches
- Tracks file movements
- Identifies generation patterns

## Test Mutation & Evolution

### 10. mcp-test-mutate
```bash
mcp-test-mutate scripttest.txt --mutations=5
```
- Mutates scripttest files
- Generates test variations
- Explores edge cases
- Validates robustness

Mutation strategies:
- Command reordering
- Input fuzzing
- Timeout variations
- Error injection

### 11. mcp-test-evolve
```bash
mcp-test-evolve --trace=trace.jsonl --test=basic.txt
```
- Evolves tests based on traces
- Adds new test cases
- Improves coverage
- Learns from failures

Example evolution:
```diff
# basic.txt
exec echo "test"
stdout 'test'

+# Evolved from trace.jsonl
+exec mcp-tool call calculate '{"x": 1, "y": 2}'
+stdout '{"result": 3}'
+
+# Error case discovered
+exec mcp-tool call calculate '{"x": "invalid"}'
+stderr 'type error'
```

### 12. mcp-test-generate
```bash
mcp-test-generate --from=manual.md --format=scripttest
```
- Converts documentation to tests
- Generates from examples
- Creates test scenarios
- Supports multiple formats

### 13. mcp-test-minimize
```bash
mcp-test-minimize failing_test.txt
```
- Reduces failing tests to minimal case
- Removes unnecessary steps
- Identifies root cause
- Speeds up debugging

## Manpage to Tool Graph

### 14. mcp-man2tool
```bash
mcp-man2tool git --output=git_tools.jsonl
```
- Parses man pages
- Extracts command structure
- Generates tool definitions
- Creates MCP servers from CLI tools

Example output:
```json
{
  "tool": "git-add",
  "description": "Add file contents to the index",
  "input_schema": {
    "type": "object",
    "properties": {
      "pathspec": {"type": "array", "items": {"type": "string"}},
      "force": {"type": "boolean"},
      "interactive": {"type": "boolean"}
    }
  }
}
```

### 15. mcp-graph-build
```bash
mcp-graph-build --tools=git_tools.jsonl --output=tool_graph.dot
```
- Builds tool dependency graphs
- Identifies tool relationships
- Visualizes workflows
- Optimizes execution paths

### 16. mcp-graph-execute
```bash
mcp-graph-execute tool_graph.dot --goal="commit changes"
```
- Executes tool graphs
- Handles dependencies
- Manages state
- Provides progress tracking

### 17. mcp-man-trace
```bash
mcp-man-trace --command="git add . && git commit -m 'test'" --output=git_trace.jsonl
```
- Records command execution as MCP trace
- Captures stdin/stdout/stderr
- Preserves timing information
- Generates tool calls

## Advanced Workflows

### 18. mcp-ci-integrate
```bash
mcp-ci-integrate --pipeline=.github/workflows/test.yml
```
- Integrates with CI/CD systems
- Adds generation steps
- Manages artifacts
- Tracks metadata

### 19. mcp-bisect
```bash
mcp-bisect --trace=failing_trace.jsonl --test=regression.txt
```
- Git bisect for traces
- Finds breaking changes
- Identifies regression points
- Automates debugging

### 20. mcp-lineage
```bash
mcp-lineage track generated_server.go
```
- Tracks code lineage
- Shows generation history
- Maps to source traces
- Visualizes evolution

## Implementation Examples

### Complete Workflow Example

```bash
# 1. Create snapshot before generation
mcp-snapshot create pre-gen

# 2. Generate code from trace
mcp-trace-codegen trace.jsonl > server.go

# 3. Annotate with metadata
mcp-git-annotate --trace=trace.jsonl server.go

# 4. Create worktree for testing
mcp-worktree create test/generated

# 5. Run tests
mcp-test-evolve --trace=trace.jsonl --test=basic.txt

# 6. Compare results
mcp-snapshot diff pre-gen post-gen

# 7. Commit with metadata
git add server.go
git commit -m "Generated from trace.jsonl

$(mcp-git-trace describe)"
```

### Test Evolution Example

```bash
# Start with basic test
cat > basic.txt << EOF
exec mcp-server
stdin '{"method": "initialize"}'
stdout '"result"'
EOF

# Evolve based on trace
mcp-test-evolve --trace=server_trace.jsonl --test=basic.txt

# Mutate for edge cases
mcp-test-mutate basic_evolved.txt --count=10

# Minimize failures
mcp-test-minimize mutant_5_failing.txt

# Generate documentation
mcp-test-generate --from=test_results.md
```

### Man Page Integration

```bash
# Extract tools from man pages
mcp-man2tool curl > curl_tools.jsonl

# Build tool graph
mcp-graph-build --tools=curl_tools.jsonl

# Generate MCP server
mcp-man2mcp curl > curl_mcp_server.go

# Test with traces
mcp-man-trace --command="curl https://api.example.com" > curl_trace.jsonl

# Generate tests
mcp-test-generate --from=curl_trace.jsonl
```

## Metadata Schema

### Generation Metadata
```yaml
mcp:generation:
  version: "1.0"
  tool:
    name: "mcp-trace-codegen"
    version: "1.2.0"
  source:
    trace: "trace.jsonl"
    checksum: "sha256:..."
  output:
    files:
      - path: "server.go"
        checksum: "sha256:..."
  timestamp: "2024-01-15T10:30:00Z"
  options:
    language: "go"
    template: "default"
```

### Test Evolution Metadata
```yaml
mcp:test:evolution:
  version: "1.0"
  original: "basic.txt"
  evolved: "basic_evolved.txt"
  source:
    trace: "trace.jsonl"
    events: [1, 5, 10]
  mutations:
    - type: "command_reorder"
      line: 5
    - type: "input_fuzz"
      line: 8
  coverage:
    before: 0.45
    after: 0.78
```

## Best Practices

1. **Always snapshot before generation**
   - Enables rollback
   - Tracks changes
   - Preserves state

2. **Use worktrees for experiments**
   - Isolates changes
   - Parallel development
   - Easy cleanup

3. **Annotate all generated code**
   - Maintains traceability
   - Enables debugging
   - Documents process

4. **Evolve tests incrementally**
   - Start simple
   - Add complexity
   - Maintain readability

5. **Version control metadata**
   - Track tool versions
   - Reference traces
   - Document decisions

These tools provide comprehensive Git integration, filesystem management, and test evolution capabilities for MCP development workflows.