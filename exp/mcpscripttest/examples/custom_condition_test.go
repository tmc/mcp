package examples

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
	"rsc.io/script"
)

// TestCustomConditions demonstrates how to create custom conditions for
// implementation-specific features in your own implementation.
func TestCustomConditions(t *testing.T) {
	// Create the options with custom conditions
	options := mcpscripttest.DefaultOptions()
	
	// Add custom implementation-specific conditions
	options.CustomConditions = map[string]script.Cond{
		// Check if the implementation supports a custom protocol extension
		"protocol_extension": script.Condition(func() error {
			if os.Getenv("MCP_PROTOCOL_EXTENSION") != "true" {
				return fmt.Errorf("protocol extension not supported")
			}
			return nil
		}),
		
		// Check if the implementation has a specific feature enabled
		"custom_feature": script.TestCondition(
			script.CondUsage{
				Summary: "check if a specific custom feature is enabled",
				Args:    "feature_name",
			},
			func(s *script.State, args ...string) error {
				if len(args) != 1 {
					return script.ErrUsage
				}
				feature := args[0]
				if os.Getenv("MCP_FEATURE_"+strings.ToUpper(feature)) != "true" {
					return fmt.Errorf("custom feature %s not enabled", feature)
				}
				return nil
			},
		),
		
		// Check if the implementation supports specific authentication methods
		"auth_method": script.TestCondition(
			script.CondUsage{
				Summary: "check if a specific authentication method is supported",
				Args:    "method",
			},
			func(s *script.State, args ...string) error {
				if len(args) != 1 {
					return script.ErrUsage
				}
				method := args[0]
				
				supportedMethods := os.Getenv("MCP_AUTH_METHODS")
				if supportedMethods == "" {
					return fmt.Errorf("no authentication methods configured")
				}
				
				for _, supported := range strings.Split(supportedMethods, ",") {
					if supported == method {
						return nil
					}
				}
				
				return fmt.Errorf("auth method %s not supported", method)
			},
		),
		
		// Check if the implementation supports specific transports
		"custom_transport": script.TestCondition(
			script.CondUsage{
				Summary: "check if a specific transport is supported",
				Args:    "transport",
			},
			func(s *script.State, args ...string) error {
				if len(args) != 1 {
					return script.ErrUsage
				}
				transport := args[0]
				
				supportedTransports := os.Getenv("MCP_CUSTOM_TRANSPORTS")
				if supportedTransports == "" {
					return fmt.Errorf("no custom transports configured")
				}
				
				for _, supported := range strings.Split(supportedTransports, ",") {
					if supported == transport {
						return nil
					}
				}
				
				return fmt.Errorf("custom transport %s not supported", transport)
			},
		),
		
		// Check if the implementation is running in a specific mode
		"runtime_mode": script.TestCondition(
			script.CondUsage{
				Summary: "check if running in a specific mode",
				Args:    "mode",
			},
			func(s *script.State, args ...string) error {
				if len(args) != 1 {
					return script.ErrUsage
				}
				mode := args[0]
				
				currentMode := os.Getenv("MCP_RUNTIME_MODE")
				if currentMode != mode {
					return fmt.Errorf("not running in %s mode (current: %s)", 
						mode, currentMode)
				}
				
				return nil
			},
		),
	}
	
	// Run the tests with custom conditions
	// The test scripts can now use conditions like:
	// [protocol_extension] echo "Protocol extension is enabled"
	// [custom_feature oauth] echo "OAuth custom feature is enabled"
	// [auth_method saml] echo "SAML authentication is supported"
	// [custom_transport grpc] echo "gRPC transport is supported"
	// [runtime_mode production] echo "Running in production mode"
	
	// Example of setting up a test environment
	os.Setenv("MCP_PROTOCOL_EXTENSION", "true")
	os.Setenv("MCP_FEATURE_OAUTH", "true")
	os.Setenv("MCP_AUTH_METHODS", "basic,bearer,saml")
	os.Setenv("MCP_CUSTOM_TRANSPORTS", "grpc,quic")
	os.Setenv("MCP_RUNTIME_MODE", "development")
	
	// Example of running a test with custom conditions
	testPattern := "./testdata/custom_conditions/*.txt"
	mcpscripttest.TestWithCoverageOptions(t, testPattern, nil, options)
}

// Example of a test script that uses custom conditions
/*
# custom_conditions_test.txt

# Protocol extension tests
[protocol_extension] echo "Protocol extension is enabled"

# Custom feature tests
[custom_feature oauth] echo "OAuth custom feature is enabled"
[custom_feature saml] echo "SAML custom feature is enabled"

# Auth method tests
[auth_method basic] echo "Basic authentication is supported"
[auth_method saml] echo "SAML authentication is supported"
[auth_method oauth] echo "OAuth authentication is supported"

# Custom transport tests
[custom_transport grpc] echo "gRPC transport is supported"
[custom_transport quic] echo "QUIC transport is supported"
[custom_transport websocket] echo "WebSocket transport is supported"

# Runtime mode tests
[runtime_mode development] echo "Running in development mode"
[runtime_mode production] echo "Running in production mode"
[runtime_mode testing] echo "Running in testing mode"

# Combining standard and custom conditions
[http] [custom_transport grpc] echo "Both HTTP and gRPC transports are supported"
*/