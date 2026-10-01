package mcpfixture_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpfixture"
	"golang.org/x/tools/txtar"
)

const trace = `mcp-send {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"demo","version":"1"}}}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25","capabilities":{"tools":{}},"serverInfo":{"name":"demo","version":"1"}}}
mcp-send {"jsonrpc":"2.0","method":"notifications/initialized"}
mcp-send {"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"echo","arguments":{}}}
mcp-recv {"jsonrpc":"2.0","id":"call","result":{"content":[{"type":"text","text":"hello"}]}}
`

func TestConvert(t *testing.T) {
	tests := []struct {
		name, input string
		options     mcpfixture.Options
		want        string
	}{
		{"linear", trace, mcpfixture.Options{Record: 4}, ""},
		{"duplicate key", strings.Replace(trace, `"method":"tools/call"`, `"method":"tools/call","method":"tools/call"`, 1), mcpfixture.Options{Record: 4}, "duplicate JSON"},
		{"unknown initialize prerequisite", strings.Replace(trace, `"protocolVersion":"2025-11-25","capabilities":{}`, `"protocolVersion":"2025-11-25","_meta":{},"capabilities":{}`, 1), mcpfixture.Options{Record: 4}, "initialize parameters"},
		{"array arguments", strings.Replace(trace, `"arguments":{}`, `"arguments":[]`, 1), mcpfixture.Options{Record: 4}, "arguments must be an object"},
		{"null arguments", strings.Replace(trace, `"arguments":{}`, `"arguments":null`, 1), mcpfixture.Options{Record: 4}, "arguments must be an object"},

		{"selection required", trace, mcpfixture.Options{}, "positive"},
		{"wrong selected record", trace, mcpfixture.Options{Record: 1}, "preceding interactions"},
		{"missing response", trace[:strings.LastIndex(trace, "mcp-recv")], mcpfixture.Options{Record: 4}, "got 4 records"},
		{"truncated tail", trace + "mcp-recv {", mcpfixture.Options{Record: 4}, "incomplete trace"},
		{"callback", strings.Replace(trace, `"method":"tools/call"`, `"method":"sampling/createMessage"`, 1), mcpfixture.Options{Record: 4}, "tools/call"},
		{"capability prerequisite", strings.Replace(trace, `"capabilities":{}`, `"capabilities":{"sampling":{}}`, 1), mcpfixture.Options{Record: 4}, "prerequisites"},
		{"latest lifecycle", strings.ReplaceAll(trace, "2025-11-25", "2026-07-28"), mcpfixture.Options{Record: 4}, "discovery lifecycle"},
		{"nil capabilities", strings.Replace(trace, `"capabilities":{"tools":{}}`, `"capabilities":null`, 1), mcpfixture.Options{Record: 4}, "initialize result"},
		{"ID type mismatch", strings.Replace(trace, `"id":"call","result"`, `"id":4,"result"`, 1), mcpfixture.Options{Record: 4}, "matching"},
		{"concurrency", strings.Replace(trace, "mcp-recv {\"jsonrpc\":\"2.0\",\"id\":\"call\"", "mcp-send {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"ping\"}\nmcp-recv {\"jsonrpc\":\"2.0\",\"id\":\"call\"", 1), mcpfixture.Options{Record: 4}, "extra interactions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := mcpfixture.Convert(strings.NewReader(tt.input), &out, tt.options)
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error=%v, want %q", err, tt.want)
				}
				if out.Len() != 0 {
					t.Fatal("wrote partial fixture on invalid input")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			a := txtar.Parse(out.Bytes())
			if len(a.Files) != 1 || !bytes.Contains(a.Comment, []byte("exec mcpfixture check")) {
				t.Fatalf("not executable txtar: %s", out.Bytes())
			}
		})
	}
}

func TestGeneratedFixtureCheck(t *testing.T) {
	var out bytes.Buffer
	if err := mcpfixture.Convert(strings.NewReader(trace), &out, mcpfixture.Options{Record: 4}); err != nil {
		t.Fatal(err)
	}
	var fixture mcpfixture.Fixture
	if err := json.Unmarshal(txtar.Parse(out.Bytes()).Files[0].Data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"hello", "changed"} {
		t.Run(text, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server := sdk.NewServer(&sdk.Implementation{Name: "fixture", Version: "1"}, nil)
			server.AddTool(&sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}, nil
			})
			ct, st := sdk.NewInMemoryTransports()
			ss, err := server.Connect(ctx, st, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, &sdk.ClientOptions{Capabilities: &sdk.ClientCapabilities{}})
			cs, err := client.Connect(ctx, ct, &sdk.ClientSessionOptions{ProtocolVersion: fixture.ProtocolVersion})
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			err = mcpfixture.Check(ctx, cs, fixture)
			if text == "hello" && err != nil {
				t.Fatal(err)
			}
			if text == "changed" && (err == nil || !strings.Contains(err.Error(), "observed result changed")) {
				t.Fatalf("changed result accepted: %v", err)
			}
		})
	}
}

