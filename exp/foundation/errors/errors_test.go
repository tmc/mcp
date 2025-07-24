package errors

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestErrorCreation(t *testing.T) {
	err := New(CodeInvalidArgument, "test error")
	
	if err.Code != CodeInvalidArgument {
		t.Errorf("Expected code %s, got %s", CodeInvalidArgument, err.Code)
	}
	
	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got %s", err.Message)
	}
	
	if err.Category != CategoryClient {
		t.Errorf("Expected category %s, got %s", CategoryClient, err.Category)
	}
	
	if err.Severity != SeverityMedium {
		t.Errorf("Expected severity %s, got %s", SeverityMedium, err.Severity)
	}
	
	// Check that timestamp is set
	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	
	// Check that stack is captured
	if len(err.Stack) == 0 {
		t.Error("Expected stack trace to be captured")
	}
}

func TestErrorFormatting(t *testing.T) {
	err := Newf(CodeNotFound, "resource %s not found", "test-resource")
	
	expected := "[not_found] resource test-resource not found"
	if err.Error() != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, err.Error())
	}
}

func TestErrorWrapping(t *testing.T) {
	originalErr := New(CodeTimeout, "operation timed out")
	wrappedErr := Wrap(originalErr, CodeInternal, "failed to complete operation")
	
	if wrappedErr.Code != CodeInternal {
		t.Errorf("Expected code %s, got %s", CodeInternal, wrappedErr.Code)
	}
	
	if wrappedErr.Wrapped != originalErr {
		t.Error("Expected wrapped error to contain original error")
	}
	
	// Test unwrap
	if unwrapped := wrappedErr.Unwrap(); unwrapped != originalErr {
		t.Error("Unwrap did not return original error")
	}
}

func TestErrorDetails(t *testing.T) {
	err := New(CodeValidation, "validation failed")
	err.WithDetail("field", "email")
	err.WithDetail("value", "invalid@")
	
	if err.Details["field"] != "email" {
		t.Errorf("Expected detail field 'email', got %v", err.Details["field"])
	}
	
	if err.Details["value"] != "invalid@" {
		t.Errorf("Expected detail value 'invalid@', got %v", err.Details["value"])
	}
}

func TestErrorContext(t *testing.T) {
	err := New(CodePermissionDenied, "access denied")
	err.WithContext("user", "john.doe")
	err.WithContext("resource", "/api/admin")
	
	if err.Context["user"] != "john.doe" {
		t.Errorf("Expected context user 'john.doe', got %v", err.Context["user"])
	}
	
	if err.Context["resource"] != "/api/admin" {
		t.Errorf("Expected context resource '/api/admin', got %v", err.Context["resource"])
	}
}

func TestErrorRetry(t *testing.T) {
	err := New(CodeUnavailable, "service unavailable")
	err.WithRetry(true, 5*time.Second, 3)
	
	if !err.IsRetryable() {
		t.Error("Expected error to be retryable")
	}
	
	if err.Retry.RetryAfter != 5*time.Second {
		t.Errorf("Expected retry after 5s, got %v", err.Retry.RetryAfter)
	}
	
	if err.Retry.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", err.Retry.MaxAttempts)
	}
}

func TestErrorUserMessage(t *testing.T) {
	err := New(CodeInternal, "database connection failed")
	err.WithUserMessage("Sorry, we're experiencing technical difficulties. Please try again later.")
	
	if !strings.Contains(err.UserMessage, "technical difficulties") {
		t.Errorf("Expected user message to contain 'technical difficulties', got %s", err.UserMessage)
	}
}

func TestErrorJSON(t *testing.T) {
	err := New(CodeNotFound, "resource not found")
	err.WithDetail("id", "12345")
	err.WithContext("path", "/api/resources/12345")
	
	data, jsonErr := err.JSON()
	if jsonErr != nil {
		t.Fatalf("Failed to marshal error to JSON: %v", jsonErr)
	}
	
	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal(data, &parsed); jsonErr != nil {
		t.Fatalf("Failed to unmarshal error JSON: %v", jsonErr)
	}
	
	if parsed["code"] != string(CodeNotFound) {
		t.Errorf("Expected code 'not_found', got %v", parsed["code"])
	}
	
	if parsed["message"] != "resource not found" {
		t.Errorf("Expected message 'resource not found', got %v", parsed["message"])
	}
}

