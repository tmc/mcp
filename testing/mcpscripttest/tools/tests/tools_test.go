package tests

import (
	"os"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"

	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

func TestMCPToolsIntegration(t *testing.T) {
	// Run the test-all-tools.txt script test
	mcpscripttest.Test(t, "../../testdata/test-all-tools.txt")
}

func TestToolsOptionsAutoCoverage(t *testing.T) {
	tests := []struct {
		name           string
		opts           *tools.ToolsOptions
		envCoverDir    string
		expectCoverage bool
	}{
		{
			name: "auto-detect enabled with GOCOVERDIR set",
			opts: &tools.ToolsOptions{
				AutoDetectCoverage: true,
				CoverMode:          tools.ToolCoverModeAuto,
			},
			envCoverDir:    "/tmp/coverage",
			expectCoverage: true,
		},
		{
			name: "auto-detect enabled without GOCOVERDIR",
			opts: &tools.ToolsOptions{
				AutoDetectCoverage: true,
				CoverMode:          tools.ToolCoverModeAuto,
			},
			envCoverDir:    "",
			expectCoverage: false,
		},
		{
			name: "explicit coverage enabled",
			opts: &tools.ToolsOptions{
				AutoDetectCoverage: false,
				CoverMode:          tools.ToolCoverModeOn,
			},
			envCoverDir:    "",
			expectCoverage: true,
		},
		{
			name: "auto-detect disabled",
			opts: &tools.ToolsOptions{
				AutoDetectCoverage: false,
				CoverMode:          tools.ToolCoverModeOff,
			},
			envCoverDir:    "/tmp/coverage",
			expectCoverage: false,
		},
		{
			name:           "default options with GOCOVERDIR",
			opts:           tools.DefaultToolsOptions(),
			envCoverDir:    "/tmp/coverage",
			expectCoverage: true,
		},
		{
			name:           "default options without GOCOVERDIR",
			opts:           tools.DefaultToolsOptions(),
			envCoverDir:    "",
			expectCoverage: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore original GOCOVERDIR
			origCoverDir := os.Getenv("GOCOVERDIR")
			defer os.Setenv("GOCOVERDIR", origCoverDir)

			// Set test GOCOVERDIR
			if tc.envCoverDir != "" {
				os.Setenv("GOCOVERDIR", tc.envCoverDir)
			} else {
				os.Unsetenv("GOCOVERDIR")
			}

			// Determine if coverage should be enabled
			coverageEnabled := tc.opts.CoverMode == tools.ToolCoverModeOn
			if tc.opts.AutoDetectCoverage && tc.opts.CoverMode == tools.ToolCoverModeAuto {
				if os.Getenv("GOCOVERDIR") != "" {
					coverageEnabled = true
				}
			}

			if coverageEnabled != tc.expectCoverage {
				t.Errorf("Expected coverage enabled: %v, got: %v", tc.expectCoverage, coverageEnabled)
			}
		})
	}
}
