package sdk2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// TestServer provides utilities for testing MCP servers.
// It follows the httptest.Server pattern from net/http/httptest.
type TestServer struct {
	*Server
	listener *TestListener
	URL      string
	
	// Client for connecting to this test server
	Client Client
	
	mu     sync.Mutex
	closed bool
}

// NewTestServer creates a new test server with the given handler.
func NewTestServer(handler Handler) *TestServer {
	if handler == nil {
		handler = DefaultServeMux
	}
	
	server := &Server{
		Handler: handler,
	}
	
	listener := NewTestListener()
	ts := &TestServer{
		Server:   server,
		listener: listener,
		URL:      "test://localhost",
	}
	
	// Create a client connected to this server
	client, _ := NewTestClient(ts)
	ts.Client = client
	
	// Start the server
	go server.Serve(ts.listener)
	
	return ts
}

// Close shuts down the test server.
func (ts *TestServer) Close() {
	ts.mu.Lock()
	if !ts.closed {
		ts.closed = true
		ts.listener.Close()
		if ts.Client != nil {
			ts.Client.Close()
		}
	}
	ts.mu.Unlock()
}

// TestListener implements Listener for testing.
type TestListener struct {
	conns chan Conn
	done  chan struct{}
	
	mu     sync.Mutex
	closed bool
}

// NewTestListener creates a new test listener.
func NewTestListener() *TestListener {
	return &TestListener{
		conns: make(chan Conn, 10),
		done:  make(chan struct{}),
	}
}

// Accept accepts a connection from the test client.
func (l *TestListener) Accept() (Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.done:
		return nil, fmt.Errorf("listener closed")
	}
}

// Close closes the test listener.
func (l *TestListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.closed {
		return nil
	}
	l.closed = true
	close(l.done)
	return nil
}

// Addr returns the listener address.
func (l *TestListener) Addr() net.Addr {
	return &testAddr{network: "test", address: "localhost"}
}

// AddConnection adds a connection to the listener.
func (l *TestListener) AddConnection(conn Conn) {
	select {
	case l.conns <- conn:
	case <-l.done:
	}
}

// testAddr implements net.Addr for testing.
type testAddr struct {
	network, address string
}

func (a *testAddr) Network() string { return a.network }
func (a *testAddr) String() string  { return a.address }

// TestConn provides a test connection implementation.
type TestConn struct {
	reader *bytes.Buffer
	writer *bytes.Buffer
	
	mu     sync.Mutex
	closed bool
}

// NewTestConn creates a new test connection.
func NewTestConn() *TestConn {
	return &TestConn{
		reader: new(bytes.Buffer),
		writer: new(bytes.Buffer),
	}
}

// Read reads from the connection.
func (c *TestConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, io.EOF
	}
	return c.reader.Read(p)
}

// Write writes to the connection.
func (c *TestConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, fmt.Errorf("connection closed")
	}
	return c.writer.Write(p)
}

// Close closes the connection.
func (c *TestConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.closed = true
	return nil
}

// LocalAddr returns the local address.
func (c *TestConn) LocalAddr() net.Addr {
	return &testAddr{network: "test", address: "local"}
}

// RemoteAddr returns the remote address.
func (c *TestConn) RemoteAddr() net.Addr {
	return &testAddr{network: "test", address: "remote"}
}

// SetDeadline sets the deadline.
func (c *TestConn) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline sets the read deadline.
func (c *TestConn) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline sets the write deadline.
func (c *TestConn) SetWriteDeadline(t time.Time) error { return nil }

// WriteString writes a string to the connection.
func (c *TestConn) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

// ReadAll reads all data from the connection.
func (c *TestConn) ReadAll() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writer.Bytes(), nil
}

// MockClient provides a mock implementation of the Client interface.
type MockClient struct {
	tools     []Tool
	resources []Resource
	prompts   []Prompt
	
	// Response handlers
	ToolHandler     func(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	ResourceHandler func(ctx context.Context, uri string) (*ResourceContent, error)
	PromptHandler   func(ctx context.Context, name string, args map[string]any) (*PromptResult, error)
	
	// Behavior configuration
	Latency time.Duration
	Error   error
	
	mu     sync.RWMutex
	closed bool
}

// NewMockClient creates a new mock client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// SetTools sets the tools returned by ListTools.
func (m *MockClient) SetTools(tools []Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools = tools
}

// SetResources sets the resources returned by ListResources.
func (m *MockClient) SetResources(resources []Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = resources
}

// SetPrompts sets the prompts returned by ListPrompts.
func (m *MockClient) SetPrompts(prompts []Prompt) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts = prompts
}

// Do implements Client.Do.
func (m *MockClient) Do(req *Request) (*Response, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	return &Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     make(Header),
		Request:    req,
	}, nil
}

// ListTools implements Client.ListTools.
func (m *MockClient) ListTools(ctx context.Context) ([]Tool, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent modification
	tools := make([]Tool, len(m.tools))
	copy(tools, m.tools)
	return tools, nil
}

