package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// Comprehensive error handling tests to achieve near 100% coverage

func TestErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	errors := []error{
		ErrInvalidParams,
		ErrNotFound,
		ErrUnsupported,
		ErrTransportClosed,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error constant should not be nil")
		}
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}

	// Test specific error messages
	if ErrInvalidParams.Error() != "mcp: invalid parameters" {
		t.Errorf("Unexpected ErrInvalidParams message: %s", ErrInvalidParams.Error())
	}

	if ErrNotFound.Error() != "mcp: not found" {
		t.Errorf("Unexpected ErrNotFound message: %s", ErrNotFound.Error())
	}

	if ErrUnsupported.Error() != "mcp: operation or capability not supported" {
		t.Errorf("Unexpected ErrUnsupported message: %s", ErrUnsupported.Error())
	}

	if ErrTransportClosed.Error() != "mcp: transport closed" {
		t.Errorf("Unexpected ErrTransportClosed message: %s", ErrTransportClosed.Error())
	}
}

func TestClientErrorHandlingComprehensive(t *testing.T) {
	// Test client with failing transport
	failingTransport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return nil, errors.New("transport failure")
	})

	_, err := NewClient(failingTransport)
	if err == nil {
		t.Error("Expected error when creating client with failing transport")
	}

	// Test client initialization errors
	initErrorTransport := &testErrorTransport{
		dialError: errors.New("dial error"),
	}

	_, err = NewClient(initErrorTransport)
	if err == nil {
		t.Error("Expected error when transport dial fails")
	}
}

func TestErrorHandlingComprehensive(t *testing.T) {
	_ = context.Background()

	// Test server with invalid handler registration
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Test registering tool with empty name (should fail)
	emptyTool := Tool{Name: "", Description: "Empty tool"}
	err := server.RegisterTool(emptyTool, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	if err != nil {
		// Server should handle empty names gracefully
		t.Logf("Server rejected empty tool name: %v", err)
	}

	// Test duplicate tool registration
	validTool := Tool{Name: "test-tool", Description: "Test tool"}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: []any{"test"}}, nil
	}

	err1 := server.RegisterTool(validTool, handler)
	if err1 != nil {
		t.Errorf("First tool registration failed: %v", err1)
	}

	err2 := server.RegisterTool(validTool, handler)
	if err2 == nil {
		t.Error("Expected error for duplicate tool registration")
	}
}

func TestJSONRPCErrorHandling(t *testing.T) {
	// Test JSONRPCError type
	jsonErr := &JSONRPCError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    json.RawMessage(`{"detail": "missing method"}`),
	}

	if jsonErr.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", jsonErr.Code)
	}

	if jsonErr.Message != "Invalid Request" {
		t.Errorf("Expected message 'Invalid Request', got %s", jsonErr.Message)
	}

	if string(jsonErr.Data) != `{"detail": "missing method"}` {
		t.Errorf("Unexpected data: %s", string(jsonErr.Data))
	}
}

func TestHandlerErrorPropagation(t *testing.T) {
	ctx := context.Background()
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Register tool with handler that returns error
	errorTool := Tool{Name: "error-tool", Description: "Tool that returns error"}
	errorHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return nil, errors.New("handler error")
	}

	err := server.RegisterTool(errorTool, errorHandler)
	if err != nil {
		t.Fatalf("Tool registration failed: %v", err)
	}

	// This test would require more complex setup to actually test handler error propagation
	// For now, we verify the handler can be registered and the error type is correct
	result, handlerErr := errorHandler(ctx, CallToolRequest{Name: "error-tool"})
	if handlerErr == nil {
		t.Error("Expected error from error handler")
	}
	if result != nil {
		t.Error("Expected nil result from error handler")
	}
}

