package sdk2

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validator defines the interface for content validators.
type Validator interface {
	Validate() error
}

// ValidateContent validates any content implementing the Validator interface.
func ValidateContent(content interface{}) error {
	if validator, ok := content.(Validator); ok {
		return validator.Validate()
	}
	return nil
}

// ContentMetadata provides additional information about content.
type ContentMetadata struct {
	Size        int64             `json:"size,omitempty"`
	Encoding    string            `json:"encoding,omitempty"`
	Language    string            `json:"language,omitempty"`
	Created     string            `json:"created,omitempty"`
	Modified    string            `json:"modified,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
	Compression string            `json:"compression,omitempty"`
}

// Enhanced Content interface with additional capabilities
type EnhancedContent interface {
	Content
	Size() int64
	Metadata() *ContentMetadata
	Validate() error
}

// Enhanced content types

// EnhancedTextContent extends TextContent with metadata and validation.
type EnhancedTextContent struct {
	Text     string           `json:"text"`
	Encoding string           `json:"encoding,omitempty"`
	Language string           `json:"language,omitempty"`
	metadata *ContentMetadata `json:"-"`
}

// ContentType implements Content.
func (t EnhancedTextContent) ContentType() string {
	if t.Encoding != "" {
		return fmt.Sprintf("text/plain; charset=%s", t.Encoding)
	}
	return "text/plain"
}

// mcpContent implements Content (sealed interface).
func (EnhancedTextContent) mcpContent() {}

// Size implements EnhancedContent.
func (t EnhancedTextContent) Size() int64 {
	return int64(utf8.RuneCountInString(t.Text))
}

// Metadata implements EnhancedContent.
func (t EnhancedTextContent) Metadata() *ContentMetadata {
	if t.metadata == nil {
		t.metadata = &ContentMetadata{
			Size:     t.Size(),
			Encoding: t.Encoding,
			Language: t.Language,
		}
	}
	return t.metadata
}

// Valid implements Content validation.
func (t EnhancedTextContent) Valid() error {
	return t.Validate()
}

// Validate implements enhanced validation.
func (t EnhancedTextContent) Validate() error {
	var errors ValidationErrors
	
	if strings.TrimSpace(t.Text) == "" {
		errors.Add("text", "required", "text content cannot be empty or whitespace only", t.Text)
	}
	
	if len(t.Text) > 1024*1024 { // 1MB limit
		errors.Add("text", "max_length", "text content exceeds maximum size of 1MB", len(t.Text))
	}
	
	if !utf8.ValidString(t.Text) {
		errors.Add("text", "encoding", "text content must be valid UTF-8", nil)
	}
	
	if t.Encoding != "" && !isValidEncoding(t.Encoding) {
		errors.Add("encoding", "format", "invalid encoding format", t.Encoding)
	}
	
	if t.Language != "" && !isValidLanguageTag(t.Language) {
		errors.Add("language", "format", "invalid language tag format", t.Language)
	}
	
	if errors.HasErrors() {
		return errors
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (t EnhancedTextContent) MarshalJSON() ([]byte, error) {
	type textContent struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Encoding string `json:"encoding,omitempty"`
		Language string `json:"language,omitempty"`
	}
	
	return json.Marshal(textContent{
		Type:     "text",
		Text:     t.Text,
		Encoding: t.Encoding,
		Language: t.Language,
	})
}

// EnhancedImageContent extends ImageContent with metadata and validation.
type EnhancedImageContent struct {
	Data        string           `json:"data"`
	MimeType    string           `json:"mimeType"`
	Width       int              `json:"width,omitempty"`
	Height      int              `json:"height,omitempty"`
	Compression float64          `json:"compression,omitempty"` // 0.0-1.0
	metadata    *ContentMetadata `json:"-"`
}

// ContentType implements Content.
func (i EnhancedImageContent) ContentType() string {
	return i.MimeType
}

// mcpContent implements Content (sealed interface).
func (EnhancedImageContent) mcpContent() {}

// Size implements EnhancedContent.
func (i EnhancedImageContent) Size() int64 {
	// Estimate size from base64 data (roughly 4/3 of actual size)
	return int64(len(i.Data) * 3 / 4)
}

// Metadata implements EnhancedContent.
func (i EnhancedImageContent) Metadata() *ContentMetadata {
	if i.metadata == nil {
		i.metadata = &ContentMetadata{
			Size: i.Size(),
			Attributes: map[string]string{
				"width":       fmt.Sprintf("%d", i.Width),
				"height":      fmt.Sprintf("%d", i.Height),
				"compression": fmt.Sprintf("%.2f", i.Compression),
			},
		}
	}
	return i.metadata
}

// Valid implements Content validation.
func (i EnhancedImageContent) Valid() error {
	return i.Validate()
}

// Validate implements enhanced validation.
func (i EnhancedImageContent) Validate() error {
	var errors ValidationErrors
	
	if strings.TrimSpace(i.Data) == "" {
		errors.Add("data", "required", "image data cannot be empty", i.Data)
	}
	
	if i.MimeType == "" {
		errors.Add("mimeType", "required", "image mime type is required", i.MimeType)
	} else if !isValidImageMimeType(i.MimeType) {
		errors.Add("mimeType", "format", "invalid image mime type", i.MimeType)
	}
	
	if i.Width < 0 {
		errors.Add("width", "min", "image width cannot be negative", i.Width)
	}
	
	if i.Height < 0 {
		errors.Add("height", "min", "image height cannot be negative", i.Height)
	}
	
	if i.Compression < 0.0 || i.Compression > 1.0 {
		errors.Add("compression", "range", "compression must be between 0.0 and 1.0", i.Compression)
	}
	
	// Validate base64 format
	if !isValidBase64(i.Data) {
		errors.Add("data", "format", "image data must be valid base64", nil)
	}
	
	// Size limits (10MB for images)
	if i.Size() > 10*1024*1024 {
		errors.Add("data", "max_size", "image data exceeds maximum size of 10MB", i.Size())
	}
	
	if errors.HasErrors() {
		return errors
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (i EnhancedImageContent) MarshalJSON() ([]byte, error) {
	type imageContent struct {
		Type        string  `json:"type"`
		Data        string  `json:"data"`
		MimeType    string  `json:"mimeType"`
		Width       int     `json:"width,omitempty"`
		Height      int     `json:"height,omitempty"`
		Compression float64 `json:"compression,omitempty"`
	}
	
	return json.Marshal(imageContent{
		Type:        "image",
		Data:        i.Data,
		MimeType:    i.MimeType,
		Width:       i.Width,
		Height:      i.Height,
		Compression: i.Compression,
	})
}

// EnhancedResourceContent extends ResourceReferenceContent with metadata.
type EnhancedResourceContent struct {
	URI         string           `json:"uri"`
	MimeType    string           `json:"mimeType,omitempty"`
	Cacheable   bool             `json:"cacheable,omitempty"`
	TTL         int64            `json:"ttl,omitempty"` // seconds
	AccessLevel string           `json:"accessLevel,omitempty"`
	metadata    *ContentMetadata `json:"-"`
}

// ContentType implements Content.
func (r EnhancedResourceContent) ContentType() string {
	if r.MimeType != "" {
		return r.MimeType
	}
	return "application/octet-stream"
}

// mcpContent implements Content (sealed interface).
func (EnhancedResourceContent) mcpContent() {}

// Size implements EnhancedContent.
func (r EnhancedResourceContent) Size() int64 {
	// For resource references, return URI size as proxy
	return int64(len(r.URI))
}

// Metadata implements EnhancedContent.
func (r EnhancedResourceContent) Metadata() *ContentMetadata {
	if r.metadata == nil {
		r.metadata = &ContentMetadata{
			Size: r.Size(),
			Attributes: map[string]string{
				"cacheable":    fmt.Sprintf("%t", r.Cacheable),
				"ttl":          fmt.Sprintf("%d", r.TTL),
				"access_level": r.AccessLevel,
			},
		}
	}
	return r.metadata
}

// Valid implements Content validation.
func (r EnhancedResourceContent) Valid() error {
	return r.Validate()
}

// Validate implements enhanced validation.
func (r EnhancedResourceContent) Validate() error {
	var errors ValidationErrors
	
	if strings.TrimSpace(r.URI) == "" {
		errors.Add("uri", "required", "resource URI cannot be empty", r.URI)
	} else {
		if _, err := url.Parse(r.URI); err != nil {
			errors.Add("uri", "format", "invalid URI format", r.URI)
		}
	}
	
	if r.MimeType != "" && !isValidMimeType(r.MimeType) {
		errors.Add("mimeType", "format", "invalid mime type format", r.MimeType)
	}
	
	if r.TTL < 0 {
		errors.Add("ttl", "min", "TTL cannot be negative", r.TTL)
	}
	
	if r.AccessLevel != "" && !isValidAccessLevel(r.AccessLevel) {
		errors.Add("accessLevel", "enum", "invalid access level", r.AccessLevel)
	}
	
	if errors.HasErrors() {
		return errors
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (r EnhancedResourceContent) MarshalJSON() ([]byte, error) {
	type resourceContent struct {
		Type        string `json:"type"`
		URI         string `json:"uri"`
		MimeType    string `json:"mimeType,omitempty"`
		Cacheable   bool   `json:"cacheable,omitempty"`
		TTL         int64  `json:"ttl,omitempty"`
		AccessLevel string `json:"accessLevel,omitempty"`
	}
	
	return json.Marshal(resourceContent{
		Type:        "resource",
		URI:         r.URI,
		MimeType:    r.MimeType,
		Cacheable:   r.Cacheable,
		TTL:         r.TTL,
		AccessLevel: r.AccessLevel,
	})
}

// Validation helper functions

func isValidEncoding(encoding string) bool {
	validEncodings := []string{"utf-8", "utf-16", "ascii", "iso-8859-1"}
	encoding = strings.ToLower(encoding)
	for _, valid := range validEncodings {
		if encoding == valid {
			return true
		}
	}
	return false
}

func isValidLanguageTag(tag string) bool {
	// Basic language tag validation (simplified BCP 47)
	match, _ := regexp.MatchString(`^[a-z]{2,3}(-[A-Z]{2})?(-[a-z0-9]+)*$`, tag)
	return match
}

func isValidImageMimeType(mimeType string) bool {
	validTypes := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"image/svg+xml", "image/bmp", "image/tiff",
	}
	for _, valid := range validTypes {
		if mimeType == valid {
			return true
		}
	}
	return false
}

func isValidMimeType(mimeType string) bool {
	// Basic MIME type validation
	match, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9][a-zA-Z0-9\-\.\+]*\/[a-zA-Z0-9][a-zA-Z0-9\-\.\+]*$`, mimeType)
	return match
}