func TestReadRejectsUnsafeFixtures(t *testing.T) {
	for _, data := range []string{
		`{"protocolVersion":"2026-07-28","clientInfo":{"name":"demo","version":"1"},"params":{"name":"echo"},"expected":{"content":[]}}`,
		`{"protocolVersion":"2025-11-25","clientInfo":{"name":"demo","version":"1"},"params":{"name":"echo","arguments":null},"expected":{"content":[]}}`,
		`{"protocolVersion":"2025-11-25","clientInfo":{"name":"demo","version":"1"},"params":{"name":"echo"},"expected":{"content":[]},"sourceRecord":1,"sourceRecord":2}`,
	} {
		if _, err := mcpfixture.Read(strings.NewReader(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestReadRequestedProtocolAndContext(t *testing.T) {
	base := `{"protocolVersion":"2025-11-25","clientInfo":{"name":"demo","version":"1"},"params":{"name":"echo"},"expected":{"content":[]}}`
	for _, field := range []string{
		`"requestedProtocolVersion":"2099-01-01",`,
		`"clientCapabilities":{"sampling":{}},`,
	} {
		if _, err := mcpfixture.Read(strings.NewReader(strings.Replace(base, `"protocolVersion":`, field+`"protocolVersion":`, 1))); err == nil {
			t.Fatalf("accepted unsupported fixture %s", field)
		}
	}
}

func TestListPrerequisiteSelection(t *testing.T) {
	for _, tt := range []struct{ method, field string }{
		{"tools/list", "tools"}, {"prompts/list", "prompts"}, {"resources/list", "resources"}, {"resources/templates/list", "resourceTemplates"},
	} {
		t.Run(tt.method, func(t *testing.T) {
			lines := strings.SplitAfter(trace, "\n")
			lists := `mcp-send {"jsonrpc":"2.0","id":2,"method":"` + tt.method + `","params":{}}` + "\n" +
				`mcp-recv {"jsonrpc":"2.0","id":2,"result":{"` + tt.field + `":[],"nextCursor":"page2"}}` + "\n" +
				`mcp-send {"jsonrpc":"2.0","id":3,"method":"` + tt.method + `","params":{"cursor":"page2"}}` + "\n" +
				`mcp-recv {"jsonrpc":"2.0","id":3,"result":{"` + tt.field + `":[]}}` + "\n"
			input := strings.Join(lines[:3], "") + lists + strings.Join(lines[3:], "")
			var out bytes.Buffer
			if err := mcpfixture.Convert(strings.NewReader(input), &out, mcpfixture.Options{Record: 8}); err != nil {
				t.Fatal(err)
			}
			fixture, err := mcpfixture.Read(bytes.NewReader(txtar.Parse(out.Bytes()).Files[0].Data))
			if err != nil {
				t.Fatal(err)
			}
			if len(fixture.Prerequisites) != 2 {
				t.Fatal("lost list pages")
			}
			malformed := strings.Replace(input, `"cursor":"page2"`, `"cursor":"unknown"`, 1)
			if err := mcpfixture.Convert(strings.NewReader(malformed), &bytes.Buffer{}, mcpfixture.Options{Record: 8}); err == nil || !strings.Contains(err.Error(), "cursor") {
				t.Fatalf("accepted unknown cursor: %v", err)
			}
			incomplete := strings.Join(lines[:3], "") + strings.Join(strings.SplitAfter(lists, "\n")[2:], "") + strings.Join(lines[3:], "")
			if err := mcpfixture.Convert(strings.NewReader(incomplete), &bytes.Buffer{}, mcpfixture.Options{Record: 6}); err == nil {
				t.Fatal("accepted cursor without preceding page")
			}
		})
	}
}
func TestRefuseStatefulPrerequisites(t *testing.T) {
	for _, method := range []string{"tools/call", "resources/read", "prompts/get", "subscriptions/listen"} {
		inserted := `mcp-send {"jsonrpc":"2.0","id":2,"method":"` + method + `","params":{}}` + "\n" + `mcp-recv {"jsonrpc":"2.0","id":2,"result":{}}` + "\n"
		lines := strings.SplitAfter(trace, "\n")
		input := strings.Join(lines[:3], "") + inserted + strings.Join(lines[3:], "")
		if err := mcpfixture.Convert(strings.NewReader(input), &bytes.Buffer{}, mcpfixture.Options{Record: 6}); err == nil {
			t.Fatalf("accepted earlier %s", method)
		}
	}
	for _, metadata := range []string{`"_meta":null,`, `"_meta":{"progressToken":"p"},`, `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"},`, `"requestState":"state",`, `"inputResponses":{"call":{"result":{}}},`} {
		input := strings.Replace(trace, `"params":{"name":"echo"`, `"params":{`+metadata+`"name":"echo"`, 1)
		if err := mcpfixture.Convert(strings.NewReader(input), &bytes.Buffer{}, mcpfixture.Options{Record: 4}); err == nil {
			t.Fatalf("accepted metadata or continuation %s", metadata)
		}
	}
}

func TestClientConfiguration(t *testing.T) {
	fixture := mcpfixture.Fixture{ClientCapabilities: json.RawMessage(`{"extensions":{"vendor/feature":{"enabled":true}},"experimental":{"fixture":{}}}`)}
	options, err := mcpfixture.ClientOptions(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if options.Capabilities.Extensions["vendor/feature"] == nil || options.Capabilities.Experimental["fixture"] == nil {
		t.Fatal("lost configuration")
	}
	for _, raw := range []string{`{"experimental":null}`, `{"extensions":{}}`} {
		fixture.ClientCapabilities = json.RawMessage(raw)
		if _, err := mcpfixture.ClientOptions(fixture); err == nil {
			t.Fatalf("accepted lossy configuration %s", raw)
		}
	}
}
