#!/bin/bash

# Advanced Fuzzing Infrastructure Demo Runner
echo "🚀 Advanced Fuzzing Infrastructure Demo"
echo "======================================"

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

echo "✅ Go found: $(go version)"

# Set up environment
export DEMO_VERBOSE=false
export COVERAGE_DEBUG=false
export FUZZ_DEBUG=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            export DEMO_VERBOSE=true
            echo "🔍 Verbose mode enabled"
            shift
            ;;
        --coverage-debug)
            export COVERAGE_DEBUG=true
            echo "📊 Coverage debug enabled"
            shift
            ;;
        --fuzz-debug)
            export FUZZ_DEBUG=true
            echo "🎯 Fuzzing debug enabled"
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose      Enable verbose output"
            echo "  --coverage-debug   Enable coverage debug output"
            echo "  --fuzz-debug       Enable fuzzing debug output"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            echo "❌ Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Create module if it doesn't exist
if [ ! -f "go.mod" ]; then
    echo "📦 Initializing Go module..."
    go mod init advanced-fuzzing-demo
fi

# Run the demo
echo "🎬 Starting demo..."
echo ""

go run main.go

echo ""
echo "✅ Demo completed successfully!"

# Show generated files
if [ -f "demo_report.json" ]; then
    echo "📄 Generated files:"
    echo "   • demo_report.json - Detailed demo report"
    
    # Show report summary if jq is available
    if command -v jq &> /dev/null; then
        echo ""
        echo "📊 Quick Report Summary:"
        echo "   Coverage: $(jq -r '.coverage.final_coverage * 100 | floor')%"
        echo "   Sessions: $(jq -r '.fuzzing.total_sessions')"
        echo "   Generated: $(jq -r '.grammar.total_generated')"
    fi
else
    echo "⚠️  No report file generated"
fi

echo ""
echo "🎉 Demo complete! Check the output above for results."