func TestErrorIs(t *testing.T) {
	err1 := New(CodeTimeout, "timeout error")
	err2 := New(CodeTimeout, "another timeout error")
	err3 := New(CodeCancelled, "cancelled error")
	
	if !err1.Is(err2) {
		t.Error("Expected errors with same code to match")
	}
	
	if err1.Is(err3) {
		t.Error("Expected errors with different codes not to match")
	}
	
	// Test helper function
	if !Is(err1, CodeTimeout) {
		t.Error("Expected Is helper to match error code")
	}
}

func TestErrorAs(t *testing.T) {
	err := New(CodeValidation, "validation error")
	
	var target *Error
	if !err.As(&target) {
		t.Error("Expected As to extract error")
	}
	
	if target != err {
		t.Error("Expected As to set target to error")
	}
	
	// Test helper function
	extracted, ok := As(err)
	if !ok {
		t.Error("Expected As helper to extract error")
	}
	
	if extracted != err {
		t.Error("Expected As helper to return error")
	}
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		err      *Error
		isTemp   bool
		isTimeout bool
		isCancel bool
		isClient bool
		isServer bool
	}{
		{
			err:       New(CodeTimeout, "timeout"),
			isTemp:    true,
			isTimeout: true,
		},
		{
			err:      New(CodeCancelled, "cancelled"),
			isCancel: true,
		},
		{
			err:      New(CodeInvalidArgument, "invalid"),
			isClient: true,
		},
		{
			err:      New(CodeInternal, "internal"),
			isServer: true,
		},
		{
			err:      New(CodeUnavailable, "unavailable").WithRetry(true, 0, 0),
			isTemp:   true,
			isServer: true,
		},
	}
	
	for _, test := range tests {
		if test.err.IsTemporary() != test.isTemp {
			t.Errorf("Expected IsTemporary=%v for %s", test.isTemp, test.err.Code)
		}
		
		if test.err.IsTimeoutError() != test.isTimeout {
			t.Errorf("Expected IsTimeoutError=%v for %s", test.isTimeout, test.err.Code)
		}
		
		if test.err.IsCancellationError() != test.isCancel {
			t.Errorf("Expected IsCancellationError=%v for %s", test.isCancel, test.err.Code)
		}
		
		if test.err.IsClientError() != test.isClient {
			t.Errorf("Expected IsClientError=%v for %s", test.isClient, test.err.Code)
		}
		
		if test.err.IsServerError() != test.isServer {
			t.Errorf("Expected IsServerError=%v for %s", test.isServer, test.err.Code)
		}
	}
}

func TestErrorChain(t *testing.T) {
	chain := NewChain()
	
	if chain.HasErrors() {
		t.Error("Expected new chain to have no errors")
	}
	
	// Add errors
	chain.Add(New(CodeValidation, "validation error 1"))
	chain.Add(New(CodeValidation, "validation error 2"))
	chain.Add(nil) // Should be ignored
	
	if !chain.HasErrors() {
		t.Error("Expected chain to have errors")
	}
	
	if len(chain.Errors()) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(chain.Errors()))
	}
	
	// Test first and last
	first := chain.First()
	if first == nil || first.Message != "validation error 1" {
		t.Error("Expected first error to be 'validation error 1'")
	}
	
	last := chain.Last()
	if last == nil || last.Message != "validation error 2" {
		t.Error("Expected last error to be 'validation error 2'")
	}
	
	// Test error string
	errString := chain.Error()
	if !strings.Contains(errString, "validation error 1") {
		t.Error("Expected error string to contain first error")
	}
	
	if !strings.Contains(errString, "validation error 2") {
		t.Error("Expected error string to contain second error")
	}
}

