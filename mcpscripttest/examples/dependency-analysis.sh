#!/bin/bash
# Example script showing how to use the dependency graph tools

set -e

echo "=== Dependency Graph Analysis Example ==="

# Step 1: Generate callgraph from test files
echo -e "\n1. Generating callgraph from test files..."
testgraph -format json testdata/demo > callgraph.json
echo "Created callgraph.json"

# Step 2: Create dependency graphs in different formats
echo -e "\n2. Creating dependency graphs..."
depgraph -input callgraph.json -format digraph > deps.digraph
depgraph -input callgraph.json -format dot > deps.dot  
depgraph -input callgraph.json -format json > deps.json
echo "Created deps.digraph, deps.dot, deps.json"

# Step 3: Answer questions using digraph tool
echo -e "\n3. Running digraph queries..."

echo -e "\n# Which programs are tested by test1.txt?"
depgraph -input callgraph.json | digraph successors test1.txt

echo -e "\n# Which tests cover mcpdiff?"
depgraph -input callgraph.json -direction program-to-test | digraph successors mcpdiff

echo -e "\n# Find all programs:"
depgraph -input callgraph.json | digraph sinks

echo -e "\n# Find all tests:"
depgraph -input callgraph.json | digraph sources

echo -e "\n# Find programs with no tests (orphaned):"
comm -23 \
  <(depgraph -input callgraph.json | digraph nodes | grep -v ".txt" | sort) \
  <(depgraph -input callgraph.json | digraph sinks | sort)

echo -e "\n# Create transitive closure graph"
depgraph -input callgraph.json -type transitive -format dot > transitive.dot
echo "Created transitive.dot"

# Step 4: Visualize if graphviz is available
if command -v dot &> /dev/null; then
    echo -e "\n4. Creating visualizations..."
    dot -Tpng deps.dot -o deps.png
    dot -Tpng transitive.dot -o transitive.png
    echo "Created deps.png and transitive.png"
fi

# Step 5: Generate coverage analysis (if coverage data exists)
if [ -d "/tmp/coverage" ]; then
    echo -e "\n5. Analyzing coverage by program..."
    coverage-by-program -depgraph deps.json -coverage /tmp/coverage -format markdown > coverage-report.md
    echo "Created coverage-report.md"
fi

echo -e "\n=== Analysis Complete ==="