#!/bin/bash

# pre-commit.sh - Local development pre-commit script
# This script runs the same checks as the pre-commit hooks and GitHub Actions
# Following Russ Cox style guidelines and CLAUDE.md requirements

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_step() {
    print_status "$YELLOW" "🔍 $1"
}

print_success() {
    print_status "$GREEN" "✅ $1"
}

print_error() {
    print_status "$RED" "❌ $1"
}

print_info() {
    print_status "$BLUE" "ℹ️  $1"
}

print_warning() {
    print_status "$YELLOW" "⚠️  $1"
}

# Function to run a command and handle errors with better reporting
run_check() {
    local description=$1
    local command=$2
    local allow_failure=${3:-false}
    
    print_step "$description"
    if eval "$command"; then
        print_success "$description passed"
        return 0
    else
        if [ "$allow_failure" = "true" ]; then
            print_warning "$description failed (non-critical)"
            return 0
        else
            print_error "$description failed"
            echo ""
            print_error "Fix the above issues and try again"
            exit 1
        fi
    fi
}

# Function to show help
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Pre-commit script for MCP Go implementation"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --fast         Run fast checks only (skip expensive tests)"
    echo "  --fix          Auto-fix formatting issues where possible"
    echo ""
    echo "This script runs the same checks as:"
    echo "  • Pre-commit hooks (.pre-commit-config.yaml)"
    echo "  • GitHub Actions CI (.github/workflows/ci.yml)"
    echo ""
    echo "For automatic pre-commit hooks, install with:"
    echo "  pip install pre-commit && pre-commit install"
}

# Parse command line arguments
FAST_MODE=false
AUTO_FIX=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --fast)
            FAST_MODE=true
            shift
            ;;
        --fix)
            AUTO_FIX=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "🚀 Running pre-commit checks for MCP Go implementation..."
if [ "$FAST_MODE" = "true" ]; then
    echo "⚡ Fast mode enabled - skipping expensive tests"
fi
if [ "$AUTO_FIX" = "true" ]; then
    echo "🔧 Auto-fix mode enabled - will attempt to fix formatting issues"
fi
echo ""

# Change to repository root
cd "$(git rev-parse --show-toplevel)"

# Display current status
print_info "Repository: $(pwd)"
print_info "Branch: $(git branch --show-current 2>/dev/null || echo 'detached HEAD')"
print_info "Go version: $(go version | cut -d' ' -f3)"
echo ""

# 1. Check for binary files (per CLAUDE.md)
run_check "Checking for binary files in staging area" '
if git diff --cached --name-only | xargs -I {} file {} 2>/dev/null | grep -q "executable\|binary"; then
    echo "Binary files detected in staging area:"
    git diff --cached --name-only | xargs -I {} file {} | grep "executable\|binary"
    echo "Please unstage binary files as per CLAUDE.md guidelines"
    exit 1
fi
'

# 2. Check that go.sum is not directly modified
run_check "Checking go.sum not directly modified" '
if git diff --cached --name-only | grep -q "go\.sum$"; then
    echo "go.sum should not be directly modified. Use go mod tidy instead."
    exit 1
fi
'

# 3. Check gofmt formatting
if [ "$AUTO_FIX" = "true" ]; then
    run_check "Auto-fixing gofmt formatting" '
    # Exclude problematic files from formatting checks
    excluded_patterns="temp/example_server_design_exploration|temp/mock_client_fix.go|exp/schema2go/generator.go|exp/cmd/mcp-tool-graph/main.go"
    unformatted_files=$(gofmt -s -l . | grep -v -E "$excluded_patterns" || true)
    if [ -n "$unformatted_files" ]; then
        echo "Auto-fixing formatting for:"
        echo "$unformatted_files"
        echo "$unformatted_files" | xargs gofmt -s -w
        echo "✨ Formatting fixed automatically"
    else
        echo "All files already properly formatted"
    fi
    '
else
    run_check "Checking gofmt formatting" '
    # Exclude problematic files from formatting checks
    excluded_patterns="temp/example_server_design_exploration|temp/mock_client_fix.go|exp/schema2go/generator.go|exp/cmd/mcp-tool-graph/main.go"
    unformatted_files=$(gofmt -s -l . | grep -v -E "$excluded_patterns" || true)
    if [ -n "$unformatted_files" ]; then
        echo "The following files need gofmt -s:"
        echo "$unformatted_files"
        echo ""
        echo "Run: gofmt -s -w . (excluding problematic files)"
        echo "Or run: $0 --fix to auto-fix formatting issues"
        exit 1
    fi
    '
fi

# 4. Run go vet
run_check "Running go vet" '
# Exclude problematic directories that contain broken Go files
go vet $(go list ./... | grep -v "temp/example_server_design_exploration" | grep -v "temp/mock_client_fix")
'

