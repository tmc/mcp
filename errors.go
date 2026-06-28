package mcp

import (
	"fmt"
	"os"
	"strings"
)

// ErrorVerbosityMode controls how much detail is included in error messages
type ErrorVerbosityMode string

const (
	// ErrorVerbosityProduction provides minimal error details for security
	ErrorVerbosityProduction ErrorVerbosityMode = "production"
	// ErrorVerbosityDevelopment provides detailed error information for debugging
	ErrorVerbosityDevelopment ErrorVerbosityMode = "development"
)

// errorVerbosity tracks the current error verbosity mode
var errorVerbosity = ErrorVerbosityProduction

func init() {
	// Check environment variable for verbosity mode
	if mode := os.Getenv("MCP_ERROR_VERBOSITY"); mode != "" {
		if mode == "development" || mode == "dev" {
			errorVerbosity = ErrorVerbosityDevelopment
		}
	}
}

// SetErrorVerbosity sets the global error verbosity mode
func SetErrorVerbosity(mode ErrorVerbosityMode) {
	errorVerbosity = mode
}

// GetErrorVerbosity returns the current error verbosity mode
func GetErrorVerbosity() ErrorVerbosityMode {
	return errorVerbosity
}

// SanitizeError sanitizes an error message based on verbosity mode
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	if errorVerbosity == ErrorVerbosityDevelopment {
		// In development, return full error details
		return err
	}

	// In production, sanitize sensitive information
	msg := err.Error()

	// Remove file paths
	msg = removePaths(msg)

	// Remove specific error details that might leak implementation
	msg = removeInternalDetails(msg)

	// Return generic error if too much detail remains
	if containsSensitiveInfo(msg) {
		return fmt.Errorf("internal server error")
	}

	return fmt.Errorf("%s", msg)
}

// SanitizeErrorMessage sanitizes an error message string
func SanitizeErrorMessage(message string) string {
	if errorVerbosity == ErrorVerbosityDevelopment {
		return message
	}

	// Remove file paths
	message = removePaths(message)

	// Remove internal details
	message = removeInternalDetails(message)

	// Check for sensitive info
	if containsSensitiveInfo(message) {
		return "internal server error"
	}

	return message
}

// removePaths removes file system paths from error messages
func removePaths(msg string) string {
	// Remove absolute paths
	words := strings.Fields(msg)
	for i, word := range words {
		if strings.Contains(word, "/") && (strings.HasPrefix(word, "/") || strings.Contains(word, ":/")) {
			words[i] = "[redacted]"
		}
	}
	return strings.Join(words, " ")
}

// removeInternalDetails removes internal implementation details
func removeInternalDetails(msg string) string {
	// Patterns to remove or replace
	replacements := map[string]string{
		"panic:":          "error:",
		"goroutine":       "",
		"runtime.":        "",
		".go:":            "",
		"database/sql:":   "database error",
		"crypto/":         "cryptographic error",
		"internal error:": "error:",
		"unexpected type": "invalid data",
	}

	for old, new := range replacements {
		msg = strings.ReplaceAll(msg, old, new)
	}

	return strings.TrimSpace(msg)
}

// containsSensitiveInfo checks if message contains sensitive patterns
func containsSensitiveInfo(msg string) bool {
	sensitivePatterns := []string{
		"password",
		"token",
		"secret",
		"key:",
		"auth:",
		"credential",
		"session",
		"sql:",
		"query:",
		"connection string",
		"database",
	}

	msgLower := strings.ToLower(msg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(msgLower, pattern) {
			return true
		}
	}

	return false
}

// NewSanitizedError creates a new error with sanitization applied
func NewSanitizedError(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	return SanitizeError(err)
}

// WrapSanitizedError wraps an error with context and sanitization
func WrapSanitizedError(err error, message string) error {
	if err == nil {
		return nil
	}

	wrapped := fmt.Errorf("%s: %w", message, err)
	return SanitizeError(wrapped)
}
