package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"golang.org/x/exp/jsonrpc2"
)

func TestCancellablePreempter_Preempt(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		params       string
		conn         *jsonrpc2.Connection
		wantErr      bool
		wantLog      string
		expectCancel bool
	}{
		{
			name:         "valid cancellation notification",
			method:       string(MethodNotificationCancelled),
			params:       `{"requestId": "123", "reason": "user cancelled"}`,
			conn:         &jsonrpc2.Connection{}, // Mock connection
			wantErr:      false,
			expectCancel: true,
		},
		{
			name:         "valid cancellation without reason",
			method:       string(MethodNotificationCancelled),
			params:       `{"requestId": "456"}`,
			conn:         &jsonrpc2.Connection{}, // Mock connection
			wantErr:      false,
			expectCancel: true,
		},
		{
			name:    "cancellation with nil connection",
			method:  string(MethodNotificationCancelled),
			params:  `{"requestId": "123", "reason": "user cancelled"}`,
			conn:    nil,
			wantErr: true,
			wantLog: "Connection is nil",
		},
		{
			name:    "invalid cancellation params",
			method:  string(MethodNotificationCancelled),
			params:  `{invalid json}`,
			conn:    &jsonrpc2.Connection{},
			wantErr: true,
			wantLog: "Failed to unmarshal cancellation params",
		},
		{
			name:    "invalid requestId format",
			method:  string(MethodNotificationCancelled),
			params:  `{"requestId": {invalid}, "reason": "test"}`,
			conn:    &jsonrpc2.Connection{},
			wantErr: true,
			wantLog: "Failed to unmarshal cancellation requestId",
		},
		{
			name:    "non-cancellation method",
			method:  "other/method",
			params:  `{}`,
			conn:    &jsonrpc2.Connection{},
			wantErr: true, // Should return ErrNotHandled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test logger that captures log output
			var logOutput strings.Builder
			logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			preempter := &CancellablePreempter{
				Conn:   tt.conn,
				Logger: logger,
			}

			ctx := context.Background()
			req := &jsonrpc2.Request{
				Method: tt.method,
				Params: json.RawMessage(tt.params),
			}

			result, err := preempter.Preempt(ctx, req)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}

				// For non-cancellation methods, should return ErrNotHandled
				if tt.method != string(MethodNotificationCancelled) {
					if err != jsonrpc2.ErrNotHandled {
						t.Errorf("Expected ErrNotHandled for non-cancellation method, got: %v", err)
					}
				}
			} else {
				if err != nil && err != jsonrpc2.ErrNotHandled {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Result should always be nil for preempter
			if result != nil {
				t.Errorf("Expected nil result, got: %v", result)
			}

			// Check log output if expected
			if tt.wantLog != "" {
				logStr := logOutput.String()
				if !strings.Contains(logStr, tt.wantLog) {
					t.Errorf("Expected log to contain %q, got: %s", tt.wantLog, logStr)
				}
			}
		})
	}
}

func TestCancellablePreempter_PreemptWithNilLogger(t *testing.T) {
	// Test that preempter works with nil logger (uses default)
	preempter := &CancellablePreempter{
		Conn:   &jsonrpc2.Connection{},
		Logger: nil, // Should use default logger
	}

	ctx := context.Background()
	req := &jsonrpc2.Request{
		Method: string(MethodNotificationCancelled),
		Params: json.RawMessage(`{"requestId": "test-123"}`),
	}

	result, err := preempter.Preempt(ctx, req)

	// Should handle gracefully with default logger
	if err != nil && err != jsonrpc2.ErrNotHandled {
		t.Errorf("Unexpected error with nil logger: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}
}

func TestCancelledNotificationParams_Unmarshaling(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CancelledNotificationParams
		wantErr bool
	}{
		{
			name:  "complete notification",
			input: `{"requestId": "test-123", "reason": "user cancelled operation"}`,
			want: CancelledNotificationParams{
				RequestID: json.RawMessage(`"test-123"`),
				Reason:    "user cancelled operation",
			},
			wantErr: false,
		},
		{
			name:  "notification without reason",
			input: `{"requestId": "456"}`,
			want: CancelledNotificationParams{
				RequestID: json.RawMessage(`"456"`),
				Reason:    "",
			},
			wantErr: false,
		},
		{
			name:  "numeric request ID",
			input: `{"requestId": 789, "reason": "timeout"}`,
			want: CancelledNotificationParams{
				RequestID: json.RawMessage(`789`),
				Reason:    "timeout",
			},
			wantErr: false,
		},
		{
			name:    "missing requestId",
			input:   `{"reason": "test"}`,
			wantErr: false, // Will have empty RequestID
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params CancelledNotificationParams
			err := json.Unmarshal([]byte(tt.input), &params)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.want.Reason != params.Reason {
				t.Errorf("Expected reason %q, got %q", tt.want.Reason, params.Reason)
			}

			// Compare RequestID as strings since they're RawMessage
			if string(tt.want.RequestID) != string(params.RequestID) {
				t.Errorf("Expected RequestID %q, got %q", string(tt.want.RequestID), string(params.RequestID))
			}
		})
	}
}

