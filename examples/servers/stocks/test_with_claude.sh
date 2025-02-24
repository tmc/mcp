#!/bin/bash; echo "Testing Stock Market server..."; go run main.go & PID=$!; sleep 2; echo "Testing health endpoint..."; curl -s http://localhost:8094/health; echo -e "
Testing stocks endpoint..."; curl -s http://localhost:8094/stocks; echo -e "
WebSocket test: Connecting to stock updates..."; echo "Use wscat -c ws://localhost:8094/ws to connect, then send:"; echo "{\"action\":\"subscribe\",\"symbols\":[\"AAPL\",\"MSFT\"]}"; go run ../../cmd/claude-desktop/main.go; trap "kill $PID" EXIT; wait $PID
