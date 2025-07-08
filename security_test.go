package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInputValidator(t *testing.T) {
	config := &SecurityConfig{
		MaxRequestSize:      1024,
		MaxStringLength:     100,
		MaxArrayLength:      10,
		MaxObjectDepth:      3,
		ForbiddenPatterns:   []string{`<script`, `javascript:`},
		AllowedContentTypes: []string{"text/plain", "application/json"},
		StrictMode:          true,
		SchemaValidation:    true,
	}

	validator, err := NewInputValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		method  string
		data    string
		wantErr bool
	}{
		{
			name:    "valid JSON request",
			method:  "tools/call",
			data:    `{"name": "test_tool", "arguments": {"arg1": "value1"}}`,
			wantErr: false,
		},
		{
			name:    "request too large",
			method:  "tools/call",
			data:    strings.Repeat("a", 2000),
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			method:  "tools/call",
			data:    `{"invalid": json}`,
			wantErr: true,
		},
		{
			name:    "forbidden pattern in string",
			method:  "tools/call",
			data:    `{"name": "test", "description": "<script>alert('xss')</script>"}`,
			wantErr: true,
		},
		{
			name:    "string too long",
			method:  "tools/call",
			data:    `{"name": "` + strings.Repeat("a", 200) + `"}`,
			wantErr: true,
		},
		{
			name:    "array too long",
			method:  "tools/call",
			data:    `{"items": [` + strings.Repeat(`"item",`, 20) + `"item"]}`,
			wantErr: true,
		},
		{
			name:    "object too deep",
			method:  "tools/call",
			data:    `{"a": {"b": {"c": {"d": {"e": "too deep"}}}}}`,
			wantErr: true,
		},
		{
			name:    "null byte in string",
			method:  "tools/call",
			data:    "{\"name\": \"test\x00null\"}",
			wantErr: true,
		},
		{
			name:    "missing required tool name",
			method:  "tools/call",
			data:    `{"arguments": {}}`,
			wantErr: true,
		},
		{
			name:    "invalid tool name format",
			method:  "tools/call",
			data:    `{"name": "invalid tool name!"}`,
			wantErr: true,
		},
		{
			name:    "valid resource read",
			method:  "resources/read",
			data:    `{"uri": "file:///path/to/resource"}`,
			wantErr: false,
		},
		{
			name:    "dangerous URI scheme",
			method:  "resources/read",
			data:    `{"uri": "javascript:alert('xss')"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(context.Background(), tt.method, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	config := DefaultSecurityConfig()
	validator, err := NewInputValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "sanitize HTML in string",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert('xss')&lt;/script&gt;",
		},
		{
			name:     "truncate long string",
			input:    strings.Repeat("a", 20000),
			expected: strings.Repeat("a", config.MaxStringLength),
		},
		{
			name:     "sanitize array",
			input:    []interface{}{"<b>bold</b>", "normal"},
			expected: []interface{}{"&lt;b&gt;bold&lt;/b&gt;", "normal"},
		},
		{
			name: "sanitize nested object",
			input: map[string]interface{}{
				"field1": "<script>",
				"field2": map[string]interface{}{
					"nested": "value",
				},
			},
			expected: map[string]interface{}{
				"field1": "&lt;script&gt;",
				"field2": map[string]interface{}{
					"nested": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.SanitizeInput(tt.input)
			if err != nil {
				t.Errorf("SanitizeInput() error = %v", err)
				return
			}

			// Compare results
			switch expected := tt.expected.(type) {
			case string:
				if result != expected {
					t.Errorf("SanitizeInput() = %v, want %v", result, expected)
				}
			case []interface{}:
				resultArr := result.([]interface{})
				if len(resultArr) != len(expected) {
					t.Errorf("SanitizeInput() array length = %d, want %d", len(resultArr), len(expected))
				}
			case map[string]interface{}:
				resultMap := result.(map[string]interface{})
				if resultMap["field1"] != expected["field1"] {
					t.Errorf("SanitizeInput() field1 = %v, want %v", resultMap["field1"], expected["field1"])
				}
			}
		})
	}
}

func TestJSONSchemaValidator(t *testing.T) {
	validator := NewJSONSchemaValidator()

	// Register a schema for tools/call
	toolSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 1},
			"arguments": {"type": "object"}
		},
		"required": ["name"]
	}`

	err := validator.RegisterSchema("tools/call", toolSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	tests := []struct {
		name    string
		msgType string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "valid tool call",
			msgType: "tools/call",
			data: map[string]interface{}{
				"name":      "test_tool",
				"arguments": map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:    "missing required field",
			msgType: "tools/call",
			data: map[string]interface{}{
				"arguments": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name:    "wrong type",
			msgType: "tools/call",
			data: map[string]interface{}{
				"name": 123, // Should be string
			},
			wantErr: true,
		},
		{
			name:    "no schema registered",
			msgType: "unknown/method",
			data:    map[string]interface{}{},
			wantErr: false, // No schema means no validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMessage(tt.msgType, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContentTypeValidator(t *testing.T) {
	validator := NewContentTypeValidator([]string{
		"text/plain",
		"application/json",
		"image/png",
		"image/jpeg",
	})

	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{
			name:        "allowed content type",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "allowed with charset",
			contentType: "text/plain; charset=utf-8",
			wantErr:     false,
		},
		{
			name:        "disallowed content type",
			contentType: "application/octet-stream",
			wantErr:     true,
		},
		{
			name:        "case insensitive",
			contentType: "TEXT/PLAIN",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateContentType(tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageContent(t *testing.T) {
	validator := NewContentTypeValidator([]string{"image/png", "image/jpeg"})

	// Valid PNG magic bytes
	validPNG := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 100))
	
	// Valid JPEG magic bytes
	validJPEG := []byte{0xFF, 0xD8, 0xFF}
	validJPEG = append(validJPEG, make([]byte, 100)...)

	tests := []struct {
		name     string
		data     []byte
		mimeType string
		wantErr  bool
	}{
		{
			name:     "valid PNG",
			data:     validPNG,
			mimeType: "image/png",
			wantErr:  false,
		},
		{
			name:     "valid JPEG",
			data:     validJPEG,
			mimeType: "image/jpeg",
			wantErr:  false,
		},
		{
			name:     "invalid PNG magic bytes",
			data:     []byte("not a png"),
			mimeType: "image/png",
			wantErr:  true,
		},
		{
			name:     "invalid JPEG magic bytes",
			data:     []byte("not a jpeg"),
			mimeType: "image/jpeg",
			wantErr:  true,
		},
		{
			name:     "disallowed MIME type",
			data:     validPNG,
			mimeType: "image/gif",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateImageContent(tt.data, tt.mimeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecureTokenStorage(t *testing.T) {
	key := []byte("test-encryption-key-32-bytes-long")
	storage, err := NewSecureTokenStorage(key)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	token := &AccessToken{
		AccessToken:  "test-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh",
		Scope:        "read write",
		ClientID:     "test-client",
		Scopes:       []string{"read", "write"},
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	// Test encryption
	encrypted, err := storage.EncryptToken(token)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	if encrypted == "" {
		t.Error("Encrypted token is empty")
	}

	// Test decryption
	decrypted, err := storage.DecryptToken(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt token: %v", err)
	}

	// Verify fields
	if decrypted.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %v, want %v", decrypted.AccessToken, token.AccessToken)
	}
	if decrypted.ClientID != token.ClientID {
		t.Errorf("ClientID = %v, want %v", decrypted.ClientID, token.ClientID)
	}

	// Test expired token
	expiredToken := &AccessToken{
		AccessToken: "expired",
		ExpiresAt:   time.Now().Add(-time.Hour),
	}

	encryptedExpired, err := storage.EncryptToken(expiredToken)
	if err != nil {
		t.Fatalf("Failed to encrypt expired token: %v", err)
	}

	_, err = storage.DecryptToken(encryptedExpired)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}

	// Test invalid encrypted data
	_, err = storage.DecryptToken("invalid-base64")
	if err == nil {
		t.Error("Expected error for invalid encrypted data")
	}

	// Test rotation check
	if storage.NeedsRotation() {
		t.Error("New storage should not need rotation")
	}

	// Test key rotation
	newKey := []byte("new-encryption-key-32-bytes-long!")
	err = storage.RotateKey(newKey)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// Old encrypted token should fail
	_, err = storage.DecryptToken(encrypted)
	if err == nil {
		t.Error("Old encrypted token should fail after key rotation")
	}
}

func TestInputValidationMiddleware(t *testing.T) {
	config := DefaultSecurityConfig()
	config.MaxRequestSize = 1024

	middleware, err := NewInputValidationMiddleware(config)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Create a test handler
	handler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &ErrorResponseImpl{}, nil
	})

	// Apply middleware
	protected := middleware.Apply(handler)

	tests := []struct {
		name       string
		request    MCPRequest
		wantError  bool
		errorCode  int
	}{
		{
			name: "valid request",
			request: &mockMCPRequest{
				method: "tools/call",
				params: json.RawMessage(`{"name": "valid_tool"}`),
			},
			wantError: false,
		},
		{
			name: "invalid JSON",
			request: &mockMCPRequest{
				method: "tools/call",
				params: json.RawMessage(`{invalid json}`),
			},
			wantError: true,
			errorCode: -32602, // Invalid params
		},
		{
			name: "forbidden pattern",
			request: &mockMCPRequest{
				method: "tools/call",
				params: json.RawMessage(`{"name": "test", "desc": "<script>alert('xss')</script>"}`),
			},
			wantError: true,
			errorCode: -32602,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := protected.Handle(context.Background(), tt.request)
			
			if tt.wantError {
				if err == nil && (resp == nil || !resp.IsError()) {
					t.Error("Expected error response")
				}
				if resp != nil && resp.IsError() {
					if respErr := resp.GetError(); respErr != nil && respErr.Code != tt.errorCode {
						t.Errorf("Error code = %d, want %d", respErr.Code, tt.errorCode)
					}
				}
			} else {
				if err != nil || (resp != nil && resp.IsError()) {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// mockMCPRequest implements MCPRequest for testing
type mockMCPRequest struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func (r *mockMCPRequest) GetMethod() string {
	return r.method
}

func (r *mockMCPRequest) GetID() interface{} {
	return r.id
}

func (r *mockMCPRequest) GetParams() json.RawMessage {
	return r.params
}

func (r *mockMCPRequest) GetContext() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

func (r *mockMCPRequest) WithContext(ctx context.Context) MCPRequest {
	return &mockMCPRequest{
		method: r.method,
		id:     r.id,
		params: r.params,
		ctx:    ctx,
	}
}

func BenchmarkInputValidation(b *testing.B) {
	config := DefaultSecurityConfig()
	validator, _ := NewInputValidator(config)
	
	data := []byte(`{
		"name": "test_tool",
		"arguments": {
			"param1": "value1",
			"param2": 123,
			"param3": ["item1", "item2", "item3"]
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateRequest(context.Background(), "tools/call", data)
	}
}

func BenchmarkTokenEncryption(b *testing.B) {
	key := []byte("benchmark-key-32-bytes-long!!!!!")
	storage, _ := NewSecureTokenStorage(key)
	
	token := &AccessToken{
		AccessToken:  "benchmark-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "benchmark-refresh",
		ClientID:     "benchmark-client",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := storage.EncryptToken(token)
		_, _ = storage.DecryptToken(encrypted)
	}
}