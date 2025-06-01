package schema2go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"
)

// Options for code generation
type Options struct {
	PackageName string
	Prefix      string
	Tags        []string
	Imports     []string
	NoValidate  bool
	NoComments  bool
	Verbose     bool
}

// Generator generates Go code from schemas
type Generator struct {
	options Options
	types   map[string]*TypeDef
	imports map[string]bool
}

// TypeDef represents a Go type definition
type TypeDef struct {
	Name        string
	GoType      string
	Description string
	Fields      []FieldDef
	EnumValues  []string
	Required    []string
	Properties  map[string]interface{}
}

// FieldDef represents a struct field
type FieldDef struct {
	Name        string
	GoName      string
	Type        string
	Description string
	Required    bool
	Tags        map[string]string
}

// NewGenerator creates a new generator
func NewGenerator(options Options) *Generator {
	if options.PackageName == "" {
		options.PackageName = "main"
	}
	if len(options.Tags) == 0 {
		options.Tags = []string{"json"}
	}

	return &Generator{
		options: options,
		types:   make(map[string]*TypeDef),
		imports: make(map[string]bool),
	}
}

// FromJSONSchema generates Go code from JSON Schema
func (g *Generator) FromJSONSchema(schemaData []byte) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return "", fmt.Errorf("invalid JSON schema: %w", err)
	}

	// Validate schema if requested
	if !g.options.NoValidate {
		if err := g.validateJSONSchema(schema); err != nil {
			return "", fmt.Errorf("schema validation failed: %w", err)
		}
	}

	// Extract root type name
	rootName := g.options.Prefix + "Root"
	if title, ok := schema["title"].(string); ok {
		rootName = g.options.Prefix + toGoName(title)
	}

	// Generate type from schema
	g.generateTypeFromSchema(rootName, schema)

	// Build Go code
	return g.buildGoCode()
}

// FromOpenAPI generates Go code from OpenAPI specification
func (g *Generator) FromOpenAPI(specData []byte) (string, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(specData, &spec); err != nil {
		// Try YAML parsing
		return "", fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	// Extract schemas from components/definitions
	var schemas map[string]interface{}
	if components, ok := spec["components"].(map[string]interface{}); ok {
		schemas, _ = components["schemas"].(map[string]interface{})
	} else if definitions, ok := spec["definitions"].(map[string]interface{}); ok {
		schemas = definitions
	}

	if schemas == nil {
		return "", fmt.Errorf("no schemas found in OpenAPI spec")
	}

	// Generate types for each schema
	for name, schema := range schemas {
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			typeName := g.options.Prefix + toGoName(name)
			g.generateTypeFromSchema(typeName, schemaMap)
		}
	}

	// Add imports for OpenAPI-specific types
	g.imports["time"] = true

	return g.buildGoCode()
}

// FromProtobuf generates Go code from Protocol Buffers (simplified)
func (g *Generator) FromProtobuf(protoData []byte) (string, error) {
	// This is a simplified implementation
	// In production, use proper protobuf parsing
	return "", fmt.Errorf("protobuf generation not implemented yet")
}

// FromMCPSchema generates Go code from MCP schema
func (g *Generator) FromMCPSchema(schemaData []byte) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return "", fmt.Errorf("invalid MCP schema: %w", err)
	}

	// Generate types for tools
	if tools, ok := schema["tools"].([]interface{}); ok {
		for i, tool := range tools {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				g.generateMCPTool(toolMap, i)
			}
		}
	}

	// Generate types for resources
	if resources, ok := schema["resources"].([]interface{}); ok {
		for i, resource := range resources {
			if resMap, ok := resource.(map[string]interface{}); ok {
				g.generateMCPResource(resMap, i)
			}
		}
	}

	// Add MCP-specific imports
	g.imports["github.com/tmc/mcp"] = true

	return g.buildGoCode()
}

