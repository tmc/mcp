#!/bin/bash
# Complete workflow example for dependency graph analysis

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Complete Dependency Analysis Workflow ===${NC}"

# Step 1: Generate test data
echo -e "\n${YELLOW}Step 1: Creating test data...${NC}"
mkdir -p testdata/workflow
cat > testdata/workflow/test_echo.txt << 'EOF'
exec echo "Hello World"
exec echo "Test Output"
EOF

cat > testdata/workflow/test_diff.txt << 'EOF'
exec mcpdiff file1.txt file2.txt
exec echo "Diff test"
exec grep "pattern" file.txt
EOF

cat > testdata/workflow/test_server.txt << 'EOF'
mcp-server-start myserver -- go run server.go
exec echo "Server test"
exec mcpdiff result1.txt result2.txt
EOF

echo "Created test files"

# Step 2: Generate callgraph
echo -e "\n${YELLOW}Step 2: Generating callgraph...${NC}"
testgraph -format json testdata/workflow > workflow-callgraph.json
echo "Created workflow-callgraph.json"

# Display callgraph summary
echo -e "\n${GREEN}Callgraph Summary:${NC}"
jq '{
  test_count: (.nodes | to_entries | map(select(.value.Type == "test")) | length),
  program_count: (.nodes | to_entries | map(select(.value.Type == "program")) | length),
  edge_count: .edges | length
}' workflow-callgraph.json

# Step 3: Create dependency graphs
echo -e "\n${YELLOW}Step 3: Creating dependency graphs...${NC}"

# Standard dependency graph
depgraph -input workflow-callgraph.json -format digraph > workflow-deps.digraph
echo "Created workflow-deps.digraph"

# Reverse dependency graph
depgraph -input workflow-callgraph.json -direction program-to-test -format digraph > workflow-reverse.digraph
echo "Created workflow-reverse.digraph"

# JSON format for further analysis
depgraph -input workflow-callgraph.json -format json > workflow-deps.json
echo "Created workflow-deps.json"

# DOT format for visualization
depgraph -input workflow-callgraph.json -format dot -group -include-meta > workflow-deps.dot
echo "Created workflow-deps.dot"

# Step 4: Analyze with digraph
echo -e "\n${YELLOW}Step 4: Running digraph analysis...${NC}"

echo -e "\n${GREEN}All tests:${NC}"
cat workflow-deps.digraph | digraph sources

echo -e "\n${GREEN}All programs:${NC}"
cat workflow-deps.digraph | digraph sinks

echo -e "\n${GREEN}Programs used by test_diff.txt:${NC}"
cat workflow-deps.digraph | digraph successors test_diff.txt

echo -e "\n${GREEN}Tests that use mcpdiff:${NC}"
cat workflow-reverse.digraph | digraph successors mcpdiff

echo -e "\n${GREEN}Testing coverage of echo:${NC}"
cat workflow-reverse.digraph | digraph successors echo

# Step 5: Find coverage gaps
echo -e "\n${YELLOW}Step 5: Finding coverage gaps...${NC}"

# Find all unique programs
all_programs=$(cat workflow-deps.digraph | digraph nodes | grep -v ".txt" | sort -u)
echo -e "\n${GREEN}All programs:${NC}"
echo "$all_programs"

# Find tested programs
tested_programs=$(cat workflow-deps.digraph | digraph sinks | sort -u)
echo -e "\n${GREEN}Tested programs:${NC}"
echo "$tested_programs"

# Find untested programs (if any)
untested=$(comm -23 <(echo "$all_programs") <(echo "$tested_programs"))
if [ -n "$untested" ]; then
    echo -e "\n${RED}Untested programs:${NC}"
    echo "$untested"
else
    echo -e "\n${GREEN}All programs have test coverage!${NC}"
fi

# Step 6: Generate visualizations
echo -e "\n${YELLOW}Step 6: Creating visualizations...${NC}"
if command -v dot &> /dev/null; then
    dot -Tpng workflow-deps.dot -o workflow-deps.png
    echo "Created workflow-deps.png"
    
    # Create a simplified view
    depgraph -input workflow-callgraph.json -format dot > workflow-simple.dot
    dot -Tpng workflow-simple.dot -o workflow-simple.png
    echo "Created workflow-simple.png"
else
    echo "Graphviz not installed, skipping PNG generation"
fi

# Step 7: Create detailed report
echo -e "\n${YELLOW}Step 7: Generating detailed report...${NC}"
cat > workflow-report.md << 'EOF'
# Dependency Analysis Report

## Overview

This report was generated from the test suite analysis.

## Test Coverage

### By Test File
EOF

# Add test coverage details
for test in $(cat workflow-deps.digraph | digraph sources); do
    echo -e "\n#### $test" >> workflow-report.md
    echo "Programs covered:" >> workflow-report.md
    cat workflow-deps.digraph | digraph successors "$test" | sed 's/^/- /' >> workflow-report.md
done

# Add program coverage details
echo -e "\n### By Program" >> workflow-report.md
for prog in $(cat workflow-deps.digraph | digraph sinks); do
    echo -e "\n#### $prog" >> workflow-report.md
    echo "Tested by:" >> workflow-report.md
    cat workflow-reverse.digraph | digraph successors "$prog" | sed 's/^/- /' >> workflow-report.md
done

echo "Created workflow-report.md"

# Step 8: Generate dependency matrix
echo -e "\n${YELLOW}Step 8: Creating dependency matrix...${NC}"
depgraph -input workflow-callgraph.json -format matrix > workflow-matrix.txt
echo "Created workflow-matrix.txt"

echo -e "\n${GREEN}Dependency Matrix:${NC}"
cat workflow-matrix.txt

# Summary
echo -e "\n${GREEN}=== Analysis Complete ===${NC}"
echo -e "\nGenerated files:"
echo "  - workflow-callgraph.json   : Raw callgraph data"
echo "  - workflow-deps.digraph     : Dependency graph for digraph tool"
echo "  - workflow-reverse.digraph  : Reverse dependencies"
echo "  - workflow-deps.json        : Dependency graph in JSON"
echo "  - workflow-deps.dot         : DOT format for visualization"
echo "  - workflow-deps.png         : Visual dependency graph"
echo "  - workflow-simple.png       : Simplified visualization"
echo "  - workflow-report.md        : Detailed coverage report"
echo "  - workflow-matrix.txt       : Adjacency matrix"

echo -e "\n${GREEN}Next steps:${NC}"
echo "1. View the PNG files to visualize dependencies"
echo "2. Read workflow-report.md for detailed coverage analysis"
echo "3. Use digraph tool for advanced queries on .digraph files"
echo "4. Integrate with coverage-by-program when coverage data is available"