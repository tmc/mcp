// Package errors provides a common error handling framework with standardized error codes
// and messaging for all MCP tools. It follows the Russ Cox coding style and provides
// structured error handling with context, wrapping, and categorization.
package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents a standardized error code.
type ErrorCode string

// Standard error codes following MCP specification and common patterns.
const (
	// General error codes
	CodeUnknown           ErrorCode = "unknown"
	CodeInvalidArgument   ErrorCode = "invalid_argument"
	CodeNotFound          ErrorCode = "not_found"
	CodeAlreadyExists     ErrorCode = "already_exists"
	CodePermissionDenied  ErrorCode = "permission_denied"
	CodeResourceExhausted ErrorCode = "resource_exhausted"
	CodeFailedPrecondition ErrorCode = "failed_precondition"
	CodeAborted           ErrorCode = "aborted"
	CodeOutOfRange        ErrorCode = "out_of_range"
	CodeUnimplemented     ErrorCode = "unimplemented"
	CodeInternal          ErrorCode = "internal"
	CodeUnavailable       ErrorCode = "unavailable"
	CodeDeadlineExceeded  ErrorCode = "deadline_exceeded"
	CodeCancelled         ErrorCode = "cancelled"
	
	// MCP-specific error codes
	CodeProtocol          ErrorCode = "protocol"
	CodeTransport         ErrorCode = "transport"
	CodeSerialization     ErrorCode = "serialization"
	CodeAuthentication    ErrorCode = "authentication"
	CodeAuthorization     ErrorCode = "authorization"
	CodeRateLimit         ErrorCode = "rate_limit"
	CodeQuotaExceeded     ErrorCode = "quota_exceeded"
	CodeConfiguration     ErrorCode = "configuration"
	CodePlugin            ErrorCode = "plugin"
	CodeTool              ErrorCode = "tool"
	CodeResource          ErrorCode = "resource"
	CodePrompt            ErrorCode = "prompt"
	
	// Tool-specific error codes
	CodeValidation        ErrorCode = "validation"
	CodeConversion        ErrorCode = "conversion"
	CodeFormatting        ErrorCode = "formatting"
	CodeParsing           ErrorCode = "parsing"
	CodeExecution         ErrorCode = "execution"
	CodeTimeout           ErrorCode = "timeout"
	CodeConnection        ErrorCode = "connection"
	CodeNetwork           ErrorCode = "network"
	CodeFileSystem        ErrorCode = "filesystem"
	CodeDatabase          ErrorCode = "database"
)

// ErrorCategory represents the category of an error.
type ErrorCategory string

const (
	CategoryClient     ErrorCategory = "client"
	CategoryServer     ErrorCategory = "server"
	CategoryTransport  ErrorCategory = "transport"
	CategoryProtocol   ErrorCategory = "protocol"
	CategoryTool       ErrorCategory = "tool"
	CategorySystem     ErrorCategory = "system"
	CategoryExternal   ErrorCategory = "external"
)

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// Error represents a structured error with additional context.
type Error struct {
	// Error code
	Code ErrorCode `json:"code"`
	
	// Error message
	Message string `json:"message"`
	
	// Error details
	Details map[string]interface{} `json:"details,omitempty"`
	
	// Error category
	Category ErrorCategory `json:"category"`
	
	// Error severity
	Severity ErrorSeverity `json:"severity"`
	
	// Timestamp when error occurred
	Timestamp time.Time `json:"timestamp"`
	
	// Context information
	Context map[string]interface{} `json:"context,omitempty"`
	
	// Stack trace
	Stack []StackFrame `json:"stack,omitempty"`
	
	// Wrapped error
	Wrapped error `json:"-"`
	
	// Retry information
	Retry RetryInfo `json:"retry,omitempty"`
	
	// User-friendly message
	UserMessage string `json:"user_message,omitempty"`
}

