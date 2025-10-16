#!/usr/bin/env bash
# JSON Performance Profiling Script for MCP Go Implementation
# Identifies bottlenecks in JSON serialization/deserialization

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
PROFILE_DIR="${PROFILE_DIR:-./perf-profiles}"
BENCH_TIME="${BENCH_TIME:-5s}"
BENCH_COUNT="${BENCH_COUNT:-5}"

echo -e "${GREEN}MCP JSON Performance Profiling${NC}"
echo "=============================================="
echo "Profile directory: $PROFILE_DIR"
echo "Benchmark time: $BENCH_TIME"
echo "Benchmark count: $BENCH_COUNT"
echo ""

# Create profile directory
mkdir -p "$PROFILE_DIR"

# Run benchmarks with CPU profiling
echo -e "${YELLOW}1. Running JSON marshaling benchmarks with CPU profiling...${NC}"
go test -bench=BenchmarkJSON -benchmem -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
    -cpuprofile="$PROFILE_DIR/json-cpu.prof" \
    -memprofile="$PROFILE_DIR/json-mem.prof" \
    ./... 2>&1 | tee "$PROFILE_DIR/json-bench-results.txt" || {
    echo -e "${YELLOW}⚠  Some JSON benchmarks may have failed${NC}"
}

echo ""
echo -e "${YELLOW}2. Running protocol marshaling benchmarks...${NC}"
go test -bench=BenchmarkMarshal -benchmem -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
    -cpuprofile="$PROFILE_DIR/protocol-cpu.prof" \
    -memprofile="$PROFILE_DIR/protocol-mem.prof" \
    ./modelcontextprotocol/... 2>&1 | tee "$PROFILE_DIR/protocol-bench-results.txt" || {
    echo -e "${YELLOW}⚠  Some protocol benchmarks may have failed${NC}"
}

echo ""
echo -e "${YELLOW}3. Running transport benchmarks...${NC}"
go test -bench=BenchmarkTransport -benchmem -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
    -cpuprofile="$PROFILE_DIR/transport-cpu.prof" \
    -memprofile="$PROFILE_DIR/transport-mem.prof" \
    . 2>&1 | tee "$PROFILE_DIR/transport-bench-results.txt" || {
    echo -e "${YELLOW}⚠  Some transport benchmarks may have failed${NC}"
}

echo ""
echo -e "${GREEN}Profile Analysis${NC}"
echo "=============================================="

# Analyze CPU profiles if available
if [ -f "$PROFILE_DIR/json-cpu.prof" ]; then
    echo ""
    echo -e "${YELLOW}Top CPU consumers in JSON operations:${NC}"
    go tool pprof -top -cum "$PROFILE_DIR/json-cpu.prof" 2>/dev/null | head -20 || {
        echo "Note: pprof analysis requires benchmarks to have run"
    }

    # Generate flame graph if available
    if command -v go-torch &> /dev/null; then
        echo ""
        echo "Generating flame graph..."
        go-torch -f "$PROFILE_DIR/json-cpu-flame.svg" "$PROFILE_DIR/json-cpu.prof" 2>/dev/null || true
    fi
fi

# Analyze memory profiles if available
if [ -f "$PROFILE_DIR/json-mem.prof" ]; then
    echo ""
    echo -e "${YELLOW}Top memory allocations in JSON operations:${NC}"
    go tool pprof -top -alloc_space "$PROFILE_DIR/json-mem.prof" 2>/dev/null | head -20 || {
        echo "Note: pprof analysis requires benchmarks to have run"
    }
fi

# Parse benchmark results for key metrics
echo ""
echo -e "${GREEN}Benchmark Summary${NC}"
echo "=============================================="

if [ -f "$PROFILE_DIR/json-bench-results.txt" ]; then
    echo ""
    echo "JSON Marshaling Performance:"
    grep -E "Benchmark.*-[0-9]+" "$PROFILE_DIR/json-bench-results.txt" | \
        awk '{printf "  %-50s %10s ns/op  %10s B/op  %8s allocs/op\n", $1, $3, $5, $7}' || true
fi

if [ -f "$PROFILE_DIR/protocol-bench-results.txt" ]; then
    echo ""
    echo "Protocol Marshaling Performance:"
    grep -E "Benchmark.*-[0-9]+" "$PROFILE_DIR/protocol-bench-results.txt" | \
        awk '{printf "  %-50s %10s ns/op  %10s B/op  %8s allocs/op\n", $1, $3, $5, $7}' || true
fi

# Identify hot spots
echo ""
echo -e "${GREEN}Identified Hot Spots${NC}"
echo "=============================================="

# Look for high allocation counts
echo "Functions with high allocation counts:"
if [ -f "$PROFILE_DIR/json-mem.prof" ]; then
    go tool pprof -top -alloc_objects "$PROFILE_DIR/json-mem.prof" 2>/dev/null | \
        grep -E "encoding/json|modelcontextprotocol|jsonrpc2" | head -10 || true
fi

# Generate optimization recommendations
echo ""
echo -e "${GREEN}Optimization Recommendations${NC}"
echo "=============================================="

cat <<EOF
Based on profiling results, consider:

1. **Reduce Allocations:**
   - Use sync.Pool for frequently allocated objects
   - Pre-allocate slices with known capacity
   - Reuse buffers where possible

2. **Optimize JSON Marshaling:**
   - Implement custom MarshalJSON for hot path types
   - Use json.RawMessage for pass-through data
   - Consider using jsoniter or other fast JSON libraries

3. **String Operations:**
   - Use strings.Builder instead of concatenation
   - Avoid unnecessary string<->[]byte conversions
   - Cache computed strings where possible

4. **Interface Conversions:**
   - Minimize interface{} usage in hot paths
   - Use concrete types where possible
   - Cache type assertions

5. **Memory Layout:**
   - Group related fields in structs
   - Use pointer receivers judiciously
   - Consider struct field ordering for alignment
EOF

echo ""
echo -e "${GREEN}Next Steps${NC}"
echo "=============================================="
echo "1. Review profiles: go tool pprof $PROFILE_DIR/json-cpu.prof"
echo "2. Compare with baseline: benchstat baseline.txt current.txt"
echo "3. Focus on high-allocation functions first"
echo "4. Test optimizations with: make test-race"
echo ""
echo "Profile files saved to: $PROFILE_DIR/"

# Generate interactive profile viewer command
echo ""
echo "To view interactive profile:"
echo "  go tool pprof -http=:8080 $PROFILE_DIR/json-cpu.prof"