// CallTool implements Client.CallTool.
func (m *MockClient) CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	handler := m.ToolHandler
	m.mu.RUnlock()
	
	if handler != nil {
		return handler(ctx, name, args)
	}
	
	// Default behavior
	return &ToolResult{
		Content: []Content{
			TextContent{Text: fmt.Sprintf("Mock result for tool %s", name)},
		},
	}, nil
}

// ListResources implements Client.ListResources.
func (m *MockClient) ListResources(ctx context.Context) ([]Resource, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	resources := make([]Resource, len(m.resources))
	copy(resources, m.resources)
	return resources, nil
}

// ReadResource implements Client.ReadResource.
func (m *MockClient) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	handler := m.ResourceHandler
	m.mu.RUnlock()
	
	if handler != nil {
		return handler(ctx, uri)
	}
	
	// Default behavior
	return &ResourceContent{
		URI:      uri,
		MimeType: "text/plain",
		Content: []Content{
			TextContent{Text: fmt.Sprintf("Mock content for resource %s", uri)},
		},
	}, nil
}

// ListPrompts implements Client.ListPrompts.
func (m *MockClient) ListPrompts(ctx context.Context) ([]Prompt, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	prompts := make([]Prompt, len(m.prompts))
	copy(prompts, m.prompts)
	return prompts, nil
}

// GetPrompt implements Client.GetPrompt.
func (m *MockClient) GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error) {
	if err := m.checkLatencyAndError(); err != nil {
		return nil, err
	}
	
	m.mu.RLock()
	handler := m.PromptHandler
	m.mu.RUnlock()
	
	if handler != nil {
		return handler(ctx, name, args)
	}
	
	// Default behavior
	return &PromptResult{
		Description: fmt.Sprintf("Mock prompt result for %s", name),
		Messages: []PromptMessage{
			{
				Role: "user",
				Content: []Content{
					TextContent{Text: fmt.Sprintf("Mock prompt for %s", name)},
				},
			},
		},
	}, nil
}

// Close implements Client.Close.
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.closed = true
	return nil
}

// checkLatencyAndError simulates latency and errors.
func (m *MockClient) checkLatencyAndError() error {
	m.mu.RLock()
	latency := m.Latency
	err := m.Error
	closed := m.closed
	m.mu.RUnlock()
	
	if closed {
		return fmt.Errorf("client closed")
	}
	
	if latency > 0 {
		time.Sleep(latency)
	}
	
	if err != nil {
		return err
	}
	
	return nil
}

// ResponseRecorder implements ResponseWriter for testing.
// It follows the httptest.ResponseRecorder pattern.
type ResponseRecorder struct {
	Code   int
	Header Header
	Body   *bytes.Buffer
	
	mu sync.Mutex
}

// NewResponseRecorder creates a new response recorder.
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:   200,
		Header: make(Header),
		Body:   new(bytes.Buffer),
	}
}

// Header implements ResponseWriter.Header.
func (rr *ResponseRecorder) Header() Header {
	return rr.Header
}

// Write implements ResponseWriter.Write.
func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	return rr.Body.Write(data)
}

// WriteHeader implements ResponseWriter.WriteHeader.
func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	rr.Code = code
}

// Result returns the response as a string.
func (rr *ResponseRecorder) Result() string {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	return rr.Body.String()
}

// TestHelper provides utilities for testing MCP applications.
type TestHelper struct {
	t interface {
		Helper()
		Fatalf(format string, args ...interface{})
		Errorf(format string, args ...interface{})
	}
}

// NewTestHelper creates a new test helper.
func NewTestHelper(t interface {
	Helper()
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}) *TestHelper {
	return &TestHelper{t: t}
}