func isValidBase64(data string) bool {
	// Basic base64 validation
	match, _ := regexp.MatchString(`^[A-Za-z0-9+/]*={0,2}$`, data)
	return match && len(data)%4 == 0
}

func isValidAccessLevel(level string) bool {
	validLevels := []string{"public", "private", "restricted", "internal"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// ContentOption defines functional options for content creation.
type ContentOption func(interface{})

// Content option constructors

// WithImageDimensions sets image dimensions.
func WithImageDimensions(width, height int) ContentOption {
	return func(v interface{}) {
		if img, ok := v.(*EnhancedImageContent); ok {
			img.Width = width
			img.Height = height
		}
	}
}

// WithImageCompression sets image compression level.
func WithImageCompression(compression float64) ContentOption {
	return func(v interface{}) {
		if img, ok := v.(*EnhancedImageContent); ok {
			img.Compression = compression
		}
	}
}

// WithResourceCaching enables/disables resource caching.
func WithResourceCaching(cacheable bool) ContentOption {
	return func(v interface{}) {
		if res, ok := v.(*EnhancedResourceContent); ok {
			res.Cacheable = cacheable
		}
	}
}

// WithResourceExpiration sets resource TTL.
func WithResourceExpiration(ttl int64) ContentOption {
	return func(v interface{}) {
		if res, ok := v.(*EnhancedResourceContent); ok {
			res.TTL = ttl
		}
	}
}

// WithTextEncoding sets text encoding.
func WithTextEncoding(encoding string) ContentOption {
	return func(v interface{}) {
		if text, ok := v.(*EnhancedTextContent); ok {
			text.Encoding = encoding
		}
	}
}

// WithTextLanguage sets text language.
func WithTextLanguage(language string) ContentOption {
	return func(v interface{}) {
		if text, ok := v.(*EnhancedTextContent); ok {
			text.Language = language
		}
	}
}

// Enhanced content constructors

// NewEnhancedTextContent creates enhanced text content with validation.
func NewEnhancedTextContent(text string, opts ...ContentOption) (*EnhancedTextContent, error) {
	content := &EnhancedTextContent{
		Text:     text,
		Encoding: "utf-8",
	}
	
	for _, opt := range opts {
		opt(content)
	}
	
	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("invalid text content: %w", err)
	}
	
	return content, nil
}

// MustNewEnhancedTextContent creates enhanced text content, panicking on error.
func MustNewEnhancedTextContent(text string, opts ...ContentOption) *EnhancedTextContent {
	content, err := NewEnhancedTextContent(text, opts...)
	if err != nil {
		panic(err)
	}
	return content
}

// NewEnhancedImageContent creates enhanced image content with validation.
func NewEnhancedImageContent(data, mimeType string, opts ...ContentOption) (*EnhancedImageContent, error) {
	content := &EnhancedImageContent{
		Data:     data,
		MimeType: mimeType,
	}
	
	for _, opt := range opts {
		opt(content)
	}
	
	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("invalid image content: %w", err)
	}
	
	return content, nil
}

