// Package mcp provides comprehensive security features for the Model Context Protocol.
// This file implements critical security components including input validation,
// rate limiting, and authentication security hardening.
package mcp

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// Security error constants
var (
	ErrValidationFailed       = errors.New("security: validation failed")
	ErrSchemaValidationFailed = errors.New("security: JSON schema validation failed")
	ErrSanitizationFailed     = errors.New("security: sanitization failed")
	ErrRateLimitExceeded      = errors.New("security: rate limit exceeded")
	ErrTokenEncryptionFailed  = errors.New("security: token encryption failed")
	ErrTokenDecryptionFailed  = errors.New("security: token decryption failed")
	ErrTokenExpired           = errors.New("security: token expired")
	ErrInvalidTokenFormat     = errors.New("security: invalid token format")
)

// SecurityConfig provides comprehensive security configuration
type SecurityConfig struct {
	// Input validation settings
	MaxRequestSize      int                      `json:"maxRequestSize"`
	MaxResponseSize     int                      `json:"maxResponseSize"`
	MaxStringLength     int                      `json:"maxStringLength"`
	MaxArrayLength      int                      `json:"maxArrayLength"`
	MaxObjectDepth      int                      `json:"maxObjectDepth"`
	AllowedContentTypes []string                 `json:"allowedContentTypes"`
	ForbiddenPatterns   []string                 `json:"forbiddenPatterns"`
	CustomValidators    map[string]ValidatorFunc `json:"-"`
	SchemaValidation    bool                     `json:"schemaValidation"`
	StrictMode          bool                     `json:"strictMode"`

	// Rate limiting settings
	RateLimitEnabled  bool                     `json:"rateLimitEnabled"`
	RequestsPerSecond int                      `json:"requestsPerSecond"`
	BurstSize         int                      `json:"burstSize"`
	WindowSize        time.Duration            `json:"windowSize"`
	PerClientLimits   map[string]RateLimitRule `json:"perClientLimits"`

	// Authentication security settings
	TokenEncryption       bool          `json:"tokenEncryption"`
	TokenRotationInterval time.Duration `json:"tokenRotationInterval"`
	MaxTokenAge           time.Duration `json:"maxTokenAge"`
	EncryptionKey         []byte        `json:"-"` // Never serialize
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		// Input validation defaults
		MaxRequestSize:      1 * 1024 * 1024,  // 1MB
		MaxResponseSize:     10 * 1024 * 1024, // 10MB
		MaxStringLength:     10000,
		MaxArrayLength:      1000,
		MaxObjectDepth:      10,
		AllowedContentTypes: []string{"text/plain", "application/json", "image/png", "image/jpeg"},
		ForbiddenPatterns:   []string{`<script`, `javascript:`, `on\w+=`, `data:text/html`},
		SchemaValidation:    true,
		StrictMode:          true,

		// Rate limiting defaults
		RateLimitEnabled:  true,
		RequestsPerSecond: 100,
		BurstSize:         20,
		WindowSize:        time.Minute,

		// Authentication security defaults
		TokenEncryption:       true,
		TokenRotationInterval: 24 * time.Hour,
		MaxTokenAge:           7 * 24 * time.Hour,
	}
}

// ValidatorFunc defines a custom validation function
type ValidatorFunc func(value interface{}) error

// InputValidator provides comprehensive input validation
type InputValidator struct {
	config           *SecurityConfig
	compiledPatterns []*regexp.Regexp
}

// NewInputValidator creates a new input validator with the given configuration
func NewInputValidator(config *SecurityConfig) (*InputValidator, error) {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	v := &InputValidator{
		config: config,
	}

	// Compile forbidden patterns
	for _, pattern := range config.ForbiddenPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid forbidden pattern '%s': %w", pattern, err)
		}
		v.compiledPatterns = append(v.compiledPatterns, re)
	}

	return v, nil
}

// ValidateRequest validates an entire MCP request
func (v *InputValidator) ValidateRequest(ctx context.Context, method string, data []byte) error {
	// Check request size
	if len(data) > v.config.MaxRequestSize {
		return fmt.Errorf("%w: request size %d exceeds maximum %d",
			ErrValidationFailed, len(data), v.config.MaxRequestSize)
	}

	// Validate JSON structure
	if !json.Valid(data) {
		return fmt.Errorf("%w: invalid JSON structure", ErrValidationFailed)
	}

	// Parse and validate content
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("%w: %v", ErrValidationFailed, err)
	}

	// Deep validation
	if err := v.validateValue(raw, 0); err != nil {
		return err
	}

	// Method-specific validation
	if err := v.validateMethodParams(method, raw); err != nil {
		return err
	}

	return nil
}

