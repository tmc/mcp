// Package main implements mcptrace2gostruct, a tool to convert MCP trace data, JSON-RPC and JSON Schema to Go structs.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	packageName = flag.String("package", "main", "Package name for the generated Go code")
	structName  = flag.String("struct", "JSONRPCRequest", "Name of the struct to generate")
	outputFile  = flag.String("out", "", "Output file (default: stdout)")
	batchMode   = flag.Bool("batch", false, "Process multiple schema files in batch mode")
	inputDir    = flag.String("dir", "", "Directory containing schema files for batch mode")
	filePattern = flag.String("pattern", "*.json", "File pattern for batch mode")
)

// JSONRPCSchema represents a JSON-RPC schema document structure
type JSONRPCSchema struct {
	Title                string                       `json:"title,omitempty"`
	Description          string                       `json:"description,omitempty"`
	Type                 string                       `json:"type,omitempty"`
	Properties           map[string]JSONRPCSchemaType `json:"properties,omitempty"`
	Required             []string                     `json:"required,omitempty"`
	AdditionalProperties bool                         `json:"additionalProperties,omitempty"`
	Schema               string                       `json:"$schema,omitempty"`
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

func main() {
	flag.Parse()

	if *batchMode {
		if *inputDir == "" {
			fmt.Fprintf(os.Stderr, "Error: -dir is required in batch mode\n")
			os.Exit(1)
		}
		processBatch()
		return
	}

	// Single file mode
	var input []byte
	var err error

	if flag.NArg() > 0 {
		// Read from specified file
		input, err = ioutil.ReadFile(flag.Arg(0))
		fmt.Fprintf(os.Stderr, "Reading from file: %s\n", flag.Arg(0))
	} else {
		// Read from stdin
		input, err = ioutil.ReadAll(os.Stdin)
		fmt.Fprintf(os.Stderr, "Reading from stdin\n")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Converting to Go structs with package name: %s\n", *packageName)

	// Try the enhanced converter first
	output, err := convertToToolsModule(input, *packageName)
	if err == nil && output != "" {
		writeOutput(output)
		return
	}

	fmt.Fprintf(os.Stderr, "Enhanced converter failed: %v, trying fallback method\n", err)

	// Fall back to original processor if the enhanced converter fails
	processInput(input)
}

func processInput(input []byte) {
	// Try to parse as a JSON-RPC response
	var response struct {
		ID      interface{}     `json:"id"`
		Result  json.RawMessage `json:"result"`
		JSONRPC string          `json:"jsonrpc"`
	}

	if err := json.Unmarshal(input, &response); err == nil && response.Result != nil {
		// It's likely a JSON-RPC response, try to extract the result
		var result struct {
			Tools []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		}

		if err := json.Unmarshal(response.Result, &result); err == nil && len(result.Tools) > 0 {
			// Process tools as separate schemas
			schemas := make(map[string][]byte, len(result.Tools))
			for _, tool := range result.Tools {
				if len(tool.InputSchema) > 0 {
					structName := convertMethodToStructName(tool.Name) + "Input"

					// Parse directly into a map structure for easier handling
					var schemaMap map[string]interface{}
					if err := json.Unmarshal(tool.InputSchema, &schemaMap); err == nil {
						// Convert back to JSON to normalize
						normalizedSchema, err := json.Marshal(schemaMap)
						if err == nil {
							schemas[structName] = normalizedSchema
						} else {
							fmt.Fprintf(os.Stderr, "Error normalizing schema for %s: %v\n", structName, err)
						}
					} else {
						fmt.Fprintf(os.Stderr, "Error parsing schema for %s: %v\n", structName, err)
					}
				}
			}

			if len(schemas) > 0 {
				output, err := GenerateMultipleStructs(schemas, *packageName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating Go structs: %v\n", err)
					os.Exit(1)
				}
				writeOutput(output)
				return
			}
		}
	}

	// Try as a JSON-RPC request
	var jsonrpcRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
	}

	// Check if it's a JSON-RPC request
	isJSONRPC := false
	if err := json.Unmarshal(input, &jsonrpcRequest); err == nil {
		if jsonrpcRequest.JSONRPC != "" && jsonrpcRequest.Method != "" {
			isJSONRPC = true
		}
	}

	var output string
	var err error

	if isJSONRPC {
		// Process as a JSON-RPC request
		methodName := jsonrpcRequest.Method
		// Convert method name to a struct name (e.g. "tools/call" -> "ToolsCall")
		structPrefix := convertMethodToStructName(methodName)
		if *structName != "JSONRPCRequest" {
			structPrefix = *structName
		}

		output, err = ParseJSONRPCRequestToStruct(input, *packageName, structPrefix)
	} else {
		// Process as a regular JSON schema
		output, err = GenerateGoStruct(input, *packageName, *structName)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Go struct: %v\n", err)
		os.Exit(1)
	}

	writeOutput(output)
}

func processBatch() {
	pattern := filepath.Join(*inputDir, *filePattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No files found matching pattern: %s\n", pattern)
		os.Exit(1)
	}

	schemas := make(map[string][]byte)
	for _, file := range files {
		// Use the filename without extension as the struct name
		base := filepath.Base(file)
		structName := strings.TrimSuffix(base, filepath.Ext(base))
		structName = convertToStructName(structName)

		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
			continue
		}

		schemas[structName] = data
	}

	output, err := GenerateMultipleStructs(schemas, *packageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Go structs: %v\n", err)
		os.Exit(1)
	}

	writeOutput(output)
}

func writeOutput(output string) {
	if *outputFile == "" {
		// Write to stdout
		fmt.Print(output)
	} else {
		// Write to file
		err := ioutil.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", *outputFile)
	}
}

// Note: convertMethodToStructName is now defined in converter.go

func convertToStructName(name string) string {
	// Handle dashes, underscores, etc.
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	for i, part := range parts {
		parts[i] = strings.Title(part)
	}

	return strings.Join(parts, "")
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

	// Check if we need the time package
	needsTimePackage := false
	for _, schema := range schemas {
		var schemaObj map[string]interface{}
		if err := json.Unmarshal(schema, &schemaObj); err != nil {
			continue
		}

		// Check for date-time format in properties
		if props, ok := schemaObj["properties"].(map[string]interface{}); ok {
			for _, propValue := range props {
				if prop, ok := propValue.(map[string]interface{}); ok {
					if format, ok := prop["format"].(string); ok && format == "date-time" {
						needsTimePackage = true
						break
					}
				}
			}
		}

		if needsTimePackage {
			break
		}
	}

	// Add imports
	if needsTimePackage {
		buf.WriteString("import (\n\t\"encoding/json\"\n\t\"time\"\n)\n\n")
	} else {
		buf.WriteString("import (\n\t\"encoding/json\"\n)\n\n")
	}

	// Process schemas in alphabetical order
	var structNames []string
	for name := range schemas {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		schema := schemas[name]

		// Debug
		fmt.Fprintf(os.Stderr, "Processing schema for %s\n", name)

		var schemaObj JSONRPCSchema
		if err := json.Unmarshal(schema, &schemaObj); err != nil {
			// Try with the improved converter instead
			if output, err := convertToToolsModule(schema, packageName); err == nil {
				return output, nil
			}
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
		fmt.Fprintf(os.Stderr, "Error formatting generated code: %v\n", err)
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

	return GenerateGoStruct(schemaJSON, packageName, structPrefix+"Request")
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
