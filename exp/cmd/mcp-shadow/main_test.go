package main

import (
	"testing"
	"time"
)

func TestFormatMCPLine(t *testing.T) {
	tests := []struct {
		name string
		msg  message
		want string
	}{
		{
			name: "basic receive",
			msg: message{
				raw:       []byte(`{"method":"test"}`),
				timestamp: time.Unix(1683000000, 550000000),
				direction: "recv",
				spanID:    "abc123",
			},
			want: `mcp-recv {"method":"test"} # 1683000000.550 spanid=abc123`,
		},
		{
			name: "primary send",
			msg: message{
				raw:       []byte(`{"result":"ok"}`),
				timestamp: time.Unix(1683000001, 120000000),
				direction: "send",
				spanID:    "def456",
				linksTo:   "abc123",
				isPrimary: true,
			},
			want: `mcp-send {"result":"ok"} # 1683000001.120 spanid=def456 linksto=abc123`,
		},
		{
			name: "shadow send with baggage",
			msg: message{
				raw:       []byte(`{"result":"shadow"}`),
				timestamp: time.Unix(1683000001, 130000000),
				direction: "send",
				spanID:    "ghi789",
				linksTo:   "abc123",
				baggage:   "shadow=true,server=test",
				isPrimary: false,
			},
			want: `mcp-send {"result":"shadow"} # 1683000001.130 spanid=ghi789 linksto=abc123 baggage=shadow=true,server=test`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMCPLine(tt.msg)
			if got != tt.want {
				t.Errorf("formatMCPLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecordRequestSpan(t *testing.T) {
	s := &shadowServer{
		requestSpans: make(map[string]string),
	}

	// Test recording a request with ID
	req := []byte(`{"jsonrpc":"2.0","method":"test","id":1}`)
	spanID := "test-span-123"
	s.recordRequestSpan(req, spanID)

	if s.requestSpans["1"] != spanID {
		t.Errorf("Expected span ID %s for request ID 1, got %s", spanID, s.requestSpans["1"])
	}

	// Test notification (no ID)
	notif := []byte(`{"jsonrpc":"2.0","method":"notification"}`)
	s.recordRequestSpan(notif, "notif-span")

	if len(s.requestSpans) != 1 {
		t.Errorf("Expected 1 request span, got %d", len(s.requestSpans))
	}
}

func TestFindRequestSpan(t *testing.T) {
	s := &shadowServer{
		requestSpans: map[string]string{
			"1": "span-123",
			"2": "span-456",
		},
	}

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "find existing span",
			data: []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`),
			want: "span-123",
		},
		{
			name: "find another span",
			data: []byte(`{"jsonrpc":"2.0","id":2,"result":"ok"}`),
			want: "span-456",
		},
		{
			name: "no ID in message",
			data: []byte(`{"jsonrpc":"2.0","method":"notification"}`),
			want: "",
		},
		{
			name: "unknown ID",
			data: []byte(`{"jsonrpc":"2.0","id":99,"result":"ok"}`),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.findRequestSpan(tt.data)
			if got != tt.want {
				t.Errorf("findRequestSpan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSpanID(t *testing.T) {
	// Test that span IDs are 16 hex characters
	spanID := generateSpanID()
	if len(spanID) != 16 {
		t.Errorf("Expected span ID length 16, got %d", len(spanID))
	}

	// Test that it's valid hex
	for _, c := range spanID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Invalid hex character in span ID: %c", c)
		}
	}

	// Test uniqueness
	spanID2 := generateSpanID()
	if spanID == spanID2 {
		t.Error("Generated identical span IDs")
	}
}

func TestGenerateTraceID(t *testing.T) {
	// Test that trace IDs are 32 hex characters
	traceID := generateTraceID()
	if len(traceID) != 32 {
		t.Errorf("Expected trace ID length 32, got %d", len(traceID))
	}

	// Test that it's valid hex
	for _, c := range traceID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Invalid hex character in trace ID: %c", c)
		}
	}
}

func TestShouldShadow(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		percent      float64
		expectShadow bool
	}{
		{
			name:         "shadow mode always",
			mode:         "shadow",
			percent:      0,
			expectShadow: true,
		},
		{
			name:         "random mode 100%",
			mode:         "random",
			percent:      100.0,
			expectShadow: true,
		},
		{
			name:         "random mode 0%",
			mode:         "random",
			percent:      0.0,
			expectShadow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*splitMode = tt.mode
			*splitPercent = tt.percent

			// For deterministic tests, we can't test random mode properly
			// Just check that it returns the expected value for 0% and 100%
			if tt.mode == "random" && tt.percent > 0 && tt.percent < 100 {
				// Skip probabilistic cases
				return
			}

			got := shouldShadow()
			if got != tt.expectShadow {
				t.Errorf("shouldShadow() = %v, want %v", got, tt.expectShadow)
			}
		})
	}
}
