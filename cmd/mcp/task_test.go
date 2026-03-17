package main

import "testing"

func TestIsTerminalTaskStatus(t *testing.T) {
	for _, status := range []string{"completed", "failed", "cancelled", "input_required"} {
		if !isTerminalTaskStatus(status) {
			t.Fatalf("status %q should be terminal", status)
		}
	}
	if isTerminalTaskStatus("working") {
		t.Fatal("working should not be terminal")
	}
}
