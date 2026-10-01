package mcpmiddleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLog(t *testing.T) {
	var output bytes.Buffer
	wantErr := errors.New("failed")
	middleware := Log(slog.New(slog.NewTextHandler(&output, nil)))
	called := false
	handler := middleware(func(_ context.Context, method string, req mcp.Request) (mcp.Result, error) {
		called = true
		if method != "tools/call" || req != nil {
			t.Fatalf("method=%q req=%v", method, req)
		}
		return nil, wantErr
	})
	_, err := handler(context.Background(), "tools/call", nil)
	if !called {
		t.Fatal("next handler not called")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v", err)
	}
	for _, want := range []string{"tools/call", "failed", "duration"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("log %q lacks %q", output.String(), want)
		}
	}
	if strings.Contains(output.String(), "secret") {
		t.Fatalf("request body leaked: %q", output.String())
	}
}

func ExampleLog() {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	_ = Log(logger)
	// Output:
}
