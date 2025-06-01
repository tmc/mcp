package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// MatchResult represents the result of a match operation
type MatchResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

// PatternMatcher is an interface for matching patterns against responses
type PatternMatcher interface {
	Match(pattern, actual []byte) MatchResult
}

// JSONMatcher implements PatternMatcher for JSON data
type JSONMatcher struct{}

// Match checks if the actual response matches the expected pattern
func (m JSONMatcher) Match(pattern, actual []byte) MatchResult {
	var patternObj, actualObj interface{}

	// Parse pattern
	if err := json.Unmarshal(pattern, &patternObj); err != nil {
		return MatchResult{
			Success:  false,
			Message:  fmt.Sprintf("Invalid pattern JSON: %v", err),
			Expected: string(pattern),
			Actual:   string(actual),
		}
	}

	// Parse actual
	if err := json.Unmarshal(actual, &actualObj); err != nil {
		return MatchResult{
			Success:  false,
			Message:  fmt.Sprintf("Invalid actual JSON: %v", err),
			Expected: string(pattern),
			Actual:   string(actual),
		}
	}

	// Perform deep match with pattern support
	result := deepMatch(patternObj, actualObj)
	if !result.Success {
		// Add the full objects for better context
		result.Expected = string(pattern)
		result.Actual = string(actual)
	}

	return result
}

// deepMatch recursively matches the pattern against the actual value
func deepMatch(pattern, actual interface{}) MatchResult {
	// Handle nil cases
	if pattern == nil {
		if actual == nil {
			return MatchResult{Success: true}
		}
		return MatchResult{
			Success: false,
			Message: "Expected nil, got non-nil value",
		}
	}
	if actual == nil {
		return MatchResult{
			Success: false,
			Message: "Expected non-nil value, got nil",
		}
	}

	// Check types
	patternType := reflect.TypeOf(pattern)
	actualType := reflect.TypeOf(actual)

	// Handle special string patterns
	if patternType.Kind() == reflect.String {
		patternStr := pattern.(string)

		// If the pattern is a special pattern like "{{any}}" or a regex with //, handle it
		if isSpecialPattern(patternStr) {
			return matchSpecialPattern(patternStr, actual)
		}
	}

	// If types don't match and it's not a special case, fail
	if patternType != actualType {
		return MatchResult{
			Success: false,
			Message: fmt.Sprintf("Type mismatch: expected %v, got %v", patternType, actualType),
		}
	}

	// Handle different types
	switch patternType.Kind() {
	case reflect.Map:
		return matchMaps(pattern.(map[string]interface{}), actual.(map[string]interface{}))
	case reflect.Slice:
		return matchSlices(pattern.([]interface{}), actual.([]interface{}))
	case reflect.String, reflect.Float64, reflect.Bool:
		if pattern == actual {
			return MatchResult{Success: true}
		}
		return MatchResult{
			Success: false,
			Message: fmt.Sprintf("Value mismatch: expected %v, got %v", pattern, actual),
		}
	default:
		// For any other type, use direct equality
		if pattern == actual {
			return MatchResult{Success: true}
		}
		return MatchResult{
			Success: false,
			Message: fmt.Sprintf("Value mismatch: expected %v, got %v", pattern, actual),
		}
	}
}

// matchMaps matches two maps recursively
func matchMaps(pattern, actual map[string]interface{}) MatchResult {
	// If the pattern has special key "{{partial}}", it means we only check for keys in the pattern
	isPartial := false
	if _, exists := pattern["{{partial}}"]; exists {
		isPartial = true
		delete(pattern, "{{partial}}")
	}

	// Check that all keys in pattern exist in actual
	for key, patternValue := range pattern {
		actualValue, exists := actual[key]
		if !exists {
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Missing key in actual: %s", key),
			}
		}

		// Recursively match values
		result := deepMatch(patternValue, actualValue)
		if !result.Success {
			// Add key context to the error message
			result.Message = fmt.Sprintf("At key '%s': %s", key, result.Message)
			return result
		}
	}

	// If not partial, check that actual doesn't have extra keys
	if !isPartial {
		for key := range actual {
			if _, exists := pattern[key]; !exists {
				return MatchResult{
					Success: false,
					Message: fmt.Sprintf("Extra key in actual: %s", key),
				}
			}
		}
	}

	return MatchResult{Success: true}
}

