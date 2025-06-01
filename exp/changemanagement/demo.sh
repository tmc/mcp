#!/bin/bash

set -e

echo "=== Intelligent Change Management Demo ==="
echo

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create demo directory
DEMO_DIR="demo-output"
rm -rf $DEMO_DIR
mkdir -p $DEMO_DIR

echo -e "${BLUE}1. Analyzing a change request...${NC}"
echo "Change: Add OAuth2 authentication to all API endpoints"
echo

mcp-change-analyze \
  -description "Add OAuth2 authentication to all API endpoints" \
  -output $DEMO_DIR/analysis.json \
  -format json

echo -e "${GREEN}✓ Analysis complete${NC}"
echo

echo -e "${BLUE}2. Finding affected tests...${NC}"
mcp-test-find \
  -change $DEMO_DIR/analysis.json \
  -codebase . \
  -output $DEMO_DIR/affected_tests.json

echo -e "${GREEN}✓ Test discovery complete${NC}"
echo

echo -e "${BLUE}3. Generating documentation...${NC}"
mcp-doc-gen \
  -change $DEMO_DIR/analysis.json \
  -output $DEMO_DIR/docs

echo -e "${GREEN}✓ Documentation generated${NC}"
echo

echo -e "${BLUE}4. Creating test mutations...${NC}"
if [ -f "testdata/sample_test.txt" ]; then
  mcp-test-mutate \
    -test testdata/sample_test.txt \
    -output $DEMO_DIR/mutations \
    -count 3
  echo -e "${GREEN}✓ Test mutations created${NC}"
else
  echo -e "${YELLOW}⚠ Sample test file not found, skipping mutations${NC}"
fi
echo

echo -e "${BLUE}5. Running complete workflow...${NC}"
mcp-change-execute \
  -description "Add OAuth2 authentication to all API endpoints" \
  -codebase . \
  -output $DEMO_DIR/complete \
  -verbose

echo
echo -e "${GREEN}=== Demo Complete ===${NC}"
echo
echo "Results saved in: $DEMO_DIR/"
echo
echo "Key files:"
echo "  - $DEMO_DIR/analysis.json         : Change analysis"
echo "  - $DEMO_DIR/affected_tests.json   : Affected tests"
echo "  - $DEMO_DIR/docs/                 : Generated documentation"
echo "  - $DEMO_DIR/mutations/            : Test mutations"
echo "  - $DEMO_DIR/complete/             : Complete workflow output"
echo

# Show analysis summary
echo -e "${BLUE}Analysis Summary:${NC}"
if command -v jq &> /dev/null; then
  jq -r '
    "Type: \(.type)
Risk: \(.risk_level)
Breaking: \(.breaking)
Components: \(.components | join(", "))
Recommendations: \(.recommendations | length)"
  ' $DEMO_DIR/analysis.json
else
  echo "(Install jq to see formatted summary)"
fi