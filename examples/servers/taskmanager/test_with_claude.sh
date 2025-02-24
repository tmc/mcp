#!/bin/bash; echo "Testing Task Manager server..."; go run main.go & PID=$!; sleep 2; echo "Testing health endpoint..."; curl -s http://localhost:8097/health; echo -e "
Creating a task..."; curl -s -X POST -H "Content-Type: application/json" -d '{"title":"Test task","priority":1}' http://localhost:8097/tasks; echo -e "
Listing tasks..."; curl -s http://localhost:8097/tasks; echo -e "
Updating task status..."; curl -s -X POST -H "Content-Type: application/json" -d '{"id":"TASK_ID","status":"in_progress"}' http://localhost:8097/tasks/update; echo -e "
WebSocket test: Use wscat -c ws://localhost:8097/ws to receive real-time task updates"; go run ../../cmd/claude-desktop/main.go; trap "kill $PID" EXIT; wait $PID
