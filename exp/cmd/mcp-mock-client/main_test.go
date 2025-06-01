package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestScenarioLoading tests the scenario loading functionality
func TestScenarioLoading(t *testing.T) {
	// Create a temporary scenario file
	dir, err := ioutil.TempDir("", "scenario-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	scenarioFile := filepath.Join(dir, "test-scenario.json")
	scenario := Scenario{
		Name:        "Test Scenario",
		Description: "A test scenario",
		Steps: []ScenarioStep{
			{
				Name:        "Step 1",
				Description: "Test step",
				Request:     json.RawMessage(`{"method":"test","id":1}`),
			},
		},
	}

	scenarioBytes, err := json.Marshal(scenario)
	if err != nil {
		t.Fatalf("Failed to marshal scenario: %v", err)
	}

	if err := ioutil.WriteFile(scenarioFile, scenarioBytes, 0644); err != nil {
		t.Fatalf("Failed to write scenario file: %v", err)
	}

	// Load the scenario
	loadedScenario, err := LoadScenario(scenarioFile)
	if err != nil {
		t.Fatalf("Failed to load scenario: %v", err)
	}

	// Verify the loaded scenario
	if loadedScenario.Name != scenario.Name {
		t.Errorf("Expected name %s, got %s", scenario.Name, loadedScenario.Name)
	}
	if loadedScenario.Description != scenario.Description {
		t.Errorf("Expected description %s, got %s", scenario.Description, loadedScenario.Description)
	}
	if len(loadedScenario.Steps) != len(scenario.Steps) {
		t.Errorf("Expected %d steps, got %d", len(scenario.Steps), len(loadedScenario.Steps))
	}
}

// TestJSONMatcher tests the JSON pattern matcher
func TestJSONMatcher(t *testing.T) {
	matcher := JSONMatcher{}

	tests := []struct {
		name     string
		pattern  string
		actual   string
		expected bool
	}{
		{
			name:     "Simple equality",
			pattern:  `{"id": 1}`,
			actual:   `{"id": 1}`,
			expected: true,
		},
		{
			name:     "Type matching",
			pattern:  `{"name": "{{string}}"}`,
			actual:   `{"name": "test"}`,
			expected: true,
		},
		{
			name:     "Regex matching",
			pattern:  `{"version": "/^[0-9]+\\.[0-9]+\\.[0-9]+$/"}`,
			actual:   `{"version": "1.2.3"}`,
			expected: true,
		},
		{
			name:     "Partial matching",
			pattern:  `{"data": {"{{partial}}": true, "required": "value"}}`,
			actual:   `{"data": {"required": "value", "extra": "field"}}`,
			expected: true,
		},
		{
			name:     "Mismatch",
			pattern:  `{"id": 1}`,
			actual:   `{"id": 2}`,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.Match([]byte(tc.pattern), []byte(tc.actual))
			if result.Success != tc.expected {
				t.Errorf("Expected success=%v, got %v: %s", tc.expected, result.Success, result.Message)
			}
		})
	}
}

// TestRunScenario tests the runScenario function
func TestRunScenario(t *testing.T) {
	// Set up global flags for testing
	*verbose = true
	*dryRun = true

	// Create a temporary scenario file
	dir, err := ioutil.TempDir("", "scenario-run-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	scenarioFile := filepath.Join(dir, "run-scenario.json")
	scenario := Scenario{
		Name:        "Run Test",
		Description: "Test running a scenario",
		Steps: []ScenarioStep{
			{
				Name:        "Step 1",
				Description: "Test step",
				Request:     json.RawMessage(`{"method":"test","id":1}`),
				Delay:       100 * time.Millisecond,
			},
		},
	}

	scenarioBytes, err := json.Marshal(scenario)
	if err != nil {
		t.Fatalf("Failed to marshal scenario: %v", err)
	}

	if err := ioutil.WriteFile(scenarioFile, scenarioBytes, 0644); err != nil {
		t.Fatalf("Failed to write scenario file: %v", err)
	}

	// Run the scenario
	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = runScenario(ctx, &buf, scenarioFile)
	if err != nil {
		t.Fatalf("Failed to run scenario: %v", err)
	}

	// Verify something was written to the buffer
	if buf.Len() == 0 && !*dryRun {
		t.Errorf("Expected output, got none")
	}
}
