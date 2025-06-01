package conditions

import (
	"encoding/json"
	"fmt"
	"strings"

	"rsc.io/script"
)

var (
	envVarCondServerSupports      = "MCP_SCRIPTTEST_SERVER_TRANSPORT_%s"
	envVarCondServerSupportsStdio = fmt.Sprintf(envVarCondServerSupports, "STDIO")
	envVarCondServerSupportsHTTP  = fmt.Sprintf(envVarCondServerSupports, "HTTP")
	envVarCondServerSupportsSSE   = fmt.Sprintf(envVarCondServerSupports, "SSE")

	envVarCondClientSupports = "MCP_SCRIPTTEST_CLIENT_CAPABILITIES"
	envVarCondServerCapabilities = "MCP_SCRIPTTEST_SERVER_CAPABILITIES"
)

// AddDefaultMCPConditions adds MCP-specific condition directives to the script engine
func AddDefaultMCPConditions(e *script.Engine) {
	// Basic transport conditions
	e.Conds["stdio"] = script.Condition("server supports stdio transport", func(s *script.State) (bool, error) {
		// Check if stdio was disabled via environment variable
		if disabled, ok := s.LookupEnv("MCP_DISABLE_STDIO"); ok && disabled == "true" {
			return false, nil
		}
		return true, nil // Default to supporting stdio
	})

	e.Conds["http"] = script.Condition("server supports http transport", func(s *script.State) (bool, error) {
		// Check if http was disabled via environment variable
		if disabled, ok := s.LookupEnv("MCP_DISABLE_HTTP"); ok && disabled == "true" {
			return false, nil
		}
		// HTTP is disabled by default
		return false, nil
	})

	e.Conds["sse"] = script.Condition("server supports sse transport", func(s *script.State) (bool, error) {
		// Check if sse was disabled via environment variable
		if disabled, ok := s.LookupEnv("MCP_DISABLE_SSE"); ok && disabled == "true" {
			return false, nil
		}
		// SSE is disabled by default
		return false, nil
	})

	// Add other capability conditions
	e.Conds["tools"] = script.Condition("server supports tools", func(s *script.State) (bool, error) {
		if disabled, ok := s.LookupEnv("MCP_DISABLE_TOOLS"); ok && disabled == "true" {
			return false, nil
		}
		return false, nil // Tools disabled by default
	})

	e.Conds["resources"] = script.Condition("server supports resources", func(s *script.State) (bool, error) {
		if disabled, ok := s.LookupEnv("MCP_DISABLE_RESOURCES"); ok && disabled == "true" {
			return false, nil
		}
		return false, nil // Resources disabled by default
	})

	e.Conds["prompts"] = script.Condition("server supports prompts", func(s *script.State) (bool, error) {
		if disabled, ok := s.LookupEnv("MCP_DISABLE_PROMPTS"); ok && disabled == "true" {
			return false, nil
		}
		return false, nil // Prompts disabled by default
	})

	e.Conds["logging"] = script.Condition("server supports logging", func(s *script.State) (bool, error) {
		if disabled, ok := s.LookupEnv("MCP_DISABLE_LOGGING"); ok && disabled == "true" {
			return false, nil
		}
		return false, nil // Logging disabled by default
	})

	// Add negated versions of all conditions
	addNegatedConditions(e)

	// Legacy conditions - these are kept for backward compatibility
	e.Conds["supports-transport"] = script.PrefixCondition(
		"Server supports <suffix>",
		func(s *script.State, suffix string) (bool, error) {
			_, ok := s.LookupEnv(fmt.Sprintf(envVarCondServerSupports, strings.ToUpper(suffix)))
			return ok, nil
		})

	e.Conds["client-supports"] = script.PrefixCondition(
		"Client supports <suffix>",
		func(s *script.State, suffix string) (bool, error) {
			ccaps, _ := s.LookupEnv(envVarCondClientSupports)
			return commonSupportsCheck(ccaps, suffix)
		})

	e.Conds["server-supports"] = script.PrefixCondition(
		"Server supports <capability>",
		func(s *script.State, suffix string) (bool, error) {
			if suffix == "" {
				return false, script.ErrUsage
			}
			scaps, _ := s.LookupEnv(envVarCondServerCapabilities) // Using the correct variable
			return commonSupportsCheck(scaps, suffix)
		},
	)

	// Add capability-specific conditions
	e.Conds["tools"] = mcpSupportsToolsCondition
	e.Conds["resources"] = mcpSupportsResourcesCondition
	e.Conds["prompts"] = mcpSupportsPromptsCondition
	e.Conds["logging"] = mcpSupportsLoggingCondition
	e.Conds["batch"] = mcpSupportsBatchCondition
	e.Conds["auth"] = mcpSupportsAuthCondition
	e.Conds["version"] = mcpSupportsVersionCondition
	e.Conds["progress"] = mcpSupportsProgressCondition
}

