package examples

import (
	"fmt"
	"os"
	"strings"

	"github.com/tmc/mcp/exp/mcpscripttest"
	"rsc.io/script"
)

// ExampleCustomConditions demonstrates how to add custom condition checks
// for implementation-specific features to the mcpscripttest framework.
func ExampleCustomConditions() {
	// Create options with custom conditions
	options := mcpscripttest.DefaultOptions()

	// Add custom conditions for your implementation
	options.CustomConditions = map[string]script.Cond{
		"supports_custom_feature": customFeatureCondition,
		"has_database_access":     databaseAccessCondition,
		"api_version":             apiVersionCondition,
	}

	// Use these options when creating an engine or running tests
	_ = options // engine := mcpscripttest.NewEngine(options)

	// You can now use these conditions in your test scripts:
	//
	// # Skip this test if custom feature is not supported
	// supports_custom_feature
	//
	// # Skip this test if database access is not available
	// has_database_access
	//
	// # Skip this test if API version is less than specified
	// api_version 2.0

	// Example of how to use the engine with these conditions
	fmt.Println("Custom conditions added to engine:",
		len(options.CustomConditions))
}

// customFeatureCondition checks if a custom feature is supported
var customFeatureCondition = script.Condition(
	"check if the implementation supports custom feature",
	func(s *script.State) (bool, error) {
		// Check for the environment variable that would enable this feature
		if os.Getenv("IMPL_CUSTOM_FEATURE_ENABLED") != "true" {
			return false, nil
		}
		return true, nil
	},
)

// databaseAccessCondition checks if database access is available
var databaseAccessCondition = script.Condition(
	"check if database access is available",
	func(s *script.State) (bool, error) {
		// Check for the environment variable with database connection string
		if os.Getenv("DATABASE_URL") == "" {
			return false, nil
		}
		return true, nil
	},
)

// apiVersionCondition checks if the API version meets the minimum requirement
var apiVersionCondition = script.PrefixCondition(
	"check if the API version is at least the specified version",
	func(s *script.State, arg string) (bool, error) {
		minVersion := arg
		currentVersion := os.Getenv("API_VERSION")
		if currentVersion == "" {
			return false, nil
		}

		// Simple string comparison (in a real implementation, you would
		// parse and compare version numbers properly)
		if strings.Compare(currentVersion, minVersion) < 0 {
			return false, nil
		}

		return true, nil
	},
)