// validateValue recursively validates values with depth tracking
func (v *InputValidator) validateValue(value interface{}, depth int) error {
	if depth > v.config.MaxObjectDepth {
		return fmt.Errorf("%w: object depth %d exceeds maximum %d",
			ErrValidationFailed, depth, v.config.MaxObjectDepth)
	}

	switch val := value.(type) {
	case string:
		return v.validateString(val)
	case []interface{}:
		return v.validateArray(val, depth)
	case map[string]interface{}:
		return v.validateObject(val, depth)
	case float64, int, int64, bool, nil:
		// Primitive types are safe
		return nil
	default:
		if v.config.StrictMode {
			return fmt.Errorf("%w: unexpected type %T", ErrValidationFailed, value)
		}
		return nil
	}
}

// validateString checks string values for security issues
func (v *InputValidator) validateString(s string) error {
	// Check length
	if len(s) > v.config.MaxStringLength {
		return fmt.Errorf("%w: string length %d exceeds maximum %d",
			ErrValidationFailed, len(s), v.config.MaxStringLength)
	}

	// Check for forbidden patterns
	for _, pattern := range v.compiledPatterns {
		if pattern.MatchString(s) {
			return fmt.Errorf("%w: forbidden pattern detected", ErrValidationFailed)
		}
	}

	// Check for null bytes
	if strings.Contains(s, "\x00") {
		return fmt.Errorf("%w: null byte in string", ErrValidationFailed)
	}

	return nil
}

// validateArray checks array values
func (v *InputValidator) validateArray(arr []interface{}, depth int) error {
	if len(arr) > v.config.MaxArrayLength {
		return fmt.Errorf("%w: array length %d exceeds maximum %d",
			ErrValidationFailed, len(arr), v.config.MaxArrayLength)
	}

	for i, item := range arr {
		if err := v.validateValue(item, depth+1); err != nil {
			return fmt.Errorf("array[%d]: %w", i, err)
		}
	}

	return nil
}

// validateObject checks object values
func (v *InputValidator) validateObject(obj map[string]interface{}, depth int) error {
	for key, value := range obj {
		// Validate key
		if err := v.validateString(key); err != nil {
			return fmt.Errorf("key '%s': %w", key, err)
		}

		// Validate value
		if err := v.validateValue(value, depth+1); err != nil {
			return fmt.Errorf("field '%s': %w", key, err)
		}
	}

	return nil
}

// validateMethodParams performs method-specific parameter validation
func (v *InputValidator) validateMethodParams(method string, params map[string]interface{}) error {
	switch Method(method) {
	case modelcontextprotocol.MethodToolsCall:
		return v.validateToolsCallParams(params)
	case modelcontextprotocol.MethodResourcesRead:
		return v.validateResourcesReadParams(params)
	case modelcontextprotocol.MethodPromptsGet:
		return v.validatePromptsGetParams(params)
	default:
		// Unknown methods are allowed but logged
		return nil
	}
}

// validateToolsCallParams validates tools/call specific parameters
func (v *InputValidator) validateToolsCallParams(params map[string]interface{}) error {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("%w: missing or invalid tool name", ErrValidationFailed)
	}

	// Validate tool name format
	if !isValidIdentifier(name) {
		return fmt.Errorf("%w: invalid tool name format", ErrValidationFailed)
	}

	// Custom validators for specific tools
	if validator, exists := v.config.CustomValidators[name]; exists {
		if args, ok := params["arguments"]; ok {
			return validator(args)
		}
	}

	return nil
}

// validateResourcesReadParams validates resources/read specific parameters
func (v *InputValidator) validateResourcesReadParams(params map[string]interface{}) error {
	uri, ok := params["uri"].(string)
	if !ok || uri == "" {
		return fmt.Errorf("%w: missing or invalid resource URI", ErrValidationFailed)
	}

	// Parse and validate URI
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("%w: invalid URI format: %v", ErrValidationFailed, err)
	}

	// Check for dangerous schemes
	if parsed.Scheme == "javascript" || parsed.Scheme == "data" {
		return fmt.Errorf("%w: forbidden URI scheme: %s", ErrValidationFailed, parsed.Scheme)
	}

	return nil
}

// validatePromptsGetParams validates prompts/get specific parameters
func (v *InputValidator) validatePromptsGetParams(params map[string]interface{}) error {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("%w: missing or invalid prompt name", ErrValidationFailed)
	}

	if !isValidIdentifier(name) {
		return fmt.Errorf("%w: invalid prompt name format", ErrValidationFailed)
	}

	return nil
}

