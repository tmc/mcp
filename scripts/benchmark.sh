#!/bin/bash

# MCP Go Performance Benchmark Script
# 
# This script runs comprehensive performance benchmarks for the MCP Go implementation,
# generates reports, and compares against baseline performance metrics.

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCHMARK_OUTPUT_DIR="$PROJECT_ROOT/benchmark-results"
BASELINE_FILE="$BENCHMARK_OUTPUT_DIR/baseline.txt"
CURRENT_FILE="$BENCHMARK_OUTPUT_DIR/current.txt"
REPORT_FILE="$BENCHMARK_OUTPUT_DIR/performance-report.md"
PROFILE_DIR="$BENCHMARK_OUTPUT_DIR/profiles"

# Benchmark configuration
BENCHMARK_TIME=${BENCHMARK_TIME:-"10s"}
BENCHMARK_COUNT=${BENCHMARK_COUNT:-5}
CPU_PROFILE=${CPU_PROFILE:-false}
MEM_PROFILE=${MEM_PROFILE:-false}
VERBOSE=${VERBOSE:-false}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OPTIONS:
    -t, --time DURATION      Benchmark duration per test (default: 10s)
    -c, --count NUM          Number of benchmark runs (default: 5)
    --cpu-profile           Enable CPU profiling
    --mem-profile          Enable memory profiling
    --compare              Compare with baseline (requires baseline file)
    --set-baseline         Set current run as new baseline
    --verbose              Enable verbose output
    --clean                Clean benchmark results directory
    -h, --help             Show this help message

EXAMPLES:
    $0                                    # Run standard benchmarks
    $0 -t 30s -c 10                     # Extended benchmarks
    $0 --cpu-profile --mem-profile      # Run with profiling
    $0 --compare                        # Compare with baseline
    $0 --set-baseline                   # Set new baseline

ENVIRONMENT VARIABLES:
    BENCHMARK_TIME         Benchmark duration (default: 10s)
    BENCHMARK_COUNT        Number of benchmark runs (default: 5)
    CPU_PROFILE           Enable CPU profiling (true/false)
    MEM_PROFILE           Enable memory profiling (true/false)
    VERBOSE               Enable verbose output (true/false)
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--time)
            BENCHMARK_TIME="$2"
            shift 2
            ;;
        -c|--count)
            BENCHMARK_COUNT="$2"
            shift 2
            ;;
        --cpu-profile)
            CPU_PROFILE=true
            shift
            ;;
        --mem-profile)
            MEM_PROFILE=true
            shift
            ;;
        --compare)
            COMPARE_MODE=true
            shift
            ;;
        --set-baseline)
            SET_BASELINE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --clean)
            CLEAN_MODE=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Clean benchmark results if requested
if [[ "${CLEAN_MODE:-false}" == "true" ]]; then
    log_info "Cleaning benchmark results directory..."
    rm -rf "$BENCHMARK_OUTPUT_DIR"
    log_success "Benchmark results cleaned"
    exit 0
fi

# Create benchmark output directory
mkdir -p "$BENCHMARK_OUTPUT_DIR"
mkdir -p "$PROFILE_DIR"

cd "$PROJECT_ROOT"

# Verify Go environment
if ! command -v go &> /dev/null; then
    log_error "Go is not installed or not in PATH"
    exit 1
fi

log_info "Go version: $(go version)"
log_info "Project root: $PROJECT_ROOT"
log_info "Benchmark output: $BENCHMARK_OUTPUT_DIR"

# Build test binary to ensure everything compiles
log_info "Building test binary..."
if ! go test -c -o /tmp/mcp-benchmark-test . 2>/dev/null; then
    log_error "Failed to build test binary. Please fix compilation errors first."
    exit 1
fi
rm -f /tmp/mcp-benchmark-test

