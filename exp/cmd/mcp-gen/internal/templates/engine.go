// Package templates provides a robust template engine for multi-language code generation
package templates

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template/parse"
	"unicode"

	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/config"
)

// TemplateEngine provides multi-language template processing
type TemplateEngine struct {
	config    *config.Config
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// TemplateData contains data passed to templates
type TemplateData struct {
	Config    *config.Config
	Package   string
	Language  string
	Tools     []Tool
	Types     []Type
	Client    *Client
	Server    *Server
	Docs      *Documentation
	Tests     *TestSuite
	Plugin    *Plugin
	Vars      map[string]interface{}
}

// Tool represents an MCP tool for template generation
type Tool struct {
	Name         string
	Description  string
	InputType    string
	OutputType   string
	InputSchema  map[string]interface{}
	OutputSchema map[string]interface{}
	Examples     []Example
	Deprecated   bool
}

// Type represents a generated type
type Type struct {
	Name        string
	Description string
	Fields      []Field
	Methods     []Method
	Annotations []string
}

// Field represents a struct/class field
type Field struct {
	Name        string
	Type        string
	Description string
	Optional    bool
	Tags        map[string]string
}

// Method represents a method/function
type Method struct {
	Name        string
	Description string
	Parameters  []Parameter
	ReturnType  string
	Body        string
	Annotations []string
}

// Parameter represents a method parameter
type Parameter struct {
	Name        string
	Type        string
	Description string
	Optional    bool
	Default     string
}

// Example represents a usage example
type Example struct {
	Name        string
	Description string
	Input       string
	Output      string
	Code        string
}

// Client represents client generation data
type Client struct {
	Name        string
	Description string
	Tools       []Tool
	Methods     []Method
}

// Server represents server generation data
type Server struct {
	Name        string
	Description string
	Tools       []Tool
	Handlers    []Handler
}

// Handler represents a tool handler
type Handler struct {
	Tool        Tool
	HandlerName string
	Signature   string
	Body        string
}

// Documentation represents documentation generation data
type Documentation struct {
	Title       string
	Description string
	Tools       []Tool
	Types       []Type
	Examples    []Example
}

// TestSuite represents test generation data
type TestSuite struct {
	Name        string
	Tools       []Tool
	TestCases   []TestCase
}

// TestCase represents a single test case
type TestCase struct {
	Name        string
	Description string
	Setup       string
	Input       string
	Expected    string
	Teardown    string
}

// Plugin represents plugin generation data
type Plugin struct {
	Name        string
	Description string
	Interface   string
	Methods     []Method
}

//go:embed templates/*
var embeddedTemplates embed.FS

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(cfg *config.Config) *TemplateEngine {
	return &TemplateEngine{
		config:    cfg,
		templates: make(map[string]*template.Template),
		funcs:     createTemplateFuncs(cfg),
	}
}

// createTemplateFuncs creates template functions for all supported languages
func createTemplateFuncs(cfg *config.Config) template.FuncMap {
	return template.FuncMap{
		// String manipulation
		"toLower":         strings.ToLower,
		"toUpper":         strings.ToUpper,
		"toTitle":         strings.Title,
		"toCamelCase":     toCamelCase,
		"toPascalCase":    toPascalCase,
		"toSnakeCase":     toSnakeCase,
		"toKebabCase":     toKebabCase,
		"toConstantCase":  toConstantCase,
		"trim":            strings.TrimSpace,
		"trimPrefix":      strings.TrimPrefix,
		"trimSuffix":      strings.TrimSuffix,
		"replace":         strings.ReplaceAll,
		"split":           strings.Split,
		"join":            strings.Join,
		"contains":        strings.Contains,
		"hasPrefix":       strings.HasPrefix,
		"hasSuffix":       strings.HasSuffix,
		
		// Language-specific naming
		"goName":          toGoName,
		"goType":          toGoType,
		"goTag":           toGoTag,
		"tsName":          toTSName,
		"tsType":          toTSType,
		"pyName":          toPyName,
		"pyType":          toPyType,
		"rustName":        toRustName,
		"rustType":        toRustType,
		"javaName":        toJavaName,
		"javaType":        toJavaType,
		
		// Type conversion
		"jsonToGoType":    jsonToGoType,
		"jsonToTSType":    jsonToTSType,
		"jsonToPyType":    jsonToPyType,
		"jsonToRustType":  jsonToRustType,
		"jsonToJavaType":  jsonToJavaType,
		
		// Formatting
		"indent":          indent,
		"comment":         comment,
		"docComment":      docComment,
		"formatCode":      formatCode,
		
		// Utility functions
		"default":         defaultValue,
		"coalesce":        coalesce,
		"dict":            dict,
		"list":            list,
		"range":           rangeFunc,
		"if":              ifFunc,
		"eq":              eq,
		"ne":              ne,
		"lt":              lt,
		"le":              le,
		"gt":              gt,
		"ge":              ge,
		"and":             and,
		"or":              or,
		"not":             not,
		
		// Template inclusion
		"include":         include,
		"template":        templateFunc,
		
		// Language-specific helpers
		"goImports":       goImports,
		"tsImports":       tsImports,
		"pyImports":       pyImports,
		"rustUses":        rustUses,
		"javaImports":     javaImports,
		
		// Code generation helpers
		"generateStruct":  generateStruct,
		"generateClass":   generateClass,
		"generateEnum":    generateEnum,
		"generateMethod":  generateMethod,
		"generateTest":    generateTest,
	}
}

// LoadTemplates loads templates from embedded files or custom directory
func (e *TemplateEngine) LoadTemplates() error {
	var templatesFS fs.FS = embeddedTemplates
	
	// Use custom template directory if specified
	if e.config.Templates.Directory != "" {
		templatesFS = os.DirFS(e.config.Templates.Directory)
	}
	
	return fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		
		content, err := fs.ReadFile(templatesFS, path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}
		
		name := strings.TrimSuffix(path, ".tmpl")
		name = strings.TrimPrefix(name, "templates/")
		
		tmpl, err := template.New(name).Funcs(e.funcs).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}
		
		e.templates[name] = tmpl
		return nil
	})
}