func mcpSupportsTransportCondition(transport string) script.Cond {
	return script.PrefixCondition(
		fmt.Sprintf("Server supports %s", transport),
		func(s *script.State, suffix string) (bool, error) {
			if suffix != "" {
				return false, script.ErrUsage
			}
			_, ok := s.LookupEnv(fmt.Sprintf(envVarCondServerSupports, strings.ToUpper(transport)))
			return ok, nil
		})
}

// addNegatedConditions adds a negated version of each condition
// This supports the [!condition] syntax in test files
func addNegatedConditions(e *script.Engine) {
	// Make a copy of the conditions map to avoid modifying during iteration
	condsCopy := make(map[string]script.Cond)
	for name, cond := range e.Conds {
		condsCopy[name] = cond
	}

	// Now add negated versions of all conditions
	for name, cond := range condsCopy {
		// Skip any conditions that are already negated
		if strings.HasPrefix(name, "!") {
			continue
		}

		// Create a new negated condition
		notName := "!" + name
		originalCond := cond // Capture the condition value
		e.Conds[notName] = script.Condition(fmt.Sprintf("NOT %s", name), func(s *script.State) (bool, error) {
			match, err := originalCond.Eval(s, "")
			if err != nil {
				return false, err
			}
			return !match, nil
		})
	}
}

// commonSupportsCheck checks if the capabilities string contains the specified capability
func commonSupportsCheck(capabilities, capability string) (bool, error) {
	if capabilities == "" {
		return false, nil
	}

	// The capabilities can be in various formats:
	// 1. JSON array: ["tools", "resources"]
	// 2. JSON object: {"tools": true, "resources": false}
	// 3. Comma-separated: "tools,resources"
	// 4. JSON object with nested capabilities

	// Try to parse as JSON array
	var capsList []string
	if err := json.Unmarshal([]byte(capabilities), &capsList); err == nil {
		for _, cap := range capsList {
			if strings.EqualFold(cap, capability) {
				return true, nil
			}
		}
		return false, nil
	}

	// Try to parse as JSON object
	var capsMap map[string]interface{}
	if err := json.Unmarshal([]byte(capabilities), &capsMap); err == nil {
		// Check if the capability exists and is truthy
		if val, ok := capsMap[capability]; ok {
			// Handle boolean values
			if boolVal, isBool := val.(bool); isBool {
				return boolVal, nil
			}
			// Handle nested objects (for capabilities with details)
			if _, isMap := val.(map[string]interface{}); isMap {
				return true, nil
			}
		}
		return false, nil
	}

	// Try comma-separated values
	for _, cap := range strings.Split(capabilities, ",") {
		if strings.EqualFold(strings.TrimSpace(cap), capability) {
			return true, nil
		}
	}

	return false, nil
}

// Capability-specific conditions
var (
	mcpSupportsToolsCondition = script.Condition("MCP server supports tools", func(s *script.State) (bool, error) {
		return checkCapability(s, "tools")
	})

	mcpSupportsResourcesCondition = script.Condition("MCP server supports resources", func(s *script.State) (bool, error) {
		return checkCapability(s, "resources")
	})

	mcpSupportsPromptsCondition = script.Condition("MCP server supports prompts", func(s *script.State) (bool, error) {
		return checkCapability(s, "prompts")
	})

	mcpSupportsLoggingCondition = script.Condition("MCP server supports logging", func(s *script.State) (bool, error) {
		return checkCapability(s, "logging")
	})

	mcpSupportsBatchCondition = script.Condition("MCP server supports batch", func(s *script.State) (bool, error) {
		return checkCapability(s, "batch")
	})

	mcpSupportsAuthCondition = script.Condition("MCP server supports auth", func(s *script.State) (bool, error) {
		return checkCapability(s, "auth")
	})

	mcpSupportsVersionCondition = script.Condition("MCP server supports version", func(s *script.State) (bool, error) {
		return checkCapability(s, "version")
	})

	mcpSupportsProgressCondition = script.Condition("MCP server supports progress", func(s *script.State) (bool, error) {
		return checkCapability(s, "progress")
	})
)

// checkCapability checks if the server has a specific capability
func checkCapability(s *script.State, capability string) (bool, error) {
	caps, _ := s.LookupEnv(envVarCondServerCapabilities)
	return commonSupportsCheck(caps, capability)
}