# Benchmark functions to run
BENCHMARK_PATTERNS=(
    # Core operations
    "BenchmarkClient_Initialize"
    "BenchmarkClient_CallTool"
    "BenchmarkClient_ListTools"
    "BenchmarkServer_HandleRequest"
    
    # Transport layer
    "BenchmarkTransport_ReadWrite"
    
    # JSON processing
    "BenchmarkJSON_Marshal"
    "BenchmarkJSON_Unmarshal"
    
    # Authentication
    "BenchmarkTokenCreation"
    "BenchmarkTokenValidation"
    "BenchmarkConcurrentTokenOperations"
    "BenchmarkTokenRotation"
    "BenchmarkPKCEVerification"
    "BenchmarkAuthorizationHeaderParsing"
    "BenchmarkAuthMemoryAllocation"
    
    # Middleware
    "BenchmarkMiddlewareChainOverhead"
    "BenchmarkRateLimiting_UnderLoad"
    "BenchmarkLoggingMiddleware_Impact"
    "BenchmarkAuthMiddleware_CacheScenarios"
    "BenchmarkMetricsMiddleware"
    "BenchmarkRecoveryMiddleware"
    "BenchmarkMiddlewareMemoryAllocation"
    
    # Concurrency
    "BenchmarkConcurrency_ClientOperations"
    
    # Memory allocation
    "BenchmarkMemory_AllocationPatterns"
    
    # Stress tests (only in extended mode)
    # "BenchmarkStress_HighThroughput"
    # "BenchmarkStress_MemoryPressure"
    
    # Performance regression detection
    "BenchmarkPerformanceBaseline"
    
    # Bottleneck analysis
    "BenchmarkBottleneckAnalysis_JSONProcessing"
    "BenchmarkBottleneckAnalysis_ContextOverhead"
    "BenchmarkBottleneckAnalysis_GoroutineOverhead"
)

# Extended benchmarks (longer running, more comprehensive)
EXTENDED_BENCHMARKS=(
    "BenchmarkStress_HighThroughput"
    "BenchmarkStress_MemoryPressure"
    "BenchmarkAuthStress_MultiClient"
    "BenchmarkAuthMemoryPressure"
    "BenchmarkMiddlewareMemoryPressure"
    "BenchmarkWithCPUProfile"
    "BenchmarkWithMemoryProfile"
)

run_benchmarks() {
    local output_file="$1"
    local patterns=("${@:2}")
    
    log_info "Running benchmarks with the following configuration:"
    log_info "  Duration: $BENCHMARK_TIME"
    log_info "  Count: $BENCHMARK_COUNT"
    log_info "  CPU Profile: $CPU_PROFILE"
    log_info "  Memory Profile: $MEM_PROFILE"
    log_info "  Output: $output_file"
    
    # Prepare benchmark arguments
    local bench_args=()
    bench_args+=("-bench=${patterns[*]}")
    bench_args+=("-benchtime=$BENCHMARK_TIME")
    bench_args+=("-count=$BENCHMARK_COUNT")
    bench_args+=("-benchmem")
    
    if [[ "$VERBOSE" == "true" ]]; then
        bench_args+=("-v")
    fi
    
    # Add profiling flags
    if [[ "$CPU_PROFILE" == "true" ]]; then
        bench_args+=("-cpuprofile=$PROFILE_DIR/cpu.prof")
    fi
    
    if [[ "$MEM_PROFILE" == "true" ]]; then
        bench_args+=("-memprofile=$PROFILE_DIR/mem.prof")
    fi
    
    # Run benchmarks
    log_info "Starting benchmark execution..."
    local start_time=$(date +%s)
    
    if go test "${bench_args[@]}" . > "$output_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Benchmarks completed in ${duration}s"
    else
        log_error "Benchmarks failed. Check output file: $output_file"
        if [[ "$VERBOSE" == "true" ]]; then
            log_error "Last 20 lines of output:"
            tail -20 "$output_file" >&2
        fi
        return 1
    fi
}

