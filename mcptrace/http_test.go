package mcptrace

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoundTripperRecordsMessages(t *testing.T) {
	var trace bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1}`)
	}))
	defer server.Close()
	client := &http.Client{Transport: &RoundTripper{Trace: NewWriter(&trace)}}
	req, err := http.NewRequest(http.MethodPost, server.URL, io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","method":"ping"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	checkTraceDirections(t, &trace, []string{"send", "recv"})
}

func TestHandlerRecordsMessages(t *testing.T) {
	var trace bytes.Buffer
	h := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1}`)
	}), NewWriter(&trace))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","method":"ping"}`))
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	checkTraceDirections(t, &trace, []string{"recv", "send"})
}

func checkTraceDirections(t *testing.T, src io.Reader, want []string) {
	t.Helper()
	r := NewReader(src)
	for i, direction := range want {
		rec, err := r.Read()
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		if rec.Direction != direction {
			t.Fatalf("record %d direction = %q, want %q", i, rec.Direction, direction)
		}
	}
	if _, err := r.Read(); err != io.EOF {
		t.Fatalf("final Read() error = %v, want EOF", err)
	}
}
