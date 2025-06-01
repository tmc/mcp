package coverage_viz

import (
	"fmt"
	"strings"
	"testing"
)

// Example function we want to test
func ParseMessage(msg string) (string, error) {
	if msg == "" {
		return "", fmt.Errorf("empty message")
	}
	
	parts := strings.Split(msg, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid format")
	}
	
	switch parts[0] {
	case "info":
		return fmt.Sprintf("INFO: %s", parts[1]), nil
	case "warn":
		return fmt.Sprintf("WARNING: %s", parts[1]), nil
	case "error":
		return fmt.Sprintf("ERROR: %s", parts[1]), nil
	default:
		return fmt.Sprintf("UNKNOWN: %s", parts[1]), nil
	}
}

// Tests that demonstrate coverage mapping
func TestParseMessage_Info(t *testing.T) {
	result, err := ParseMessage("info:test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "INFO: test message" {
		t.Errorf("expected 'INFO: test message', got %s", result)
	}
}

func TestParseMessage_Warn(t *testing.T) {
	result, err := ParseMessage("warn:caution")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "WARNING: caution" {
		t.Errorf("expected 'WARNING: caution', got %s", result)
	}
}

func TestParseMessage_Empty(t *testing.T) {
	_, err := ParseMessage("")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestParseMessage_InvalidFormat(t *testing.T) {
	_, err := ParseMessage("no colon here")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

// This test demonstrates partial coverage of the switch statement
func TestParseMessage_PartialCoverage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"info:msg", "INFO: msg"},
		{"warn:msg", "WARNING: msg"},
		// Note: missing "error" and "default" cases
	}
	
	for _, tc := range tests {
		result, err := ParseMessage(tc.input)
		if err != nil {
			t.Errorf("unexpected error for %s: %v", tc.input, err)
		}
		if result != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, result)
		}
	}
}