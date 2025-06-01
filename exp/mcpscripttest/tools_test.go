package mcpscripttest

import (
	"os"
	"testing"
)

func TestMCPToolsIntegration(t *testing.T) {
	// Run the test-all-tools.txt script test
	Test(t, "testdata/test-all-tools.txt")
}

func TestToolsOptionsAutoCoverage(t *testing.T) {
	tests := []struct {
		name           string
		opts           *ToolsOptions
		envCoverDir    string
		expectCoverage bool
	}{
		{
			name: "auto-detect enabled with GOCOVERDIR set",
			opts: &ToolsOptions{
				AutoDetectCoverage: true,
				InstallCoverage:    false,
			},
			envCoverDir:    "/tmp/coverage",
			expectCoverage: true,
		},
		{
			name: "auto-detect enabled without GOCOVERDIR",
			opts: &ToolsOptions{
				AutoDetectCoverage: true,
				InstallCoverage:    false,
			},
			envCoverDir:    "",
			expectCoverage: false,
		},
		{
			name: "explicit coverage enabled",
			opts: &ToolsOptions{
				AutoDetectCoverage: false,
				InstallCoverage:    true,
			},
			envCoverDir:    "",
			expectCoverage: true,
		},
		{
			name: "auto-detect disabled",
			opts: &ToolsOptions{
				AutoDetectCoverage: false,
				InstallCoverage:    false,
			},
			envCoverDir:    "/tmp/coverage",
			expectCoverage: false,
		},
		{
			name:           "default options with GOCOVERDIR",
			opts:           DefaultToolsOptions(),
			envCoverDir:    "/tmp/coverage",
			expectCoverage: true,
		},
		{
			name:           "default options without GOCOVERDIR",
			opts:           DefaultToolsOptions(),
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
			coverageEnabled := tc.opts.InstallCoverage
			if tc.opts.AutoDetectCoverage && !tc.opts.InstallCoverage {
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