func TestErrorRegistry(t *testing.T) {
	registry := NewRegistry()
	
	handlerCalled := false
	handler := HandlerFunc(func(err *Error) error {
		handlerCalled = true
		return nil
	})
	
	registry.Register(CodeTimeout, handler)
	
	// Handle timeout error
	err := New(CodeTimeout, "timeout error")
	if handleErr := registry.Handle(err); handleErr != nil {
		t.Fatalf("Failed to handle error: %v", handleErr)
	}
	
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	
	// Handle error without registered handler
	handlerCalled = false
	err2 := New(CodeNotFound, "not found")
	if handleErr := registry.Handle(err2); handleErr != nil {
		t.Fatalf("Failed to handle error: %v", handleErr)
	}
	
	if handlerCalled {
		t.Error("Expected handler not to be called for different error code")
	}
}

func TestFromError(t *testing.T) {
	// Test with nil
	if err := FromError(nil); err != nil {
		t.Error("Expected FromError(nil) to return nil")
	}
	
	// Test with structured error
	structuredErr := New(CodeValidation, "validation error")
	if err := FromError(structuredErr); err != structuredErr {
		t.Error("Expected FromError to return structured error as-is")
	}
	
	// Test with standard error
	standardErr := context.DeadlineExceeded
	converted := FromError(standardErr)
	if converted.Code != CodeUnknown {
		t.Errorf("Expected code %s, got %s", CodeUnknown, converted.Code)
	}
	
	if converted.Wrapped != standardErr {
		t.Error("Expected wrapped error to be original error")
	}
}

func TestFromContext(t *testing.T) {
	// Test with active context
	ctx := context.Background()
	if err := FromContext(ctx); err != nil {
		t.Error("Expected FromContext with active context to return nil")
	}
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	err := FromContext(ctx)
	if err == nil {
		t.Fatal("Expected FromContext with cancelled context to return error")
	}
	
	if err.Code != CodeCancelled {
		t.Errorf("Expected code %s, got %s", CodeCancelled, err.Code)
	}
	
	// Test with deadline exceeded
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	time.Sleep(10 * time.Millisecond)
	
	err = FromContext(ctx)
	if err == nil {
		t.Fatal("Expected FromContext with deadline exceeded to return error")
	}
	
	if err.Code != CodeDeadlineExceeded {
		t.Errorf("Expected code %s, got %s", CodeDeadlineExceeded, err.Code)
	}
}

func TestCommonConstructors(t *testing.T) {
	tests := []struct {
		fn   func(string) *Error
		code ErrorCode
		msg  string
	}{
		{InvalidArgument, CodeInvalidArgument, "invalid"},
		{NotFound, CodeNotFound, "not found"},
		{AlreadyExists, CodeAlreadyExists, "exists"},
		{PermissionDenied, CodePermissionDenied, "denied"},
		{Internal, CodeInternal, "internal"},
		{Unavailable, CodeUnavailable, "unavailable"},
		{Timeout, CodeTimeout, "timeout"},
		{Cancelled, CodeCancelled, "cancelled"},
		{Protocol, CodeProtocol, "protocol"},
		{Transport, CodeTransport, "transport"},
		{Configuration, CodeConfiguration, "config"},
		{Validation, CodeValidation, "validation"},
	}
	
	for _, test := range tests {
		err := test.fn(test.msg)
		if err.Code != test.code {
			t.Errorf("Expected code %s, got %s", test.code, err.Code)
		}
		
		if err.Message != test.msg {
			t.Errorf("Expected message '%s', got '%s'", test.msg, err.Message)
		}
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected ErrorCategory
	}{
		{CodeProtocol, CategoryProtocol},
		{CodeTransport, CategoryTransport},
		{CodeConnection, CategoryTransport},
		{CodeNetwork, CategoryTransport},
		{CodeTool, CategoryTool},
		{CodeExecution, CategoryTool},
		{CodeInvalidArgument, CategoryClient},
		{CodeNotFound, CategoryClient},
		{CodeInternal, CategoryServer},
		{CodeUnimplemented, CategoryServer},
		{CodeFileSystem, CategorySystem},
		{CodeDatabase, CategorySystem},
		{CodeUnknown, CategorySystem},
	}
	
	for _, test := range tests {
		category := inferCategory(test.code)
		if category != test.expected {
			t.Errorf("Expected category %s for code %s, got %s", test.expected, test.code, category)
		}
	}
}

