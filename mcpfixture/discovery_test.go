package mcpfixture_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpfixture"
	"github.com/tmc/mcp/mcptrace"
	"golang.org/x/tools/txtar"
)

// Record real SDK HTTP messages, rather than constructing the modern lifecycle.
func TestDiscoveryRoundTrip(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			responseText := "hello"
			server := sdk.NewServer(&sdk.Implementation{Name: "fixture", Version: "1"}, nil)
			server.AddTool(&sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: responseText}}}, nil
			})
			handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
			httpServer := httptest.NewServer(handler)
			defer httpServer.Close()
			var recorded bytes.Buffer
			httpClient := &http.Client{Transport: &mcptrace.RoundTripper{Trace: mcptrace.NewWriter(&recorded)}}
			client := sdk.NewClient(&sdk.Implementation{Name: "recorded-client", Version: "1"}, &sdk.ClientOptions{Capabilities: &sdk.ClientCapabilities{Experimental: map[string]any{"fixture": map[string]any{}}}})
			session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: httpServer.URL, HTTPClient: httpClient}, &sdk.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			var listParams *sdk.ListToolsParams
			if version >= "2026-07-28" {
				listParams = &sdk.ListToolsParams{}
			}
			if _, err := session.ListTools(ctx, listParams); err != nil {
				t.Fatal(err)
			}
			if _, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "echo", Arguments: map[string]any{}}); err != nil {
				t.Fatal(err)
			}
			// Legal read-only exchanges after the selected response are outside its boundary.
			if _, err := session.ListTools(ctx, &sdk.ListToolsParams{}); err != nil {
				t.Fatal(err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			trace := recorded.String()
			reader := mcptrace.NewReader(strings.NewReader(trace))
			selected := 0
			for record := 1; ; record++ {
				r, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				var message struct {
					Method string `json:"method"`
				}
				if err := json.Unmarshal(r.Message, &message); err != nil {
					t.Fatal(err)
				}
				if message.Method == "tools/call" {
					selected = record
				}
			}
			if selected <= 4 {
				t.Fatalf("expected selected record after discovery/list prerequisites, got %d", selected)
			}
			var converted bytes.Buffer
			if err := mcpfixture.Convert(strings.NewReader(trace), &converted, mcpfixture.Options{Record: selected}); err != nil {
				t.Fatalf("convert: %v\n%s", err, trace)
			}
			fixture, err := mcpfixture.Read(bytes.NewReader(txtar.Parse(converted.Bytes()).Files[0].Data))
			if err != nil {
				t.Fatal(err)
			}
			if len(fixture.Prerequisites) != 1 || fixture.ClientInfo.Name != "recorded-client" || fixture.ProtocolVersion != version {
				t.Fatalf("fixture lost context: %+v", fixture)
			}
			options, err := mcpfixture.ClientOptions(fixture)
			if err != nil {
				t.Fatal(err)
			}
			checkClient := sdk.NewClient(fixture.ClientInfo, options)
			checked, err := checkClient.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: httpServer.URL}, &sdk.ClientSessionOptions{ProtocolVersion: fixture.RequestedProtocolVersion})
			if err != nil {
				t.Fatal(err)
			}
			defer checked.Close()
			if err := mcpfixture.Check(ctx, checked, fixture); err != nil {
				t.Fatal(err)
			}
			responseText = "changed"
			if err := mcpfixture.Check(ctx, checked, fixture); err == nil || !strings.Contains(err.Error(), "observed result changed") {
				t.Fatalf("changed result accepted: %v", err)
			}
			if version == "2026-07-28" {
				for _, method := range []string{"tools/list", "tools/call"} {
					for _, missing := range []bool{false, true} {
						bad := alterContext(t, trace, method, missing)
						err := mcpfixture.Convert(strings.NewReader(bad), io.Discard, mcpfixture.Options{Record: selected})
						if err == nil || !strings.Contains(err.Error(), "request context") {
							t.Fatalf("accepted conflicting or missing %s context: %v", method, err)
						}
					}
				}
			}
			if err := mcpfixture.Convert(strings.NewReader(trace+"mcp-recv {"), io.Discard, mcpfixture.Options{Record: selected}); err == nil {
				t.Fatal("accepted truncated trailing trace")
			}
		})
	}
}

func alterContext(t *testing.T, trace, method string, missing bool) string {
	t.Helper()
	var out bytes.Buffer
	writer := mcptrace.NewWriter(&out)
	reader := mcptrace.NewReader(strings.NewReader(trace))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var message map[string]any
		decoder := json.NewDecoder(bytes.NewReader(record.Message))
		decoder.UseNumber()
		if err := decoder.Decode(&message); err != nil {
			t.Fatal(err)
		}
		if message["method"] == method {
			params := message["params"].(map[string]any)
			meta := params["_meta"].(map[string]any)
			if missing {
				delete(meta, sdk.MetaKeyClientInfo)
			} else {
				meta[sdk.MetaKeyProtocolVersion] = "2025-11-25"
			}
			record.Message, err = json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Write(record); err != nil {
			t.Fatal(err)
		}
	}
	return out.String()
}
