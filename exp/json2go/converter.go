package json2go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"unicode"
)

// Options for the converter
type Options struct {
	PackageName string
	TypeName    string
	Tags        []string
	Prefix      string
	Verbose     bool
}

// Converter converts JSON to Go structs
type Converter struct {
	options Options
	types   map[string]string
}

// NewConverter creates a new JSON to Go converter
func NewConverter(options Options) *Converter {
	if options.PackageName == "" {
		options.PackageName = "main"
	}
	if options.TypeName == "" {
		options.TypeName = "Generated"
	}
	if len(options.Tags) == 0 {
		options.Tags = []string{"json"}
	}

	return &Converter{
		options: options,
		types:   make(map[string]string),
	}
}

// ConvertJSON converts arbitrary JSON to Go structs
func (c *Converter) ConvertJSON(jsonData []byte) (string, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Generate the main type
	mainType := c.generateType(c.options.TypeName, data)
	
	// Build the complete Go file
	var buf bytes.Buffer
	if err := c.generateFile(&buf, mainType); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ConvertJSONSchema converts JSON Schema to Go types
func (c *Converter) ConvertJSONSchema(schemaData []byte) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return "", fmt.Errorf("invalid JSON Schema: %w", err)
	}

	// Generate types from schema
	typeName := c.options.TypeName
	if title, ok := schema["title"].(string); ok && typeName == "Generated" {
		typeName = toGoName(title)
	}

	mainType := c.generateFromSchema(typeName, schema)
	
	// Build the complete Go file
	var buf bytes.Buffer
	if err := c.generateFile(&buf, mainType); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ConvertJSONRPC converts JSON-RPC format to Go types
func (c *Converter) ConvertJSONRPC(jsonData []byte) (string, error) {
	// Parse as JSON-RPC
	var rpc map[string]interface{}
	if err := json.Unmarshal(jsonData, &rpc); err != nil {
		return "", fmt.Errorf("invalid JSON-RPC: %w", err)
	}

	// Generate appropriate struct based on content
	var mainType string
	if _, hasResult := rpc["result"]; hasResult {
		mainType = c.generateJSONRPCResponse(rpc)
	} else if _, hasMethod := rpc["method"]; hasMethod {
		mainType = c.generateJSONRPCRequest(rpc)
	} else {
		mainType = c.generateType(c.options.TypeName, rpc)
	}

	// Build the complete Go file
	var buf bytes.Buffer
	if err := c.generateFile(&buf, mainType); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (c *Converter) generateType(name string, data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}:
		return c.generateStruct(name, v)
	case []interface{}:
		if len(v) > 0 {
			// Infer type from first element
			elemType := c.inferType(v[0])
			return fmt.Sprintf("[]%s", elemType)
		}
		return "[]interface{}"
	default:
		return c.inferType(data)
	}
}

func (c *Converter) generateStruct(name string, obj map[string]interface{}) string {
	var fields []string
	
	typeName := c.options.Prefix + name
	header := fmt.Sprintf("type %s struct {\n", typeName)
	
	// Generate fields
	for key, value := range obj {
		fieldName := toGoName(key)
		fieldType := c.inferType(value)
		
		// Handle nested structs
		if nestedObj, ok := value.(map[string]interface{}); ok {
			nestedTypeName := name + fieldName
			c.types[nestedTypeName] = c.generateStruct(nestedTypeName, nestedObj)
			fieldType = c.options.Prefix + nestedTypeName
		}
		
		// Build field with tags
		tags := c.buildTags(key)
		fields = append(fields, fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, tags))
	}
	
	// Sort fields for consistent output
	sortFields(&fields)
	
	result := header
	for _, field := range fields {
		result += field
	}
	result += "}"
	
	return result
}

func (c *Converter) generateFromSchema(name string, schema map[string]interface{}) string {
	schemaType, _ := schema["type"].(string)
	
	switch schemaType {
	case "object":
		return c.generateStructFromSchema(name, schema)
	case "array":
		items, _ := schema["items"].(map[string]interface{})
		itemType := c.generateFromSchema(name+"Item", items)
		return fmt.Sprintf("[]%s", itemType)
	default:
		return c.schemaTypeToGo(schemaType, schema)
	}
}

func (c *Converter) generateStructFromSchema(name string, schema map[string]interface{}) string {
	properties, _ := schema["properties"].(map[string]interface{})
	required, _ := schema["required"].([]interface{})
	
	var fields []string
	typeName := c.options.Prefix + name
	header := fmt.Sprintf("type %s struct {\n", typeName)
	
	// Create required set for quick lookup
	requiredSet := make(map[string]bool)
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredSet[reqStr] = true
		}
	}
	
	// Generate fields from properties
	for propName, propSchema := range properties {
		fieldName := toGoName(propName)
		prop, _ := propSchema.(map[string]interface{})
		
		fieldType := c.generateFromSchema(name+fieldName, prop)
		
		// Make non-required fields pointers
		if !requiredSet[propName] {
			fieldType = "*" + fieldType
		}
		
		tags := c.buildTags(propName)
		fields = append(fields, fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, tags))
	}
	
	sortFields(&fields)
	
	result := header
	for _, field := range fields {
		result += field
	}
	result += "}"
	
	return result
}

