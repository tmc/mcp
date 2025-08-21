# CI/CD Pipeline for MCP Go Implementation

This document describes the comprehensive CI/CD setup for the MCP Go implementation, following Russ Cox style guidelines and CLAUDE.md requirements.

## Overview

The CI/CD pipeline consists of three main components:

1. **GitHub Actions CI** (`.github/workflows/ci.yml`) - Automated testing and validation
2. **Pre-commit Hooks** (`.pre-commit-config.yaml`) - Local development quality gates
3. **Local Pre-commit Script** (`scripts/pre-commit.sh`) - Manual quality checks

## GitHub Actions CI Pipeline

### Workflow Structure

The CI pipeline runs on:
- Push to `main`, `next`, `develop` branches
- Pull requests to `main`, `next` branches  
- Manual workflow dispatch

### Jobs and Matrix Testing

#### 1. Code Quality (`quality`)
- **Platform**: ubuntu-latest
- **Checks**:
  - gofmt formatting validation
  - go vet static analysis
  - go mod tidy verification
  - Binary file detection (per CLAUDE.md)
  - golangci-lint comprehensive linting

#### 2. Build Verification (`build`)
- **Matrix**: ubuntu-latest, macos-latest
- **Actions**:
  - Build all packages (excluding problematic temp files)
  - Build core tools in `cmd/`
  - Build experimental tools in `exp/cmd/` (with conditional dependencies)
  - Verify tool functionality with `--help` tests

#### 3. Test Matrix (`test`)
- **Go Versions**: 1.20, 1.21, 1.22, 1.23
- **Platforms**: ubuntu-latest, macos-latest
- **Features**:
  - Race detection tests (Go 1.23 on Linux)
  - Synctest validation (Go 1.23 on Linux with `GOEXPERIMENT=synctest`)
  - JUnit XML test result artifacts
  - Test timeout: 10 minutes

#### 4. Coverage Reporting (`coverage`)
- **Platform**: ubuntu-latest
- **Features**:
  - Atomic coverage mode
  - HTML coverage reports
  - Codecov integration
  - Coverage threshold checking (45.0%)
  - 30-day artifact retention

#### 5. Integration Tests (`integration`)
- **Platform**: ubuntu-latest
- **Timeout**: 15 minutes
- **Tests**:
  - Tool integration verification
  - mcpscripttest framework tests
  - Basic server functionality

#### 6. Security Checks (`security`)
- **Platform**: ubuntu-latest
- **Tools**:
  - Gosec security scanner with SARIF output
  - Go vulnerability scanning with `govulncheck`
  - GitHub security events integration

#### 7. CI Success Gate (`ci-success`)
- **Dependencies**: All previous jobs
- **Purpose**: Final status validation for merge requirements

### Environment Variables

```yaml
CGO_ENABLED: 0                    # Pure Go builds
GO_VERSION_FILE: go.mod          # Use go.mod for Go version
GOTESTSUM_FORMAT: standard-verbose # Test output format  
TEST_TIMEOUT: 10m                # Test timeout
COVERAGE_THRESHOLD: 45.0         # Minimum coverage percentage
```

## Pre-commit Hooks

### Installation

```bash
pip install pre-commit
pre-commit install
```

### Hook Categories

#### General File Checks
- Large file prevention (500KB limit)
- Binary file detection
- Merge conflict detection  
- YAML validation
- Whitespace fixing

#### Go-Specific Hooks
- **Binary file check**: Prevents committing binaries per CLAUDE.md
- **go.sum protection**: Prevents direct go.sum modification
- **gofmt formatting**: Auto-fixes or validates Go formatting
- **go vet**: Static analysis
- **go mod tidy**: Dependency management validation
- **Build testing**: Package and test compilation
- **Tool building**: Core tool compilation verification
- **Smoke tests**: Basic functionality verification

#### Security and Linting
- Go module vendoring
- Cyclomatic complexity checking (threshold: 15)
- Unit test validation
- YAML linting

### Configuration

```yaml
# Auto-fix PRs with pre-commit.ci
ci:
  autofix_prs: true
  autoupdate_schedule: weekly
  skip: [go-unit-tests, smoke-test]  # Skip expensive tests
```

## Local Pre-commit Script

### Usage

