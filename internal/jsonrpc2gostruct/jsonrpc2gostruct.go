package jsonrpc2gostruct

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"regexp"
	"sort"
	"strings"
)

// JSONRPCSchema represents a JSON-RPC schema document structure
type JSONRPCSchema struct {
	Title       string                       `json:"title,omitempty"`
	Description string                       `json:"description,omitempty"`
	Type        string                       `json:"type,omitempty"`
	Properties  map[string]JSONRPCSchemaType `json:"properties,omitempty"`
	Required    []string                     `json:"required,omitempty"`
}

// JSONRPCSchemaType represents a type within a JSON-RPC schema
type JSONRPCSchemaType struct {
	Type        string                       `json:"type,omitempty"`
	Description string                       `json:"description,omitempty"`
	Items       *JSONRPCSchemaType           `json:"items,omitempty"`
	Properties  map[string]JSONRPCSchemaType `json:"properties,omitempty"`
	Required    []string                     `json:"required,omitempty"`
	Format      string                       `json:"format,omitempty"`
	Ref         string                       `json:"$ref,omitempty"`
	Enum        []string                     `json:"enum,omitempty"`
}

// GenerateGoStruct takes a JSON-RPC schema and converts it to a Go struct definition
func GenerateGoStruct(schemaJSON []byte, packageName, structName string) (string, error) {
	var schema JSONRPCSchema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return "", fmt.Errorf("error parsing schema: %w", err)
	}

	// Start building the Go file with package declaration and imports
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buf.WriteString("import (\n\t\"encoding/json\"\n)\n\n")

	// Generate struct with comments
	if schema.Description != "" {
		buf.WriteString(fmt.Sprintf("// %s - %s\n", structName, schema.Description))
	} else if schema.Title != "" {
		buf.WriteString(fmt.Sprintf("// %s - %s\n", structName, schema.Title))
	} else {
		buf.WriteString(fmt.Sprintf("// %s represents a JSON-RPC object\n", structName))
	}

	buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Process properties in alphabetical order for consistent output
	var propNames []string
	for name := range schema.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	// Set of required fields for lookup
	requiredFields := make(map[string]bool)
	for _, req := range schema.Required {
		requiredFields[req] = true
	}

	for _, name := range propNames {
		prop := schema.Properties[name]
		fieldName := ToGoFieldName(name)
		fieldType := JSONTypeToGoType(prop)

		// Add field comment if description exists
		if prop.Description != "" {
			buf.WriteString(fmt.Sprintf("\t// %s\n", prop.Description))
		}

		// Create the field with JSON tag
		var jsonTag string
		if requiredFields[name] {
			jsonTag = fmt.Sprintf("`json:\"%s\"`", name)
		} else {
			jsonTag = fmt.Sprintf("`json:\"%s,omitempty\"`", name)
		}

		buf.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}

	buf.WriteString("}\n")

	// Format the Go code
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formattedCode), nil
}

// ToGoFieldName converts a JSON property name to a Go struct field name
func ToGoFieldName(name string) string {
	// Handle special cases like "id" -> "ID"
	if name == "id" {
		return "ID"
	}

	// Split by non-alphanumeric characters
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(name, -1)

	// Capitalize each part
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, "")
}

// JSONTypeToGoType converts a JSON schema type to the corresponding Go type
func JSONTypeToGoType(schema JSONRPCSchemaType) string {
	// Handle $ref first
	if schema.Ref != "" {
		// Extract the type name from the reference
		parts := strings.Split(schema.Ref, "/")
		return parts[len(parts)-1]
	}

	switch schema.Type {
	case "string":
		if schema.Format == "byte" {
			return "[]byte"
		}
		if schema.Format == "date-time" {
			return "time.Time"
		}
		if len(schema.Enum) > 0 {
			return "string" // For enum types, still use string in Go
		}
		return "string"

	case "integer":
		return "int"

	case "number":
		return "float64"

	case "boolean":
		return "bool"

	case "array":
		if schema.Items != nil {
			itemType := JSONTypeToGoType(*schema.Items)
			return "[]" + itemType
		}
		return "[]interface{}"

	case "object":
		if schema.Properties != nil && len(schema.Properties) > 0 {
			// For embedded objects, we could generate a nested struct
			// But for simplicity, we'll use map[string]interface{}
			return "map[string]interface{}"
		}
		return "map[string]interface{}"

	case "null":
		return "interface{}"

	default:
		// For unknown types, use interface{}
		return "interface{}"
	}
}