func TestCancellablePreempter_RequestIDParsing(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		wantErr   bool
		expectID  interface{}
	}{
		{
			name:      "string request ID",
			requestID: `"string-id-123"`,
			wantErr:   false,
			expectID:  "string-id-123",
		},
		{
			name:      "numeric request ID",
			requestID: `42`,
			wantErr:   false,
			expectID:  float64(42), // JSON numbers unmarshal as float64
		},
		{
			name:      "null request ID",
			requestID: `null`,
			wantErr:   false,
			expectID:  nil,
		},
		{
			name:      "invalid request ID",
			requestID: `{invalid}`,
			wantErr:   true,
		},
		{
			name:      "array request ID (invalid)",
			requestID: `["not", "valid"]`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logOutput strings.Builder
			logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			preempter := &CancellablePreempter{
				Conn:   &jsonrpc2.Connection{},
				Logger: logger,
			}

			ctx := context.Background()
			params := `{"requestId": ` + tt.requestID + `}`
			req := &jsonrpc2.Request{
				Method: string(MethodNotificationCancelled),
				Params: json.RawMessage(params),
			}

			result, err := preempter.Preempt(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				// Should log the error
				logStr := logOutput.String()
				if !strings.Contains(logStr, "Failed to unmarshal cancellation requestId") {
					t.Error("Expected error log for invalid requestId")
				}
			} else {
				if err != nil && err != jsonrpc2.ErrNotHandled {
					t.Errorf("Unexpected error: %v", err)
				}
				// Should log successful processing
				logStr := logOutput.String()
				if !strings.Contains(logStr, "Received cancellation notification") {
					t.Error("Expected success log for valid cancellation")
				}
			}

			if result != nil {
				t.Errorf("Expected nil result, got: %v", result)
			}
		})
	}
}

func TestCancellablePreempter_Integration(t *testing.T) {
	// Test realistic cancellation scenario
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	preempter := &CancellablePreempter{
		Conn:   &jsonrpc2.Connection{}, // In real usage, this would be a live connection
		Logger: logger,
	}

	ctx := context.Background()

	// Simulate cancellation notification from client
	cancellationRequest := &jsonrpc2.Request{
		Method: string(MethodNotificationCancelled),
		Params: json.RawMessage(`{
			"requestId": "req-abc-123",
			"reason": "User pressed cancel button"
		}`),
	}

	result, err := preempter.Preempt(ctx, cancellationRequest)

	// Should handle without error (returns ErrNotHandled to indicate processing complete)
	if err != jsonrpc2.ErrNotHandled {
		t.Errorf("Expected ErrNotHandled, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}

	// Verify proper logging
	logStr := logOutput.String()
	expectedLogContents := []string{
		"Received cancellation notification",
		"req-abc-123",
		"User pressed cancel button",
	}

	for _, content := range expectedLogContents {
		if !strings.Contains(logStr, content) {
			t.Errorf("Expected log to contain %q, got: %s", content, logStr)
		}
	}
}
