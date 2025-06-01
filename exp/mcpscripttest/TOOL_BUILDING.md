# Go Tool Building Guide

This guide explains how to build and manage Go tools, including coverage-enabled tools and custom toolchain integration.

## Understanding Go Tools

Go tools are executables that can be invoked via the `go tool` command. They are stored in `$GOROOT/pkg/tool/$GOOS_$GOARCH/`.

### Built-in Tools vs Custom Tools

**Built-in tools** (like compile, link, asm):
- Pre-compiled during Go installation
- Located in the Go toolchain directory
- Invoked via `go tool <name>`

**Custom tools**:
- Must be built separately
- Can be installed globally or per-module
- Various methods for integration

## Building Tools

### Basic Tool Building

```bash
# Build a tool to current directory
go build ./cmd/mytool

# Build with specific output name/location
go build -o mytool ./cmd/mytool

# Install globally (to $GOPATH/bin or $GOBIN)
go install ./cmd/mytool
```

### Building with Coverage

Go 1.20+ supports building executables with coverage instrumentation:

```bash
# Build with coverage enabled
go build -cover -o mytool ./cmd/mytool

# Build with specific coverage mode
go build -cover -covermode=atomic -o mytool ./cmd/mytool

# Coverage modes:
# - set:    did each statement run?
# - count:  how many times did each statement run?
# - atomic: like count but safe for concurrent use
```

### Running Coverage-Enabled Tools

```bash
# Set coverage output directory
export GOCOVERDIR=/tmp/coverage

# Run the tool
./mytool

# Analyze coverage
go tool covdata percent -i=/tmp/coverage
go tool covdata textfmt -i=/tmp/coverage -o=coverage.txt
go tool cover -html=coverage.txt -o=coverage.html
```

## Tool Management Approaches

### 1. Global Installation

```bash
# Install to $GOPATH/bin or $GOBIN
go install github.com/example/tool@latest

# Run directly (if in PATH)
tool --help
```

### 2. Module-Based Tools (go.mod)

Traditional approach (pre-Go 1.24):
```go
// tools.go
//go:build tools
// +build tools

package tools

import (
    _ "github.com/example/tool"
)
```

New approach (Go 1.24+):
```bash
# Add tool to module
go get -tool github.com/example/tool@latest

# Run via go tool
go tool tool-name
```

### 3. go run Approach

```bash
# Run without installing
go run github.com/example/tool@latest

# Run with coverage
go run -cover github.com/example/tool@latest

# Run with specific version
go run github.com/example/tool@v1.2.3
```

## Cannot Do: go tool with Build Flags

The `go tool` command **cannot** pass build flags. It only runs pre-built binaries:

```bash
# This does NOT work
go tool -cover mytool  # ❌ Not supported

# go tool only supports -n flag
go tool -n compile     # ✓ Prints command without executing
```

## Workarounds for Coverage-Enabled Tools

### 1. Build and Run Separately

```bash
#!/bin/bash
# build-and-run-with-coverage.sh

TOOL=$1
shift

# Build with coverage
TMPDIR=$(mktemp -d)
go build -cover -o "$TMPDIR/tool" "$TOOL"

# Run with coverage collection
GOCOVERDIR="${GOCOVERDIR:-./coverage}" "$TMPDIR/tool" "$@"

# Cleanup
rm -rf "$TMPDIR"
```

### 2. Makefile Approach

```makefile
# Makefile
.PHONY: build-tools coverage-tools

build-tools:
	go build -o bin/tool1 ./cmd/tool1
	go build -o bin/tool2 ./cmd/tool2

coverage-tools:
	go build -cover -o bin/tool1-cov ./cmd/tool1
	go build -cover -o bin/tool2-cov ./cmd/tool2

run-coverage:
	GOCOVERDIR=./coverage bin/tool1-cov

coverage-report:
	go tool covdata percent -i=./coverage
	go tool covdata textfmt -i=./coverage -o=coverage.txt
	go tool cover -html=coverage.txt
```

### 3. Docker-Based Approach

```dockerfile
# Dockerfile.coverage
FROM golang:1.21

WORKDIR /app
COPY . .

# Build with coverage
RUN go build -cover -o /usr/local/bin/mytool ./cmd/mytool

ENV GOCOVERDIR=/coverage
VOLUME ["/coverage"]

ENTRYPOINT ["mytool"]
```

## Reading Build Information

Tools can include build metadata that can be inspected:

```go
package main

import (
    "debug/buildinfo"
    "fmt"
    "runtime/debug"
)

// At runtime, from within the tool
func showBuildInfo() {
    if info, ok := debug.ReadBuildInfo(); ok {
        fmt.Printf("Go version: %s\n", info.GoVersion)
        for _, setting := range info.Settings {
            if setting.Key == "-cover" {
                fmt.Printf("Built with coverage: %s\n", setting.Value)
            }
        }
    }
}

// From external binary
func inspectBinary(path string) {
    info, err := buildinfo.ReadFile(path)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Tool built with Go %s\n", info.GoVersion)
}
```

## Best Practices

1. **Use go run for development**: Quick iteration with coverage support
2. **Use go install for production**: Clean, optimized binaries
3. **Separate coverage builds**: Use different names/locations for coverage-enabled tools
4. **Document tool requirements**: Specify minimum Go version and build flags
5. **Version your tools**: Use semantic versioning and module versioning

## Example: Complete Tool Workflow

```bash
# 1. Development with coverage
go run -cover ./cmd/mytool

# 2. Build for testing
go build -cover -o mytool-test ./cmd/mytool
GOCOVERDIR=./test-coverage ./mytool-test

# 3. Build for production
go build -trimpath -ldflags="-s -w" -o mytool ./cmd/mytool

# 4. Install globally
go install ./cmd/mytool

# 5. Cross-compile
GOOS=linux GOARCH=amd64 go build -o mytool-linux ./cmd/mytool
GOOS=windows GOARCH=amd64 go build -o mytool.exe ./cmd/mytool
```

## Future Developments

The Go team is actively working on improving tool management:

1. Enhanced `tool` directive in go.mod (Go 1.24+)
2. Potential support for build flags in tool management
3. Better integration between module system and tooling

For now, use the appropriate workaround based on your needs.

## References

- [Go Command Documentation](https://golang.org/cmd/go/)
- [Build Mode Documentation](https://golang.org/cmd/go/#hdr-Build_modes)
- [Coverage Documentation](https://go.dev/blog/integration-test-coverage)
- [Module Management](https://golang.org/ref/mod)