generate_report() {
    local current_file="$1"
    local baseline_file="$2"
    local report_file="$3"
    
    log_info "Generating performance report..."
    
    cat > "$report_file" << EOF
# MCP Go Performance Benchmark Report

Generated on: $(date)
Go Version: $(go version)
Platform: $(uname -s)/$(uname -m)
CPU Info: $(grep -m1 'model name' /proc/cpuinfo 2>/dev/null || echo "N/A")

## Configuration

- Benchmark Time: $BENCHMARK_TIME
- Benchmark Count: $BENCHMARK_COUNT
- CPU Profiling: $CPU_PROFILE
- Memory Profiling: $MEM_PROFILE

## Summary

EOF

    # Extract key metrics from current run
    if [[ -f "$current_file" ]]; then
        log_info "Extracting metrics from current run..."
        
        echo "### Current Run Results" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        
        # Extract benchmark results (filter out build/test output)
        grep -E "^Benchmark" "$current_file" | head -20 >> "$report_file" 2>/dev/null || true
        
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
        
        # Extract performance baseline metrics
        if grep -q "BenchmarkPerformanceBaseline" "$current_file"; then
            echo "### Performance Baseline Metrics" >> "$report_file"
            echo "" >> "$report_file"
            grep -A 5 -B 5 "BenchmarkPerformanceBaseline" "$current_file" >> "$report_file" || true
            echo "" >> "$report_file"
        fi
    fi
    
    # Compare with baseline if available
    if [[ -f "$baseline_file" ]] && [[ "${COMPARE_MODE:-false}" == "true" ]]; then
        log_info "Comparing with baseline..."
        
        echo "### Comparison with Baseline" >> "$report_file"
        echo "" >> "$report_file"
        
        # Simple comparison (could be enhanced with benchcmp or similar tools)
        echo "Baseline file: $baseline_file" >> "$report_file"
        echo "Current file: $current_file" >> "$report_file"
        echo "" >> "$report_file"
        
        # Extract and compare key benchmarks
        local key_benchmarks=("BenchmarkClient_CallTool" "BenchmarkTokenValidation" "BenchmarkMiddlewareChainOverhead")
        
        for benchmark in "${key_benchmarks[@]}"; do
            echo "#### $benchmark" >> "$report_file"
            echo "" >> "$report_file"
            
            echo "Baseline:" >> "$report_file"
            echo '```' >> "$report_file"
            grep "$benchmark" "$baseline_file" | head -3 >> "$report_file" 2>/dev/null || echo "Not found" >> "$report_file"
            echo '```' >> "$report_file"
            
            echo "Current:" >> "$report_file"
            echo '```' >> "$report_file"
            grep "$benchmark" "$current_file" | head -3 >> "$report_file" 2>/dev/null || echo "Not found" >> "$report_file"
            echo '```' >> "$report_file"
            echo "" >> "$report_file"
        done
    fi
    
    # Add bottleneck analysis
    echo "### Bottleneck Analysis" >> "$report_file"
    echo "" >> "$report_file"
    
    if grep -q "BenchmarkBottleneckAnalysis" "$current_file"; then
        echo "#### Identified Performance Areas" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        grep "BenchmarkBottleneckAnalysis" "$current_file" >> "$report_file" || true
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add memory allocation analysis
    echo "### Memory Allocation Analysis" >> "$report_file"
    echo "" >> "$report_file"
    
    if grep -q "allocs/op" "$current_file"; then
        echo "Top memory allocating benchmarks:" >> "$report_file"
        echo '```' >> "$report_file"
        grep "allocs/op" "$current_file" | sort -k5 -nr | head -10 >> "$report_file" || true
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add profiling information if available
    if [[ "$CPU_PROFILE" == "true" ]] && [[ -f "$PROFILE_DIR/cpu.prof" ]]; then
        echo "### CPU Profiling" >> "$report_file"
        echo "" >> "$report_file"
        echo "CPU profile saved to: \`$PROFILE_DIR/cpu.prof\`" >> "$report_file"
        echo "" >> "$report_file"
        echo "To analyze:" >> "$report_file"
        echo '```bash' >> "$report_file"
        echo "go tool pprof $PROFILE_DIR/cpu.prof" >> "$report_file"
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if [[ "$MEM_PROFILE" == "true" ]] && [[ -f "$PROFILE_DIR/mem.prof" ]]; then
        echo "### Memory Profiling" >> "$report_file"
        echo "" >> "$report_file"
        echo "Memory profile saved to: \`$PROFILE_DIR/mem.prof\`" >> "$report_file"
        echo "" >> "$report_file"
        echo "To analyze:" >> "$report_file"
        echo '```bash' >> "$report_file"
        echo "go tool pprof $PROFILE_DIR/mem.prof" >> "$report_file"
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add optimization recommendations
    echo "### Optimization Opportunities" >> "$report_file"
    echo "" >> "$report_file"
    
    # Analyze results and provide recommendations
    if grep -q "BenchmarkJSON" "$current_file"; then
        echo "#### JSON Processing" >> "$report_file"
        local json_performance=$(grep "BenchmarkJSON" "$current_file" | head -1)
        echo "- Current JSON processing performance: \`$json_performance\`" >> "$report_file"
        echo "- Consider using streaming JSON parsing for large payloads" >> "$report_file"
        echo "- Implement JSON schema validation caching" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if grep -q "BenchmarkMiddleware" "$current_file"; then
        echo "#### Middleware Chain" >> "$report_file"
        local middleware_overhead=$(grep "BenchmarkMiddlewareChainOverhead" "$current_file" | head -1)
        echo "- Current middleware overhead: \`$middleware_overhead\`" >> "$report_file"
        echo "- Consider middleware ordering optimization" >> "$report_file"
        echo "- Implement conditional middleware execution" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if grep -q "BenchmarkAuth" "$current_file"; then
        echo "#### Authentication" >> "$report_file"
        local auth_performance=$(grep "BenchmarkTokenValidation" "$current_file" | head -1)
        echo "- Current token validation performance: \`$auth_performance\`" >> "$report_file"
        echo "- Optimize token cache hit ratio" >> "$report_file"
        echo "- Consider token pre-validation for high-frequency operations" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    echo "### Files Generated" >> "$report_file"
    echo "" >> "$report_file"
    echo "- Benchmark results: \`$current_file\`" >> "$report_file"
    echo "- Performance report: \`$report_file\`" >> "$report_file"
    
    if [[ "$CPU_PROFILE" == "true" ]]; then
        echo "- CPU profile: \`$PROFILE_DIR/cpu.prof\`" >> "$report_file"
    fi
    
    if [[ "$MEM_PROFILE" == "true" ]]; then
        echo "- Memory profile: \`$PROFILE_DIR/mem.prof\`" >> "$report_file"
    fi
    
    log_success "Performance report generated: $report_file"
}

