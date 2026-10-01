package mcpcatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func catalog(input, output string) *Snapshot {
	var in, out any
	if input != "" {
		in = json.RawMessage(input)
	}
	if output != "" {
		out = json.RawMessage(output)
	}
	return &Snapshot{Format: 1, Context: Context{Scope: "project", Principal: "local", Client: "default"}, Initialize: &mcp.InitializeResult{ProtocolVersion: "2026-07-28", ServerInfo: &mcp.Implementation{Name: "test"}, Capabilities: &mcp.ServerCapabilities{}}, Tools: []*mcp.Tool{{Name: "search", InputSchema: in, OutputSchema: out}}}
}
func TestCompare(t *testing.T) {
	const narrow = `{"type":"object","properties":{"q":{"type":"string","enum":["a"]}},"required":["q"],"additionalProperties":false}`
	const wide = `{"type":"object","properties":{"q":{"type":"string"}},"additionalProperties":false}`
	const unsupported = `{"type":"object","properties":{"q":{"type":"string","pattern":"a"}}}`
	for _, tt := range []struct {
		name, inOld, inNew, outOld, outNew string
		want                               Kind
	}{
		{"unchanged", narrow, narrow, "", "", ""},
		{"wider input", narrow, wide, "", "", Changed},
		{"narrower input", wide, narrow, "", "", Incompatible},
		{"narrower output", narrow, narrow, wide, narrow, Changed},
		{"wider output", narrow, narrow, narrow, wide, Incompatible},
		{"unchanged unsupported", unsupported, unsupported, "", "", Unknown},
		{"removed output", narrow, narrow, narrow, "", Unknown},
		{"unconstrained property", `{"type":"object"}`, `{"type":"object","properties":{"q":{"type":"string"}}}`, "", "", Incompatible},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changes := Compare(catalog(tt.inOld, tt.outOld), catalog(tt.inNew, tt.outNew))
			if tt.want == "" {
				if len(changes) != 0 {
					t.Fatalf("got %+v", changes)
				}
				return
			}
			if len(changes) != 1 || changes[0].Kind != tt.want {
				t.Fatalf("got %+v, want %s", changes, tt.want)
			}
		})
	}
}
func TestContext(t *testing.T) {
	a := catalog(`{"type":"object"}`, "")
	b := catalog(`{"type":"object"}`, "")
	b.Context.Principal = "other"
	if c := Compare(a, b); len(c) != 1 || c[0].Kind != Unknown {
		t.Fatalf("got %+v", c)
	}
}
func TestReadWrite(t *testing.T) {
	s := catalog(`{"type":"object"}`, "")
	s.Tools = append(s.Tools, &mcp.Tool{Name: "alpha", InputSchema: json.RawMessage(`{"type":"object"}`)})
	var a, b bytes.Buffer
	if err := Write(&a, s); err != nil {
		t.Fatal(err)
	}
	read, err := Read(&a)
	if err != nil {
		t.Fatal(err)
	}
	if read.Tools[0].Name != "alpha" || s.Tools[0].Name != "search" {
		t.Fatal("ordering modified original or not normalized")
	}
	if err := Write(&a, read); err != nil {
		t.Fatal(err)
	}
	if err := Write(&b, s); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("nondeterministic output")
	}
	if _, err := Read(strings.NewReader(b.String() + " {}")); err == nil {
		t.Fatal("accepted trailing object")
	}
	s.Tools = append(s.Tools, s.Tools[0])
	if err := Write(&a, s); err == nil {
		t.Fatal("accepted duplicate tools")
	}
}
func TestCapture(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "search", InputSchema: json.RawMessage(`{"type":"object"}`)}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Error("capture called tool")
		return nil, nil
	})
	a, b := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, a, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	s, err := Capture(ctx, cs, Context{Scope: "test", Principal: "local", Client: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tools) != 1 || s.Tools[0].Name != "search" {
		t.Fatalf("got %+v", s.Tools)
	}
	s.Initialize.ServerInfo.Name = "detached"
	if cs.InitializeResult().ServerInfo.Name == "detached" {
		t.Fatal("snapshot aliases session")
	}
}

func TestInvalidSchemasUnknown(t *testing.T) {
	for _, s := range []string{
		`{"type":"object","properties":{"q":{"type":"string","enum":["a",null]}}}`,
		`{"type":"object","properties":{"q":{"type":"string","enum":["a","a"]}}}`,
		`{"type":"object","description":null}`,
		`{"type":"object","properties":{"q":{"type":"string"}},"required":[null]}`,
	} {
		if c := Compare(catalog(s, ""), catalog(s, "")); len(c) != 1 || c[0].Kind != Unknown {
			t.Fatalf("schema %s: got %+v", s, c)
		}
	}
}
