all: build

build:
	go build -o mcp-proxy main.go

test: build test-stdio test-http

test-stdio:
	@echo "Testing STDIO proxy..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
		./mcp-proxy -- npx @modelcontextprotocol/server-everything stdio | \
		grep -q '"result"' && echo "✓ STDIO proxy test passed" || echo "✗ STDIO proxy test failed"

test-http:
	@echo "Testing HTTP proxy..."
	@echo "Note: Requires manual testing with separate server"
	@echo "1. Start server: npx @modelcontextprotocol/server-everything sse"
	@echo "2. Start proxy: ./mcp-proxy -transport=http"
	@echo "3. Connect client: mcp-connect -transport=sse -url=http://localhost:8080"

test-verbose:
	@echo "Testing verbose mode..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
		./mcp-proxy -v -t -- npx @modelcontextprotocol/server-everything stdio | \
		tee test-verbose.log
	@echo "Check test-verbose.log for output"

clean:
	rm -f mcp-proxy test-verbose.log

install: build
	go install

.PHONY: all build test test-stdio test-http test-verbose clean install