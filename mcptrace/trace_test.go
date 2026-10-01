package mcptrace

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	when := time.Unix(1734567890, 123000000)
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.Write(Record{Direction: "recv", Message: []byte(`{"jsonrpc":"2.0","id":1,"note":" # not a timestamp"}`), Time: when, Metadata: []string{"spanid=req1", "linksto=resp1"}}); err != nil {
		t.Fatal(err)
	}
	r := NewReader(&buf)
	got, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got.Direction != "recv" || string(got.Message) != `{"jsonrpc":"2.0","id":1,"note":" # not a timestamp"}` || !got.Time.Equal(when) {
		t.Fatalf("Read() = %#v", got)
	}
	if !reflect.DeepEqual(got.Metadata, []string{"spanid=req1", "linksto=resp1"}) {
		t.Fatalf("Read() metadata = %#v", got.Metadata)
	}
	if _, err := r.Read(); err != io.EOF {
		t.Fatalf("Read() error = %v, want EOF", err)
	}
}

func ExampleWriter() {
	var buf bytes.Buffer
	_ = NewWriter(&buf).Write(Record{Direction: "send", Message: []byte(`{"jsonrpc":"2.0","method":"ping"}`)})
	fmt.Print(buf.String())
	// Output: mcp-send {"jsonrpc":"2.0","method":"ping"}
}

func ExampleReader() {
	r := NewReader(strings.NewReader("mcp-send {\"jsonrpc\":\"2.0\",\"method\":\"ping\"}\n"))
	record, _ := r.Read()
	fmt.Printf("%s %s", record.Direction, record.Message)
	// Output: send {"jsonrpc":"2.0","method":"ping"}
}
