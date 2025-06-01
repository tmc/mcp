.PHONY: build test clean run demo docker-demo

build:
	GOWORK=off go build -o mcptrace-to-otel .

test:
	GOWORK=off go test -v

clean:
	rm -f mcptrace-to-otel

install:
	GOWORK=off go install .

run: build
	./mcptrace-to-otel -f example.mcp

# Demo targets
demo-stdout: build
	@echo "=== Converting to stdout ==="
	./mcptrace-to-otel -f example.mcp -type stdout

demo-jaeger: build
	@echo "=== Converting to Jaeger ==="
	@echo "Make sure Jaeger is running at localhost:14268"
	./mcptrace-to-otel -f example.mcp -type jaeger -endpoint http://localhost:14268/api/traces

demo-otlp: build
	@echo "=== Converting to OTLP ==="
	@echo "Make sure OTLP collector is running at localhost:4317"
	./mcptrace-to-otel -f example.mcp -type otlp-grpc -endpoint localhost:4317

# Docker demo with Jaeger
docker-demo:
	@echo "Starting Jaeger in Docker..."
	docker run -d --name jaeger \
	  -e COLLECTOR_OTLP_ENABLED=true \
	  -p 16686:16686 \
	  -p 14268:14268 \
	  jaegertracing/all-in-one:latest
	@echo "Waiting for Jaeger to start..."
	@sleep 5
	@echo "Converting and sending trace to Jaeger..."
	$(MAKE) demo-jaeger
	@echo "\nView traces at: http://localhost:16686"
	@echo "To stop Jaeger: docker stop jaeger && docker rm jaeger"

# Generate dependencies
deps:
	GOWORK=off go mod tidy
	GOWORK=off go mod download

# Docker build
docker-build:
	docker build -t mcptrace-to-otel:latest .

# Run in Docker
docker-run: docker-build
	docker run --rm mcptrace-to-otel:latest -f example.mcp -type stdout

# Full stack demo with docker compose
stack-demo:
	docker compose up -d
	@sleep 10
	./demo.sh

# Clean up Docker resources
clean-docker:
	docker compose down
	docker rmi mcptrace-to-otel:latest || true