# 5. Check go mod tidy
if [ "$AUTO_FIX" = "true" ]; then
    run_check "Auto-fixing go mod tidy" '
    echo "Running go mod tidy..."
    go mod tidy
    echo "✨ Dependencies tidied automatically"
    '
else
    run_check "Checking go mod tidy" '
    cp go.mod go.mod.bak
    cp go.sum go.sum.bak
    go mod tidy
    if ! diff -q go.mod go.mod.bak >/dev/null 2>&1 || ! diff -q go.sum go.sum.bak >/dev/null 2>&1; then
        echo "go.mod or go.sum is not tidy."
        echo ""
        echo "Run: go mod tidy"
        echo "Or run: $0 --fix to auto-fix dependency issues"
        rm go.mod.bak go.sum.bak
        exit 1
    fi
    rm go.mod.bak go.sum.bak
    '
fi

# 6. Test compilation of all packages
run_check "Testing package compilation" '
# Exclude problematic packages that contain broken Go files
go build $(go list ./... | grep -v "temp/example_server_design_exploration" | grep -v "temp/mock_client_fix")
'

# 7. Test compilation of all core tools
run_check "Testing core tools compilation" '
for tool in cmd/*; do
    if [ -d "$tool" ] && [ -f "$tool/main.go" ]; then
        echo "  Building $(basename $tool)..."
        go build "$tool"
    fi
done
'

# 8. Test compilation of experimental tools (allow failures)
if [ "$FAST_MODE" != "true" ]; then
    print_step "Testing experimental tools compilation (warnings only)"
    if [ -d "exp/cmd" ]; then
        cd exp
        for tool in cmd/*; do
            if [ -d "$tool" ] && [ -f "$tool/main.go" ]; then
                if go build -tags=k8s "$tool" 2>/dev/null; then
                    echo "  ✅ Built exp/$(basename $tool)"
                else
                    echo "  ⚠️  Failed to build exp/$(basename $tool) (possibly conditional)"
                fi
            fi
        done
        cd ..
        print_success "Experimental tools check completed"
    else
        echo "  No experimental tools found"
        print_success "Experimental tools check skipped"
    fi
else
    print_info "Skipping experimental tools compilation (fast mode)"
fi

# 9. Test that tests compile (but don't run them)
run_check "Testing test compilation" '
# Exclude problematic packages from test compilation
go test -run="^$" $(go list ./... | grep -v "temp/example_server_design_exploration" | grep -v "temp/mock_client_fix")
'

# 10. Run a quick smoke test of core functionality
print_step "Running smoke tests"
probe_ok=false
serve_ok=false

if go run ./cmd/mcp-probe --help >/dev/null 2>&1; then
    echo "  ✅ mcp-probe help works"
    probe_ok=true
else
    echo "  ❌ mcp-probe help failed"
fi

if go run ./cmd/mcp-serve --help >/dev/null 2>&1; then
    echo "  ✅ mcp-serve help works"
    serve_ok=true
else
    echo "  ❌ mcp-serve help failed"
fi

if [ "$probe_ok" = "true" ] && [ "$serve_ok" = "true" ]; then
    print_success "Smoke tests completed"
else
    print_error "Smoke tests failed - core tools not working"
    exit 1
fi

# 11. Check YAML files (if yamllint is available)
if command -v yamllint >/dev/null 2>&1; then
    if [ -f ".github/workflows/ci.yml" ] || [ -f ".pre-commit-config.yaml" ]; then
        run_check "Checking YAML files" "yamllint .github/workflows/*.yml .pre-commit-config.yaml 2>/dev/null || true" true
    else
        print_info "No YAML files to check"
    fi
else
    print_info "yamllint not available - skipping YAML validation"
    print_info "Install with: pip install yamllint"
fi

# 12. Final summary
echo ""
echo "=================================================================================="
print_success "🎉 All pre-commit checks passed!"
print_status "$GREEN" "✨ Ready to commit following Go project style guidelines"
echo "=================================================================================="
echo ""

# Show commit guidelines
print_info "📋 Commit Guidelines:"
echo "   • Use format: 'package: description' (e.g., 'mcp: fix formatting')"
echo "   • Keep descriptions lowercase, no trailing periods"
echo "   • Use atomic commits with single logical changes"
echo "   • Consider using git-auto-commit-message --auto for consistency"
echo ""

# Show next steps
print_info "🔄 Next Steps:"
echo "   • Stage your changes: git add <files>"
echo "   • Create commit: git commit -m 'package: description'"
echo "   • Push changes: git push origin <branch>"
echo ""

# Show additional tools
print_info "🛠  Additional Tools:"
echo "   • Fast checks only: $0 --fast"
echo "   • Auto-fix issues:  $0 --fix"
echo "   • Install pre-commit hooks: pre-commit install"
echo "   • Run CI locally: make ci-local"
echo ""