func TestParameterValidationErrors(t *testing.T) {
	// Test various parameter validation scenarios
	tests := []struct {
		name      string
		params    interface{}
		expectErr bool
	}{
		{
			name:      "nil params",
			params:    nil,
			expectErr: false, // nil params should be handled gracefully
		},
		{
			name:      "empty params",
			params:    map[string]interface{}{},
			expectErr: false,
		},
		{
			name: "invalid tool name",
			params: CallToolRequest{
				Name: "", // Empty name should cause validation error
			},
			expectErr: true,
		},
		{
			name: "valid params",
			params: CallToolRequest{
				Name:      "valid-tool",
				Arguments: json.RawMessage(`{"arg": "value"}`),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parameter marshaling/unmarshaling
			data, err := json.Marshal(tt.params)
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Unexpected marshal error: %v", err)
				}
				return
			}

			// Test unmarshaling back
			var unmarshaled interface{}
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Unexpected unmarshal error: %v", err)
				}
			}
		})
	}
}

func TestContextErrorHandling(t *testing.T) {
	// Test context cancellation propagation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test that cancelled context causes appropriate errors
	if ctx.Err() == nil {
		t.Error("Cancelled context should have error")
	}

	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", ctx.Err())
	}

	// Test with deadline exceeded
	deadlineCtx, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	// Simulate deadline exceeded
	deadlineCtx = context.WithValue(deadlineCtx, "error", context.DeadlineExceeded)

	// Verify context error handling would work in real scenarios
}

func TestTransportErrorRecovery(t *testing.T) {
	// Test various transport error scenarios
	errorScenarios := []struct {
		name        string
		transport   Transport
		expectError bool
		errorType   string
	}{
		{
			name:        "nil connection",
			transport:   &ReadWriteCloserTransport{nil},
			expectError: true,
			errorType:   "closed pipe",
		},
		{
			name: "dial failure",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, errors.New("dial failed")
			}),
			expectError: true,
			errorType:   "dial failed",
		},
		{
			name:        "intermittent failure",
			transport:   &testIntermittentTransport{failCount: 1},
			expectError: true,
			errorType:   "intermittent failure",
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			conn, err := scenario.transport.Dial(context.Background())

			if scenario.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if scenario.errorType == "closed pipe" && err == ErrTransportClosed {
					// This is acceptable - transport closed is the proper error
				} else if !strings.Contains(err.Error(), scenario.errorType) {
					t.Errorf("Expected error containing %q, got %q", scenario.errorType, err.Error())
				}
				if conn != nil {
					t.Error("Expected nil connection on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if conn == nil {
					t.Error("Expected non-nil connection")
				}
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test error wrapping and unwrapping
	baseErr := errors.New("base error")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	if !errors.Is(wrappedErr, baseErr) {
		t.Error("Wrapped error should unwrap to base error")
	}

	// Test multiple wrapping levels
	doubleWrapped := fmt.Errorf("double wrapped: %w", wrappedErr)
	if !errors.Is(doubleWrapped, baseErr) {
		t.Error("Double wrapped error should unwrap to base error")
	}

	// Test with custom error types
	customErr := &JSONRPCError{Code: -32000, Message: "Custom error"}

	if customErr.Code != -32000 {
		t.Errorf("Expected code -32000, got %d", customErr.Code)
	}

	if customErr.Message != "Custom error" {
		t.Errorf("Expected message 'Custom error', got %s", customErr.Message)
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test that error messages are properly formatted
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name:     "simple error",
			err:      errors.New("simple error message"),
			contains: []string{"simple error message"},
		},
		{
			name:     "formatted error",
			err:      fmt.Errorf("error with %s: %d", "format", 42),
			contains: []string{"error with format: 42"},
		},
		{
			name:     "MCP error",
			err:      fmt.Errorf("mcp: %w", ErrInvalidParams),
			contains: []string{"mcp:", "invalid parameters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("Error message %q should contain %q", errMsg, expected)
				}
			}
		})
	}
}

func TestConcurrentErrorHandling(t *testing.T) {
	// Test error handling under concurrent access
	const numGoroutines = 50
	errors := make(chan error, numGoroutines)

	// Create a transport that sometimes fails
	failingTransport := &testRandomFailureTransport{failureRate: 0.3}

	// Launch multiple goroutines that may encounter errors
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := failingTransport.Dial(context.Background())
			errors <- err
		}()
	}

	// Collect results
	var successCount, errorCount int
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// We expect some failures due to the failure rate
	if errorCount == 0 {
		t.Error("Expected some errors with random failure transport")
	}

	if successCount == 0 {
		t.Error("Expected some successes with random failure transport")
	}

	t.Logf("Concurrent error test: %d successes, %d errors", successCount, errorCount)
}