// StackFrame represents a frame in the call stack.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// RetryInfo provides information about retry behavior.
type RetryInfo struct {
	// Whether the error is retryable
	Retryable bool `json:"retryable"`
	
	// Suggested retry delay
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	
	// Maximum retry attempts
	MaxAttempts int `json:"max_attempts,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	return e.Wrapped
}

// Is checks if the error matches the target error.
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return false
}

// As extracts the error as the target type.
func (e *Error) As(target interface{}) bool {
	if t, ok := target.(**Error); ok {
		*t = e
		return true
	}
	return false
}

// String returns a string representation of the error.
func (e *Error) String() string {
	return e.Error()
}

// JSON returns a JSON representation of the error.
func (e *Error) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// WithDetail adds a detail to the error.
func (e *Error) WithDetail(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithContext adds context to the error.
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithUserMessage sets a user-friendly message.
func (e *Error) WithUserMessage(message string) *Error {
	e.UserMessage = message
	return e
}

// WithRetry sets retry information.
func (e *Error) WithRetry(retryable bool, retryAfter time.Duration, maxAttempts int) *Error {
	e.Retry = RetryInfo{
		Retryable:   retryable,
		RetryAfter:  retryAfter,
		MaxAttempts: maxAttempts,
	}
	return e
}

// IsRetryable returns whether the error is retryable.
func (e *Error) IsRetryable() bool {
	return e.Retry.Retryable
}

// IsTemporary returns whether the error is temporary.
func (e *Error) IsTemporary() bool {
	return e.IsRetryable() && e.Severity != SeverityCritical
}

// IsTimeoutError returns whether the error is a timeout error.
func (e *Error) IsTimeoutError() bool {
	return e.Code == CodeTimeout || e.Code == CodeDeadlineExceeded
}

// IsCancellationError returns whether the error is a cancellation error.
func (e *Error) IsCancellationError() bool {
	return e.Code == CodeCancelled
}

// IsClientError returns whether the error is a client error.
func (e *Error) IsClientError() bool {
	return e.Category == CategoryClient
}

// IsServerError returns whether the error is a server error.
func (e *Error) IsServerError() bool {
	return e.Category == CategoryServer
}

// New creates a new error with the given code and message.
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Category:  inferCategory(code),
		Severity:  inferSeverity(code),
		Timestamp: time.Now(),
		Stack:     captureStack(),
	}
}

// Newf creates a new error with formatted message.
func Newf(code ErrorCode, format string, args ...interface{}) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, code ErrorCode, message string) *Error {
	if err == nil {
		return nil
	}
	
	// If already a structured error, wrap it
	if e, ok := err.(*Error); ok {
		return &Error{
			Code:      code,
			Message:   message,
			Category:  inferCategory(code),
			Severity:  inferSeverity(code),
			Timestamp: time.Now(),
			Stack:     captureStack(),
			Wrapped:   e,
		}
	}
	
	// Wrap standard error
	return &Error{
		Code:      code,
		Message:   message,
		Category:  inferCategory(code),
		Severity:  inferSeverity(code),
		Timestamp: time.Now(),
		Stack:     captureStack(),
		Wrapped:   err,
	}
}

// Wrapf wraps an existing error with formatted message.
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *Error {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// FromError converts a standard error to a structured error.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	
	// If already a structured error, return as-is
	if e, ok := err.(*Error); ok {
		return e
	}
	
	// Convert standard error
	return &Error{
		Code:      CodeUnknown,
		Message:   err.Error(),
		Category:  CategorySystem,
		Severity:  SeverityMedium,
		Timestamp: time.Now(),
		Stack:     captureStack(),
		Wrapped:   err,
	}
}

// FromContext creates an error from context cancellation.
func FromContext(ctx context.Context) *Error {
	if ctx.Err() == nil {
		return nil
	}
	
	if ctx.Err() == context.Canceled {
		return New(CodeCancelled, "operation was cancelled")
	}
	
	if ctx.Err() == context.DeadlineExceeded {
		return New(CodeDeadlineExceeded, "operation deadline exceeded")
	}
	
	return Wrap(ctx.Err(), CodeUnknown, "context error")
}

// Chain represents a chain of errors.
type Chain struct {
	errors []*Error
}

// NewChain creates a new error chain.
func NewChain() *Chain {
	return &Chain{}
}

// Add adds an error to the chain.
func (c *Chain) Add(err error) *Chain {
	if err == nil {
		return c
	}
	
	if e, ok := err.(*Error); ok {
		c.errors = append(c.errors, e)
	} else {
		c.errors = append(c.errors, FromError(err))
	}
	
	return c
}

// HasErrors returns whether the chain has any errors.
func (c *Chain) HasErrors() bool {
	return len(c.errors) > 0
}

// Errors returns all errors in the chain.
func (c *Chain) Errors() []*Error {
	return c.errors
}

// Error returns a combined error message.
func (c *Chain) Error() string {
	if len(c.errors) == 0 {
		return ""
	}
	
	if len(c.errors) == 1 {
		return c.errors[0].Error()
	}
	
	var messages []string
	for _, err := range c.errors {
		messages = append(messages, err.Error())
	}
	
	return strings.Join(messages, "; ")
}

// First returns the first error in the chain.
func (c *Chain) First() *Error {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[0]
}

// Last returns the last error in the chain.
func (c *Chain) Last() *Error {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[len(c.errors)-1]
}

// Handler represents an error handler.
type Handler interface {
	HandleError(err *Error) error
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc func(*Error) error

// HandleError implements the Handler interface.
func (f HandlerFunc) HandleError(err *Error) error {
	return f(err)
}

// Registry manages error handlers.
type Registry struct {
	handlers map[ErrorCode][]Handler
}

// NewRegistry creates a new error handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[ErrorCode][]Handler),
	}
}

// Register registers an error handler for a specific error code.
func (r *Registry) Register(code ErrorCode, handler Handler) {
	r.handlers[code] = append(r.handlers[code], handler)
}

// Handle handles an error using registered handlers.
func (r *Registry) Handle(err *Error) error {
	if err == nil {
		return nil
	}
	
	// Get handlers for this error code
	handlers := r.handlers[err.Code]
	
	// Try generic handlers
	if len(handlers) == 0 {
		handlers = r.handlers[CodeUnknown]
	}
	
	// Execute handlers
	for _, handler := range handlers {
		if handlerErr := handler.HandleError(err); handlerErr != nil {
			return handlerErr
		}
	}
	
	return nil
}

// Helper functions

// inferCategory infers the error category from the error code.
func inferCategory(code ErrorCode) ErrorCategory {
	switch code {
	case CodeProtocol:
		return CategoryProtocol
	case CodeTransport, CodeConnection, CodeNetwork:
		return CategoryTransport
	case CodeTool, CodeExecution:
		return CategoryTool
	case CodeInvalidArgument, CodeNotFound, CodeAlreadyExists:
		return CategoryClient
	case CodeInternal, CodeUnimplemented, CodeUnavailable:
		return CategoryServer
	case CodeFileSystem, CodeDatabase:
		return CategorySystem
	default:
		return CategorySystem
	}
}

// inferSeverity infers the error severity from the error code.
func inferSeverity(code ErrorCode) ErrorSeverity {
	switch code {
	case CodeInternal, CodeUnavailable:
		return SeverityCritical
	case CodePermissionDenied, CodeAuthentication, CodeAuthorization:
		return SeverityHigh
	case CodeInvalidArgument, CodeNotFound, CodeValidation:
		return SeverityMedium
	case CodeCancelled, CodeTimeout:
		return SeverityLow
	default:
		return SeverityMedium
	}
}

// captureStack captures the current call stack.
func captureStack() []StackFrame {
	var stack []StackFrame
	
	// Skip the first few frames (this function and error creation)
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		stack = append(stack, StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})
	}
	
	return stack
}

// Common error constructors

// InvalidArgument creates an invalid argument error.
func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message)
}

// NotFound creates a not found error.
func NotFound(message string) *Error {
	return New(CodeNotFound, message)
}

// AlreadyExists creates an already exists error.
func AlreadyExists(message string) *Error {
	return New(CodeAlreadyExists, message)
}

// PermissionDenied creates a permission denied error.
func PermissionDenied(message string) *Error {
	return New(CodePermissionDenied, message)
}

// Internal creates an internal error.
func Internal(message string) *Error {
	return New(CodeInternal, message)
}

// Unavailable creates an unavailable error.
func Unavailable(message string) *Error {
	return New(CodeUnavailable, message)
}

// Timeout creates a timeout error.
func Timeout(message string) *Error {
	return New(CodeTimeout, message)
}

// Cancelled creates a cancelled error.
func Cancelled(message string) *Error {
	return New(CodeCancelled, message)
}

// Protocol creates a protocol error.
func Protocol(message string) *Error {
	return New(CodeProtocol, message)
}

// Transport creates a transport error.
func Transport(message string) *Error {
	return New(CodeTransport, message)
}

// Configuration creates a configuration error.
func Configuration(message string) *Error {
	return New(CodeConfiguration, message)
}

// Validation creates a validation error.
func Validation(message string) *Error {
	return New(CodeValidation, message)
}

// Default error registry
var defaultRegistry = NewRegistry()

// RegisterHandler registers a handler with the default registry.
func RegisterHandler(code ErrorCode, handler Handler) {
	defaultRegistry.Register(code, handler)
}

// HandleError handles an error using the default registry.
func HandleError(err *Error) error {
	return defaultRegistry.Handle(err)
}

// Is checks if an error is of a specific type.
func Is(err error, code ErrorCode) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}

// As extracts a structured error from an error.
func As(err error) (*Error, bool) {
	if e, ok := err.(*Error); ok {
		return e, true
	}
	return nil, false
}

// Code returns the error code of an error.
func Code(err error) ErrorCode {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return CodeUnknown
}

// Category returns the error category of an error.
func Category(err error) ErrorCategory {
	if e, ok := err.(*Error); ok {
		return e.Category
	}
	return CategorySystem
}

// Severity returns the error severity of an error.
func Severity(err error) ErrorSeverity {
	if e, ok := err.(*Error); ok {
		return e.Severity
	}
	return SeverityMedium
}