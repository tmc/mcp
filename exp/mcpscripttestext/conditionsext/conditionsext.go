// Package conditionsext provides advanced condition extensions for mcpscripttest.
package conditionsext

import (
	"rsc.io/script"
)

// DefaultCommands returns any additional commands provided by the conditions extension.
func DefaultCommands() map[string]script.Cmd {
	return map[string]script.Cmd{
		// No additional commands for basic conditions extension
	}
}

// DefaultConditions returns the default advanced conditions for mcpscripttest.
func DefaultConditions() map[string]script.Cond {
	return map[string]script.Cond{
		// Transport conditions
		"stdio": stdioCond,
		"http":  httpCond,
		"sse":   sseCond,

		// Capability conditions
		"tools":     toolsCond,
		"resources": resourcesCond,
		"prompts":   promptsCond,
		"logging":   loggingCond,
		"sampling":  samplingCond,

		// Server capability conditions
		"tools_list_changed":     toolsListChangedCond,
		"resources_subscribe":    resourcesSubscribeCond,
		"resources_list_changed": resourcesListChangedCond,
		"prompts_list_changed":   promptsListChangedCond,

		// Client capability conditions
		"client_sampling": clientSamplingCond,
		"client_roots":    clientRootsCond,

		// Protocol version conditions
		"protocol_version": protocolVersionCond,

		// Implementation info conditions
		"server_name":    serverNameCond,
		"server_version": serverVersionCond,
		"client_name":    clientNameCond,
		"client_version": clientVersionCond,

		// Test environment conditions
		"test_coverage": testCoverageCond,
		"test_debug":    testDebugCond,
		"test_timeout":  testTimeoutCond,
	}
}

// TODO: These condition implementations would need to be moved from the conditions package
// For now, we'll create placeholder implementations

var stdioCond = script.Condition("server supports stdio transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from conditions/conditions.go
	return true, nil
})

var httpCond = script.Condition("server supports http transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from conditions/conditions.go
	return false, nil
})

var sseCond = script.Condition("server supports sse transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from conditions/conditions.go
	return false, nil
})

var toolsCond = script.Condition("server supports tools capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for tools capability detection
	return true, nil
})

var resourcesCond = script.Condition("server supports resources capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for resources capability detection
	return true, nil
})

var promptsCond = script.Condition("server supports prompts capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for prompts capability detection
	return true, nil
})

var loggingCond = script.Condition("server supports logging capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for logging capability detection
	return false, nil
})

var samplingCond = script.Condition("server supports sampling capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for sampling capability detection
	return false, nil
})

var toolsListChangedCond = script.Condition("server supports tools list changed notifications", func(s *script.State) (bool, error) {
	// TODO: Implementation for tools list changed capability detection
	return false, nil
})

var resourcesSubscribeCond = script.Condition("server supports resource subscriptions", func(s *script.State) (bool, error) {
	// TODO: Implementation for resource subscription capability detection
	return false, nil
})

var resourcesListChangedCond = script.Condition("server supports resources list changed notifications", func(s *script.State) (bool, error) {
	// TODO: Implementation for resources list changed capability detection
	return false, nil
})

var promptsListChangedCond = script.Condition("server supports prompts list changed notifications", func(s *script.State) (bool, error) {
	// TODO: Implementation for prompts list changed capability detection
	return false, nil
})

var clientSamplingCond = script.Condition("client supports sampling", func(s *script.State) (bool, error) {
	// TODO: Implementation for client sampling capability detection
	return false, nil
})

var clientRootsCond = script.Condition("client supports roots capability", func(s *script.State) (bool, error) {
	// TODO: Implementation for client roots capability detection
	return false, nil
})

var protocolVersionCond = script.PrefixCondition(
	"check protocol version",
	func(s *script.State, version string) (bool, error) {
		// TODO: Implementation for protocol version checking
		return version == "2024-11-05", nil
	})

var serverNameCond = script.PrefixCondition(
	"check server name",
	func(s *script.State, name string) (bool, error) {
		// TODO: Implementation for server name checking
		return false, nil
	})

var serverVersionCond = script.PrefixCondition(
	"check server version",
	func(s *script.State, version string) (bool, error) {
		// TODO: Implementation for server version checking
		return false, nil
	})

var clientNameCond = script.PrefixCondition(
	"check client name",
	func(s *script.State, name string) (bool, error) {
		// TODO: Implementation for client name checking
		return false, nil
	})

var clientVersionCond = script.PrefixCondition(
	"check client version",
	func(s *script.State, version string) (bool, error) {
		// TODO: Implementation for client version checking
		return false, nil
	})

var testCoverageCond = script.Condition("test coverage is enabled", func(s *script.State) (bool, error) {
	_, hasCoverDir := s.LookupEnv("GOCOVERDIR")
	return hasCoverDir, nil
})

var testDebugCond = script.Condition("test debug mode is enabled", func(s *script.State) (bool, error) {
	debug, hasDebug := s.LookupEnv("MCP_DEBUG")
	return hasDebug && debug == "true", nil
})

var testTimeoutCond = script.PrefixCondition(
	"check test timeout setting",
	func(s *script.State, timeout string) (bool, error) {
		actualTimeout, hasTimeout := s.LookupEnv("MCP_TOOL_TIMEOUT")
		return hasTimeout && actualTimeout == timeout, nil
	})
