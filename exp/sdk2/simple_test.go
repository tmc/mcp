package sdk2

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBasicTypes(t *testing.T) {
	// Test TextContent
	text := TextContent{Text: "Hello, world!"}
	if text.ContentType() != "text/plain" {
		t.Errorf("TextContent.ContentType() = %s, want text/plain", text.ContentType())
	}

	data, err := json.Marshal(text)
	if err != nil {
		t.Fatalf("json.Marshal(TextContent) failed: %v", err)
	}

	expected := `{"type":"text","text":"Hello, world!"}`
	if string(data) != expected {
		t.Errorf("TextContent JSON = %s, want %s", string(data), expected)
	}

	// Test ImageContent
	image := ImageContent{Data: "base64data", MimeType: "image/png"}
	if image.ContentType() != "image/png" {
		t.Errorf("ImageContent.ContentType() = %s, want image/png", image.ContentType())
	}

	data, err = json.Marshal(image)
	if err != nil {
		t.Fatalf("json.Marshal(ImageContent) failed: %v", err)
	}

	expected = `{"type":"image","data":"base64data","mimeType":"image/png"}`
	if string(data) != expected {
		t.Errorf("ImageContent JSON = %s, want %s", string(data), expected)
	}
}

func TestRequestID(t *testing.T) {
	// Test string ID
	id := RequestID{Value: "test-id"}
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal(RequestID) failed: %v", err)
	}
	if string(data) != `"test-id"` {
		t.Errorf("RequestID JSON = %s, want \"test-id\"", string(data))
	}
	if id.String() != "test-id" {
		t.Errorf("RequestID.String() = %s, want test-id", id.String())
	}

	// Test number ID
	id = RequestID{Value: int64(123)}
	data, err = json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal(RequestID) failed: %v", err)
	}
	if string(data) != "123" {
		t.Errorf("RequestID JSON = %s, want 123", string(data))
	}
	if id.String() != "123" {
		t.Errorf("RequestID.String() = %s, want 123", id.String())
	}

	// Test nil ID
	id = RequestID{Value: nil}
	data, err = json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal(RequestID) failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("RequestID JSON = %s, want null", string(data))
	}
	if id.String() != "<nil>" {
		t.Errorf("RequestID.String() = %s, want <nil>", id.String())
	}
}

func TestServeMux(t *testing.T) {
	mux := NewServeMux()

	// Test registering a handler
	called := false
	mux.HandleFunc("test/method", func(w ResponseWriter, r *Request) {
		called = true
		w.WriteHeader(StatusOK)
	})

	// Create a mock request
	req := &Request{
		Method:  "test/method",
		Context: context.Background(),
	}

	// Create a mock response writer
	rw := &mockResponseWriter{header: make(Header)}

	// Call the handler
	mux.ServeRequest(rw, req)

	if !called {
		t.Error("Handler was not called")
	}

	if rw.statusCode != StatusOK {
		t.Errorf("Status code = %d, want %d", rw.statusCode, StatusOK)
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		called = true
		w.WriteHeader(StatusOK)
	})

	req := &Request{
		Method:  "test",
		Context: context.Background(),
	}
	rw := &mockResponseWriter{header: make(Header)}

	handler.ServeRequest(rw, req)

	if !called {
		t.Error("HandlerFunc was not called")
	}
}

func TestNotFoundHandler(t *testing.T) {
	handler := NotFoundHandler()
	req := &Request{
		Method:  "unknown",
		Context: context.Background(),
	}
	rw := &mockResponseWriter{header: make(Header)}

	handler.ServeRequest(rw, req)

	if rw.statusCode != StatusNotFound {
		t.Errorf("Status code = %d, want %d", rw.statusCode, StatusNotFound)
	}

	if !strings.Contains(rw.body, "404") {
		t.Errorf("Response body should contain '404', got: %s", rw.body)
	}
}

func TestConstants(t *testing.T) {
	if ProtocolVersion != "2025-03-26" {
		t.Errorf("ProtocolVersion = %s, want 2025-03-26", ProtocolVersion)
	}

	if DefaultPort != "3000" {
		t.Errorf("DefaultPort = %s, want 3000", DefaultPort)
	}

	// Test method constants
	expectedMethods := map[string]string{
		MethodInitialize:     "initialize",
		MethodInitialized:    "notifications/initialized",
		MethodToolsList:      "tools/list",
		MethodToolsCall:      "tools/call",
		MethodResourcesList:  "resources/list",
		MethodResourcesRead:  "resources/read",
		MethodPromptsList:    "prompts/list",
		MethodPromptsGet:     "prompts/get",
		MethodLoggingLog:     "logging/setLevel",
		MethodProgress:       "notifications/progress",
		MethodCancelled:      "notifications/cancelled",
	}

	for constant, expected := range expectedMethods {
		if constant != expected {
			t.Errorf("Method constant mismatch: got %s, want %s", constant, expected)
		}
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		code int
		text string
	}{
		{StatusOK, "OK"},
		{StatusBadRequest, "Bad Request"},
		{StatusNotFound, "Not Found"},
		{StatusInternalServerError, "Internal Server Error"},
		{999, "Unknown Status Code 999"},
	}

	for _, test := range tests {
		got := StatusText(test.code)
		if got != test.text {
			t.Errorf("StatusText(%d) = %s, want %s", test.code, got, test.text)
		}
	}
}

func TestRequestWithContext(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.WithValue(context.Background(), "key", "value")

	req := &Request{
		Method:  "test",
		Context: ctx1,
	}

	req2 := req.WithContext(ctx2)

	// Should be a different request
	if req == req2 {
		t.Error("WithContext should return a new request")
	}

	// Should have the new context
	if req2.Context != ctx2 {
		t.Error("WithContext should set the new context")
	}

	// Original should be unchanged
	if req.Context != ctx1 {
		t.Error("Original request context should be unchanged")
	}

	// Should panic with nil context
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithContext(nil) should panic")
		}
	}()
	req.WithContext(nil)
}

// Mock response writer for testing
type mockResponseWriter struct {
	header     Header
	body       string
	statusCode int
}

func (w *mockResponseWriter) Header() Header {
	return w.header
}

func (w *mockResponseWriter) Write(data []byte) (int, error) {
	w.body += string(data)
	return len(data), nil
}

func (w *mockResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func TestClientConfig(t *testing.T) {
	config := &ClientConfig{
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want %v", config.Timeout, 10*time.Second)
	}

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", config.MaxRetries)
	}

	if config.ClientInfo.Name != "test-client" {
		t.Errorf("ClientInfo.Name = %s, want test-client", config.ClientInfo.Name)
	}
}

func BenchmarkTextContentMarshal(b *testing.B) {
	content := TextContent{Text: "Hello, world!"}
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRequestIDMarshal(b *testing.B) {
	id := RequestID{Value: "test-id-123"}
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(id)
		if err != nil {
			b.Fatal(err)
		}
	}
}