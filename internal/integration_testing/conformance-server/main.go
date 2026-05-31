package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcp "github.com/tmc/mcp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "HTTP listen address")
	urlFile := flag.String("url-file", "", "path to write the MCP endpoint URL")
	flag.Parse()

	server := mcp.NewServer("tmc-mcp-conformance-fixture", "0.0.1")
	if err := server.RegisterTool(mcp.Tool{
		Name:        "echo",
		Description: "Echo a message.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string"}
			}
		}`),
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Message string `json:"message"`
		}
		if len(req.Arguments) > 0 {
			if err := json.Unmarshal(req.Arguments, &args); err != nil {
				return nil, err
			}
		}
		if args.Message == "" {
			args.Message = "ok"
		}
		return &mcp.CallToolResult{
			Content: []any{mcp.TextContent{Type: "text", Text: args.Message}},
		}, nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "register echo tool: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Logger: logger}))

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	endpoint, err := endpointURL(ln.Addr())
	if err != nil {
		fmt.Fprintf(os.Stderr, "endpoint url: %v\n", err)
		os.Exit(1)
	}
	if *urlFile != "" {
		if err := os.WriteFile(*urlFile, []byte(endpoint+"\n"), 0o666); err != nil {
			fmt.Fprintf(os.Stderr, "write url file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(endpoint)
	}
	fmt.Fprintf(os.Stderr, "serving MCP conformance fixture at %s\n", endpoint)

	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		err := httpServer.Serve(ln)
		if err == http.ErrServerClosed {
			err = nil
		}
		errc <- err
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-sigc:
		fmt.Fprintf(os.Stderr, "received %s, shutting down\n", sig)
	case err := <-errc:
		if err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		os.Exit(1)
	}
	if err := <-errc; err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func endpointURL(addr net.Addr) (string, error) {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", err
	}
	return "http://" + net.JoinHostPort(host, port) + "/mcp", nil
}
