package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
)

// SchemaObject represents a JsonSchema object with its properties
type SchemaObject struct {
	Type                 string                  `json:"type,omitempty"`
	Description          string                  `json:"description,omitempty"`
	Properties           map[string]SchemaObject `json:"properties,omitempty"`
	Required             []string                `json:"required,omitempty"`
	Items                *SchemaObject           `json:"items,omitempty"`
	Ref                  string                  `json:"$ref,omitempty"`
	Format               string                  `json:"format,omitempty"`
	AdditionalProperties interface{}             `json:"additionalProperties,omitempty"`
	Enum                 []interface{}           `json:"enum,omitempty"`
}

// convertResponseToStructs converts a JSON response containing tool definitions to Go structs
func convertResponseToStructs(data []byte, packageName string) (string, error) {
	// Try to parse the input first - it could be either a direct JSON-RPC response
	// or it might be a JSON file containing multiple tool definitions

	// First try to parse as a JSON-RPC response
	var response struct {
		Result struct {
			Tools []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}

	// Try parsing as a response from a list_tools call
	if err := json.Unmarshal(data, &response); err != nil || len(response.Result.Tools) == 0 {
		// It's not a JSON-RPC response with tools, try parsing as direct schema file
		return convertSchemaFile(data, packageName)
	}

	fmt.Fprintf(os.Stderr, "Found %d tools in response\n", len(response.Result.Tools))

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buf.WriteString("import (\n\t\"encoding/json\"\n)\n\n")

	for _, tool := range response.Result.Tools {
		// Skip tools without input schema
		if len(tool.InputSchema) == 0 {
			fmt.Fprintf(os.Stderr, "Skipping tool %s - no input schema\n", tool.Name)
			continue
		}

		structName := convertMethodToStructName(tool.Name) + "Input"
		fmt.Fprintf(os.Stderr, "Generating struct for %s\n", structName)

		// Generate struct for this tool
		structDef, err := generateStructFromSchema(tool.InputSchema, structName, tool.Description)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating struct for %s: %v\n", structName, err)
			continue
		}

		buf.WriteString(structDef)
		buf.WriteString("\n\n")
	}

	// Format Go code
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format code: %v\n", err)
		fmt.Fprintf(os.Stderr, "Code: %s\n", buf.String())
		return buf.String(), fmt.Errorf("formatting error: %w", err)
	}

	return string(formattedCode), nil
}

// convertSchemaFile converts a file containing JSON Schema directly to Go structs
func convertSchemaFile(data []byte, packageName string) (string, error) {
	// Try parsing the file as a JSON Schema object
	var schemaObj SchemaObject
	if err := json.Unmarshal(data, &schemaObj); err != nil {
		return "", fmt.Errorf("failed to parse as JSON Schema: %w", err)
	}

	// If it's a root schema with type "object" and properties, process it directly
	if schemaObj.Type == "object" && len(schemaObj.Properties) > 0 {
		fmt.Fprintf(os.Stderr, "Processing as direct JSON Schema\n")

		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
		buf.WriteString("import (\n\t\"encoding/json\"\n)\n\n")

		// Use the filename or "Schema" as the struct name
		structName := "Schema"

		structDef, err := generateStructDefinition(schemaObj, structName, schemaObj.Description)
		if err != nil {
			return "", fmt.Errorf("failed to generate struct: %w", err)
		}

		buf.WriteString(structDef)

		// Format Go code
		formattedCode, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to format code: %v\n", err)
			fmt.Fprintf(os.Stderr, "Code: %s\n", buf.String())
			return buf.String(), fmt.Errorf("formatting error: %w", err)
		}

		return string(formattedCode), nil
	}

	// Try parsing as a JSON object that might have tools as top-level keys
	var toolsMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &toolsMap); err != nil {
		return "", fmt.Errorf("failed to parse as tools map: %w", err)
	}

	// If we have multiple top-level keys, treat them as tool definitions
	if len(toolsMap) > 0 {
		fmt.Fprintf(os.Stderr, "Processing as tools map with %d entries\n", len(toolsMap))

		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
		buf.WriteString("import (\n\t\"encoding/json\"\n)\n\n")

		// Sort keys for consistent output
		var toolNames []string
		for name := range toolsMap {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)

		for _, toolName := range toolNames {
			schema := toolsMap[toolName]

			// Parse the schema
			var schemaObj SchemaObject
			if err := json.Unmarshal(schema, &schemaObj); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing schema for %s: %v\n", toolName, err)
				continue
			}

			// Generate a struct name from the tool name
			structName := toGoName(toolName)

			// Generate struct definition
			structDef, err := generateStructDefinition(schemaObj, structName, schemaObj.Description)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating struct for %s: %v\n", structName, err)
				continue
			}

			buf.WriteString(structDef)
			buf.WriteString("\n\n")
		}

		// Format Go code
		formattedCode, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to format code: %v\n", err)
			fmt.Fprintf(os.Stderr, "Code: %s\n", buf.String())
			return buf.String(), fmt.Errorf("formatting error: %w", err)
		}

		return string(formattedCode), nil
	}

	return "", fmt.Errorf("could not parse input as a valid JSON Schema")
}