func (g *Generator) generateTypeFromSchema(name string, schema map[string]interface{}) {
	schemaType, _ := schema["type"].(string)
	
	switch schemaType {
	case "object":
		g.generateObjectType(name, schema)
	case "array":
		g.generateArrayType(name, schema)
	case "string":
		if enum, ok := schema["enum"].([]interface{}); ok {
			g.generateEnumType(name, schema, enum)
		} else {
			g.types[name] = &TypeDef{
				Name:   name,
				GoType: "string",
			}
		}
	case "number":
		g.types[name] = &TypeDef{
			Name:   name,
			GoType: "float64",
		}
	case "integer":
		g.types[name] = &TypeDef{
			Name:   name,
			GoType: "int",
		}
	case "boolean":
		g.types[name] = &TypeDef{
			Name:   name,
			GoType: "bool",
		}
	default:
		// Handle anyOf, oneOf, allOf
		if anyOf, ok := schema["anyOf"].([]interface{}); ok {
			g.generateUnionType(name, anyOf)
		} else if oneOf, ok := schema["oneOf"].([]interface{}); ok {
			g.generateUnionType(name, oneOf)
		} else {
			g.types[name] = &TypeDef{
				Name:   name,
				GoType: "interface{}",
			}
		}
	}
}

func (g *Generator) generateObjectType(name string, schema map[string]interface{}) {
	typeDef := &TypeDef{
		Name:        name,
		GoType:      "struct",
		Description: getDescription(schema),
		Fields:      []FieldDef{},
	}

	// Get properties
	properties, _ := schema["properties"].(map[string]interface{})
	required, _ := schema["required"].([]interface{})
	
	// Create required set for quick lookup
	requiredSet := make(map[string]bool)
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredSet[reqStr] = true
		}
	}

	// Generate fields
	if properties != nil {
		// Sort property names for consistent output
		var propNames []string
		for name := range properties {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)

		for _, propName := range propNames {
			prop := properties[propName].(map[string]interface{})
			field := g.generateField(propName, prop, requiredSet[propName])
			typeDef.Fields = append(typeDef.Fields, field)
		}
	}

	g.types[name] = typeDef
}

func (g *Generator) generateField(name string, schema map[string]interface{}, required bool) FieldDef {
	field := FieldDef{
		Name:        name,
		GoName:      toGoName(name),
		Description: getDescription(schema),
		Required:    required,
		Tags:        make(map[string]string),
	}

	// Determine Go type
	fieldType := g.schemaTypeToGo(schema)
	
	// Make optional fields pointers
	if !required && !isBasicType(fieldType) {
		fieldType = "*" + fieldType
	}
	
	field.Type = fieldType

	// Add struct tags
	for _, tag := range g.options.Tags {
		switch tag {
		case "json":
			jsonTag := name
			if !required {
				jsonTag += ",omitempty"
			}
			field.Tags["json"] = jsonTag
		case "yaml":
			yamlTag := name
			if !required {
				yamlTag += ",omitempty"
			}
			field.Tags["yaml"] = yamlTag
		case "xml":
			field.Tags["xml"] = name + ",omitempty"
		}
	}

	return field
}

