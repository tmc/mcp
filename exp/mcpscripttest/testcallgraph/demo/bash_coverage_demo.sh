#!/bin/bash

# Demo script to show bash coverage analysis with testcallgraph

# Create a test bash script
cat > test_script.sh << 'EOF'
#!/bin/bash

# Function definitions
function setup() {
    echo "Setting up environment..."
    mkdir -p output
}

function process_data() {
    local file=$1
    echo "Processing $file..."
    if [[ -f "$file" ]]; then
        cat "$file" | wc -l
    else
        echo "File not found: $file"
        return 1
    fi
}

function cleanup() {
    echo "Cleaning up..."
    rm -rf output
}

# Main execution
echo "Starting script..."
setup

# Process some files
for file in data1.txt data2.txt missing.txt; do
    process_data "$file"
done

cleanup
echo "Done."
EOF

chmod +x test_script.sh

# Create a test file that executes the bash script
cat > bash_test.txt << 'EOF'
# Test that executes our bash script
exec bash test_script.sh

# Test with coverage collection  
exec kcov --exclude-pattern=/usr coverage_out ./test_script.sh

# Another script execution
bash ./test_script.sh --verbose
EOF

# Create some test data files
echo "Line 1" > data1.txt
echo "Line 1" > data2.txt

# Run testcallgraph with bash mode
echo "=== Running testcallgraph with bash analysis ==="
go run ../../cmd/testcallgraph/main.go -bash -v bash_test.txt

# Show the output formats
echo -e "\n=== JSON output ==="
go run ../../cmd/testcallgraph/main.go -bash -format json bash_test.txt

echo -e "\n=== DOT output for visualization ==="
go run ../../cmd/testcallgraph/main.go -bash -format dot bash_test.txt > bash_callgraph.dot
echo "Generated bash_callgraph.dot - convert to SVG with: dot -Tsvg bash_callgraph.dot > bash_callgraph.svg"

# Show statistics
echo -e "\n=== Statistics and Coverage ==="
go run ../../cmd/testcallgraph/main.go -bash -stats bash_test.txt

# Clean up
rm -f test_script.sh bash_test.txt data1.txt data2.txt bash_callgraph.dot