// SanitizeInput sanitizes input data by removing or escaping potentially dangerous content
func (v *InputValidator) SanitizeInput(data interface{}) (interface{}, error) {
	switch val := data.(type) {
	case string:
		return v.sanitizeString(val), nil
	case []interface{}:
		return v.sanitizeArray(val)
	case map[string]interface{}:
		return v.sanitizeObject(val)
	default:
		return data, nil
	}
}

// sanitizeString cleans string values
func (v *InputValidator) sanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// HTML escape if needed
	if v.config.StrictMode {
		// Replace ampersand FIRST to avoid double-encoding
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&#x27;")
	}

	// Truncate if too long
	if len(s) > v.config.MaxStringLength {
		s = s[:v.config.MaxStringLength]
	}

	return s
}

// sanitizeArray cleans array values
func (v *InputValidator) sanitizeArray(arr []interface{}) ([]interface{}, error) {
	result := make([]interface{}, 0, len(arr))

	for _, item := range arr {
		sanitized, err := v.SanitizeInput(item)
		if err != nil {
			return nil, err
		}
		result = append(result, sanitized)

		// Limit array size
		if len(result) >= v.config.MaxArrayLength {
			break
		}
	}

	return result, nil
}

// sanitizeObject cleans object values
func (v *InputValidator) sanitizeObject(obj map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range obj {
		// Sanitize key
		sanitizedKey := v.sanitizeString(key)

		// Sanitize value
		sanitizedValue, err := v.SanitizeInput(value)
		if err != nil {
			return nil, err
		}

		result[sanitizedKey] = sanitizedValue
	}

	return result, nil
}

// JSONSchemaValidator provides JSON schema validation for MCP messages
type JSONSchemaValidator struct {
	schemas map[string]*jsonschema.Schema
	mu      sync.RWMutex
}

// NewJSONSchemaValidator creates a new JSON schema validator
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{
		schemas: make(map[string]*jsonschema.Schema),
	}
}

// RegisterSchema registers a JSON schema for a specific message type
func (v *JSONSchemaValidator) RegisterSchema(messageType string, schemaJSON string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Create a compiler
	compiler := jsonschema.NewCompiler()

	// Add schema as a resource
	schemaURL := fmt.Sprintf("schema://%s", messageType)
	if err := compiler.AddResource(schemaURL, strings.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile schema
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	v.schemas[messageType] = schema
	return nil
}

// ValidateMessage validates a message against its registered schema
func (v *JSONSchemaValidator) ValidateMessage(messageType string, data interface{}) error {
	v.mu.RLock()
	schema, exists := v.schemas[messageType]
	v.mu.RUnlock()

	if !exists {
		// No schema registered for this type
		return nil
	}

	// Validate data against schema
	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidationFailed, err)
	}

	return nil
}

// ValidateRequest validates an MCP request against its schema (RequestValidator interface)
func (v *JSONSchemaValidator) ValidateRequest(ctx context.Context, req MCPRequest) error {
	if req.GetMethod() == "" {
		return fmt.Errorf("request method is required")
	}

	// Extract params for validation
	if params := req.GetParams(); params != nil {
		var paramsObj interface{}
		if err := json.Unmarshal(params, &paramsObj); err == nil {
			return v.ValidateMessage(req.GetMethod(), paramsObj)
		}
	}

	return nil
}

// ValidateResponse validates an MCP response against its schema (RequestValidator interface)
func (v *JSONSchemaValidator) ValidateResponse(ctx context.Context, resp MCPResponse) error {
	if resp == nil {
		return fmt.Errorf("response cannot be nil")
	}

	// Validate response result if available
	if result := resp.GetResult(); result != nil {
		return v.ValidateMessage("response", result)
	}

	return nil
}

// ContentTypeValidator validates content types and MIME types
type ContentTypeValidator struct {
	allowedTypes map[string]bool
	mu           sync.RWMutex
}

// NewContentTypeValidator creates a new content type validator
func NewContentTypeValidator(allowedTypes []string) *ContentTypeValidator {
	v := &ContentTypeValidator{
		allowedTypes: make(map[string]bool),
	}

	for _, contentType := range allowedTypes {
		v.allowedTypes[strings.ToLower(contentType)] = true
	}

	return v
}

// ValidateContentType checks if a content type is allowed
func (v *ContentTypeValidator) ValidateContentType(contentType string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Extract base content type (without parameters)
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	if !v.allowedTypes[base] {
		return fmt.Errorf("%w: content type '%s' not allowed", ErrValidationFailed, contentType)
	}

	return nil
}