func (g *Generator) schemaTypeToGo(schema map[string]interface{}) string {
	schemaType, _ := schema["type"].(string)
	
	switch schemaType {
	case "string":
		if format, ok := schema["format"].(string); ok {
			switch format {
			case "date-time":
				g.imports["time"] = true
				return "time.Time"
			case "date":
				g.imports["time"] = true
				return "time.Time"
			case "uuid":
				return "string" // Could use a UUID type
			case "email":
				return "string"
			case "uri", "url":
				return "string"
			default:
				return "string"
			}
		}
		return "string"
	case "number":
		return "float64"
	case "integer":
		if format, ok := schema["format"].(string); ok {
			switch format {
			case "int32":
				return "int32"
			case "int64":
				return "int64"
			default:
				return "int"
			}
		}
		return "int"
	case "boolean":
		return "bool"
	case "array":
		items, _ := schema["items"].(map[string]interface{})
		if items != nil {
			itemType := g.schemaTypeToGo(items)
			return "[]" + itemType
		}
		return "[]interface{}"
	case "object":
		// Check if it's a map
		if additionalProps, ok := schema["additionalProperties"].(map[string]interface{}); ok {
			valueType := g.schemaTypeToGo(additionalProps)
			return "map[string]" + valueType
		}
		// Generate nested type
		if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
			// Generate inline type or reference
			return "interface{}" // Simplified for now
		}
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func (g *Generator) generateArrayType(name string, schema map[string]interface{}) {
	items, _ := schema["items"].(map[string]interface{})
	if items == nil {
		g.types[name] = &TypeDef{
			Name:   name,
			GoType: "[]interface{}",
		}
		return
	}

	itemType := g.schemaTypeToGo(items)
	g.types[name] = &TypeDef{
		Name:        name,
		GoType:      "[]" + itemType,
		Description: getDescription(schema),
	}
}

func (g *Generator) generateEnumType(name string, schema map[string]interface{}, values []interface{}) {
	// Create type alias for enum
	g.types[name] = &TypeDef{
		Name:        name,
		GoType:      "string",
		Description: getDescription(schema),
		EnumValues:  []string{},
	}

	// Convert enum values
	for _, val := range values {
		if strVal, ok := val.(string); ok {
			g.types[name].EnumValues = append(g.types[name].EnumValues, strVal)
		}
	}
}

func (g *Generator) generateUnionType(name string, schemas []interface{}) {
	// Simplified union type generation
	g.types[name] = &TypeDef{
		Name:        name,
		GoType:      "interface{}",
		Description: "Union type",
	}
}

func (g *Generator) generateMCPTool(tool map[string]interface{}, index int) {
	name, _ := tool["name"].(string)
	if name == "" {
		name = fmt.Sprintf("Tool%d", index)
	}
	
	typeName := g.options.Prefix + toGoName(name) + "Tool"
	
	// Generate tool struct
	typeDef := &TypeDef{
		Name:        typeName,
		GoType:      "struct",
		Description: getDescription(tool),
		Fields:      []FieldDef{},
	}

	// Add standard tool fields
	typeDef.Fields = append(typeDef.Fields, FieldDef{
		GoName: "Name",
		Type:   "string",
		Tags:   map[string]string{"json": "name"},
	})
	
	typeDef.Fields = append(typeDef.Fields, FieldDef{
		GoName: "Description",
		Type:   "string",
		Tags:   map[string]string{"json": "description"},
	})

	// Add input schema if present
	if inputSchema, ok := tool["inputSchema"].(map[string]interface{}); ok {
		inputTypeName := typeName + "Input"
		g.generateTypeFromSchema(inputTypeName, inputSchema)
		
		typeDef.Fields = append(typeDef.Fields, FieldDef{
			GoName: "InputSchema",
			Type:   "*" + inputTypeName,
			Tags:   map[string]string{"json": "inputSchema,omitempty"},
		})
	}

	g.types[typeName] = typeDef
}

func (g *Generator) generateMCPResource(resource map[string]interface{}, index int) {
	name, _ := resource["name"].(string)
	if name == "" {
		name = fmt.Sprintf("Resource%d", index)
	}
	
	typeName := g.options.Prefix + toGoName(name) + "Resource"
	
	// Generate resource struct
	typeDef := &TypeDef{
		Name:        typeName,
		GoType:      "struct",
		Description: getDescription(resource),
		Fields:      []FieldDef{},
	}

	// Add standard resource fields
	typeDef.Fields = append(typeDef.Fields, FieldDef{
		GoName: "Name",
		Type:   "string",
		Tags:   map[string]string{"json": "name"},
	})
	
	typeDef.Fields = append(typeDef.Fields, FieldDef{
		GoName: "Description",
		Type:   "string",
		Tags:   map[string]string{"json": "description"},
	})
	
	typeDef.Fields = append(typeDef.Fields, FieldDef{
		GoName: "URI",
		Type:   "string",
		Tags:   map[string]string{"json": "uri"},
	})

	g.types[typeName] = typeDef
}

func (g *Generator) buildGoCode() (string, error) {
	var buf bytes.Buffer
	
	// Template for Go file
	tmpl := `package {{ .Package }}

{{ if .Imports }}
import (
{{ range .Imports }}	{{ . }}
{{ end }})
{{ end }}

{{ range .Types }}
{{ if not $.NoComments }}{{ if .Description }}// {{ .Name }} {{ .Description }}
{{ end }}{{ end }}type {{ .Name }} {{ if eq .GoType "struct" }}struct {
{{ range .Fields }}	{{ .GoName }} {{ .Type }} {{ .Tags }}{{ if .Description }} // {{ .Description }}{{ end }}
{{ end }}}{{ else }}{{ .GoType }}{{ end }}

{{ if .EnumValues }}// {{ .Name }} values
const (
{{ range $i, $val := .EnumValues }}	{{ $.Prefix }}{{ .Name }}{{ toGoName $val }} {{ .Name }} = "{{ $val }}"
{{ end }})
{{ end }}
{{ end }}
`

	// Execute template
	t, err := template.New("gocode").Funcs(template.FuncMap{
		"toGoName": toGoName,
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	// Prepare template data
	var sortedTypes []*TypeDef
	for _, typ := range g.types {
		sortedTypes = append(sortedTypes, typ)
	}
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Name < sortedTypes[j].Name
	})

	var importList []string
	for imp := range g.imports {
		importList = append(importList, fmt.Sprintf(`"%s"`, imp))
	}
	sort.Strings(importList)

	data := map[string]interface{}{
		"Package":   g.options.PackageName,
		"Imports":   importList,
		"Types":     sortedTypes,
		"NoComments": g.options.NoComments,
		"Prefix":    g.options.Prefix,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	// Format the code
	source := buf.Bytes()
	formatted, err := format.Source(source)
	if err != nil {
		return string(source), nil // Return unformatted if formatting fails
	}

	return string(formatted), nil
}

// Helper functions

func (g *Generator) validateJSONSchema(schema map[string]interface{}) error {
	// Basic validation
	if _, ok := schema["type"]; !ok && 
	   _, ok := schema["properties"]; !ok &&
	   _, ok := schema["$ref"]; !ok {
		return fmt.Errorf("schema must have type, properties, or $ref")
	}
	return nil
}

func toGoName(s string) string {
	// Convert to CamelCase
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	
	name := strings.Join(words, "")
	
	// Ensure it starts with uppercase
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	
	// Handle special cases
	replacements := map[string]string{
		"Id":   "ID",
		"Url":  "URL",
		"Uri":  "URI",
		"Api":  "API",
		"Json": "JSON",
		"Xml":  "XML",
		"Http": "HTTP",
	}
	
	for old, new := range replacements {
		name = strings.ReplaceAll(name, old, new)
	}
	
	return name
}

func getDescription(schema map[string]interface{}) string {
	if desc, ok := schema["description"].(string); ok {
		return desc
	}
	return ""
}

func isBasicType(goType string) bool {
	basicTypes := map[string]bool{
		"string":  true,
		"int":     true,
		"int32":   true,
		"int64":   true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
	}
	return basicTypes[goType]
}

// FieldDef.Tags() returns formatted struct tags
func (f FieldDef) Tags() string {
	if len(f.Tags) == 0 {
		return ""
	}
	
	var tagParts []string
	for key, value := range f.Tags {
		tagParts = append(tagParts, fmt.Sprintf(`%s:"%s"`, key, value))
	}
	
	return "`" + strings.Join(tagParts, " ") + "`"
}