func TestInferSeverity(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected ErrorSeverity
	}{
		{CodeInternal, SeverityCritical},
		{CodeUnavailable, SeverityCritical},
		{CodePermissionDenied, SeverityHigh},
		{CodeAuthentication, SeverityHigh},
		{CodeAuthorization, SeverityHigh},
		{CodeInvalidArgument, SeverityMedium},
		{CodeNotFound, SeverityMedium},
		{CodeValidation, SeverityMedium},
		{CodeCancelled, SeverityLow},
		{CodeTimeout, SeverityLow},
		{CodeUnknown, SeverityMedium},
	}
	
	for _, test := range tests {
		severity := inferSeverity(test.code)
		if severity != test.expected {
			t.Errorf("Expected severity %s for code %s, got %s", test.expected, test.code, severity)
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	err := New(CodeValidation, "validation error")
	
	// Test Code helper
	if code := Code(err); code != CodeValidation {
		t.Errorf("Expected code %s, got %s", CodeValidation, code)
	}
	
	// Test Category helper
	if category := Category(err); category != err.Category {
		t.Errorf("Expected category %s, got %s", err.Category, category)
	}
	
	// Test Severity helper
	if severity := Severity(err); severity != err.Severity {
		t.Errorf("Expected severity %s, got %s", err.Severity, severity)
	}
	
	// Test with non-structured error
	standardErr := context.DeadlineExceeded
	
	if code := Code(standardErr); code != CodeUnknown {
		t.Errorf("Expected code %s for standard error, got %s", CodeUnknown, code)
	}
	
	if category := Category(standardErr); category != CategorySystem {
		t.Errorf("Expected category %s for standard error, got %s", CategorySystem, category)
	}
	
	if severity := Severity(standardErr); severity != SeverityMedium {
		t.Errorf("Expected severity %s for standard error, got %s", SeverityMedium, severity)
	}
}

func TestStackTrace(t *testing.T) {
	err := New(CodeInternal, "test error")
	
	if len(err.Stack) == 0 {
		t.Fatal("Expected stack trace to be captured")
	}
	
	// Check first frame
	frame := err.Stack[0]
	if frame.Function == "" {
		t.Error("Expected function name in stack frame")
	}
	
	if frame.File == "" {
		t.Error("Expected file name in stack frame")
	}
	
	if frame.Line == 0 {
		t.Error("Expected line number in stack frame")
	}
}

func TestDefaultRegistry(t *testing.T) {
	handlerCalled := false
	handler := HandlerFunc(func(err *Error) error {
		handlerCalled = true
		return nil
	})
	
	RegisterHandler(CodeTimeout, handler)
	
	err := New(CodeTimeout, "timeout error")
	if handleErr := HandleError(err); handleErr != nil {
		t.Fatalf("Failed to handle error: %v", handleErr)
	}
	
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(CodeInternal, "benchmark error")
	}
}

func BenchmarkErrorWithDetails(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := New(CodeInternal, "benchmark error")
		err.WithDetail("field1", "value1")
		err.WithDetail("field2", "value2")
		err.WithContext("context1", "value1")
		err.WithContext("context2", "value2")
	}
}

func BenchmarkErrorJSON(b *testing.B) {
	err := New(CodeInternal, "benchmark error")
	err.WithDetail("field", "value")
	err.WithContext("context", "value")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err.JSON()
	}
}

func BenchmarkErrorChain(b *testing.B) {
	errors := make([]*Error, 10)
	for i := range errors {
		errors[i] = New(CodeInternal, "error")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := NewChain()
		for _, err := range errors {
			chain.Add(err)
		}
		chain.Error()
	}
}