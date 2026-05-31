# Scripts

This directory contains development and CI/CD scripts for the MCP Go implementation.

## Scripts

### `check-root-dep-contract.sh`

Checks the root package runtime dependency contract for the v1 release gate.
Runtime dependencies must be the standard library, `golang.org/x/*`, or the
approved root API exceptions recorded in
`docs/design/v1-release-exemplary-gate.md`.

**Usage:**
```bash
bash ./scripts/check-root-dep-contract.sh
make check-deps
```

### `mcp-conformance.sh`

Runs the upstream MCP server conformance harness against an already-running
HTTP MCP server endpoint. The script pins the harness to
`@modelcontextprotocol/conformance@0.1.16` and prefers the local Node 24
installation when present because Node 20 lacks `fs.globSync`.

**Usage:**
```bash
MCP_CONFORMANCE_URL=http://127.0.0.1:3000/mcp ./scripts/mcp-conformance.sh
make conformance MCP_CONFORMANCE_URL=http://127.0.0.1:3000/mcp
```

Use `--dry-run` to print the resolved command without running the harness.

### `pre-commit.sh`

Local development pre-commit script that runs the same checks as the CI pipeline.

**Usage:**
```bash
./scripts/pre-commit.sh
```

**What it checks:**
- No binary files in staging area (per CLAUDE.md guidelines)
- go.sum not directly modified (use `go mod tidy` instead)
- Code formatting with `gofmt -s`
- `go vet` passes
- `go.mod` and `go.sum` are tidy
- All packages compile successfully
- All core tools compile
- Experimental tools compile (warnings only for conditional builds)
- All tests compile
- Basic smoke tests for core tools
- YAML file validation (if yamllint available)

**Style Guidelines:**
- Follows Russ Cox style guidelines as specified in CLAUDE.md
- Enforces Go project commit message format: `package: description`
- Fast-fail on first error for quick feedback

**Integration with Git:**
To run automatically on commit, add to your `.git/hooks/pre-commit`:
```bash
#!/bin/bash
./scripts/pre-commit.sh
```

## CI/CD Integration

The pre-commit script mirrors the checks run in GitHub Actions CI pipeline:
- Same formatting and linting rules
- Same build verification
- Same test compilation checks
- Consistent error messages and feedback

This ensures local development matches CI environment and prevents CI failures.
