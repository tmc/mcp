package main

import (
	"testing"
	"time"
)

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *TraceHeader
		wantNil bool
	}{
		{
			name: "basic header",
			line: "# mcptrace:v1",
			want: &TraceHeader{
				Version: "1",
				Baggage: make(map[string]string),
			},
		},
		{
			name: "header with trace parent",
			line: "# mcptrace:v1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want: &TraceHeader{
				Version:     "1",
				TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				Baggage:     make(map[string]string),
			},
		},
		{
			name: "header with baggage",
			line: "# mcptrace:v1 baggage=env=test,user=123",
			want: &TraceHeader{
				Version: "1",
				Baggage: map[string]string{
					"env":  "test",
					"user": "123",
				},
			},
		},
		{
			name: "full header",
			line: "# mcptrace:v1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 tracestate=vendor=value baggage=key=value",
			want: &TraceHeader{
				Version:     "1",
				TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				TraceState:  "vendor=value",
				Baggage: map[string]string{
					"key": "value",
				},
			},
		},
		{
			name:    "invalid header",
			line:    "# not-mcptrace",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeader(tt.line)
			
			if tt.wantNil {
				if got != nil {
					t.Errorf("parseHeader() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("parseHeader() = nil, want %v", tt.want)
				return
			}

			if got.Version != tt.want.Version {
				t.Errorf("Version = %v, want %v", got.Version, tt.want.Version)
			}
			if got.TraceParent != tt.want.TraceParent {
				t.Errorf("TraceParent = %v, want %v", got.TraceParent, tt.want.TraceParent)
			}
			if got.TraceState != tt.want.TraceState {
				t.Errorf("TraceState = %v, want %v", got.TraceState, tt.want.TraceState)
			}
			if len(got.Baggage) != len(tt.want.Baggage) {
				t.Errorf("Baggage length = %v, want %v", len(got.Baggage), len(tt.want.Baggage))
			}
			for k, v := range tt.want.Baggage {
				if got.Baggage[k] != v {
					t.Errorf("Baggage[%s] = %v, want %v", k, got.Baggage[k], v)
				}
			}
		})
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *MCPMessage
		wantNil bool
	}{
		{
			name: "basic recv line",
			line: `mcp-recv {"method":"test","id":1} # 1683000000.500`,
			want: &MCPMessage{
				Direction: "recv",
				JSON: map[string]interface{}{
					"method": "test",
					"id":     float64(1),
				},
				Timestamp: time.Unix(1683000000, 500000000),
				Baggage:   make(map[string]string),
			},
		},
		{
			name: "send line with span",
			line: `mcp-send {"result":"ok","id":1} # 1683000001.000 spanid=abc123`,
			want: &MCPMessage{
				Direction: "send",
				JSON: map[string]interface{}{
					"result": "ok",
					"id":     float64(1),
				},
				Timestamp: time.Unix(1683000001, 0),
				SpanID:    "abc123",
				Baggage:   make(map[string]string),
			},
		},
		{
			name: "line with links and baggage",
			line: `mcp-send {"result":"shadow"} # 1683000002.500 spanid=def456 linksto=abc123 baggage=shadow=true,test=1`,
			want: &MCPMessage{
				Direction: "send",
				JSON: map[string]interface{}{
					"result": "shadow",
				},
				Timestamp: time.Unix(1683000002, 500000000),
				SpanID:    "def456",
				LinksTo:   "abc123",
				Baggage: map[string]string{
					"shadow": "true",
					"test":   "1",
				},
			},
		},
		{
			name: "shadow line (commented)",
			line: `# mcp-send {"result":"shadow"} # 1683000003.000 spanid=ghi789`,
			want: &MCPMessage{
				Direction: "send",
				JSON: map[string]interface{}{
					"result": "shadow",
				},
				Timestamp: time.Unix(1683000003, 0),
				SpanID:    "ghi789",
				Baggage:   make(map[string]string),
			},
		},
		{
			name:    "invalid line",
			line:    "not a valid mcp line",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.line)

			if tt.wantNil {
				if got != nil {
					t.Errorf("parseLine() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("parseLine() = nil, want %v", tt.want)
				return
			}

			if got.Direction != tt.want.Direction {
				t.Errorf("Direction = %v, want %v", got.Direction, tt.want.Direction)
			}
			if got.SpanID != tt.want.SpanID {
				t.Errorf("SpanID = %v, want %v", got.SpanID, tt.want.SpanID)
			}
			if got.LinksTo != tt.want.LinksTo {
				t.Errorf("LinksTo = %v, want %v", got.LinksTo, tt.want.LinksTo)
			}
			if !got.Timestamp.Equal(tt.want.Timestamp) {
				t.Errorf("Timestamp = %v, want %v", got.Timestamp, tt.want.Timestamp)
			}

			// Compare JSON
			for k, v := range tt.want.JSON {
				if got.JSON[k] != v {
					t.Errorf("JSON[%s] = %v, want %v", k, got.JSON[k], v)
				}
			}

			// Compare baggage
			if len(got.Baggage) != len(tt.want.Baggage) {
				t.Errorf("Baggage length = %v, want %v", len(got.Baggage), len(tt.want.Baggage))
			}
			for k, v := range tt.want.Baggage {
				if got.Baggage[k] != v {
					t.Errorf("Baggage[%s] = %v, want %v", k, got.Baggage[k], v)
				}
			}
		})
	}
}

func TestParseBaggage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "single pair",
			input: "key=value",
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:  "multiple pairs",
			input: "env=prod,user=123,debug=true",
			expected: map[string]string{
				"env":   "prod",
				"user":  "123",
				"debug": "true",
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "malformed pairs",
			input: "key=value,invalid,another=test",
			expected: map[string]string{
				"key":     "value",
				"another": "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]string)
			parseBaggage(tt.input, result)

			if len(result) != len(tt.expected) {
				t.Errorf("parseBaggage() resulted in %d pairs, want %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseBaggage()[%s] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}