all: mcp-serve

mcp-serve: main.go
	go build -o mcp-serve

clean:
	rm -f mcp-serve

install: mcp-serve
	go install

.PHONY: all clean install