// AssertNoError asserts that err is nil.
func (h *TestHelper) AssertNoError(err error) {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError asserts that err is not nil.
func (h *TestHelper) AssertError(err error) {
	h.t.Helper()
	if err == nil {
		h.t.Fatalf("expected error, got nil")
	}
}

// AssertEqual asserts that two values are equal.
func (h *TestHelper) AssertEqual(got, want interface{}) {
	h.t.Helper()
	if got != want {
		h.t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual asserts that two values are not equal.
func (h *TestHelper) AssertNotEqual(got, notWant interface{}) {
	h.t.Helper()
	if got == notWant {
		h.t.Errorf("got %v, expected it to be different", got)
	}
}

// AssertTrue asserts that a condition is true.
func (h *TestHelper) AssertTrue(condition bool, msg string) {
	h.t.Helper()
	if !condition {
		h.t.Errorf("assertion failed: %s", msg)
	}
}

// AssertFalse asserts that a condition is false.
func (h *TestHelper) AssertFalse(condition bool, msg string) {
	h.t.Helper()
	if condition {
		h.t.Errorf("assertion failed: %s", msg)
	}
}

// AssertContains asserts that a string contains a substring.
func (h *TestHelper) AssertContains(str, substr string) {
	h.t.Helper()
	if !contains(str, substr) {
		h.t.Errorf("string %q does not contain %q", str, substr)
	}
}

// AssertJSON asserts that two JSON strings are equivalent.
func (h *TestHelper) AssertJSON(got, want string) {
	h.t.Helper()
	
	var gotData, wantData interface{}
	
	if err := json.Unmarshal([]byte(got), &gotData); err != nil {
		h.t.Fatalf("failed to unmarshal got JSON: %v", err)
	}
	
	if err := json.Unmarshal([]byte(want), &wantData); err != nil {
		h.t.Fatalf("failed to unmarshal want JSON: %v", err)
	}
	
	gotJSON, _ := json.Marshal(gotData)
	wantJSON, _ := json.Marshal(wantData)
	
	if string(gotJSON) != string(wantJSON) {
		h.t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

// contains checks if a string contains a substring (simple implementation).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// NewTestClient creates a client connected to a test server.
func NewTestClient(server *TestServer) (Client, error) {
	// Create test connections
	serverConn := NewTestConn()
	clientConn := NewTestConn()
	
	// Cross-connect them (server writes go to client reads, etc.)
	serverConn.reader = clientConn.writer
	clientConn.reader = serverConn.writer
	
	// Add server connection to the test listener
	server.listener.AddConnection(serverConn)
	
	// Create client with the client side of the connection
	config := &ClientConfig{
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		ClientInfo: ClientInfo{Name: "test-client", Version: "0.1.0"},
	}
	
	return newClient(context.Background(), clientConn, config)
}

// BenchmarkHelper provides utilities for benchmarking MCP operations.
type BenchmarkHelper struct {
	b interface {
		ResetTimer()
		StopTimer()
		StartTimer()
	}
}

// NewBenchmarkHelper creates a new benchmark helper.
func NewBenchmarkHelper(b interface {
	ResetTimer()
	StopTimer()
	StartTimer()
}) *BenchmarkHelper {
	return &BenchmarkHelper{b: b}
}

// ResetTimer resets the benchmark timer.
func (h *BenchmarkHelper) ResetTimer() {
	h.b.ResetTimer()
}

// StopTimer stops the benchmark timer.
func (h *BenchmarkHelper) StopTimer() {
	h.b.StopTimer()
}

// StartTimer starts the benchmark timer.
func (h *BenchmarkHelper) StartTimer() {
	h.b.StartTimer()
}

// LoadTestHelper provides utilities for load testing.
type LoadTestHelper struct {
	concurrency int
	duration    time.Duration
}

// NewLoadTestHelper creates a new load test helper.
func NewLoadTestHelper(concurrency int, duration time.Duration) *LoadTestHelper {
	return &LoadTestHelper{
		concurrency: concurrency,
		duration:    duration,
	}
}

// RunLoadTest runs a load test with the given function.
func (h *LoadTestHelper) RunLoadTest(fn func() error) *LoadTestResult {
	result := &LoadTestResult{
		StartTime: time.Now(),
	}
	
	var wg sync.WaitGroup
	results := make(chan error, h.concurrency*100) // Buffer for results
	
	// Start workers
	for i := 0; i < h.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			start := time.Now()
			for time.Since(start) < h.duration {
				err := fn()
				results <- err
				if err != nil {
					result.mu.Lock()
					result.Errors++
					result.mu.Unlock()
				} else {
					result.mu.Lock()
					result.Successes++
					result.mu.Unlock()
				}
			}
		}()
	}
	
	// Wait for completion
	wg.Wait()
	close(results)
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TotalRequests = result.Successes + result.Errors
	
	if result.Duration > 0 {
		result.RequestsPerSecond = float64(result.TotalRequests) / result.Duration.Seconds()
	}
	
	return result
}

// LoadTestResult contains the results of a load test.
type LoadTestResult struct {
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	TotalRequests     int64
	Successes         int64
	Errors            int64
	RequestsPerSecond float64
	
	mu sync.Mutex
}

// ErrorRate returns the error rate as a percentage.
func (r *LoadTestResult) ErrorRate() float64 {
	if r.TotalRequests == 0 {
		return 0
	}
	return float64(r.Errors) / float64(r.TotalRequests) * 100
}

// String returns a string representation of the results.
func (r *LoadTestResult) String() string {
	return fmt.Sprintf(
		"Load Test Results:\n"+
			"  Duration: %v\n"+
			"  Total Requests: %d\n"+
			"  Successes: %d\n"+
			"  Errors: %d\n"+
			"  Error Rate: %.2f%%\n"+
			"  Requests/sec: %.2f",
		r.Duration,
		r.TotalRequests,
		r.Successes,
		r.Errors,
		r.ErrorRate(),
		r.RequestsPerSecond,
	)
}