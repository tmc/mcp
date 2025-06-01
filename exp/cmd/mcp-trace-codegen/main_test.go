package main

import (
	"strings"
	"testing"
)

func TestParseTraceLine(t *testing.T) {
	tcg := NewTraceCodeGenerator(Options{})
	
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{
			name:    "valid initialize request",
			line:    `2024-01-15T10:00:00 -> initialize {"method":"initialize","params":{}}`,
			wantErr: false,
		},
		{
			name:    "valid tool call",
			line:    `2024-01-15T10:00:00 -> tools/call {"method":"tools/call","params":{"name":"test"}}`,
			wantErr: false,
		},
		{
			name:    "invalid format",
			line:    `invalid line`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := tcg.parseTraceLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTraceLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && entry == nil {
				t.Error("parseTraceLine() returned nil entry without error")
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	tcg := NewTraceCodeGenerator(Options{packageName: "test"})
	
	// Process some trace lines to build state
	lines := []string{
		`2024-01-15T10:00:00 -> initialize {"method":"initialize","params":{"clientInfo":{"name":"test","version":"1.0"}}}`,
		`2024-01-15T10:00:01 <- initialize {"result":{"serverInfo":{"name":"server","version":"1.0"}}}`,
	}
	
	for _, line := range lines {
		if err := tcg.ProcessLine(line); err != nil {
			t.Fatalf("ProcessLine() error: %v", err)
		}
	}
	
	code := tcg.generateCode()
	
	// Basic checks
	if !strings.Contains(code, "package test") {
		t.Error("Generated code missing package declaration")
	}
	
	if !strings.Contains(code, "import") {
		t.Error("Generated code missing imports")
	}
	
	if !strings.Contains(code, "MCPServer") {
		t.Error("Generated code missing server type")
	}
}