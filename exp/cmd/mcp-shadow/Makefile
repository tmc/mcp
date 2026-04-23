.PHONY: build test clean

build:
	go build -o mcp-shadow .

test:
	go test -v

clean:
	rm -f mcp-shadow *.mcp

install:
	go install .

demo: build
	./test.sh