```bash
# Run all checks
./scripts/pre-commit.sh

# Fast mode (skip expensive tests)  
./scripts/pre-commit.sh --fast

# Auto-fix issues
./scripts/pre-commit.sh --fix

# Show help
./scripts/pre-commit.sh --help
```

### Features

#### Command Line Options
- `--help`: Show usage information
- `--fast`: Skip expensive operations (experimental tools, etc.)
- `--fix`: Automatically fix formatting and dependency issues

#### Check Categories
1. **Binary file detection** (per CLAUDE.md requirement)
2. **go.sum protection** (prevents direct modification)  
3. **Code formatting** (gofmt with auto-fix support)
4. **Static analysis** (go vet)
5. **Dependency management** (go mod tidy with auto-fix)
6. **Compilation testing** (packages and tests)
7. **Tool building** (core and experimental tools)
8. **Smoke testing** (basic functionality verification)
9. **YAML validation** (if yamllint available)

#### Output Features
- Colored output with clear status indicators
- Progress tracking with step-by-step reporting
- Detailed error messages with fix suggestions
- Comprehensive final summary with guidelines

## Development Workflow Integration

### Pre-commit Integration

The three components work together to ensure code quality:

1. **Local Development**: Use `scripts/pre-commit.sh` for manual checks
2. **Git Hooks**: Pre-commit hooks run automatically on `git commit`
3. **CI Pipeline**: GitHub Actions validates all changes in CI

### Commit Standards

Following Go project conventions:

```bash
# Format: package: description
git commit -m "mcp: fix code formatting issues"
git commit -m "middleware: add compression support" 
git commit -m "all: update dependencies"
```

### Quality Gates

#### Local Quality Gates
- Code formatting (gofmt)
- Static analysis (go vet)
- Dependency tidiness (go mod tidy)
- Compilation verification
- Basic functionality testing

#### CI Quality Gates  
- Multi-version Go compatibility
- Cross-platform builds
- Race condition detection
- Security scanning
- Coverage requirements
- Integration testing

## File Structure

```
.github/
  workflows/
    ci.yml                    # Main CI pipeline
.pre-commit-config.yaml       # Pre-commit hook configuration
.yamllint                     # YAML linting configuration
scripts/
  pre-commit.sh              # Local pre-commit script
docs/
  CI_CD.md                   # This documentation
```

## Troubleshooting

### Common Issues

#### Build Failures
```bash
# Check excluded patterns match problematic files
excluded_patterns="temp/example_server_design_exploration|temp/mock_client_fix.go"
```

#### Test Failures
```bash
# Run tests with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run with synctest
GOEXPERIMENT=synctest go test -tags=synctest ./...
```

#### Pre-commit Issues
```bash
# Update hooks to latest versions
pre-commit autoupdate

# Run specific hook
pre-commit run gofmt

# Skip hooks temporarily  
git commit --no-verify
```

### Performance Optimization

#### Fast Development Cycle
```bash
# Use fast mode for quick checks
./scripts/pre-commit.sh --fast

# Auto-fix common issues
./scripts/pre-commit.sh --fix
```

#### CI Optimization  
- Parallel job execution with dependencies
- Caching for Go modules and build artifacts
- Conditional expensive operations
- Matrix optimization for coverage

## Security Considerations

### Access Controls
- **GitHub Actions**: Limited permissions with `contents: read`
- **Security Events**: Write access for SARIF uploads only
- **Artifacts**: Limited retention periods

### Vulnerability Management
- **govulncheck**: Automated vulnerability scanning
- **Gosec**: Security-focused static analysis
- **Dependency Scanning**: Regular security updates

### Binary File Protection
Per CLAUDE.md requirements:
- Pre-commit hooks prevent binary file commits
- CI validates no binaries in repository
- Clear error messages guide developers

## Maintenance

### Regular Updates
- **Weekly**: Pre-commit hook updates via pre-commit.ci
- **Monthly**: Review and update Go versions in CI matrix
- **Quarterly**: Review security scanning tools and thresholds

### Monitoring
- **Coverage Trends**: Monitor coverage improvements over time
- **Build Performance**: Track CI execution times
- **Security Alerts**: Respond to vulnerability reports

### Documentation
- Keep this document synchronized with pipeline changes
- Update examples when adding new tools or checks
- Document any project-specific exclusions or exceptions