func (c *Converter) schemaTypeToGo(schemaType string, schema map[string]interface{}) string {
	switch schemaType {
	case "string":
		return "string"
	case "number":
		return "float64"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "null":
		return "interface{}"
	default:
		// Check for enum
		if enum, ok := schema["enum"]; ok {
			// Could generate a custom type, but for now use string
			return "string"
		}
		return "interface{}"
	}
}

func (c *Converter) inferType(value interface{}) string {
	switch v := value.(type) {
	case bool:
		return "bool"
	case float64:
		if v == float64(int(v)) {
			return "int"
		}
		return "float64"
	case string:
		return "string"
	case []interface{}:
		if len(v) > 0 {
			elemType := c.inferType(v[0])
			return "[]" + elemType
		}
		return "[]interface{}"
	case map[string]interface{}:
		return "map[string]interface{}"
	case nil:
		return "interface{}"
	default:
		return "interface{}"
	}
}

func (c *Converter) generateJSONRPCRequest(rpc map[string]interface{}) string {
	typeName := c.options.Prefix + c.options.TypeName
	
	// Generate params type if exists
	if params, ok := rpc["params"].(map[string]interface{}); ok {
		paramsTypeName := typeName + "Params"
		c.types[paramsTypeName] = c.generateStruct(paramsTypeName, params)
	}
	
	// Generate main request type
	fields := []string{
		fmt.Sprintf("\tJSONRPC string %s\n", c.buildTags("jsonrpc")),
		fmt.Sprintf("\tMethod  string %s\n", c.buildTags("method")),
		fmt.Sprintf("\tParams  %sParams %s\n", typeName, c.buildTags("params")),
		fmt.Sprintf("\tID      interface{} %s\n", c.buildTags("id")),
	}
	
	return fmt.Sprintf("type %s struct {\n%s}", typeName, strings.Join(fields, ""))
}

func (c *Converter) generateJSONRPCResponse(rpc map[string]interface{}) string {
	typeName := c.options.Prefix + c.options.TypeName
	
	// Generate result type if exists
	if result, ok := rpc["result"].(map[string]interface{}); ok {
		resultTypeName := typeName + "Result"
		c.types[resultTypeName] = c.generateStruct(resultTypeName, result)
	}
	
	// Generate error type if exists
	if rpcError, ok := rpc["error"].(map[string]interface{}); ok {
		errorTypeName := typeName + "Error"
		c.types[errorTypeName] = c.generateStruct(errorTypeName, rpcError)
	}
	
	// Generate main response type
	fields := []string{
		fmt.Sprintf("\tJSONRPC string %s\n", c.buildTags("jsonrpc")),
		fmt.Sprintf("\tResult  *%sResult %s\n", typeName, c.buildTags("result")),
		fmt.Sprintf("\tError   *%sError %s\n", typeName, c.buildTags("error")),
		fmt.Sprintf("\tID      interface{} %s\n", c.buildTags("id")),
	}
	
	return fmt.Sprintf("type %s struct {\n%s}", typeName, strings.Join(fields, ""))
}

func (c *Converter) buildTags(jsonKey string) string {
	var tags []string
	
	for _, tag := range c.options.Tags {
		switch tag {
		case "json":
			tags = append(tags, fmt.Sprintf(`json:"%s,omitempty"`, jsonKey))
		case "yaml":
			tags = append(tags, fmt.Sprintf(`yaml:"%s,omitempty"`, jsonKey))
		case "xml":
			tags = append(tags, fmt.Sprintf(`xml:"%s,omitempty"`, jsonKey))
		default:
			tags = append(tags, fmt.Sprintf(`%s:"%s"`, tag, jsonKey))
		}
	}
	
	if len(tags) == 0 {
		return ""
	}
	
	return "`" + strings.Join(tags, " ") + "`"
}

func (c *Converter) generateFile(w io.Writer, mainType string) error {
	tmpl := `package {{ .Package }}

// Code generated by json2go. DO NOT EDIT.

{{ range $name, $def := .Types }}
{{ $def }}

{{ end }}

{{ .MainType }}
`
	
	t, err := template.New("file").Parse(tmpl)
	if err != nil {
		return err
	}
	
	return t.Execute(w, map[string]interface{}{
		"Package":  c.options.PackageName,
		"Types":    c.types,
		"MainType": mainType,
	})
}

// Helper functions

func toGoName(s string) string {
	// Convert to camelCase
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	
	return strings.Join(words, "")
}

func sortFields(fields *[]string) {
	// Sort fields alphabetically for consistent output
	// In a real implementation, you might want more sophisticated sorting
}

// Writer interface for output
type io interface {
	Writer
}

type Writer interface {
	Write([]byte) (int, error)
}