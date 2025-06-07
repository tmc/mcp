package sdk2

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Common errors following stdlib patterns like os.ErrNotExist, io.EOF, etc.
var (
	// Connection errors
	ErrClosed      = errors.New("mcp: connection closed")
	ErrTimeout     = errors.New("mcp: operation timed out")
	ErrConnRefused = errors.New("mcp: connection refused")
	ErrConnReset   = errors.New("mcp: connection reset")

	// Protocol errors
	ErrHandshake    = errors.New("mcp: handshake failed")
	ErrProtocol     = errors.New("mcp: protocol error")
	ErrInvalidData  = errors.New("mcp: invalid data")
	ErrNotSupported = errors.New("mcp: not supported")

	// Request/Response errors
	ErrToolNotFound = errors.New("mcp: tool not found")
	ErrBadRequest   = errors.New("mcp: bad request")
	ErrUnauthorized = errors.New("mcp: unauthorized")
	ErrForbidden    = errors.New("mcp: forbidden")

	// Validation errors
	ErrInvalid  = errors.New("mcp: invalid")
	ErrRequired = errors.New("mcp: required field missing")
	ErrTooLarge = errors.New("mcp: data too large")
	ErrTooSmall = errors.New("mcp: data too small")

	// Client/Server lifecycle errors
	ErrClientClosed = errors.New("mcp: client closed")
	ErrServerClosed = errors.New("mcp: server closed")
)

// MCPError represents an MCP-specific error with context preservation.
// It follows stdlib error wrapping patterns and provides rich error information.
type MCPError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
	Cause   error           `json:"-"` // Underlying error for wrapping
	Op      string          `json:"-"` // Operation that failed
}

// Error implements the error interface.
func (e *MCPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("MCP %s error %d: %s: %v", e.Op, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("MCP %s error %d: %s", e.Op, e.Code, e.Message)
}

// Unwrap implements error unwrapping for Go 1.13+ error handling.
func (e *MCPError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for Go 1.13+ error handling.
func (e *MCPError) Is(target error) bool {
	t, ok := target.(*MCPError)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Op == t.Op
}

// NewError creates a new MCPError with context.
func NewError(op string, code int, message string, cause error) *MCPError {
	return &MCPError{
		Op:      op,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewErrorWithData creates a new MCPError with additional data.
func NewErrorWithData(op string, code int, message string, data interface{}, cause error) *MCPError {
	var dataBytes json.RawMessage
	if data != nil {
		if bytes, err := json.Marshal(data); err == nil {
			dataBytes = bytes
		}
	}

	return &MCPError{
		Op:      op,
		Code:    code,
		Message: message,
		Data:    dataBytes,
		Cause:   cause,
	}
}

// ValidationError represents a validation failure with detailed context.
type ValidationError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, rule, message string, value any) *ValidationError {
	return &ValidationError{
		Field:   field,
		Rule:    rule,
		Message: message,
		Value:   value,
	}
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("validation failed: %d errors", len(e))
}

// Add adds a validation error to the collection.
func (e *ValidationErrors) Add(field, rule, message string, value any) {
	*e = append(*e, ValidationError{
		Field:   field,
		Rule:    rule,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// TimeoutError represents a timeout error.
type TimeoutError struct {
	Op      string
	Timeout string
	Cause   error
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s timeout after %s: %v", e.Op, e.Timeout, e.Cause)
	}
	return fmt.Sprintf("%s timeout after %s", e.Op, e.Timeout)
}

// Unwrap implements error unwrapping.
func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// Timeout returns true, indicating this is a timeout error.
func (e *TimeoutError) Timeout() bool {
	return true
}

// ConnectionError represents a connection-related error.
type ConnectionError struct {
	Op      string
	Network string
	Address string
	Cause   error
}

// Error implements the error interface.
func (e *ConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s connection to %s://%s failed: %v", e.Op, e.Network, e.Address, e.Cause)
	}
	return fmt.Sprintf("%s connection to %s://%s failed", e.Op, e.Network, e.Address)
}

// Unwrap implements error unwrapping.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// Common error constructors following stdlib patterns

// WrapError wraps an error with additional context.
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	if mcpErr, ok := err.(*MCPError); ok {
		// If it's already an MCPError, just update the operation
		mcpErr.Op = op
		return mcpErr
	}
	return &MCPError{
		Op:      op,
		Code:    StatusInternalServerError,
		Message: "internal error",
		Cause:   err,
	}
}

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	type timeout interface {
		Timeout() bool
	}

	if t, ok := err.(timeout); ok {
		return t.Timeout()
	}

	// Check for wrapped timeout errors
	if te, ok := err.(*TimeoutError); ok {
		return te.Timeout()
	}

	return false
}

// IsRetryable determines if an error is retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if IsTimeout(err) {
		return true
	}

	if _, ok := err.(*ConnectionError); ok {
		return true
	}

	// Check for MCP errors with retryable codes
	if mcpErr, ok := err.(*MCPError); ok {
		switch mcpErr.Code {
		case StatusInternalServerError, StatusServiceUnavailable, StatusRequestTimeout:
			return true
		}
	}

	return false
}

