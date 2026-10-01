package mcptrace

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIOTransportRecordsSDKSession(t *testing.T) {
	var trace bytes.Buffer
	left, right := net.Pipe()
	traceWriter := NewWriter(&trace)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1"}, nil)
	serverSession, err := server.Connect(ctx, &mcp.IOTransport{Reader: right, Writer: right}, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, &mcp.IOTransport{
		Reader: ReadCloser(left, traceWriter, "send"),
		Writer: WriteCloser(left, traceWriter, "recv"),
	}, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	if err := clientSession.Close(); err != nil {
		t.Fatal(err)
	}
	_ = serverSession.Close()
	checkTraceDirectionsAtLeast(t, &trace, 2)
}

func TestStreamableHTTPRecordsSDKSessions(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-03-26"} {
		t.Run(version, func(t *testing.T) {
			var trace bytes.Buffer
			var serverTrace bytes.Buffer
			server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1"}, &mcp.ServerOptions{
				SupportedProtocolVersions: []string{version},
			})
			httpOpts := &mcp.StreamableHTTPOptions{Stateless: version == "2026-07-28"}
			handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, httpOpts)
			httpServer := httptest.NewServer(Handler(handler, NewWriter(&serverTrace)))
			defer httpServer.Close()
			client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
			transport := &mcp.StreamableClientTransport{
				Endpoint:             httpServer.URL,
				HTTPClient:           &http.Client{Transport: &RoundTripper{Trace: NewWriter(&trace)}},
				DisableStandaloneSSE: true,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			session, err := client.Connect(ctx, transport, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			checkTraceDirectionsAtLeast(t, &trace, 2)
			checkTraceDirectionsAtLeast(t, &serverTrace, 2)
		})
	}
}

func checkTraceDirectionsAtLeast(t *testing.T, src io.Reader, min int) {
	t.Helper()
	r := NewReader(src)
	count := 0
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count < min {
		t.Fatalf("recorded %d messages, want at least %d", count, min)
	}
}
