#!/bin/bash
# Script to run mcpscripttest with proper test environment

# Change to the directory containing the script
cd "$(dirname "$0")"

# Run as a go test to avoid testing.Short() panic
go test -run TestConformance -args -- "$@"