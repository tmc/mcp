# Scripts

This directory contains development and CI/CD scripts for the MCP Go implementation.

## Scripts

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