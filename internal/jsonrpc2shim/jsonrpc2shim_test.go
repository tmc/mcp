package jsonrpc2shim

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestMakeID(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "int64 ID",
			input:   int64(123),
			wantErr: false,
		},
		{
			name:    "string ID",
			input:   "test-id",
			wantErr: false,
		},
		{
			name:    "nil ID",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   true, // bool is not a valid ID type
			wantErr: true,
		},
		{
			name:    "float is invalid",
			input:   3.14,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := MakeID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the ID is created correctly
			switch v := tt.input.(type) {
			case int64:
				if id != Int64ID(v) {
					t.Errorf("int64 ID mismatch")
				}
			case string:
				if id != StringID(v) {
					t.Errorf("string ID mismatch")
				}
			case nil:
				// nil case is handled
			}
		})
	}
}

func TestNewIntID(t *testing.T) {
	id := NewIntID(42)
	expected := Int64ID(42)
	if id != expected {
		t.Errorf("NewIntID(42) = %v, want %v", id, expected)
	}
}

func TestConnectionConfig(t *testing.T) {
	// Test that we can create a ConnectionConfig
	var reader Reader
	var writer Writer
	var closer io.Closer
	var preempter Preempter

	cfg := ConnectionConfig{
		Reader:    reader,
		Writer:    writer,
		Closer:    closer,
		Preempter: preempter,
		Bind: func(conn *Connection) Handler {
			return HandlerFunc(func(ctx context.Context, req *Request) (interface{}, error) {
				return nil, nil
			})
		},
		OnDone: func() {
			// callback
		},
	}

	// Verify fields are set
	if cfg.Reader != reader {
		t.Error("Reader field not set correctly")
	}
	if cfg.Writer != writer {
		t.Error("Writer field not set correctly")
	}
	if cfg.Closer != closer {
		t.Error("Closer field not set correctly")
	}
	if cfg.Preempter != preempter {
		t.Error("Preempter field not set correctly")
	}
	if cfg.Bind == nil {
		t.Error("Bind function should not be nil")
	}
	if cfg.OnDone == nil {
		t.Error("OnDone function should not be nil")
	}
}

func TestNewConnection(t *testing.T) {
	ctx := context.Background()

	bindCalled := false
	cfg := ConnectionConfig{
		Bind: func(conn *Connection) Handler {
			bindCalled = true
			return HandlerFunc(func(ctx context.Context, req *Request) (interface{}, error) {
				return "test response", nil
			})
		},
	}

	conn := NewConnection(ctx, cfg)
	if conn == nil {
		t.Error("NewConnection should return a connection")
	}
	if !bindCalled {
		t.Error("Bind function should have been called")
	}
}

func TestNotification(t *testing.T) {
	// Test Notification struct
	notif := Notification{
		Method: "test/notification",
		Params: nil,
	}

	if notif.Method != "test/notification" {
		t.Errorf("Method = %v, want %v", notif.Method, "test/notification")
	}
	if notif.Params != nil {
		t.Errorf("Params should be nil")
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants are defined
	errors := []error{
		ErrClientClosing,
		ErrServerClosing,
		ErrNotHandled,
		ErrInvalidRequest,
		ErrInvalidParams,
	}

	for i, err := range errors {
		if err == nil {
			t.Errorf("Error constant %d should not be nil", i)
		}
		if err.Error() == "" {
			t.Errorf("Error constant %d should have a message", i)
		}
	}

	// Test specific error messages
	if !strings.Contains(ErrClientClosing.Error(), "client") {
		t.Error("ErrClientClosing should mention client")
	}
	if !strings.Contains(ErrServerClosing.Error(), "server") {
		t.Error("ErrServerClosing should mention server")
	}
	if !strings.Contains(ErrNotHandled.Error(), "not handled") {
		t.Error("ErrNotHandled should mention 'not handled'")
	}
	if !strings.Contains(ErrInvalidRequest.Error(), "invalid") {
		t.Error("ErrInvalidRequest should mention 'invalid'")
	}
	if !strings.Contains(ErrInvalidParams.Error(), "invalid") {
		t.Error("ErrInvalidParams should mention 'invalid'")
	}
}

func TestTypeAliases(t *testing.T) {
	// Test that type aliases are properly defined by trying to use them
	var conn *Connection
	var handler Handler
	var handlerFunc HandlerFunc
	var id ID
	var message *Message
	var reader Reader
	var writer Writer
	var preempter Preempter
	var request *Request
	var response *Response

	// Basic nil checks to ensure types are defined
	_ = conn
	_ = handler
	_ = handlerFunc
	_ = id
	_ = message
	_ = reader
	_ = writer
	_ = preempter
	_ = request
	_ = response

	t.Log("All type aliases are properly defined")
}

func TestForwardedFunctions(t *testing.T) {
	// Test that forwarded functions are available
	if EncodeMessage == nil {
		t.Error("EncodeMessage should be available")
	}
	if DecodeMessage == nil {
		t.Error("DecodeMessage should be available")
	}

	// Test ID creation functions
	intID := Int64ID(123)
	stringID := StringID("test")

	if intID == (ID{}) {
		t.Error("Int64ID should create a valid ID")
	}
	if stringID == (ID{}) {
		t.Error("StringID should create a valid ID")
	}
}