// generateStructFromSchema generates a Go struct from a JSON Schema
func generateStructFromSchema(schemaData []byte, structName, description string) (string, error) {
	var schemaObj SchemaObject
	if err := json.Unmarshal(schemaData, &schemaObj); err != nil {
		return "", fmt.Errorf("failed to parse schema: %w", err)
	}

	return generateStructDefinition(schemaObj, structName, description)
}

// generateStructDefinition generates a Go struct definition from a SchemaObject
func generateStructDefinition(schema SchemaObject, structName, description string) (string, error) {
	var buf bytes.Buffer

	// Add struct comment
	if description != "" {
		buf.WriteString(fmt.Sprintf("// %s - %s\n", structName, description))
	} else {
		buf.WriteString(fmt.Sprintf("// %s represents a schema object\n", structName))
	}

	buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// If it's not an object or has no properties, create an empty struct
	if schema.Type != "object" || len(schema.Properties) == 0 {
		buf.WriteString("}\n")
		return buf.String(), nil
	}

	// Create a map of required fields
	requiredFields := make(map[string]bool)
	for _, field := range schema.Required {
		requiredFields[field] = true
	}

	// Sort property names for consistent output
	var propNames []string
	for propName := range schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)

	for _, propName := range propNames {
		prop := schema.Properties[propName]
		fieldName := toGoName(propName)

		// Add field description as comment
		if prop.Description != "" {
			buf.WriteString(fmt.Sprintf("\t// %s\n", prop.Description))
		}

		// Determine field type
		fieldType := schemaTypeToGoType(prop)

		// Add JSON tag, with omitempty for non-required fields
		jsonTag := propName
		if !requiredFields[propName] {
			jsonTag += ",omitempty"
		}

		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, jsonTag))
	}

	buf.WriteString("}\n")
	return buf.String(), nil
}

// schemaTypeToGoType converts a JSON Schema type to a Go type
func schemaTypeToGoType(schema SchemaObject) string {
	// Handle references first
	if schema.Ref != "" {
		parts := strings.Split(schema.Ref, "/")
		refType := parts[len(parts)-1]
		// Convert reference to Go struct name
		return toGoName(refType)
	}

	switch schema.Type {
	case "string":
		if schema.Format == "date-time" {
			return "time.Time"
		}
		if schema.Format == "byte" || schema.Format == "binary" {
			return "[]byte"
		}
		if len(schema.Enum) > 0 {
			return "string" // Enum types are still strings in Go
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
			itemType := schemaTypeToGoType(*schema.Items)
			return "[]" + itemType
		}
		return "[]interface{}"

	case "object":
		// For embedded objects, use map[string]interface{} for simplicity
		// A more sophisticated solution would generate nested structs
		if len(schema.Properties) > 0 {
			return "map[string]interface{}"
		}

		// Handle additionalProperties
		if schema.AdditionalProperties != nil {
			// If additionalProperties is a boolean, return a generic map
			if _, ok := schema.AdditionalProperties.(bool); ok {
				return "map[string]interface{}"
			}

			// If additionalProperties is an object, try to extract the type
			if propObj, ok := schema.AdditionalProperties.(map[string]interface{}); ok {
				if typeVal, ok := propObj["type"].(string); ok {
					switch typeVal {
					case "string":
						return "map[string]string"
					case "integer":
						return "map[string]int"
					case "number":
						return "map[string]float64"
					case "boolean":
						return "map[string]bool"
					}
				}
			}

			// Default map type
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

// toGoName converts a JSON property name to a Go field name
func toGoName(s string) string {
	// Special case for ID
	if s == "id" {
		return "ID"
	}

	// Handle common JSON-to-Go mappings
	switch strings.ToLower(s) {
	case "url":
		return "URL"
	case "uri":
		return "URI"
	case "ip":
		return "IP"
	case "html":
		return "HTML"
	case "json":
		return "JSON"
	case "xml":
		return "XML"
	case "http":
		return "HTTP"
	case "https":
		return "HTTPS"
	}

	// Convert to camel case
	var result strings.Builder
	nextUpper := true

	for _, c := range s {
		if !isAlphanumeric(c) {
			nextUpper = true
			continue
		}

		if nextUpper {
			result.WriteRune(toUpper(c))
			nextUpper = false
		} else {
			result.WriteRune(c)
		}
	}

	return result.String()
}

// isAlphanumeric checks if a character is a letter or number
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// toUpper converts a rune to uppercase
func toUpper(c rune) rune {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// convertMethodToStructName converts a method name (like "tools/call") to a Go struct name (like "ToolsCall")
func convertMethodToStructName(method string) string {
	parts := strings.Split(method, "/")
	for i, part := range parts {
		parts[i] = toGoName(part)
	}
	return strings.Join(parts, "")
}

// convertToToolsModule is the main entry point for the converter
func convertToToolsModule(data []byte, packageName string) (string, error) {
	output, err := convertResponseToStructs(data, packageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Conversion failed: %v\n", err)
		return "", err
	}

	fmt.Fprintf(os.Stderr, "Conversion successful\n")
	return output, nil
}
