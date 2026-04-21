package main

import (
	"os"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpspec"
	"github.com/tmc/mcp/exp/sourcegen"
)

func TestMCPSpecGeneration(t *testing.T) {
	// Read example spec
	data, err := os.ReadFile("testdata/example.mcpspec")
	if err != nil {
		t.Fatalf("Failed to read example.mcpspec: %v", err)
	}

	// Parse spec
	spec, err := mcpspec.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Generate code
	gen := sourcegen.NewGenerator("generated")
	output, err := gen.GenerateFromSpec(spec)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Verify required elements
	checks := []string{
		`ServerName    = "weather-server"`,
		`ServerVersion = "1.0.0"`,
		`type GetWeatherInput struct`,
		`func (t *GetWeatherImpl) Execute`,
		`type CurrentWeatherResource interface`,
		`func (r *CurrentWeatherImpl) Read`,
		`type WeatherReportPrompt interface`,
		`func (p *WeatherReportImpl) Get`,
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Generated output missing expected string: %s", check)
		}
	}
}