// ExecuteTemplate executes a template with the given data
func (e *TemplateEngine) ExecuteTemplate(name string, data *TemplateData) (string, error) {
	tmpl, exists := e.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}
	
	// Add config and custom vars to data
	if data.Config == nil {
		data.Config = e.config
	}
	if data.Vars == nil {
		data.Vars = make(map[string]interface{})
	}
	
	// Merge custom vars from config
	for k, v := range e.config.Templates.CustomVars {
		data.Vars[k] = v
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	
	result := buf.String()
	
	// Format code if it's a known language
	if formatted, err := e.formatCode(result, data.Language); err == nil {
		result = formatted
	}
	
	return result, nil
}

// formatCode formats code based on language
func (e *TemplateEngine) formatCode(code, language string) (string, error) {
	switch language {
	case "go":
		return formatGoCode(code)
	case "typescript":
		return formatTSCode(code)
	case "python":
		return formatPyCode(code)
	case "rust":
		return formatRustCode(code)
	case "java":
		return formatJavaCode(code)
	default:
		return code, nil
	}
}

// Helper functions for template functions

func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	if len(words) == 0 {
		return s
	}
	
	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		result += strings.Title(strings.ToLower(words[i]))
	}
	return result
}

func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	
	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.Title(strings.ToLower(word)))
	}
	return result.String()
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

func toConstantCase(s string) string {
	return strings.ToUpper(toSnakeCase(s))
}

func toGoName(s string) string {
	return toPascalCase(s)
}

