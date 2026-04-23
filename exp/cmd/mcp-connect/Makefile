all: build

build:
	go build -o mcp-connect main.go

test: build test-stdio test-sse test-http

test-stdio:
	@echo "Testing STDIO transport..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-connect | grep -q "result" && echo "✓ STDIO test passed" || echo "✗ STDIO test failed"

test-sse:
	@echo "Testing SSE transport..."
	@# Start SSE server in background
	@npx @modelcontextprotocol/server-everything sse > /dev/null 2>&1 & SSE_PID=$$!; \
	sleep 3; \
	echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-connect -transport=sse | grep -q "result" && echo "✓ SSE test passed" || echo "✗ SSE test failed"; \
	kill $$SSE_PID 2>/dev/null || true

test-http:
	@echo "Testing HTTP transport..."
	@# Start HTTP server in background
	@npx @modelcontextprotocol/server-everything streamableHttp > /dev/null 2>&1 & HTTP_PID=$$!; \
	sleep 3; \
	echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-connect -transport=http | grep -q "result" && echo "✓ HTTP test passed" || echo "✗ HTTP test failed"; \
	kill $$HTTP_PID 2>/dev/null || true

test-script: build
	@echo "Testing script mode..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' > test-script.txt
	@echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' >> test-script.txt
	@./mcp-connect -script=test-script.txt | grep -q "result" && echo "✓ Script test passed" || echo "✗ Script test failed"
	@rm -f test-script.txt

clean:
	rm -f mcp-connect

install: build
	go install

.PHONY: all build test test-stdio test-sse test-http test-script clean install