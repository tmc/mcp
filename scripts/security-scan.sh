#!/usr/bin/env bash
# Security scanning automation for MCP Go implementation
# Runs gosec and govulncheck with comprehensive reporting

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GOSEC_REPORT_DIR="${GOSEC_REPORT_DIR:-./security-reports}"
GOSEC_REPORT_FORMAT="${GOSEC_REPORT_FORMAT:-json,sarif,html,text}"
GOSEC_SEVERITY="${GOSEC_SEVERITY:-medium}" # low, medium, high
GOSEC_CONFIDENCE="${GOSEC_CONFIDENCE:-medium}" # low, medium, high

# Ensure report directory exists
mkdir -p "$GOSEC_REPORT_DIR"

echo -e "${GREEN}MCP Security Scanning Suite${NC}"
echo "=============================================="
echo "Report directory: $GOSEC_REPORT_DIR"
echo ""

# Check for required tools
echo -e "${YELLOW}Checking for required tools...${NC}"
if ! command -v gosec &> /dev/null; then
    echo -e "${RED}ERROR: gosec not found. Installing...${NC}"
    go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

if ! command -v govulncheck &> /dev/null; then
    echo -e "${RED}ERROR: govulncheck not found. Installing...${NC}"
    go install golang.org/x/vuln/cmd/govulncheck@latest
fi

echo -e "${GREEN}✓ Tools available${NC}"
echo ""

# Run gosec
echo -e "${YELLOW}Running gosec security scanner...${NC}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Run gosec with multiple output formats
gosec \
    -severity "$GOSEC_SEVERITY" \
    -confidence "$GOSEC_CONFIDENCE" \
    -exclude-generated \
    -fmt json \
    -out "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.json" \
    ./... 2>&1 | tee "$GOSEC_REPORT_DIR/gosec-output-$TIMESTAMP.txt" || {
    GOSEC_EXIT=$?
    echo -e "${YELLOW}⚠ gosec found issues (exit code: $GOSEC_EXIT)${NC}"
}

# Generate SARIF format for GitHub Advanced Security
gosec \
    -severity "$GOSEC_SEVERITY" \
    -confidence "$GOSEC_CONFIDENCE" \
    -exclude-generated \
    -fmt sarif \
    -out "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.sarif" \
    ./... 2>/dev/null || true

# Generate HTML report
gosec \
    -severity "$GOSEC_SEVERITY" \
    -confidence "$GOSEC_CONFIDENCE" \
    -exclude-generated \
    -fmt html \
    -out "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.html" \
    ./... 2>/dev/null || true

echo -e "${GREEN}✓ gosec scan complete${NC}"
echo "  Reports: $GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.*"
echo ""

# Run govulncheck
echo -e "${YELLOW}Running govulncheck vulnerability scanner...${NC}"
govulncheck -json ./... > "$GOSEC_REPORT_DIR/govulncheck-report-$TIMESTAMP.json" 2>&1 | tee "$GOSEC_REPORT_DIR/govulncheck-output-$TIMESTAMP.txt" || {
    GOVULN_EXIT=$?
    echo -e "${YELLOW}⚠ govulncheck found vulnerabilities (exit code: $GOVULN_EXIT)${NC}"
}

echo -e "${GREEN}✓ govulncheck scan complete${NC}"
echo "  Report: $GOSEC_REPORT_DIR/govulncheck-report-$TIMESTAMP.json"
echo ""

# Generate summary
echo -e "${GREEN}Security Scan Summary${NC}"
echo "=============================================="

# Parse gosec results
if [ -f "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.json" ]; then
    GOSEC_ISSUES=$(jq -r '.Stats.found // 0' "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.json" 2>/dev/null || echo "0")
    echo "gosec issues found: $GOSEC_ISSUES"

    if [ "$GOSEC_ISSUES" -gt 0 ]; then
        echo ""
        echo "Issue breakdown by severity:"
        jq -r '.Issues[] | "\(.severity): \(.rule_id) - \(.file):\(.line)"' "$GOSEC_REPORT_DIR/gosec-report-$TIMESTAMP.json" 2>/dev/null | sort | uniq -c | sort -rn | head -10 || true
    fi
fi

# Parse govulncheck results
echo ""
if [ -f "$GOSEC_REPORT_DIR/govulncheck-report-$TIMESTAMP.json" ]; then
    VULN_COUNT=$(jq '[.[] | select(.finding != null)] | length' "$GOSEC_REPORT_DIR/govulncheck-report-$TIMESTAMP.json" 2>/dev/null || echo "0")
    echo "govulncheck vulnerabilities: $VULN_COUNT"

    if [ "$VULN_COUNT" -gt 0 ]; then
        echo ""
        echo "Vulnerability details:"
        jq -r '.[] | select(.finding != null) | "\(.finding.osv): \(.finding.fixed_version // "no fix available")"' "$GOSEC_REPORT_DIR/govulncheck-report-$TIMESTAMP.json" 2>/dev/null | head -10 || true
    fi
fi

echo ""
echo -e "${GREEN}Scan complete!${NC}"
echo "View detailed reports in: $GOSEC_REPORT_DIR"

# Exit with error if critical issues found
if [ "${GOSEC_ISSUES:-0}" -gt 0 ] || [ "${VULN_COUNT:-0}" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Security issues detected - review reports${NC}"
    exit 1
fi

exit 0
