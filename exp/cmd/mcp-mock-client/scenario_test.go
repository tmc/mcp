package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestScenarioRunning tests the scenario-based functionality of mcp-mock-client
func TestScenarioRunning(t *testing.T) {
	scenarioFile := filepath.Join("testdata", "scenarios", "basic_scenario.json")

	// Ensure the directory exists
	err := os.MkdirAll(filepath.Dir(scenarioFile), 0755)
	if err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create a sample scenario for testing
	scenario := Scenario{
		Name:        "Basic API Test",
		Description: "Tests basic API functionality",
		Steps: []ScenarioStep{
			{
				Name:        "Initialize",
				Description: "Initialize the connection",
				Request:     json.RawMessage(`{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`),
				Expectations: []Expectation{
					{
						Type:    "response",
						Pattern: json.RawMessage(`{"jsonrpc":"2.0","result":{"capabilities":{}},"id":1}`),
					},
				},
			},
			{
				Name:        "List Tools",
				Description: "List available tools",
				Request:     json.RawMessage(`{"jsonrpc":"2.0","method":"listTools","params":{},"id":2}`),
				Expectations: []Expectation{
					{
						Type:    "response",
						Pattern: json.RawMessage(`{"jsonrpc":"2.0","result":{"tools":[]},"id":2}`),
					},
				},
			},
		},
	}

	// Write scenario to file
	scenarioBytes, err := json.MarshalIndent(scenario, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal scenario: %v", err)
	}
	err = os.WriteFile(scenarioFile, scenarioBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write scenario file: %v", err)
	}

	// Run the scenario with a mock input/output
	var outBuf bytes.Buffer
	ctx := context.Background()
	err = runScenario(ctx, &outBuf, scenarioFile)
	if err != nil {
		t.Fatalf("Failed to run scenario: %v", err)
	}

	// Verify the output is not empty when not in dry run mode
	output := outBuf.String()
	if !*dryRun && len(output) == 0 {
		t.Errorf("Expected output, got empty buffer")
	}
}

// TestMockResponder implements the ResponseReader interface for testing scenarios
type TestMockResponder struct {
	Responses []string
	index     int
}

func (m *TestMockResponder) ReadResponse() ([]byte, error) {
	if m.index >= len(m.Responses) {
		return nil, io.EOF
	}
	resp := m.Responses[m.index]
	m.index++
	return []byte(resp), nil
}
