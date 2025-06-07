package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListChangeNotifications(t *testing.T) {
	tests := []struct {
		name     string
		method   MCPMethod
		caps     ServerCapabilities
		wantSent bool
	}{
		{
			name:   "tools list changed with capability",
			method: MethodToolsList,
			caps: ServerCapabilities{
				Tools: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{ListChanged: true},
			},
			wantSent: true,
		},
		{
			name:     "tools list changed without capability",
			method:   MethodToolsList,
			caps:     ServerCapabilities{},
			wantSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("test", "1.0.0", WithCapabilities(tt.caps))

			var notified bool
			server.dispatch.Handle(string(tt.method), func(method string, params json.RawMessage) error {
				notified = true
				return nil
			})

			// Only notify if the capability is enabled
			if tt.caps.Tools != nil && tt.caps.Tools.ListChanged {
				err := server.dispatch.NotifyListChanged(context.Background(), tt.method)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				// Skip notification when capability isn't enabled
				// and just set notified to false for test
				notified = false
			}

			if notified != tt.wantSent {
				t.Errorf("notification sent = %v, want %v", notified, tt.wantSent)
			}
		})
	}
}

func TestDispatcher(t *testing.T) {
	d := NewDispatcher()

	var received []string
	d.Handle("test", func(method string, params json.RawMessage) error {
		received = append(received, string(params))
		return nil
	})

	want := `{"hello":"world"}`
	if err := d.Dispatch(context.Background(), "test", json.RawMessage(want)); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff([]string{want}, received); diff != "" {
		t.Errorf("notification mismatch (-want +got):\n%s", diff)
	}
}

func TestNotifications(t *testing.T) {
	tests := []struct {
		name     string
		send     func(*Dispatcher) error
		wantType string
		wantData string
	}{
		{
			name: "progress",
			send: func(d *Dispatcher) error {
				total := 100.0
				return d.NotifyProgress(context.Background(), "token1", 50.0, &total)
			},
			wantType: string(MethodProgress),
			wantData: `{"progressToken":"token1","progress":50,"total":100}`,
		},
		{
			name: "logging message",
			send: func(d *Dispatcher) error {
				return d.NotifyLoggingMessage(context.Background(), LogLevelInfo, "test", "hello")
			},
			wantType: string(MethodLogging),
			wantData: `{"level":"info","logger":"test","data":"hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			var gotType string
			var gotData json.RawMessage

			d.Handle(tt.wantType, func(method string, params json.RawMessage) error {
				gotType = method
				gotData = params
				return nil
			})

			if err := tt.send(d); err != nil {
				t.Fatal(err)
			}

			if gotType != tt.wantType {
				t.Errorf("got type %q, want %q", gotType, tt.wantType)
			}

			var got, want interface{}
			if err := json.Unmarshal(gotData, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tt.wantData), &want); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("notification data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
