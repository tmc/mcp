// Package codegen - Schema analysis and type generation
package codegen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/config"
	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/templates"
)

// GenerateTypeName generates a type name from a tool name and suffix
func (a *SchemaAnalyzer) GenerateTypeName(toolName, suffix string) string {
	baseName := a.toTypeName(toolName)
	return baseName + suffix
}

// SchemaToTypes converts a JSON schema to template types
func (a *SchemaAnalyzer) SchemaToTypes(schema map[string]interface{}) []templates.Type {
	var types []templates.Type

	// Handle root schema
	if rootType := a.schemaToType("Root", schema); rootType != nil {
		types = append(types, *rootType)
	}

	// Handle definitions
	if definitions, ok := schema["definitions"].(map[string]interface{}); ok {
		for name, def := range definitions {
			if defMap, ok := def.(map[string]interface{}); ok {
				if defType := a.schemaToType(a.toTypeName(name), defMap); defType != nil {
					types = append(types, *defType)
				}
			}
		}
	}

	return types
}

// schemaToType converts a single schema to a template type
func (a *SchemaAnalyzer) schemaToType(name string, schema map[string]interface{}) *templates.Type {
	typ := &templates.Type{
		Name:        name,
		Description: a.getDescription(schema),
		Fields:      []templates.Field{},
		Methods:     []templates.Method{},
		Annotations: []string{},
	}

	// Handle object type
	if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			required := a.getRequiredFields(schema)

			for propName, propSchema := range properties {
				if propMap, ok := propSchema.(map[string]interface{}); ok {
					field := a.schemaToField(propName, propMap, required)
					typ.Fields = append(typ.Fields, field)
				}
			}
		}
	}

	// Handle enum type
	if enum, ok := schema["enum"].([]interface{}); ok {
		typ.Annotations = append(typ.Annotations, "enum")
		for _, value := range enum {
			// For enums, we might want to generate constants instead
			fieldName := a.toFieldName(fmt.Sprintf("%v", value))
			field := templates.Field{
				Name:        fieldName,
				Type:        a.getEnumValueType(value),
				Description: fmt.Sprintf("Enum value: %v", value),
				Optional:    false,
			}
			typ.Fields = append(typ.Fields, field)
		}
	}

	return typ
}

// schemaToField converts a schema property to a template field
func (a *SchemaAnalyzer) schemaToField(name string, schema map[string]interface{}, required []string) templates.Field {
	field := templates.Field{
		Name:        a.toFieldName(name),
		Type:        a.schemaToFieldType(schema),
		Description: a.getDescription(schema),
		Optional:    !a.isRequired(name, required),
		Tags:        make(map[string]string),
	}

	// Add language-specific tags
	field.Tags["json"] = name
	if field.Optional {
		field.Tags["json"] = name + ",omitempty"
	}

	// Add validation tags
	if min, ok := schema["minimum"].(float64); ok {
		field.Tags["validate"] = fmt.Sprintf("min=%v", min)
	}
	if max, ok := schema["maximum"].(float64); ok {
		if existing, ok := field.Tags["validate"]; ok {
			field.Tags["validate"] = existing + fmt.Sprintf(",max=%v", max)
		} else {
			field.Tags["validate"] = fmt.Sprintf("max=%v", max)
		}
	}
	if minLen, ok := schema["minLength"].(float64); ok {
		field.Tags["validate"] = fmt.Sprintf("minlen=%v", int(minLen))
	}
	if maxLen, ok := schema["maxLength"].(float64); ok {
		if existing, ok := field.Tags["validate"]; ok {
			field.Tags["validate"] = existing + fmt.Sprintf(",maxlen=%v", int(maxLen))
		} else {
			field.Tags["validate"] = fmt.Sprintf("maxlen=%v", int(maxLen))
		}
	}
	if pattern, ok := schema["pattern"].(string); ok {
		field.Tags["validate"] = fmt.Sprintf("regexp=%s", pattern)
	}

	return field
}

// schemaToFieldType converts a schema to a field type string
func (a *SchemaAnalyzer) schemaToFieldType(schema map[string]interface{}) string {
	switch a.config.Language {
	case "go":
		return a.schemaToGoType(schema)
	case "typescript":
		return a.schemaToTSType(schema)
	case "python":
		return a.schemaToPyType(schema)
	case "rust":
		return a.schemaToRustType(schema)
	case "java":
		return a.schemaToJavaType(schema)
	default:
		return "interface{}"
	}
}

// Language-specific type conversion methods

