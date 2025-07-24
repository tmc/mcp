#!/bin/bash

# MCP Code Generation Tools Demo
# This script demonstrates the capabilities of the mcp-gen toolchain

set -e

echo "🚀 MCP Code Generation Tools Demo"
echo "=================================="

# Create demo directory
DEMO_DIR="./demo-output"
rm -rf "$DEMO_DIR"
mkdir -p "$DEMO_DIR"

echo ""
echo "📋 Available Tools:"
echo "  - mcp-gen: Multi-language code generator"
echo "  - mcp-scaffold: Project scaffolding tool"
echo "  - mcp-migrate: Migration assistant"

echo ""
echo "🔧 Building tools..."

# Build the tools
go build -o "$DEMO_DIR/mcp-gen" .
go build -o "$DEMO_DIR/mcp-scaffold" ../mcp-scaffold
go build -o "$DEMO_DIR/mcp-migrate" ../mcp-migrate

echo "✅ Tools built successfully"

echo ""
echo "📊 Demonstrating mcp-gen capabilities:"
echo "======================================"

# Demo 1: Generate Go client from schema
echo ""
echo "1️⃣ Generating Go client from time server schema..."
"$DEMO_DIR/mcp-gen" client -lang go -output "$DEMO_DIR/go-client" -package github.com/demo/timeclient examples/time-server-schema.json

echo "✅ Generated Go client files:"
find "$DEMO_DIR/go-client" -name "*.go" -exec basename {} \; | sort

# Demo 2: Generate TypeScript client
echo ""
echo "2️⃣ Generating TypeScript client..."
"$DEMO_DIR/mcp-gen" client -lang typescript -output "$DEMO_DIR/ts-client" -package time-client examples/time-server-schema.json

echo "✅ Generated TypeScript client files:"
find "$DEMO_DIR/ts-client" -name "*.ts" -exec basename {} \; | sort

# Demo 3: Generate Python client
echo ""
echo "3️⃣ Generating Python client..."
"$DEMO_DIR/mcp-gen" client -lang python -output "$DEMO_DIR/py-client" -package time_client examples/time-server-schema.json

echo "✅ Generated Python client files:"
find "$DEMO_DIR/py-client" -name "*.py" -exec basename {} \; | sort

# Demo 4: Generate server stub
echo ""
echo "4️⃣ Generating Go server stub..."
"$DEMO_DIR/mcp-gen" server -lang go -output "$DEMO_DIR/go-server" -package timeserver examples/time-server-schema.json

echo "✅ Generated Go server files:"
find "$DEMO_DIR/go-server" -name "*.go" -exec basename {} \; | sort

# Demo 5: Generate documentation
echo ""
echo "5️⃣ Generating documentation..."
"$DEMO_DIR/mcp-gen" docs -output "$DEMO_DIR/docs" examples/time-server-schema.json

echo "✅ Generated documentation:"
find "$DEMO_DIR/docs" -name "*.md" -exec basename {} \;

# Demo 6: Generate tests
echo ""
echo "6️⃣ Generating test suites..."
"$DEMO_DIR/mcp-gen" tests -lang go -output "$DEMO_DIR/tests" examples/time-server-schema.json

echo "✅ Generated test files:"
find "$DEMO_DIR/tests" -name "*_test.go" -exec basename {} \;

echo ""
echo "🏗️ Demonstrating mcp-scaffold capabilities:"
echo "============================================"

# Demo 7: Scaffold basic project
echo ""
echo "7️⃣ Scaffolding basic Go project..."
"$DEMO_DIR/mcp-scaffold" server -lang go -template basic -output "$DEMO_DIR/scaffold-basic" -author demo basic-server

echo "✅ Scaffolded basic project structure:"
find "$DEMO_DIR/scaffold-basic" -type f | head -10

# Demo 8: Scaffold advanced project
echo ""
echo "8️⃣ Scaffolding advanced TypeScript project..."
"$DEMO_DIR/mcp-scaffold" client -lang typescript -template advanced -output "$DEMO_DIR/scaffold-advanced" -author demo advanced-client

echo "✅ Scaffolded advanced project structure:"
find "$DEMO_DIR/scaffold-advanced" -type f | head -10

echo ""
echo "🔄 Demonstrating mcp-migrate capabilities:"
echo "=========================================="

# Demo 9: Analyze project for migration
echo ""
echo "9️⃣ Analyzing Go project for migration..."
"$DEMO_DIR/mcp-migrate" analyze -lang go -path "$DEMO_DIR/go-client" -from 1.0 -to 2.0

echo "✅ Migration analysis completed"

# Demo 10: Create migration plan
echo ""
echo "🔟 Creating migration plan..."
"$DEMO_DIR/mcp-migrate" plan -lang go -path "$DEMO_DIR/go-client" -from 1.0 -to 2.0

echo "✅ Migration plan created"

echo ""
echo "📊 Demo Results Summary:"
echo "======================="
echo "Generated files by tool:"
echo ""

echo "📁 mcp-gen outputs:"
echo "  - Go client: $(find "$DEMO_DIR/go-client" -name "*.go" | wc -l) files"
echo "  - TypeScript client: $(find "$DEMO_DIR/ts-client" -name "*.ts" | wc -l) files"
echo "  - Python client: $(find "$DEMO_DIR/py-client" -name "*.py" | wc -l) files"
echo "  - Go server: $(find "$DEMO_DIR/go-server" -name "*.go" | wc -l) files"
echo "  - Documentation: $(find "$DEMO_DIR/docs" -name "*.md" | wc -l) files"
echo "  - Tests: $(find "$DEMO_DIR/tests" -name "*_test.go" | wc -l) files"

echo ""
echo "📁 mcp-scaffold outputs:"
echo "  - Basic project: $(find "$DEMO_DIR/scaffold-basic" -type f | wc -l) files"
echo "  - Advanced project: $(find "$DEMO_DIR/scaffold-advanced" -type f | wc -l) files"

echo ""
echo "📁 mcp-migrate outputs:"
echo "  - Migration analysis: completed"
echo "  - Migration plan: created"

echo ""
echo "🎯 Key Features Demonstrated:"
echo "  ✅ Multi-language code generation (Go, TypeScript, Python)"
echo "  ✅ Type-safe client generation from JSON schemas"
echo "  ✅ Server stub generation with TODO implementations"
echo "  ✅ Comprehensive test suite generation"
echo "  ✅ API documentation generation"
echo "  ✅ Project scaffolding with templates"
echo "  ✅ Migration analysis and planning"

echo ""
echo "📂 All demo files are available in: $DEMO_DIR"
echo ""
echo "💡 Next Steps:"
echo "  1. Examine generated files in $DEMO_DIR/"
echo "  2. Customize templates for your specific needs"
echo "  3. Integrate tools into your CI/CD pipeline"
echo "  4. Extend with custom plugins and generators"

echo ""
echo "🎉 Demo completed successfully!"
echo "   For more information, see README.md"