// ValidateImageContent validates image content for security
func (v *ContentTypeValidator) ValidateImageContent(data []byte, mimeType string) error {
	// Validate MIME type
	if err := v.ValidateContentType(mimeType); err != nil {
		return err
	}

	// Basic image validation by checking magic bytes
	switch mimeType {
	case "image/png":
		if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
			return fmt.Errorf("%w: invalid PNG format", ErrValidationFailed)
		}
	case "image/jpeg":
		if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF {
			return fmt.Errorf("%w: invalid JPEG format", ErrValidationFailed)
		}
	}

	return nil
}

// SecureTokenStorage provides encrypted token storage
type SecureTokenStorage struct {
	cipher       cipher.AEAD
	rotationTime time.Time
	mu           sync.RWMutex
}

// NewSecureTokenStorage creates a new secure token storage with encryption
func NewSecureTokenStorage(key []byte) (*SecureTokenStorage, error) {
	if len(key) != 32 {
		// Generate key from provided key using SHA256
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &SecureTokenStorage{
		cipher:       aead,
		rotationTime: time.Now().Add(24 * time.Hour),
	}, nil
}

// EncryptToken encrypts a token for secure storage
func (s *SecureTokenStorage) EncryptToken(token *AccessToken) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Serialize token
	plaintext, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenEncryptionFailed, err)
	}

	// Generate nonce
	nonce := make([]byte, s.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenEncryptionFailed, err)
	}

	// Encrypt
	ciphertext := s.cipher.Seal(nonce, nonce, plaintext, nil)

	// Encode as base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts a token from secure storage
func (s *SecureTokenStorage) DecryptToken(encrypted string) (*AccessToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Decode from base64
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenDecryptionFailed, err)
	}

	// Extract nonce
	if len(ciphertext) < s.cipher.NonceSize() {
		return nil, ErrInvalidTokenFormat
	}

	nonce, ciphertext := ciphertext[:s.cipher.NonceSize()], ciphertext[s.cipher.NonceSize():]

	// Decrypt
	plaintext, err := s.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenDecryptionFailed, err)
	}

	// Deserialize token
	var token AccessToken
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenDecryptionFailed, err)
	}

	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return &token, nil
}

// NeedsRotation checks if the encryption key needs rotation
func (s *SecureTokenStorage) NeedsRotation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.rotationTime)
}

// RotateKey rotates the encryption key
func (s *SecureTokenStorage) RotateKey(newKey []byte) error {
	if len(newKey) != 32 {
		hash := sha256.Sum256(newKey)
		newKey = hash[:]
	}

	block, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cipher = aead
	s.rotationTime = time.Now().Add(24 * time.Hour)

	return nil
}

// InputValidationMiddleware provides input validation as middleware
type InputValidationMiddleware struct {
	validator        *InputValidator
	schemaValidator  *JSONSchemaValidator
	contentValidator *ContentTypeValidator
}

// NewInputValidationMiddleware creates validation middleware
func NewInputValidationMiddleware(config *SecurityConfig) (*InputValidationMiddleware, error) {
	validator, err := NewInputValidator(config)
	if err != nil {
		return nil, err
	}

	return &InputValidationMiddleware{
		validator:        validator,
		schemaValidator:  NewJSONSchemaValidator(),
		contentValidator: NewContentTypeValidator(config.AllowedContentTypes),
	}, nil
}

// Apply implements the Middleware interface
func (m *InputValidationMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Extract raw params for validation
		if params := req.GetParams(); params != nil {
			if err := m.validator.ValidateRequest(ctx, req.GetMethod(), params); err != nil {
				return NewErrorResponse(err.Error(), -32602), nil // Invalid params error
			}

			// Schema validation if available
			var paramsObj interface{}
			if err := json.Unmarshal(params, &paramsObj); err == nil {
				if err := m.schemaValidator.ValidateMessage(req.GetMethod(), paramsObj); err != nil {
					return NewErrorResponse(err.Error(), -32602), nil
				}
			}
		}

		// Process request
		resp, err := next.Handle(ctx, req)

		// Validate response size
		if resp != nil {
			if respData, err := json.Marshal(resp.GetResult()); err == nil {
				if len(respData) > m.validator.config.MaxResponseSize {
					return NewErrorResponse("Response too large", -32603), nil
				}
			}
		}

		return resp, err
	})
}

func (m *InputValidationMiddleware) Name() string {
	return "input_validation"
}

func (m *InputValidationMiddleware) Priority() int {
	return 950 // High priority, after recovery but before auth
}