func (a *SchemaAnalyzer) schemaToGoType(schema map[string]interface{}) string {
	if ref, ok := schema["$ref"].(string); ok {
		return a.refToGoType(ref)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "string":
			if format, ok := schema["format"].(string); ok {
				switch format {
				case "date-time":
					return "time.Time"
				case "uri":
					return "string"
				case "email":
					return "string"
				}
			}
			return "string"
		case "integer":
			return "int"
		case "number":
			return "float64"
		case "boolean":
			return "bool"
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemType := a.schemaToGoType(items)
				return "[]" + itemType
			}
			return "[]interface{}"
		case "object":
			if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				// Generate inline struct for simple objects
				return a.generateInlineGoStruct(schema)
			}
			return "map[string]interface{}"
		}
	}

	// Handle oneOf, anyOf, allOf
	if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		return a.handleGoUnionType(oneOf)
	}
	if anyOf, ok := schema["anyOf"].([]interface{}); ok {
		return a.handleGoUnionType(anyOf)
	}
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		return a.handleGoIntersectionType(allOf)
	}

	return "interface{}"
}

func (a *SchemaAnalyzer) schemaToTSType(schema map[string]interface{}) string {
	if ref, ok := schema["$ref"].(string); ok {
		return a.refToTSType(ref)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "string":
			if enum, ok := schema["enum"].([]interface{}); ok {
				var values []string
				for _, v := range enum {
					values = append(values, fmt.Sprintf("'%v'", v))
				}
				return strings.Join(values, " | ")
			}
			return "string"
		case "integer", "number":
			return "number"
		case "boolean":
			return "boolean"
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemType := a.schemaToTSType(items)
				return itemType + "[]"
			}
			return "any[]"
		case "object":
			if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				return a.generateInlineTSInterface(schema)
			}
			return "Record<string, any>"
		}
	}

	// Handle oneOf, anyOf, allOf
	if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		return a.handleTSUnionType(oneOf)
	}
	if anyOf, ok := schema["anyOf"].([]interface{}); ok {
		return a.handleTSUnionType(anyOf)
	}
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		return a.handleTSIntersectionType(allOf)
	}

	return "any"
}

func (a *SchemaAnalyzer) schemaToPyType(schema map[string]interface{}) string {
	if ref, ok := schema["$ref"].(string); ok {
		return a.refToPyType(ref)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "string":
			if enum, ok := schema["enum"].([]interface{}); ok {
				// Return Literal type for enums
				var values []string
				for _, v := range enum {
					values = append(values, fmt.Sprintf("'%v'", v))
				}
				return "Literal[" + strings.Join(values, ", ") + "]"
			}
			return "str"
		case "integer":
			return "int"
		case "number":
			return "float"
		case "boolean":
			return "bool"
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemType := a.schemaToPyType(items)
				return "List[" + itemType + "]"
			}
			return "List[Any]"
		case "object":
			if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				return a.generateInlinePyTypedDict(schema)
			}
			return "Dict[str, Any]"
		}
	}

	// Handle oneOf, anyOf, allOf
	if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		return a.handlePyUnionType(oneOf)
	}
	if anyOf, ok := schema["anyOf"].([]interface{}); ok {
		return a.handlePyUnionType(anyOf)
	}
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		return a.handlePyIntersectionType(allOf)
	}

	return "Any"
}

func (a *SchemaAnalyzer) schemaToRustType(schema map[string]interface{}) string {
	if ref, ok := schema["$ref"].(string); ok {
		return a.refToRustType(ref)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "string":
			return "String"
		case "integer":
			return "i64"
		case "number":
			return "f64"
		case "boolean":
			return "bool"
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemType := a.schemaToRustType(items)
				return "Vec<" + itemType + ">"
			}
			return "Vec<serde_json::Value>"
		case "object":
			if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				return a.generateInlineRustStruct(schema)
			}
			return "serde_json::Map<String, serde_json::Value>"
		}
	}

	// Handle oneOf, anyOf, allOf
	if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		return a.handleRustUnionType(oneOf)
	}
	if anyOf, ok := schema["anyOf"].([]interface{}); ok {
		return a.handleRustUnionType(anyOf)
	}
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		return a.handleRustIntersectionType(allOf)
	}

	return "serde_json::Value"
}

func (a *SchemaAnalyzer) schemaToJavaType(schema map[string]interface{}) string {
	if ref, ok := schema["$ref"].(string); ok {
		return a.refToJavaType(ref)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "string":
			return "String"
		case "integer":
			return "Integer"
		case "number":
			return "Double"
		case "boolean":
			return "Boolean"
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemType := a.schemaToJavaType(items)
				return "List<" + itemType + ">"
			}
			return "List<Object>"
		case "object":
			if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				return a.generateInlineJavaClass(schema)
			}
			return "Map<String, Object>"
		}
	}

	// Handle oneOf, anyOf, allOf
	if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		return a.handleJavaUnionType(oneOf)
	}
	if anyOf, ok := schema["anyOf"].([]interface{}); ok {
		return a.handleJavaUnionType(anyOf)
	}
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		return a.handleJavaIntersectionType(allOf)
	}

	return "Object"
}

// Helper methods

func (a *SchemaAnalyzer) getDescription(schema map[string]interface{}) string {
	if desc, ok := schema["description"].(string); ok {
		return desc
	}
	return ""
}