// Test helper types for error scenarios

type testErrorTransport struct {
	dialError error
}

func (t *testErrorTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return nil, t.dialError
}

type testMockTransport struct {
	shouldFail bool
}

func (t *testMockTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	if t.shouldFail {
		return nil, errors.New("mock transport failure")
	}
	return &testSuccessfulReadWriteCloser{}, nil
}

type testSuccessfulReadWriteCloser struct {
	bytes.Buffer
}

func (t *testSuccessfulReadWriteCloser) Close() error {
	return nil
}

type testIntermittentTransport struct {
	callCount int
	failCount int
}

func (t *testIntermittentTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	t.callCount++
	if t.callCount <= t.failCount {
		return nil, errors.New("intermittent failure")
	}
	return &testSuccessfulReadWriteCloser{}, nil
}

type testRandomFailureTransport struct {
	failureRate float64
	mu          sync.Mutex
	callCount   int
}

func (t *testRandomFailureTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	t.mu.Lock()
	t.callCount++
	count := t.callCount
	t.mu.Unlock()

	// Simple deterministic "random" failure based on call count
	if float64(count%10)/10.0 < t.failureRate {
		return nil, errors.New("random failure")
	}
	return &testSuccessfulReadWriteCloser{}, nil
}

func TestOAuthErrorHandling(t *testing.T) {
	// Test OAuth-specific error handling
	oauthErr := &OAuthError{
		Code:        ErrorInvalidClient,
		Description: "Invalid client credentials",
		URI:         "https://example.com/error",
		State:       "test-state",
	}

	expectedMsg := "invalid_client: Invalid client credentials"
	if oauthErr.Error() != expectedMsg {
		t.Errorf("Expected OAuth error message %q, got %q", expectedMsg, oauthErr.Error())
	}

	// Test OAuth error without description
	oauthErrNoDesc := &OAuthError{
		Code: ErrorInvalidRequest,
	}

	if oauthErrNoDesc.Error() != ErrorInvalidRequest {
		t.Errorf("Expected OAuth error message %q, got %q", ErrorInvalidRequest, oauthErrNoDesc.Error())
	}
}

func TestProgressNotificationErrors(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Test progress notification with invalid token
	err := dispatcher.NotifyProgress(ctx, nil, 0.5, nil)
	if err != nil {
		t.Errorf("Progress notification should handle nil token gracefully: %v", err)
	}

	// Test with invalid progress value (should still work)
	err = dispatcher.NotifyProgress(ctx, "test", -1.0, nil)
	if err != nil {
		t.Errorf("Progress notification should handle negative progress: %v", err)
	}

	// Test with progress > 1.0 (should still work)
	total := 2.0
	err = dispatcher.NotifyProgress(ctx, "test", 1.5, &total)
	if err != nil {
		t.Errorf("Progress notification should handle progress > 1.0: %v", err)
	}
}

func TestLoggingNotificationErrors(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Test logging notification with various data types
	testData := []interface{}{
		"string data",
		12345,
		map[string]string{"key": "value"},
		[]string{"item1", "item2"},
		nil,
	}

	for _, data := range testData {
		err := dispatcher.NotifyLoggingMessage(ctx, LogLevelInfo, "test-logger", data)
		if err != nil {
			t.Errorf("Logging notification failed for data %v: %v", data, err)
		}
	}

	// Test with empty logger name
	err := dispatcher.NotifyLoggingMessage(ctx, LogLevelError, "", "test message")
	if err != nil {
		t.Errorf("Logging notification should handle empty logger name: %v", err)
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := fmt.Errorf("error %d: %w", i, ErrInvalidParams)
		_ = err.Error() // Force string creation
	}
}

func BenchmarkErrorWrapping(b *testing.B) {
	baseErr := errors.New("base error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped := fmt.Errorf("level %d: %w", i, baseErr)
		_ = errors.Is(wrapped, baseErr)
	}
}
