package mcpspec

import (
	"encoding/json"
	"fmt"
	"os"
)

// Parse parses an MCPSpec from bytes
// Currently only supports JSON. For YAML support, we'd need to add a dependency.
func Parse(data []byte) (*MCPSpec, error) {
	var spec MCPSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshaling spec: %w", err)
	}
	return &spec, nil
}

// ParseFile parses an MCPSpec from a file
func ParseFile(path string) (*MCPSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return Parse(data)
}