func (a *SchemaAnalyzer) getRequiredFields(schema map[string]interface{}) []string {
	if required, ok := schema["required"].([]interface{}); ok {
		var fields []string
		for _, field := range required {
			if fieldStr, ok := field.(string); ok {
				fields = append(fields, fieldStr)
			}
		}
		return fields
	}
	return []string{}
}

func (a *SchemaAnalyzer) isRequired(fieldName string, required []string) bool {
	for _, req := range required {
		if req == fieldName {
			return true
		}
	}
	return false
}

func (a *SchemaAnalyzer) getEnumValueType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	default:
		return "interface{}"
	}
}

func (a *SchemaAnalyzer) toTypeName(name string) string {
	return toPascalCase(name)
}

func (a *SchemaAnalyzer) toFieldName(name string) string {
	switch a.config.Language {
	case "go":
		return toPascalCase(name)
	case "typescript":
		return toCamelCase(name)
	case "python":
		return toSnakeCase(name)
	case "rust":
		return toSnakeCase(name)
	case "java":
		return toCamelCase(name)
	default:
		return name
	}
}

// Reference resolution methods

func (a *SchemaAnalyzer) refToGoType(ref string) string {
	// Extract type name from $ref
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toPascalCase(parts[len(parts)-1])
	}
	return "interface{}"
}

func (a *SchemaAnalyzer) refToTSType(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toPascalCase(parts[len(parts)-1])
	}
	return "any"
}

func (a *SchemaAnalyzer) refToPyType(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toPascalCase(parts[len(parts)-1])
	}
	return "Any"
}

func (a *SchemaAnalyzer) refToRustType(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toPascalCase(parts[len(parts)-1])
	}
	return "serde_json::Value"
}

func (a *SchemaAnalyzer) refToJavaType(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toPascalCase(parts[len(parts)-1])
	}
	return "Object"
}

// Inline type generation methods

func (a *SchemaAnalyzer) generateInlineGoStruct(schema map[string]interface{}) string {
	// For now, return a map type - inline structs are complex
	return "map[string]interface{}"
}

func (a *SchemaAnalyzer) generateInlineTSInterface(schema map[string]interface{}) string {
	// For now, return a Record type - inline interfaces are complex
	return "Record<string, any>"
}

func (a *SchemaAnalyzer) generateInlinePyTypedDict(schema map[string]interface{}) string {
	// For now, return a Dict type - inline TypedDict is complex
	return "Dict[str, Any]"
}

func (a *SchemaAnalyzer) generateInlineRustStruct(schema map[string]interface{}) string {
	// For now, return a Map type - inline structs are complex
	return "serde_json::Map<String, serde_json::Value>"
}

func (a *SchemaAnalyzer) generateInlineJavaClass(schema map[string]interface{}) string {
	// For now, return a Map type - inline classes are complex
	return "Map<String, Object>"
}

// Union/intersection type handling

func (a *SchemaAnalyzer) handleGoUnionType(schemas []interface{}) string {
	// Go doesn't have native union types, use interface{}
	return "interface{}"
}

func (a *SchemaAnalyzer) handleTSUnionType(schemas []interface{}) string {
	var types []string
	for _, schema := range schemas {
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			types = append(types, a.schemaToTSType(schemaMap))
		}
	}
	return strings.Join(types, " | ")
}

func (a *SchemaAnalyzer) handlePyUnionType(schemas []interface{}) string {
	var types []string
	for _, schema := range schemas {
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			types = append(types, a.schemaToPyType(schemaMap))
		}
	}
	return "Union[" + strings.Join(types, ", ") + "]"
}

func (a *SchemaAnalyzer) handleRustUnionType(schemas []interface{}) string {
	// Rust uses enums for union types - return generic for now
	return "serde_json::Value"
}

func (a *SchemaAnalyzer) handleJavaUnionType(schemas []interface{}) string {
	// Java doesn't have native union types, use Object
	return "Object"
}

func (a *SchemaAnalyzer) handleGoIntersectionType(schemas []interface{}) string {
	// Go doesn't have native intersection types, use interface{}
	return "interface{}"
}

func (a *SchemaAnalyzer) handleTSIntersectionType(schemas []interface{}) string {
	var types []string
	for _, schema := range schemas {
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			types = append(types, a.schemaToTSType(schemaMap))
		}
	}
	return strings.Join(types, " & ")
}

func (a *SchemaAnalyzer) handlePyIntersectionType(schemas []interface{}) string {
	// Python doesn't have native intersection types
	return "Any"
}

func (a *SchemaAnalyzer) handleRustIntersectionType(schemas []interface{}) string {
	// Rust doesn't have native intersection types
	return "serde_json::Value"
}

func (a *SchemaAnalyzer) handleJavaIntersectionType(schemas []interface{}) string {
	// Java doesn't have native intersection types
	return "Object"
}

// Utility functions

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