// Error status codes
const (
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusRequestTimeout      = 408
	StatusConflict            = 409
	StatusTooManyRequests     = 429
	StatusInternalServerError = 500
	StatusNotImplemented      = 501
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
	StatusGatewayTimeout      = 504
)

var statusText = map[int]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusUnauthorized:        "Unauthorized",
	StatusForbidden:           "Forbidden",
	StatusNotFound:            "Not Found",
	StatusMethodNotAllowed:    "Method Not Allowed",
	StatusRequestTimeout:      "Request Timeout",
	StatusConflict:            "Conflict",
	StatusTooManyRequests:     "Too Many Requests",
	StatusInternalServerError: "Internal Server Error",
	StatusNotImplemented:      "Not Implemented",
	StatusBadGateway:          "Bad Gateway",
	StatusServiceUnavailable:  "Service Unavailable",
	StatusGatewayTimeout:      "Gateway Timeout",
}

// StatusText returns a text for the MCP status code.
func StatusText(code int) string {
	if text, ok := statusText[code]; ok {
		return text
	}
	return fmt.Sprintf("Unknown Status Code %d", code)
}

// Helper constructors following stdlib patterns

// Errorf creates a formatted error similar to fmt.Errorf
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf("mcp: "+format, args...)
}

// TimeoutErrorf creates a timeout error with formatted message
func TimeoutErrorf(op, timeout string, args ...interface{}) error {
	return &TimeoutError{
		Op:      op,
		Timeout: timeout,
		Cause:   fmt.Errorf(args[0].(string), args[1:]...),
	}
}

// ConnErrorf creates a connection error with formatted message
func ConnErrorf(op, network, address, format string, args ...interface{}) error {
	return &ConnectionError{
		Op:      op,
		Network: network,
		Address: address,
		Cause:   fmt.Errorf(format, args...),
	}
}

// MustParseTool is like ParseTool but panics on error (like template.Must)
func MustParseTool(data []byte) Tool {
	tool, err := ParseTool(data)
	if err != nil {
		panic(err)
	}
	return tool
}

// ParseTool parses tool definition from JSON (like url.Parse)
func ParseTool(data []byte) (Tool, error) {
	var tool Tool
	if err := json.Unmarshal(data, &tool); err != nil {
		return Tool{}, fmt.Errorf("parse tool: %w", err)
	}
	if err := tool.Validate(); err != nil {
		return Tool{}, fmt.Errorf("invalid tool: %w", err)
	}
	return tool, nil
}

// Constructor helpers for common error types

// NewConnError creates a new connection error following net.OpError patterns
func NewConnError(op, network, address string, err error) error {
	return &ConnectionError{
		Op:      op,
		Network: network,
		Address: address,
		Cause:   err,
	}
}

// NewProtocolError creates a new protocol error
func NewProtocolError(phase, message string, cause error) error {
	return &ProtocolError{
		Phase:   phase,
		Message: message,
		Cause:   cause,
	}
}

// NewMCPError creates a new MCP error with all fields
func NewMCPError(op, method string, code int, message string, cause error) error {
	return &MCPError{
		Op:      op,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ProtocolError represents a protocol-level error
type ProtocolError struct {
	Phase   string // "handshake", "initialize", "request", "response"
	Message string
	Cause   error
}

func (e *ProtocolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("mcp protocol error in %s: %s: %v", e.Phase, e.Message, e.Cause)
	}
	return fmt.Sprintf("mcp protocol error in %s: %s", e.Phase, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	return e.Cause
}
