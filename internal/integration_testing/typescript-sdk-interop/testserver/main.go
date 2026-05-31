package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/tmc/mcp"
)

func main() {
	httpAddr := flag.String("http", "", "HTTP listen address for streamable HTTP")
	urlFile := flag.String("url-file", "", "path to write the streamable HTTP endpoint URL")
	flag.Parse()

	server := mcp.NewServer("tmc-mcp-typescript-stdio-smoke", "0.0.1")
	if err := server.RegisterTool(echoTool(), echo); err != nil {
		fmt.Fprintf(os.Stderr, "register echo tool: %v\n", err)
		os.Exit(1)
	}
	if *httpAddr != "" {
		if err := serveHTTP(*httpAddr, *urlFile, server); err != nil {
			fmt.Fprintf(os.Stderr, "serve http: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := server.Serve(context.Background(), mcp.StdioTransport()); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func serveHTTP(addr, urlFile string, server *mcp.Server) error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil))

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()

	endpoint, err := endpointURL(ln.Addr())
	if err != nil {
		return fmt.Errorf("endpoint url: %w", err)
	}
	if urlFile != "" {
		if err := os.WriteFile(urlFile, []byte(endpoint+"\n"), 0o666); err != nil {
			return fmt.Errorf("write url file: %w", err)
		}
	} else {
		fmt.Println(endpoint)
	}

	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func endpointURL(addr net.Addr) (string, error) {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", err
	}
	return "http://" + net.JoinHostPort(host, port) + "/mcp", nil
}

func echoTool() mcp.Tool {
	return mcp.Tool{
		Name:        "echo",
		Description: "Echo a message for TypeScript SDK interop smoke tests.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			},
			"required": ["message"]
		}`),
	}
}

func echo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		return nil, fmt.Errorf("decode echo arguments: %w", err)
	}
	if args.Message == "" {
		return nil, fmt.Errorf("message is required")
	}
	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: "echo: " + args.Message,
			},
		},
	}, nil
}
