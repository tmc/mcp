package mcpcorpus

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func call(arguments string) string {
	return `mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":` + arguments + `}}` + "\n"
}
func TestExtract(t *testing.T) {
	trace := call(`{"q":"a"}`) + call(`{"q":"a"}`) + call(`{ "q": "a" }`)
	seeds, err := Extract(strings.NewReader(trace), "search")
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 2 {
		t.Fatalf("got %d seeds", len(seeds))
	}
	if seeds[0].Name >= seeds[1].Name {
		t.Fatal("not sorted")
	}
	reverse, err := Extract(strings.NewReader(call(`{ "q": "a" }`)+call(`{"q":"a"}`)), "search")
	if err != nil {
		t.Fatal(err)
	}
	for i := range seeds {
		if seeds[i].Name != reverse[i].Name || !bytes.Equal(seeds[i].Arguments, reverse[i].Arguments) {
			t.Fatal("order changed seeds")
		}
	}
	other, err := Extract(strings.NewReader(trace), "other")
	if err != nil || len(other) != 0 {
		t.Fatalf("got %+v, %v", other, err)
	}
}
func TestMalformed(t *testing.T) {
	for _, tt := range []struct{ name, trace string }{
		{"null", call("null")}, {"array", call("[]")}, {"scalar", call(`"a"`)},
		{"absent", `mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search"}}`},
		{"notification", `mcp-send {"jsonrpc":"2.0","method":"tools/call","params":{"name":"search","arguments":{}}}`},
		{"invalid version", `mcp-send {"jsonrpc":"1.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{}}}`},
		{"invalid id", `mcp-send {"jsonrpc":"2.0","id":{},"method":"tools/call","params":{"name":"search","arguments":{}}}`},
		{"duplicate selection", `mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"other","name":"search","arguments":{}}}`},
		{"duplicate argument key", call(`{"q":1,"q":2}`)},
		{"truncated", call(`{}`) + `mcp-send {"jsonrpc":`},
		{"batch", `mcp-send []`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			seeds, err := Extract(strings.NewReader(tt.trace), "search")
			if err == nil || seeds != nil {
				t.Fatalf("got seeds=%+v err=%v", seeds, err)
			}
		})
	}
}
func TestDirectionsAndSessions(t *testing.T) {
	// Extraction does not infer a client direction, correlate reused IDs, or require initialization.
	trace := strings.TrimSuffix(call(`{"q":"a"}`), "\n") + " # 1 session=a\n" +
		strings.TrimSuffix(strings.ReplaceAll(call(`{"q":"b"}`), "mcp-send", "mcp-recv"), "\n") + " # 2 session=b\n"
	seeds, err := Extract(strings.NewReader(trace), "search")
	if err != nil || len(seeds) != 2 {
		t.Fatalf("got %+v, %v", seeds, err)
	}
}
func TestWriteFuzz(t *testing.T) {
	raw := []byte(`{"q":"a"}`)
	var out bytes.Buffer
	if err := WriteFuzz(&out, Seed{Arguments: raw}); err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/fuzz/FuzzArguments/simple")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), expected) {
		t.Fatalf("got %q, want %q", out.Bytes(), expected)
	}
	out.Reset()
	if err := WriteJSON(&out, Seed{Arguments: raw}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), append(raw, '\n')) {
		t.Fatal("JSON bytes changed")
	}
	if err := WriteFuzz(&out, Seed{Arguments: []byte("null")}); err == nil {
		t.Fatal("accepted null")
	}
}

// FuzzArguments exercises the native testing.F parser against the seed fixture
// that TestWriteFuzz requires WriteFuzz to reproduce byte for byte.
func FuzzArguments(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) { var arguments map[string]any; _ = json.Unmarshal(data, &arguments) })
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestShortWrite(t *testing.T) {
	seed := Seed{Arguments: []byte(`{}`)}
	if err := WriteFuzz(shortWriter{}, seed); err == nil {
		t.Fatal("accepted short fuzz write")
	}
	if err := WriteJSON(shortWriter{}, seed); err == nil {
		t.Fatal("accepted short JSON write")
	}
}
