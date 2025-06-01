package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

// TestSimple tests that the tool compiles and works
func TestSimple(t *testing.T) {
	// We'll test the functionality directly using the functions in the package
	tempDir := t.TempDir()

	// Create a simple test schema
	schemaJSON := `{
  "type": "object",
  "description": "A simple test schema",
  "properties": {
    "name": {
      "type": "string",
      "description": "Name field"
    },
    "count": {
      "type": "integer",
      "description": "Count field"
    }
  }
}`

	schemaPath := filepath.Join(tempDir, "schema.json")
	if err := ioutil.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Process the schema
	schemaData, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// Test using the convertToToolsModule function directly
	output, err := convertToToolsModule(schemaData, "test")
	if err != nil {
		t.Fatalf("Failed to convert schema: %v", err)
	}

	// Check output
	t.Logf("Generated output: %s", output)

	// Print to a file for inspection
	outputPath := filepath.Join(tempDir, "output.go")
	if err := ioutil.WriteFile(outputPath, []byte(output), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	t.Logf("Wrote output to: %s", outputPath)
}