func toGoType(jsonType string) string {
	switch jsonType {
	case "string":
		return "string"
	case "number":
		return "float64"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func toGoTag(name string, tags map[string]string) string {
	if tags == nil {
		return fmt.Sprintf("`json:\"%s\"`", name)
	}
	
	var parts []string
	for key, value := range tags {
		parts = append(parts, fmt.Sprintf("%s:\"%s\"", key, value))
	}
	return "`" + strings.Join(parts, " ") + "`"
}

func toTSName(s string) string {
	return toCamelCase(s)
}

func toTSType(jsonType string) string {
	switch jsonType {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "any[]"
	case "object":
		return "Record<string, any>"
	default:
		return "any"
	}
}

func toPyName(s string) string {
	return toSnakeCase(s)
}

func toPyType(jsonType string) string {
	switch jsonType {
	case "string":
		return "str"
	case "number":
		return "float"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		return "List[Any]"
	case "object":
		return "Dict[str, Any]"
	default:
		return "Any"
	}
}

func toRustName(s string) string {
	return toSnakeCase(s)
}

func toRustType(jsonType string) string {
	switch jsonType {
	case "string":
		return "String"
	case "number":
		return "f64"
	case "integer":
		return "i64"
	case "boolean":
		return "bool"
	case "array":
		return "Vec<serde_json::Value>"
	case "object":
		return "serde_json::Map<String, serde_json::Value>"
	default:
		return "serde_json::Value"
	}
}

func toJavaName(s string) string {
	return toCamelCase(s)
}

func toJavaType(jsonType string) string {
	switch jsonType {
	case "string":
		return "String"
	case "number":
		return "Double"
	case "integer":
		return "Integer"
	case "boolean":
		return "Boolean"
	case "array":
		return "List<Object>"
	case "object":
		return "Map<String, Object>"
	default:
		return "Object"
	}
}

func jsonToGoType(schema map[string]interface{}) string {
	if typ, ok := schema["type"].(string); ok {
		return toGoType(typ)
	}
	return "interface{}"
}

func jsonToTSType(schema map[string]interface{}) string {
	if typ, ok := schema["type"].(string); ok {
		return toTSType(typ)
	}
	return "any"
}

func jsonToPyType(schema map[string]interface{}) string {
	if typ, ok := schema["type"].(string); ok {
		return toPyType(typ)
	}
	return "Any"
}

func jsonToRustType(schema map[string]interface{}) string {
	if typ, ok := schema["type"].(string); ok {
		return toRustType(typ)
	}
	return "serde_json::Value"
}

func jsonToJavaType(schema map[string]interface{}) string {
	if typ, ok := schema["type"].(string); ok {
		return toJavaType(typ)
	}
	return "Object"
}

func indent(spaces int, text string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func comment(text string) string {
	return "// " + strings.ReplaceAll(text, "\n", "\n// ")
}

func docComment(text string) string {
	return "// " + strings.ReplaceAll(text, "\n", "\n// ")
}

func formatCode(code, language string) string {
	switch language {
	case "go":
		if formatted, err := format.Source([]byte(code)); err == nil {
			return string(formatted)
		}
	}
	return code
}

func formatGoCode(code string) (string, error) {
	formatted, err := format.Source([]byte(code))
	if err != nil {
		return code, err
	}
	return string(formatted), nil
}

func formatTSCode(code string) (string, error) {
	// TODO: Implement TypeScript formatting
	return code, nil
}

func formatPyCode(code string) (string, error) {
	// TODO: Implement Python formatting
	return code, nil
}

func formatRustCode(code string) (string, error) {
	// TODO: Implement Rust formatting
	return code, nil
}

func formatJavaCode(code string) (string, error) {
	// TODO: Implement Java formatting
	return code, nil
}

// Utility template functions

func defaultValue(def interface{}, val interface{}) interface{} {
	if val == nil || val == "" {
		return def
	}
	return val
}

func coalesce(values ...interface{}) interface{} {
	for _, v := range values {
		if v != nil && v != "" {
			return v
		}
	}
	return nil
}

func dict(values ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(values); i += 2 {
		if i+1 < len(values) {
			key := fmt.Sprintf("%v", values[i])
			result[key] = values[i+1]
		}
	}
	return result
}

func list(values ...interface{}) []interface{} {
	return values
}

func rangeFunc(count int) []int {
	result := make([]int, count)
	for i := 0; i < count; i++ {
		result[i] = i
	}
	return result
}

func ifFunc(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func eq(a, b interface{}) bool {
	return a == b
}

func ne(a, b interface{}) bool {
	return a != b
}

func lt(a, b interface{}) bool {
	// TODO: Implement proper comparison
	return false
}

func le(a, b interface{}) bool {
	return lt(a, b) || eq(a, b)
}

func gt(a, b interface{}) bool {
	return !le(a, b)
}

func ge(a, b interface{}) bool {
	return !lt(a, b)
}

func and(values ...bool) bool {
	for _, v := range values {
		if !v {
			return false
		}
	}
	return true
}

func or(values ...bool) bool {
	for _, v := range values {
		if v {
			return true
		}
	}
	return false
}

func not(value bool) bool {
	return !value
}

func include(name string) string {
	// TODO: Implement template inclusion
	return ""
}

func templateFunc(name string, data interface{}) string {
	// TODO: Implement template function
	return ""
}

func goImports(types []Type) []string {
	imports := make(map[string]bool)
	for _, t := range types {
		for _, field := range t.Fields {
			if strings.Contains(field.Type, "time.Time") {
				imports["time"] = true
			}
			if strings.Contains(field.Type, "context.Context") {
				imports["context"] = true
			}
		}
	}
	
	var result []string
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

func tsImports(types []Type) []string {
	// TODO: Implement TypeScript imports
	return []string{}
}

func pyImports(types []Type) []string {
	// TODO: Implement Python imports
	return []string{}
}

func rustUses(types []Type) []string {
	// TODO: Implement Rust uses
	return []string{}
}

func javaImports(types []Type) []string {
	// TODO: Implement Java imports
	return []string{}
}

func generateStruct(typ Type, language string) string {
	// TODO: Implement struct generation
	return ""
}

func generateClass(typ Type, language string) string {
	// TODO: Implement class generation
	return ""
}

func generateEnum(typ Type, language string) string {
	// TODO: Implement enum generation
	return ""
}

func generateMethod(method Method, language string) string {
	// TODO: Implement method generation
	return ""
}

func generateTest(testCase TestCase, language string) string {
	// TODO: Implement test generation
	return ""
}