package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcp "github.com/tmc/mcp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "HTTP listen address")
	urlFile := flag.String("url-file", "", "path to write the MCP endpoint URL")
	flag.Parse()

	server := mcp.NewServer("tmc-mcp-conformance-fixture", "0.0.1", mcp.WithCompletionHandler(func(ctx context.Context, req mcp.CompleteRequest) (*mcp.CompleteResult, error) {
		var result mcp.CompleteResult
		result.Completion.Values = []string{}
		return &result, nil
	}))
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
	registerConformanceFixtures(server)

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

func registerConformanceFixtures(server *mcp.Server) {
	registerConformanceTools(server)
	registerConformanceResources(server)
	registerConformancePrompts(server)
}

func registerConformanceTools(server *mcp.Server) {
	mustRegisterTool(server, mcp.Tool{
		Name:        "test_simple_text",
		Description: "Tests simple text content response.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textToolResult("This is a simple text response for testing."), nil
	})

	mustRegisterTool(server, mcp.Tool{
		Name:        "test_image_content",
		Description: "Tests image content response.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := imageData()
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []any{mcp.ImageContent{Type: "image", Data: data, MimeType: "image/png"}},
		}, nil
	})

	mustRegisterTool(server, mcp.Tool{
		Name:        "test_audio_content",
		Description: "Tests audio content response.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := audioData()
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []any{audioContent{Type: "audio", Data: data, MimeType: "audio/wav"}},
		}, nil
	})

	mustRegisterTool(server, mcp.Tool{
		Name:        "test_embedded_resource",
		Description: "Tests embedded resource content response.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []any{
				embeddedTextResource("test://embedded-resource", "text/plain", "This is an embedded resource content."),
			},
		}, nil
	})

	mustRegisterTool(server, mcp.Tool{
		Name:        "test_multiple_content_types",
		Description: "Tests response with multiple content types.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := imageData()
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []any{
				mcp.TextContent{Type: "text", Text: "Multiple content types test:"},
				mcp.ImageContent{Type: "image", Data: data, MimeType: "image/png"},
				embeddedTextResource("test://mixed-content-resource", "application/json", `{"test":"data","value":123}`),
			},
		}, nil
	})

	mustRegisterTool(server, mcp.Tool{
		Name:        "test_error_handling",
		Description: "Tests error response handling.",
		InputSchema: noArgumentsSchema,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []any{
				mcp.TextContent{Type: "text", Text: "This tool intentionally returns an error for testing"},
			},
		}, nil
	})
}

func registerConformanceResources(server *mcp.Server) {
	mustRegisterResource(server, mcp.Resource{
		URI:         "test://static-text",
		Name:        "static-text",
		Description: "A static text resource for testing.",
		MimeType:    "text/plain",
	}, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "This is the content of the static text resource.",
			},
		}, nil
	})

	mustRegisterResource(server, mcp.Resource{
		URI:         "test://static-binary",
		Name:        "static-binary",
		Description: "A static binary resource for testing.",
		MimeType:    "image/png",
	}, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.BlobResourceContents{
				URI:      req.URI,
				MimeType: "image/png",
				Blob:     testImageBase64,
			},
		}, nil
	})

	mustRegisterResourceTemplate(server, mcp.ResourceTemplate{
		Template:    "test://template/{id}/data",
		Description: "A resource template with parameter substitution.",
	}, templateResourceHandler)

	// The root server's template matcher is exact-only; expose the conformance URI
	// directly so the fixture can still exercise the handler without root API edits.
	mustRegisterResource(server, mcp.Resource{
		URI:         "test://template/123/data",
		Name:        "template-123-data",
		Description: "A concrete resource served by the template fixture.",
		MimeType:    "application/json",
	}, templateResourceHandler)
}

