#!/bin/bash
# replay-diff.sh - Script to replay MCP recordings and show differences
# Usage: ./replay-diff.sh [options] baseline.mcp command_to_run
# 
# This script:
# 1. Takes a baseline MCP recording and a command to run
# 2. Replays the baseline to the command
# 3. Captures the new output
# 4. Shows the differences between baseline and new output

set -e

# Parse arguments
BASELINE=""
REPLAY_OPTS=""
DIFF_OPTS=""
COMMAND=""

show_usage() {
    echo "Usage: $0 [options] baseline.mcp command_to_run"
    echo ""
    echo "Options:"
    echo "  --replay-opts \"opts\"   Options to pass to mcp-replay (e.g. \"--speed 2.0\")"
    echo "  --diff-opts \"opts\"     Options to pass to mcpdiff (e.g. \"-v -i\")"
    echo "  --output file          Save new recording to file"
    echo ""
    echo "Example:"
    echo "  $0 baseline.mcp \"python server.py\""
    echo "  $0 --replay-opts \"-speed 2.0\" --diff-opts \"-v -word-diff\" baseline.mcp \"node app.js\""
    exit 1
}

# Parse arguments
OUTPUT=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --replay-opts)
            REPLAY_OPTS="$2"
            shift 2
            ;;
        --diff-opts)
            DIFF_OPTS="$2"
            shift 2
            ;;
        --output)
            OUTPUT="$2"
            shift 2
            ;;
        --help|-h)
            show_usage
            ;;
        *)
            if [[ -z "$BASELINE" ]]; then
                BASELINE="$1"
                shift
            else
                COMMAND="$COMMAND $1"
                shift
            fi
            ;;
    esac
done

# Validate arguments
if [[ -z "$BASELINE" || -z "$COMMAND" ]]; then
    echo "Error: Missing required arguments"
    show_usage
fi

if [[ ! -f "$BASELINE" ]]; then
    echo "Error: Baseline file '$BASELINE' does not exist"
    exit 1
fi

# Create a temporary file for new recording
if [[ -z "$OUTPUT" ]]; then
    NEW_RECORDING=$(mktemp /tmp/mcp-replay.XXXXXX)
    trap "rm -f $NEW_RECORDING" EXIT
else
    NEW_RECORDING="$OUTPUT"
fi

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCPDIFF_PATH="$SCRIPT_DIR/../mcpdiff/mcpdiff"
MCPREPLAY_PATH="$SCRIPT_DIR/mcp-replay"

# Ensure tools exist
if [[ ! -f "$MCPREPLAY_PATH" ]]; then
    echo "Building mcp-replay..."
    (cd "$SCRIPT_DIR" && go build)
fi

if [[ ! -f "$MCPDIFF_PATH" ]]; then
    echo "Building mcpdiff..."
    (cd "$(dirname "$MCPDIFF_PATH")" && go build)
fi

# Run the replay and capture output
echo "Replaying $BASELINE to $COMMAND and capturing output to $NEW_RECORDING..."
eval "$MCPREPLAY_PATH $REPLAY_OPTS $BASELINE" | eval "$COMMAND" > "$NEW_RECORDING"

# Run diff
echo "Comparing baseline to new recording..."
eval "$MCPDIFF_PATH $DIFF_OPTS $BASELINE $NEW_RECORDING"

if [[ -n "$OUTPUT" ]]; then
    echo "New recording saved to $OUTPUT"
fi