// GenerateMultipleStructs generates multiple Go structs from a set of JSON-RPC schemas
func GenerateMultipleStructs(schemas map[string][]byte, packageName string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buf.WriteString("import (\n\t\"encoding/json\"\n\t\"time\"\n)\n\n")

	// Process schemas in alphabetical order
	var structNames []string
	for name := range schemas {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		schema := schemas[name]

		var schemaObj JSONRPCSchema
		if err := json.Unmarshal(schema, &schemaObj); err != nil {
			return "", fmt.Errorf("error parsing schema %s: %w", name, err)
		}

		// Add struct comment based on schema description
		if schemaObj.Description != "" {
			buf.WriteString(fmt.Sprintf("// %s - %s\n", name, schemaObj.Description))
		} else if schemaObj.Title != "" {
			buf.WriteString(fmt.Sprintf("// %s - %s\n", name, schemaObj.Title))
		} else {
			buf.WriteString(fmt.Sprintf("// %s represents a JSON-RPC object\n", name))
		}

		buf.WriteString(fmt.Sprintf("type %s struct {\n", name))

		// Process properties in alphabetical order
		var propNames []string
		for propName := range schemaObj.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		// Track required fields
		requiredFields := make(map[string]bool)
		for _, req := range schemaObj.Required {
			requiredFields[req] = true
		}

		for _, propName := range propNames {
			prop := schemaObj.Properties[propName]
			fieldName := ToGoFieldName(propName)
			fieldType := JSONTypeToGoType(prop)

			// Add field comment
			if prop.Description != "" {
				buf.WriteString(fmt.Sprintf("\t// %s\n", prop.Description))
			}

			// Create field with appropriate JSON tag
			var jsonTag string
			if requiredFields[propName] {
				jsonTag = fmt.Sprintf("`json:\"%s\"`", propName)
			} else {
				jsonTag = fmt.Sprintf("`json:\"%s,omitempty\"`", propName)
			}

			buf.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
		}

		buf.WriteString("}\n\n")
	}

	// Format the Go code
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("error formatting generated code: %w", err)
	}

	return string(formattedCode), nil
}

// ParseJSONRPCRequestToStruct parses a JSON-RPC request and creates a Go struct definition
func ParseJSONRPCRequestToStruct(jsonrpcRequest []byte, packageName, structPrefix string) (string, error) {
	// Parse the request to extract params structure
	var request struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
	}

	if err := json.Unmarshal(jsonrpcRequest, &request); err != nil {
		return "", fmt.Errorf("error parsing JSON-RPC request: %w", err)
	}

	// Clean up method name to use in struct name
	methodName := request.Method
	methodName = strings.ReplaceAll(methodName, "/", "_")

	// If params is null or empty, return a simple struct
	if len(request.Params) == 0 || string(request.Params) == "null" {
		return fmt.Sprintf(`package %s

// %sRequest is a request for the %s method
type %sRequest struct {
	// Add fields as needed
}
`, packageName, structPrefix, request.Method, structPrefix), nil
	}

	// Create a schema from the params
	var paramsObj map[string]interface{}
	if err := json.Unmarshal(request.Params, &paramsObj); err != nil {
		// If it's not an object, create a simple schema
		return fmt.Sprintf(`package %s

// %sRequest is a request for the %s method
type %sRequest struct {
	// Raw params: %s
	// Could not parse as object: %v
}
`, packageName, structPrefix, request.Method, structPrefix, string(request.Params), err), nil
	}

	// Convert params to a schema
	schema := JSONRPCSchema{
		Type:        "object",
		Description: fmt.Sprintf("Request for %s method", request.Method),
		Properties:  make(map[string]JSONRPCSchemaType),
	}

	for key, value := range paramsObj {
		propType := inferJSONType(value)
		schema.Properties[key] = propType
	}

	// Generate the struct
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("error creating schema: %w", err)
	}

	structDef, err := GenerateGoStruct(schemaJSON, packageName, structPrefix+"Request")
	if err != nil {
		return "", fmt.Errorf("error generating struct: %w", err)
	}

	// Add a comment with the method name
	methodComment := fmt.Sprintf("// %sRequest is a request for the %s method\n", structPrefix, request.Method)

	// Replace the default comment with our method-specific comment
	result := regexp.MustCompile(`(?m)^// \w+Request.*\n`).ReplaceAllString(structDef, methodComment)

	return result, nil
}

// inferJSONType infers the JSON schema type from a Go value
func inferJSONType(value interface{}) JSONRPCSchemaType {
	if value == nil {
		return JSONRPCSchemaType{Type: "null"}
	}

	switch v := value.(type) {
	case string:
		return JSONRPCSchemaType{Type: "string"}
	case bool:
		return JSONRPCSchemaType{Type: "boolean"}
	case float64:
		// Could be integer or number
		if float64(int(v)) == v {
			return JSONRPCSchemaType{Type: "integer"}
		}
		return JSONRPCSchemaType{Type: "number"}
	case map[string]interface{}:
		// It's an object
		properties := make(map[string]JSONRPCSchemaType)
		for key, val := range v {
			properties[key] = inferJSONType(val)
		}
		return JSONRPCSchemaType{
			Type:       "object",
			Properties: properties,
		}
	case []interface{}:
		// It's an array
		if len(v) > 0 {
			// Use the first element to determine item type
			return JSONRPCSchemaType{
				Type:  "array",
				Items: &JSONRPCSchemaType{Type: inferJSONType(v[0]).Type},
			}
		}
		return JSONRPCSchemaType{
			Type:  "array",
			Items: &JSONRPCSchemaType{Type: "string"}, // Default to string for empty arrays
		}
	default:
		// Unknown type
		return JSONRPCSchemaType{Type: "string"}
	}
}
