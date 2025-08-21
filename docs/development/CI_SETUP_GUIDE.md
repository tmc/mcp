# CI/CD Setup Guide

Quick setup guide for developers working with the MCP Go implementation.

## Quick Start

### 1. Install Pre-commit (Recommended)

```bash
# Install pre-commit
pip install pre-commit

# Install hooks in your repo
cd /path/to/mcp
pre-commit install

# Test the setup
pre-commit run --all-files
```

### 2. Use Local Script (Alternative)

```bash
# Run all checks
./scripts/pre-commit.sh

# Fast mode for quick feedback
./scripts/pre-commit.sh --fast

# Auto-fix formatting issues
./scripts/pre-commit.sh --fix
```

### 3. Integrate with Your IDE

#### VS Code
Add to `.vscode/settings.json`:
```json
{
  "go.formatTool": "goimports",
  "go.lintOnSave": "package",
  "go.vetOnSave": "package",
  "editor.formatOnSave": true,
  "git.enableSmartCommit": false
}
```

#### GoLand/IntelliJ
- Enable "Go fmt on save"
- Enable "Go vet on save"
- Set up pre-commit hook integration

## Development Workflow

### Daily Development

1. **Before coding:**
   ```bash
   git pull origin main
   ./scripts/pre-commit.sh --fast  # Quick check
   ```

2. **While coding:**
   - Pre-commit hooks run automatically on `git commit`
   - Use `./scripts/pre-commit.sh --fix` to auto-fix issues

3. **Before pushing:**
   ```bash
   ./scripts/pre-commit.sh  # Full check
   make test                # Run tests locally
   ```

### Commit Standards

Follow Go project conventions:
```bash
# Good examples
git commit -m "mcp: fix connection handling"
git commit -m "middleware: add timeout support"
git commit -m "all: update dependencies"

# Bad examples (don't do this)
git commit -m "Fix bugs"
git commit -m "Update code"
git commit -m "WIP: stuff"
```

## Troubleshooting

### Common Issues

#### "Binary files detected"
```bash
# Find binary files
git ls-files | xargs file | grep binary

# Remove from staging
git reset HEAD <binary-file>
```

#### "gofmt formatting required"
```bash
# Auto-fix
./scripts/pre-commit.sh --fix

# Or manually
gofmt -s -w .
```

#### "go.sum not tidy"
```bash
# Auto-fix
./scripts/pre-commit.sh --fix

# Or manually  
go mod tidy
```

#### "Tests failing"
```bash
# Run specific test
go test -v ./path/to/package

# Run with race detection
go test -race ./path/to/package

# Check for problematic packages
go test ./... | grep FAIL
```

### Skip Hooks Temporarily

```bash
# Skip pre-commit hooks (use sparingly!)
git commit --no-verify

# Skip specific hook
SKIP=gofmt git commit

# Run specific hook only
pre-commit run gofmt --all-files
```

## CI Pipeline Understanding

### What Runs When

| Event | Jobs |
|-------|------|
| Push to main/next | All jobs (quality, build, test, coverage, integration, security) |
| Pull Request | All jobs |
| Manual dispatch | All jobs |

### Job Dependencies

```
quality ────────┐
build ──────────┤
                ├─→ coverage ─┐
test ───────────┤             ├─→ ci-success
integration ────┤             │
security ───────┘─────────────┘
```

### Expected Duration
- **Quality**: ~2-3 minutes
- **Build**: ~3-4 minutes per OS
- **Test**: ~5-8 minutes per Go version/OS
- **Coverage**: ~3-4 minutes  
- **Integration**: ~5-7 minutes
- **Security**: ~2-3 minutes

**Total**: ~15-25 minutes (parallel execution)

## Performance Tips

### Speed Up Local Development

```bash
# Use fast mode for quick feedback
./scripts/pre-commit.sh --fast

# Auto-fix common issues
./scripts/pre-commit.sh --fix

# Run specific make targets
make fmt vet build
```

### Optimize CI Usage

- **Small PRs**: Faster CI, easier review
- **Atomic commits**: Clear history, easier debugging
- **Draft PRs**: Use for WIP to avoid unnecessary CI runs
- **Squash merging**: Clean main branch history

## Advanced Usage

### Custom Exclusions

Edit `.pre-commit-config.yaml` to exclude specific files:
```yaml
exclude: |
  (?x)^(
    temp/.*|
    your-custom-exclusion/.*|
    \.generated\.go$
  )$
```

### Local CI Simulation

```bash
# Simulate CI locally
make ci-local

# Run specific test matrix
GOEXPERIMENT=synctest go test -tags=synctest ./...
go test -race ./...
```

### Coverage Analysis

```bash
# Generate coverage report
make test-coverage

# View in browser
open coverage/coverage.html
```

## Getting Help

### Resources
- **Main Documentation**: `docs/CI_CD.md`
- **Make Targets**: `make help`
- **Script Help**: `./scripts/pre-commit.sh --help`

### Common Commands Reference

```bash
# Development
./scripts/pre-commit.sh           # Full check
./scripts/pre-commit.sh --fast    # Quick check  
./scripts/pre-commit.sh --fix     # Auto-fix

# Pre-commit
pre-commit run --all-files        # Run all hooks
pre-commit autoupdate             # Update hooks
pre-commit uninstall             # Remove hooks

# Make targets
make test                        # Run tests
make fmt                         # Format code
make vet                         # Static analysis  
make build                       # Build packages
make ci-local                    # Simulate CI

# Git workflow
git add .                        # Stage changes
git commit                       # Commit (runs hooks)
git push origin <branch>         # Push changes
```

## Need Help?

1. **Check this guide** for common solutions
2. **Read error messages** - they usually contain fix instructions  
3. **Use auto-fix mode** - `./scripts/pre-commit.sh --fix`
4. **Review CI logs** - detailed error information in GitHub Actions
5. **Ask team members** - someone else may have seen the issue

Remember: The CI/CD pipeline is designed to help you write better code, not to slow you down. Use the tools and don't fight them!