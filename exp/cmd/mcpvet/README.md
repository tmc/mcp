# mcpvet - Go Vet Tool for MCP Scripts

mcpvet is a custom Go analysis tool built on top of the `go/analysis` package to validate MCP scripttest files.

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/mcpvet@latest
```

## Usage

mcpvet can be used in two ways:

### 1. As a standalone tool

```bash
mcpvet [flags] [packages]
```

Example:
```bash
mcpvet ./...
```

### 2. With go vet

```bash
go vet -vettool=$(which mcpvet) [flags] [packages]
```

Example:
```bash
go vet -vettool=$(which mcpvet) ./...
```

## What it Checks

The scripttest analyzer looks for issues in MCP scripttest files (`.txt` files), including:

1. Invalid command syntax 
2. Missing required arguments for specific commands
3. Unbalanced async commands (start/stop/wait)
4. Potentially problematic paths and environment variables
5. Unknown commands

## Example Output

```
/path/to/testdata/example.txt:10: error (args): mcp-replay command requires a recording file argument
  mcp-replay
/path/to/testdata/example.txt:15: warning (async): stdin command found, but no active async command detected
  stdin hello
/path/to/testdata/example.txt:23: warning (portability): Absolute path '/tmp/test' may cause test to be non-portable
  cat /tmp/test
```

## Integration with CI/CD

You can add mcpvet to your CI/CD pipeline to ensure that script tests are valid before they're merged:

```yaml
- name: Validate scripttest files
  run: go vet -vettool=$(go env GOPATH)/bin/mcpvet ./...
```

## Development

To add new checks, modify the analyzer in the `scripttest` package.