func registerConformancePrompts(server *mcp.Server) {
	mustRegisterPrompt(server, mcp.Prompt{
		Name:        "test_simple_prompt",
		Description: "A simple prompt without arguments.",
	}, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "This is a simple prompt for testing."},
				},
			},
		}, nil
	})

	mustRegisterPrompt(server, mcp.Prompt{
		Name:        "test_prompt_with_arguments",
		Description: "A prompt with required arguments.",
		Arguments: []mcp.PromptArgument{
			{Name: "arg1", Description: "First test argument", Required: true},
			{Name: "arg2", Description: "Second test argument", Required: true},
		},
	}, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		arg1 := promptArg(req, "arg1")
		arg2 := promptArg(req, "arg2")
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Prompt with arguments: arg1='%s', arg2='%s'", arg1, arg2),
					},
				},
			},
		}, nil
	})

	mustRegisterPrompt(server, mcp.Prompt{
		Name:        "test_prompt_with_embedded_resource",
		Description: "A prompt that includes an embedded resource.",
		Arguments: []mcp.PromptArgument{
			{Name: "resourceUri", Description: "URI of the resource to embed", Required: true},
		},
	}, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		uri := promptArg(req, "resourceUri")
		if uri == "" {
			uri = "test://example-resource"
		}
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: embeddedTextResource(uri, "text/plain", "Embedded resource content for testing."),
				},
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "Please process the embedded resource above."},
				},
			},
		}, nil
	})

	mustRegisterPrompt(server, mcp.Prompt{
		Name:        "test_prompt_with_image",
		Description: "A prompt that includes image content.",
	}, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		data, err := imageData()
		if err != nil {
			return nil, err
		}
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.ImageContent{Type: "image", Data: data, MimeType: "image/png"},
				},
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "Please analyze the image above."},
				},
			},
		}, nil
	})
}

func templateResourceHandler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	id := templateID(req.URI)
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.URI,
			MimeType: "application/json",
			Text:     fmt.Sprintf(`{"id":"%s","templateTest":true,"data":"Data for ID: %s"}`, id, id),
		},
	}, nil
}

func textToolResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []any{mcp.TextContent{Type: "text", Text: text}},
	}
}

type audioContent struct {
	Type     string `json:"type"`
	Data     []byte `json:"data"`
	MimeType string `json:"mimeType"`
}

type embeddedResourceContent struct {
	Type     string `json:"type"`
	Resource any    `json:"resource"`
}

type embeddedTextResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

func embeddedTextResource(uri, mimeType, text string) embeddedResourceContent {
	return embeddedResourceContent{
		Type: "resource",
		Resource: embeddedTextResourceContents{
			URI:      uri,
			MimeType: mimeType,
			Text:     text,
		},
	}
}

func promptArg(req mcp.GetPromptRequest, name string) string {
	if req.Arguments == nil {
		return ""
	}
	v, ok := req.Arguments[name]
	if !ok {
		return ""
	}
	return fmt.Sprint(v)
}

func templateID(uri string) string {
	const (
		prefix = "test://template/"
		suffix = "/data"
	)
	if !strings.HasPrefix(uri, prefix) || !strings.HasSuffix(uri, suffix) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(uri, prefix), suffix)
}

func imageData() ([]byte, error) {
	return decodeTestData("image", testImageBase64)
}

func audioData() ([]byte, error) {
	return decodeTestData("audio", testAudioBase64)
}

func decodeTestData(name, enc string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("decode %s data: %w", name, err)
	}
	return data, nil
}

func mustRegisterTool(server *mcp.Server, tool mcp.Tool, handler mcp.ToolHandlerFunc) {
	if err := server.RegisterTool(tool, handler); err != nil {
		fmt.Fprintf(os.Stderr, "register tool %s: %v\n", tool.Name, err)
		os.Exit(1)
	}
}

func mustRegisterResource(server *mcp.Server, resource mcp.Resource, handler mcp.ReadResourceHandlerFunc) {
	if err := server.RegisterResource(resource, handler); err != nil {
		fmt.Fprintf(os.Stderr, "register resource %s: %v\n", resource.URI, err)
		os.Exit(1)
	}
}

func mustRegisterResourceTemplate(server *mcp.Server, template mcp.ResourceTemplate, handler mcp.ResourceTemplateHandlerFunc) {
	if err := server.RegisterResourceTemplate(template, handler); err != nil {
		fmt.Fprintf(os.Stderr, "register resource template %s: %v\n", template.Template, err)
		os.Exit(1)
	}
}

func mustRegisterPrompt(server *mcp.Server, prompt mcp.Prompt, handler mcp.GetPromptHandlerFunc) {
	if err := server.RegisterPrompt(prompt, handler); err != nil {
		fmt.Fprintf(os.Stderr, "register prompt %s: %v\n", prompt.Name, err)
		os.Exit(1)
	}
}

var noArgumentsSchema = json.RawMessage(`{"type":"object","properties":{}}`)

const (
	testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
	testAudioBase64 = "UklGRiYAAABXQVZFZm10IBAAAAABAAEAQB8AAAB9AAACABAAZGF0YQIAAAA="
)
