package mcpscripttest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFuzzingScripts runs the mcpscripttest tests for fuzzing functionality
func TestFuzzingScripts(t *testing.T) {
	// Get the path to our scripttest files
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "fuzzing")

	// Define test cases
	tests := []struct {
		name string
		file string
		skip string // reason to skip
	}{
		{
			name: "simple fuzzing",
			file: "simple_fuzzing_test.txt",
		},
		{
			name: "basic fuzzing",
			file: "basic_fuzzing_test.txt",
			skip: "requires external server setup",
		},
		{
			name: "coverage guided fuzzing",
			file: "coverage_guided_fuzzing_test.txt",
		},
		{
			name: "specialized generators",
			file: "specialized_generators_test.txt",
		},
		{
			name: "smart generator",
			file: "smart_generator_test.txt",
		},
		{
			name: "cooperative fuzzing",
			file: "cooperative_fuzzing_test.txt",
		},
		{
			name: "visualization",
			file: "visualization_test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			// Use the Test function from mcpscripttest to run the script
			opts := DefaultOptions()
			opts.AdditionalEnvVars = []string{"FUZZING_TEST"}
			opts.IncludeDefaultMCPCommands = true

			// Set environment variable
			os.Setenv("FUZZING_TEST", "1")
			defer os.Unsetenv("FUZZING_TEST")

			Test(t, filepath.Join(testdataDir, tt.file), opts)
		})
	}
}

// TestFuzzingIntegration tests the integration between fuzzing and mcpscripttest
func TestFuzzingIntegration(t *testing.T) {
	// Skip this for now as it needs the fuzzing package
	t.Skip("Fuzzing integration test - requires fuzzing package import")
}
