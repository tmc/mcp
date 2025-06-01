all: mcp-send

mcp-send: main.go
	go build -o mcp-send

clean:
	rm -f mcp-send

install: mcp-send
	go install

.PHONY: all clean install