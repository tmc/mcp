# Installation Guide

This guide walks you through installing the MCP tools and setting up your development environment.

## Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for using Makefiles)

## Installing MCP Tools

### From Source

1. Clone the repository:
```bash
git clone https://github.com/tmc/mcp.git
cd mcp
```

2. Build all tools:
```bash
go build ./...
```

3. Install tools to your PATH:
```bash
# Install specific tools
go install ./cmd/mcp-spy
go install ./cmd/mcp-replay
go install ./cmd/mcpdiff
go install ./cmd/mcp-sort
go install ./cmd/mcp-connect
go install ./cmd/mcp-proxy
go install ./cmd/mcp-serve
go install ./cmd/mcp-shadow

# Or install all at once
go install ./cmd/...
```

### Using Go Get

For individual tools:
```bash
go get -u github.com/tmc/mcp/cmd/mcp-spy@latest
go get -u github.com/tmc/mcp/cmd/mcp-replay@latest
# ... etc
```

## Verifying Installation

Check that tools are installed correctly:

```bash
# Check versions
mcp-spy -version
mcp-replay -version

# Test with a simple echo
echo '{"jsonrpc":"2.0","method":"test","id":1}' | mcp-spy -- cat
```

## Setting Up Your Environment

### Environment Variables

Optional environment variables:

```bash
# Set default MCP trace directory
export MCP_TRACE_DIR="$HOME/.mcp/traces"

# Enable verbose logging by default
export MCP_VERBOSE=1
```

### Shell Completion

For bash completion:
```bash
# Add to ~/.bashrc
source <(mcp-connect -completion-script-bash)
```

For zsh completion:
```bash
# Add to ~/.zshrc
source <(mcp-connect -completion-script-zsh)
```

## Development Dependencies

For developing MCP servers and clients, you'll also need:

```bash
# Core dependencies
go get github.com/tmc/mcp

# Optional: testing utilities
go get github.com/stretchr/testify
```

## Next Steps

Now that you have MCP installed, proceed to the [Quick Start Guide](./quickstart.md) to learn how to use the tools.

## Troubleshooting

### Command not found

If tools aren't found after installation:

1. Check your PATH:
```bash
echo $PATH
```

2. Ensure Go's bin directory is in PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

3. Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.)

### Build Errors

If you encounter build errors:

1. Verify Go version:
```bash
go version
# Should be 1.21 or later
```

2. Update dependencies:
```bash
go mod download
go mod tidy
```

3. Clean build cache:
```bash
go clean -cache
```

## Platform-Specific Notes

### macOS

On macOS, you may need to install additional tools:
```bash
# Install timeout command (gtimeout)
brew install coreutils

# Install pretty-print (pp) if desired
brew install pp
```

### Windows

On Windows, use PowerShell or Git Bash. Some examples may need adaptation for Windows command syntax.

### Linux

Most distributions include all necessary tools. Ensure you have `make` installed if using Makefiles:
```bash
# Debian/Ubuntu
sudo apt-get install build-essential

# Fedora/RHEL
sudo dnf install make
```