package golangsdkinterop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	officialmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	tmcmcp "github.com/tmc/mcp"
)

const runAsServerEnv = "TMC_MCP_GOLANG_SDK_INTEROP_SERVER"

func TestMain(m *testing.M) {
	if os.Getenv(runAsServerEnv) == "1" {
		os.Unsetenv(runAsServerEnv)
		if err := runEchoServer(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}
	os.Exit(m.Run())
}

func TestOfficialGoSDKCommandTransportStdioSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), runAsServerEnv+"=1")

	client := officialmcp.NewClient(&officialmcp.Implementation{
		Name:    "tmc-mcp-interop-client",
		Version: "v0.0.0",
	}, nil)
	session, err := client.Connect(ctx, &officialmcp.CommandTransport{
		Command:           cmd,
		TerminateDuration: time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("connect official Go SDK client to tmc/mcp server: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Errorf("close session: %v", err)
		}
	}()

	tools, err := session.ListTools(ctx, &officialmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("listed %d tools, want 1", len(tools.Tools))
	}
	if got := tools.Tools[0].Name; got != "echo" {
		t.Fatalf("listed tool %q, want echo", got)
	}

	result, err := session.CallTool(ctx, &officialmcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"message": "hello from official Go SDK"},
	})
	if err != nil {
		t.Fatalf("call echo tool: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("echo returned %d content items, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*officialmcp.TextContent)
	if !ok {
		t.Fatalf("echo returned %T, want *mcp.TextContent", result.Content[0])
	}
	if got, want := text.Text, "echo: hello from official Go SDK"; got != want {
		t.Fatalf("echo text = %q, want %q", got, want)
	}
}

func runEchoServer(ctx context.Context) error {
	server := tmcmcp.NewServer("tmc-mcp-go-sdk-interop-server", "v0.0.0")
	if err := server.RegisterTool(tmcmcp.Tool{
		Name:        "echo",
		Description: "Echoes a message.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string"}
			},
			"required": ["message"]
		}`),
	}, echoTool); err != nil {
		return fmt.Errorf("register echo tool: %w", err)
	}

	err := server.Serve(ctx, tmcmcp.StdioTransport())
	if expectedServerClose(err) {
		return nil
	}
	return err
}

func echoTool(ctx context.Context, req tmcmcp.CallToolRequest) (*tmcmcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		return nil, fmt.Errorf("decode echo arguments: %w", err)
	}
	return &tmcmcp.CallToolResult{
		Content: []any{tmcmcp.TextContent{
			Type: "text",
			Text: "echo: " + args.Message,
		}},
	}, nil
}

func expectedServerClose(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, tmcmcp.ErrTransportClosed) {
		return true
	}
	return strings.Contains(err.Error(), "EOF")
}
