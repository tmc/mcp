package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

func TestError(t *testing.T) {
	err := &Error{
		Code:    CodeInternalError,
		Message: "internal error",
		Data:    "additional data",
	}

	expected := "jsonrpc2 error: code=-32603 message=internal error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestID_IsValid(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want bool
	}{
		{"numeric ID", ID{Num: 123}, true},
		{"string ID", ID{Str: "test"}, true},
		{"null ID", ID{raw: json.RawMessage("null")}, true},
		{"zero ID", ID{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsValid(); got != tt.want {
				t.Errorf("ID.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64ID(t *testing.T) {
	id := Int64ID(42)
	if id.Num != 42 {
		t.Errorf("Int64ID(42).Num = %d, want 42", id.Num)
	}
	if id.Str != "" {
		t.Errorf("Int64ID(42).Str = %q, want empty", id.Str)
	}
}

func TestStringID(t *testing.T) {
	id := StringID("test")
	if id.Str != "test" {
		t.Errorf("StringID(\"test\").Str = %q, want \"test\"", id.Str)
	}
	if id.Num != 0 {
		t.Errorf("StringID(\"test\").Num = %d, want 0", id.Num)
	}
}

func TestRequest_NotificationMethod(t *testing.T) {
	tests := []struct {
		name string
		req  *Request
		want bool
	}{
		{"request with valid ID", &Request{ID: Int64ID(1)}, false},
		{"notification without ID", &Request{ID: ID{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.NotificationMethod(); got != tt.want {
				t.Errorf("Request.NotificationMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testReadWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (rw *testReadWriteCloser) Close() error {
	rw.closed = true
	return nil
}

func (rw *testReadWriteCloser) Write(p []byte) (n int, err error) {
	return rw.Buffer.Write(p)
}

type connCaptureReadWriteCloser struct {
	mu       sync.Mutex
	readBuf  *bytes.Buffer
	writeBuf bytes.Buffer
	closed   bool
}

func newConnCaptureReadWriteCloser(data []byte) *connCaptureReadWriteCloser {
	return &connCaptureReadWriteCloser{
		readBuf: bytes.NewBuffer(data),
	}
}

func (rw *connCaptureReadWriteCloser) Read(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.closed {
		return 0, io.EOF
	}
	return rw.readBuf.Read(p)
}

func (rw *connCaptureReadWriteCloser) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.closed {
		return 0, io.ErrClosedPipe
	}
	return rw.writeBuf.Write(p)
}

func (rw *connCaptureReadWriteCloser) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.closed = true
	return nil
}

func (rw *connCaptureReadWriteCloser) WrittenBytes() []byte {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return append([]byte(nil), rw.writeBuf.Bytes()...)
}

func decodeWrittenMessage(t *testing.T, rw *connCaptureReadWriteCloser) Message {
	t.Helper()

	data := bytes.TrimSpace(rw.WrittenBytes())
	if len(data) == 0 {
		t.Fatal("no written message captured")
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal captured message: %v", err)
	}
	return msg
}

func TestNewBufferedStream(t *testing.T) {
	rw := &testReadWriteCloser{Buffer: bytes.NewBuffer(nil)}
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})

	if stream == nil {
		t.Fatal("NewBufferedStream returned nil")
	}

	if stream.reader == nil {
		t.Error("BufferedStream.reader is nil")
	}

	if stream.writer != rw {
		t.Error("BufferedStream.writer not set correctly")
	}

	if stream.closer != rw {
		t.Error("BufferedStream.closer not set correctly")
	}
}

func TestBufferedStream_ReadWrite(t *testing.T) {
	rw := &testReadWriteCloser{Buffer: bytes.NewBuffer(nil)}
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})

	// Test writing a message
	msg := Message{
		Version: Version,
		Method:  "test/method",
		Params:  json.RawMessage(`{"param":"value"}`),
	}

	ctx := context.Background()
	err := stream.Write(ctx, msg)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test reading the message back
	readMsg, err := stream.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if readMsg.Version != msg.Version {
		t.Errorf("Read message version = %q, want %q", readMsg.Version, msg.Version)
	}

	if readMsg.Method != msg.Method {
		t.Errorf("Read message method = %q, want %q", readMsg.Method, msg.Method)
	}

	if !bytes.Equal(readMsg.Params, msg.Params) {
		t.Errorf("Read message params = %q, want %q", readMsg.Params, msg.Params)
	}
}

func TestBufferedStream_Close(t *testing.T) {
	rw := &testReadWriteCloser{Buffer: bytes.NewBuffer(nil)}
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})

	err := stream.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !rw.closed {
		t.Error("Underlying closer was not called")
	}
}

func TestVSCodeObjectCodec(t *testing.T) {
	codec := VSCodeObjectCodec{}

	// Test encode
	data := map[string]interface{}{
		"method": "test",
		"id":     123,
	}

	encoded, err := codec.Encode(data)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Test decode
	var decoded map[string]interface{}
	err = codec.Decode(encoded, &decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded["method"] != data["method"] {
		t.Errorf("Decoded method = %v, want %v", decoded["method"], data["method"])
	}

	// ID comes back as float64 from JSON
	if decoded["id"] != float64(123) {
		t.Errorf("Decoded id = %v, want %v", decoded["id"], float64(123))
	}
}

type testHandler struct {
	requests []*Request
	mu       sync.Mutex
}

func (h *testHandler) Handle(ctx context.Context, conn *Conn, req *Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests = append(h.requests, req)

	// Echo back the method name as result
	if !req.NotificationMethod() {
		conn.Reply(ctx, req.ID, map[string]string{"echo": req.Method})
	}
}

func (h *testHandler) getRequests() []*Request {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]*Request, len(h.requests))
	copy(result, h.requests)
	return result
}

func TestConn_NewConn(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	if conn == nil {
		t.Fatal("NewConn returned nil")
	}

	if conn.stream != stream {
		t.Error("Conn.stream not set correctly")
	}

	if conn.handler != handler {
		t.Error("Conn.handler not set correctly")
	}
}

func TestConn_Reply(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	// Test numeric ID reply
	id := Int64ID(123)
	result := map[string]string{"result": "success"}

	err := conn.Reply(ctx, id, result)
	if err != nil {
		t.Fatalf("Reply failed: %v", err)
	}

	// Read back the response
	response := decodeWrittenMessage(t, rw)

	if response.Version != Version {
		t.Errorf("Response version = %q, want %q", response.Version, Version)
	}

	var resultData map[string]string
	err = json.Unmarshal(response.Result, &resultData)
	if err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if resultData["result"] != "success" {
		t.Errorf("Response result = %v, want %v", resultData, result)
	}
}

func TestConn_ReplyWithData(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	// Test string ID reply
	id := StringID("test-id")
	result := json.RawMessage(`{"raw":"data"}`)

	err := conn.ReplyWithData(ctx, id, result)
	if err != nil {
		t.Fatalf("ReplyWithData failed: %v", err)
	}

	// Read back the response
	response := decodeWrittenMessage(t, rw)

	if !bytes.Equal(response.Result, result) {
		t.Errorf("Response result = %q, want %q", response.Result, result)
	}

	var responseID string
	err = json.Unmarshal(response.ID, &responseID)
	if err != nil {
		t.Fatalf("Unmarshal response ID failed: %v", err)
	}

	if responseID != "test-id" {
		t.Errorf("Response ID = %q, want %q", responseID, "test-id")
	}
}

func TestConn_ReplyWithError(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	id := Int64ID(456)
	errorObj := &Error{
		Code:    CodeInvalidParams,
		Message: "invalid parameters",
		Data:    "test data",
	}

	err := conn.ReplyWithError(ctx, id, errorObj)
	if err != nil {
		t.Fatalf("ReplyWithError failed: %v", err)
	}

	// Read back the response
	response := decodeWrittenMessage(t, rw)

	if response.Error == nil {
		t.Fatal("Response error is nil")
	}

	if response.Error.Code != CodeInvalidParams {
		t.Errorf("Response error code = %d, want %d", response.Error.Code, CodeInvalidParams)
	}

	if response.Error.Message != "invalid parameters" {
		t.Errorf("Response error message = %q, want %q", response.Error.Message, "invalid parameters")
	}
}

func TestConn_Notify(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	params := map[string]string{"param": "value"}

	err := conn.Notify(ctx, "test/notification", params)
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	// Read back the notification
	notification := decodeWrittenMessage(t, rw)

	if notification.Method != "test/notification" {
		t.Errorf("Notification method = %q, want %q", notification.Method, "test/notification")
	}

	var notificationParams map[string]string
	err = json.Unmarshal(notification.Params, &notificationParams)
	if err != nil {
		t.Fatalf("Unmarshal notification params failed: %v", err)
	}

	if notificationParams["param"] != "value" {
		t.Errorf("Notification params = %v, want %v", notificationParams, params)
	}

	// Notifications should not have an ID
	if len(notification.ID) > 0 {
		t.Errorf("Notification has ID %q, should be empty", notification.ID)
	}
}

func TestConn_NotifyWithData(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	params := json.RawMessage(`{"raw":"notification"}`)

	err := conn.NotifyWithData(ctx, "test/raw", params)
	if err != nil {
		t.Fatalf("NotifyWithData failed: %v", err)
	}

	// Read back the notification
	notification := decodeWrittenMessage(t, rw)

	if !bytes.Equal(notification.Params, params) {
		t.Errorf("Notification params = %q, want %q", notification.Params, params)
	}
}

func TestConn_DisconnectNotify(t *testing.T) {
	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	conn := NewConn(ctx, stream, handler)

	ch := conn.DisconnectNotify()
	select {
	case <-ch:
		// Expected - channel should be closed immediately in current implementation
	case <-time.After(100 * time.Millisecond):
		t.Error("DisconnectNotify channel was not closed")
	}
}

func TestEncodeDecodeMessage(t *testing.T) {
	msg := Message{
		Version: Version,
		Method:  "test",
		Params:  json.RawMessage(`{"test":true}`),
		ID:      json.RawMessage(`123`),
	}

	// Test encoding
	data, err := EncodeMessage(msg)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	// Test decoding
	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if decoded.Version != msg.Version {
		t.Errorf("Decoded version = %q, want %q", decoded.Version, msg.Version)
	}

	if decoded.Method != msg.Method {
		t.Errorf("Decoded method = %q, want %q", decoded.Method, msg.Method)
	}

	if !bytes.Equal(decoded.Params, msg.Params) {
		t.Errorf("Decoded params = %q, want %q", decoded.Params, msg.Params)
	}

	if !bytes.Equal(decoded.ID, msg.ID) {
		t.Errorf("Decoded ID = %q, want %q", decoded.ID, msg.ID)
	}
}

type testPipeReadWriteCloser struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func newTestPipe() *testPipeReadWriteCloser {
	r, w := io.Pipe()
	return &testPipeReadWriteCloser{reader: r, writer: w}
}

func (p *testPipeReadWriteCloser) Read(data []byte) (int, error) {
	return p.reader.Read(data)
}

func (p *testPipeReadWriteCloser) Write(data []byte) (int, error) {
	return p.writer.Write(data)
}

func (p *testPipeReadWriteCloser) Close() error {
	p.writer.Close()
	return p.reader.Close()
}

func TestConn_Integration(t *testing.T) {
	// Create a more realistic test with actual message exchange
	pipe := newTestPipe()
	stream := NewBufferedStream(pipe, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = NewConn(ctx, stream, handler)

	// Write a request message to simulate incoming request
	go func() {
		requestMsg := Message{
			Version: Version,
			ID:      json.RawMessage(`"req-123"`),
			Method:  "test/echo",
			Params:  json.RawMessage(`{"message":"hello"}`),
		}

		time.Sleep(10 * time.Millisecond) // Small delay to ensure connection is ready
		err := stream.Write(ctx, requestMsg)
		if err != nil {
			t.Errorf("Failed to write request: %v", err)
		}
	}()

	// Give some time for the request to be processed
	time.Sleep(100 * time.Millisecond)

	// Check that the handler received the request
	requests := handler.getRequests()
	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	req := requests[0]
	if req.Method != "test/echo" {
		t.Errorf("Request method = %q, want %q", req.Method, "test/echo")
	}

	if req.ID.Str != "req-123" {
		t.Errorf("Request ID = %q, want %q", req.ID.Str, "req-123")
	}

	var params map[string]string
	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal params: %v", err)
	}

	if params["message"] != "hello" {
		t.Errorf("Request params message = %q, want %q", params["message"], "hello")
	}
}

func TestDial(t *testing.T) {
	// Create a simple in-memory connection for testing
	rw := newConnCaptureReadWriteCloser(nil)
	handler := &testHandler{}

	ctx := context.Background()
	conn, err := Dial(ctx, rw, handler)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if conn == nil {
		t.Fatal("Dial returned nil connection")
	}

	if conn.handler != handler {
		t.Error("Connection handler not set correctly")
	}
}

func TestHandlerFunc(t *testing.T) {
	var called bool
	var receivedReq *Request

	handlerFunc := HandlerFunc(func(ctx context.Context, conn *Conn, req *Request) {
		called = true
		receivedReq = req
	})

	rw := newConnCaptureReadWriteCloser(nil)
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	conn := NewConn(context.Background(), stream, &testHandler{})

	req := &Request{
		ID:     StringID("test"),
		Method: "test/method",
	}

	handlerFunc.Handle(context.Background(), conn, req)

	if !called {
		t.Error("HandlerFunc was not called")
	}

	if receivedReq != req {
		t.Error("HandlerFunc did not receive the correct request")
	}
}

func TestConn_ReadMessages_ErrorHandling(t *testing.T) {
	// Test reading invalid JSON
	rw := &testReadWriteCloser{Buffer: bytes.NewBufferString("invalid json\n")}
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = NewConn(ctx, stream, handler)

	// Give some time for the goroutine to process the invalid message
	time.Sleep(50 * time.Millisecond)

	// Should not have received any valid requests
	requests := handler.getRequests()
	if len(requests) != 0 {
		t.Errorf("Expected 0 requests for invalid JSON, got %d", len(requests))
	}
}

func TestConn_ReadMessages_EOF(t *testing.T) {
	// Test handling EOF
	rw := &testReadWriteCloser{Buffer: bytes.NewBuffer(nil)}
	stream := NewBufferedStream(rw, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx := context.Background()
	_ = NewConn(ctx, stream, handler)

	// Close the buffer to simulate EOF
	rw.Close()

	// Give some time for the goroutine to process EOF
	time.Sleep(50 * time.Millisecond)

	// Should handle EOF gracefully without crashing
	requests := handler.getRequests()
	if len(requests) != 0 {
		t.Errorf("Expected 0 requests after EOF, got %d", len(requests))
	}
}

func TestConn_ReadMessages_Notification(t *testing.T) {
	pipe := newTestPipe()
	stream := NewBufferedStream(pipe, VSCodeObjectCodec{})
	handler := &testHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = NewConn(ctx, stream, handler)

	// Send a notification (no ID)
	go func() {
		notificationMsg := Message{
			Version: Version,
			Method:  "test/notification",
			Params:  json.RawMessage(`{"notify":"true"}`),
		}

		time.Sleep(10 * time.Millisecond)
		err := stream.Write(ctx, notificationMsg)
		if err != nil {
			t.Errorf("Failed to write notification: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	requests := handler.getRequests()
	if len(requests) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(requests))
	}

	req := requests[0]
	if req.Method != "test/notification" {
		t.Errorf("Notification method = %q, want %q", req.Method, "test/notification")
	}

	if req.NotificationMethod() != true {
		t.Error("Expected notification to be detected as such")
	}
}

// Benchmark tests
func BenchmarkEncodeMessage(b *testing.B) {
	msg := Message{
		Version: Version,
		Method:  "benchmark/test",
		Params:  json.RawMessage(`{"test":"data","number":123,"array":[1,2,3]}`),
		ID:      json.RawMessage(`456`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncodeMessage(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeMessage(b *testing.B) {
	data := []byte(`{"jsonrpc":"2.0","method":"benchmark/test","params":{"test":"data","number":123,"array":[1,2,3]},"id":456}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecodeMessage(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