// MustNewEnhancedImageContent creates enhanced image content, panicking on error.
func MustNewEnhancedImageContent(data, mimeType string, opts ...ContentOption) *EnhancedImageContent {
	content, err := NewEnhancedImageContent(data, mimeType, opts...)
	if err != nil {
		panic(err)
	}
	return content
}

// NewEnhancedResourceContent creates enhanced resource content with validation.
func NewEnhancedResourceContent(uri string, opts ...ContentOption) (*EnhancedResourceContent, error) {
	content := &EnhancedResourceContent{
		URI:       uri,
		Cacheable: true, // Default to cacheable
	}
	
	for _, opt := range opts {
		opt(content)
	}
	
	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource content: %w", err)
	}
	
	return content, nil
}

// MustNewEnhancedResourceContent creates enhanced resource content, panicking on error.
func MustNewEnhancedResourceContent(uri string, opts ...ContentOption) *EnhancedResourceContent {
	content, err := NewEnhancedResourceContent(uri, opts...)
	if err != nil {
		panic(err)
	}
	return content
}

// ValidateStruct performs struct validation using reflection and tags.
func ValidateStruct(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", val.Kind())
	}
	
	var errors ValidationErrors
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		// Check for validation tags
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}
		
		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		// Perform validation based on tags
		if err := validateField(fieldName, field, validateTag); err != nil {
			if valErr, ok := err.(*ValidationError); ok {
				errors = append(errors, *valErr)
			} else {
				errors.Add(fieldName, "validation", err.Error(), field.Interface())
			}
		}
	}
	
	if errors.HasErrors() {
		return errors
	}
	return nil
}