// matchSlices matches two slices recursively
func matchSlices(pattern, actual []interface{}) MatchResult {
	// If the pattern has a single item with "{{items}}", it's a template for all items
	if len(pattern) == 1 {
		if patternMap, ok := pattern[0].(map[string]interface{}); ok {
			if _, hasItemsToken := patternMap["{{items}}"]; hasItemsToken {
				// This is an items template
				delete(patternMap, "{{items}}")
				for i, actualItem := range actual {
					result := deepMatch(patternMap, actualItem)
					if !result.Success {
						result.Message = fmt.Sprintf("Item %d: %s", i, result.Message)
						return result
					}
				}
				return MatchResult{Success: true}
			}
		}
	}

	// Regular slice matching
	if len(pattern) != len(actual) {
		return MatchResult{
			Success: false,
			Message: fmt.Sprintf("Array length mismatch: expected %d, got %d", len(pattern), len(actual)),
		}
	}

	for i, patternItem := range pattern {
		result := deepMatch(patternItem, actual[i])
		if !result.Success {
			result.Message = fmt.Sprintf("Array item %d: %s", i, result.Message)
			return result
		}
	}

	return MatchResult{Success: true}
}

// isSpecialPattern checks if a string is a special pattern
func isSpecialPattern(pattern string) bool {
	// Check for {{...}} patterns
	if strings.HasPrefix(pattern, "{{") && strings.HasSuffix(pattern, "}}") {
		return true
	}

	// Check for regex patterns /regex/
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		return true
	}

	return false
}

// matchSpecialPattern matches a special pattern against a value
func matchSpecialPattern(pattern string, actual interface{}) MatchResult {
	// Handle {{...}} patterns
	if strings.HasPrefix(pattern, "{{") && strings.HasSuffix(pattern, "}}") {
		token := pattern[2 : len(pattern)-2]

		switch token {
		case "any", "*":
			// Matches anything
			return MatchResult{Success: true}
		case "string":
			if _, ok := actual.(string); ok {
				return MatchResult{Success: true}
			}
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Expected string, got %T", actual),
			}
		case "number":
			if _, ok := actual.(float64); ok {
				return MatchResult{Success: true}
			}
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Expected number, got %T", actual),
			}
		case "boolean":
			if _, ok := actual.(bool); ok {
				return MatchResult{Success: true}
			}
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Expected boolean, got %T", actual),
			}
		case "object":
			if _, ok := actual.(map[string]interface{}); ok {
				return MatchResult{Success: true}
			}
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Expected object, got %T", actual),
			}
		case "array":
			if _, ok := actual.([]interface{}); ok {
				return MatchResult{Success: true}
			}
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Expected array, got %T", actual),
			}
		default:
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Unknown pattern token: %s", token),
			}
		}
	}

	// Handle regex patterns /regex/
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		regexStr := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile(regexStr)
		if err != nil {
			return MatchResult{
				Success: false,
				Message: fmt.Sprintf("Invalid regex: %v", err),
			}
		}

		// Convert actual to string if possible
		var actualStr string
		switch v := actual.(type) {
		case string:
			actualStr = v
		case float64:
			actualStr = fmt.Sprintf("%g", v)
		case bool:
			actualStr = fmt.Sprintf("%t", v)
		default:
			// Convert to JSON
			data, err := json.Marshal(actual)
			if err != nil {
				return MatchResult{
					Success: false,
					Message: fmt.Sprintf("Cannot convert %T to string for regex matching", actual),
				}
			}
			actualStr = string(data)
		}

		if re.MatchString(actualStr) {
			return MatchResult{Success: true}
		}
		return MatchResult{
			Success: false,
			Message: fmt.Sprintf("Regex %s did not match %s", regexStr, actualStr),
		}
	}

	// If it's not a recognized special pattern, treat it as literal
	if actual == pattern {
		return MatchResult{Success: true}
	}
	return MatchResult{
		Success: false,
		Message: fmt.Sprintf("Value mismatch: expected %v, got %v", pattern, actual),
	}
}