# Main execution
main() {
    log_info "Starting MCP Go Performance Benchmark"
    log_info "======================================"
    
    # Run benchmarks
    if ! run_benchmarks "$CURRENT_FILE" "${BENCHMARK_PATTERNS[@]}"; then
        log_error "Benchmark execution failed"
        exit 1
    fi
    
    # Set baseline if requested
    if [[ "${SET_BASELINE:-false}" == "true" ]]; then
        log_info "Setting current run as new baseline..."
        cp "$CURRENT_FILE" "$BASELINE_FILE"
        log_success "Baseline updated: $BASELINE_FILE"
    fi
    
    # Generate report
    generate_report "$CURRENT_FILE" "$BASELINE_FILE" "$REPORT_FILE"
    
    # Display summary
    log_info "======================================"
    log_success "Benchmark execution completed"
    log_info "Results saved to: $BENCHMARK_OUTPUT_DIR"
    
    if [[ -f "$REPORT_FILE" ]]; then
        log_info "Performance report:"
        echo ""
        head -50 "$REPORT_FILE"
        echo ""
        log_info "Full report: $REPORT_FILE"
    fi
    
    # Show quick performance metrics
    if [[ -f "$CURRENT_FILE" ]]; then
        echo ""
        log_info "Quick Performance Metrics:"
        echo "=========================="
        
        # Extract key performance indicators
        local baseline_ops=$(grep "BenchmarkPerformanceBaseline" "$CURRENT_FILE" | awk '{print $3}' | head -1)
        if [[ -n "$baseline_ops" ]]; then
            log_success "Baseline operations/sec: $baseline_ops"
        fi
        
        # Show top 3 fastest benchmarks
        echo ""
        log_info "Top performing benchmarks:"
        grep "^Benchmark" "$CURRENT_FILE" | sort -k3 -nr | head -3 | while read line; do
            log_success "$line"
        done
        
        # Show potential issues (slow benchmarks)
        echo ""
        log_info "Potential bottlenecks (slowest benchmarks):"
        grep "^Benchmark" "$CURRENT_FILE" | sort -k3 -n | head -3 | while read line; do
            log_warning "$line"
        done
    fi
    
    echo ""
    log_info "To analyze profiles, use:"
    if [[ "$CPU_PROFILE" == "true" ]]; then
        echo "  go tool pprof $PROFILE_DIR/cpu.prof"
    fi
    if [[ "$MEM_PROFILE" == "true" ]]; then
        echo "  go tool pprof $PROFILE_DIR/mem.prof"
    fi
    
    echo ""
    log_info "To compare with baseline next time, use:"
    echo "  $0 --compare"
}

# Run main function
main "$@"