func validateField(fieldName string, field reflect.Value, tag string) error {
	rules := strings.Split(tag, ",")
	
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		
		switch rule {
		case "required":
			if field.Kind() == reflect.String && field.String() == "" {
				return NewValidationError(fieldName, "required", "field is required", nil)
			}
			if field.Kind() == reflect.Slice && field.Len() == 0 {
				return NewValidationError(fieldName, "required", "field is required", nil)
			}
			if field.IsZero() {
				return NewValidationError(fieldName, "required", "field is required", nil)
			}
		case "nonzero":
			if field.IsZero() {
				return NewValidationError(fieldName, "nonzero", "field cannot be zero", field.Interface())
			}
		}
		
		// Handle parameterized rules
		if strings.Contains(rule, "=") {
			parts := strings.SplitN(rule, "=", 2)
			if len(parts) == 2 {
				ruleName := parts[0]
				ruleValue := parts[1]
				
				if err := validateParameterizedRule(fieldName, field, ruleName, ruleValue); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

func validateParameterizedRule(fieldName string, field reflect.Value, ruleName, ruleValue string) error {
	switch ruleName {
	case "min":
		// Implementation would depend on field type
	case "max":
		// Implementation would depend on field type
	case "minlen":
		// For strings and slices
	case "maxlen":
		// For strings and slices
	}
	return nil
}