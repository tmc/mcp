#!/bin/bash; echo "Testing Calculator server..."; go run main.go & PID=$!; sleep 2; echo "Testing health endpoint..."; curl -s http://localhost:8096/health; echo -e "
Testing calculation endpoint..."; curl -s -X POST -H "Content-Type: application/json" -d '{"op":"add","a":5,"b":3}' http://localhost:8096/calculate; echo -e "
WebSocket test: Use wscat -c ws://localhost:8096/ws and send:"; echo '{"op":"multiply","a":4,"b":6}'; go run ../../cmd/claude-desktop/main.go; trap "kill $PID" EXIT; wait $PID
