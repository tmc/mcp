// Package sourcegen generates Go source code from MCP tool descriptions and JSON schemas
package sourcegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/tmc/mcp/exp/mcpspec"
)

// Generator generates Go source code from schemas
type Generator struct {
	packageName string
	imports     map[string]bool
	types       map[string]string
	refs        map[string]*mcpspec.JSONSchema
	typeCount   int // Counter for generating unique type names
}

// NewGenerator creates a new source generator
func NewGenerator(packageName string) *Generator {
	return &Generator{
		packageName: packageName,
		imports:     make(map[string]bool),
		types:       make(map[string]string),
		refs:        make(map[string]*mcpspec.JSONSchema),
		typeCount:   0,
	}
}

// GetPackageName returns the package name
func (g *Generator) GetPackageName() string {
	return g.packageName
}

// GetImports returns the imports map
func (g *Generator) GetImports() map[string]bool {
	return g.imports
}

// GenerateFromSpec generates Go source from an MCPSpec
func (g *Generator) GenerateFromSpec(spec *mcpspec.MCPSpec) (string, error) {
	var buf bytes.Buffer

	// Add package declaration
	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)

	// Generate Server Metadata constant
	fmt.Fprintf(&buf, "const (\n")
	fmt.Fprintf(&buf, "\tServerName = \"%s\"\n", spec.Server.Name)
	fmt.Fprintf(&buf, "\tServerVersion = \"%s\"\n", spec.Server.Version)
	fmt.Fprintf(&buf, ")\n\n")

	// Generate Tools
	for _, tool := range spec.Tools {
		toolCode, err := g.generateToolCode(&tool)
		if err != nil {
			return "", fmt.Errorf("generating tool %s: %w", tool.Name, err)
		}
		buf.WriteString(toolCode)
		buf.WriteString("\n")
	}

	// Generate Resources
	for _, resource := range spec.Resources {
		resourceCode, err := g.generateResourceCode(&resource)
		if err != nil {
			return "", fmt.Errorf("generating resource %s: %w", resource.Name, err)
		}
		buf.WriteString(resourceCode)
		buf.WriteString("\n")
	}

	// Generate Prompts
	for _, prompt := range spec.Prompts {
		promptCode, err := g.generatePromptCode(&prompt)
		if err != nil {
			return "", fmt.Errorf("generating prompt %s: %w", prompt.Name, err)
		}
		buf.WriteString(promptCode)
		buf.WriteString("\n")
	}

	// Add imports
	return g.finalizeSource(&buf)
}

// GenerateFromTool generates Go source from an MCP tool definition
func (g *Generator) GenerateFromTool(tool *mcpspec.ToolDefinition) (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)

	code, err := g.generateToolCode(tool)
	if err != nil {
		return "", err
	}
	buf.WriteString(code)

	return g.finalizeSource(&buf)
}

func (g *Generator) generateToolCode(tool *mcpspec.ToolDefinition) (string, error) {
	// Parse input schema
	var inputSchema mcpspec.JSONSchema
	if len(tool.InputSchema) > 0 {
		if err := json.Unmarshal(tool.InputSchema, &inputSchema); err != nil {
			return "", fmt.Errorf("parsing input schema: %w", err)
		}
	}

	// Parse return type
	var returnType mcpspec.JSONSchema
	if len(tool.ReturnType) > 0 {
		if err := json.Unmarshal(tool.ReturnType, &returnType); err != nil {
			return "", fmt.Errorf("parsing return type: %w", err)
		}
	} else {
		// Use standard MCP CallToolResult format
		returnType = mcpspec.JSONSchema{
			Type: "object",
			Properties: map[string]*mcpspec.JSONSchema{
				"_meta": {
					Type:                 "object",
					Description:          "Optional metadata",
					AdditionalProperties: true,
				},
				"content": {
					Type:        "array",
					Description: "The content returned by the tool",
					Items: &mcpspec.JSONSchema{
						Type: "object",
						Properties: map[string]*mcpspec.JSONSchema{
							"type": {
								Type:        "string",
								Description: "The type of content (must be: text, image, audio, resource)",
							},
							"text": {
								Type:        "string",
								Description: "The text content (when type is 'text')",
							},
							"data": {
								Type:        "string",
								Description: "Base64 encoded data (when type is 'image' or 'audio')",
							},
							"mimeType": {
								Type:        "string",
								Description: "MIME type of the content (for image/audio)",
							},
							"uri": {
								Type:        "string",
								Description: "URI of the resource (when type is 'resource')",
							},
							"annotations": {
								Type: "object",
								Properties: map[string]*mcpspec.JSONSchema{
									"audience": {
										Type: "array",
										Items: &mcpspec.JSONSchema{
											Type: "string",
										},
										Description: "Audience for this content",
									},
									"priority": {
										Type:        "number",
										Description: "Priority of this content",
									},
								},
							},
						},
					},
				},
				"isError": {
					Type:        "boolean",
					Description: "Whether this result is an error",
				},
			},
			Required: []string{"content"},
		}
	}

	// Generate types
	var buf bytes.Buffer

	toolName := toGoName(tool.Name)

	// Generate input type
	inputTypeName := toolName + "Input"
	if err := g.generateType(&buf, inputTypeName, &inputSchema); err != nil {
		return "", fmt.Errorf("generating input type: %w", err)
	}

	// Generate output type
	outputTypeName := toolName + "Output"
	if err := g.generateType(&buf, outputTypeName, &returnType); err != nil {
		return "", fmt.Errorf("generating output type: %w", err)
	}

	// Generate tool interface
	fmt.Fprintf(&buf, "// %s %s\n", toolName, tool.Description)
	fmt.Fprintf(&buf, "type %sTool interface {\n", toolName)
	fmt.Fprintf(&buf, "\t// Execute runs the %s tool\n", tool.Name)
	fmt.Fprintf(&buf, "\tExecute(ctx context.Context, input *%s) (*%s, error)\n", inputTypeName, outputTypeName)
	fmt.Fprintf(&buf, "}\n\n")

	// Add context import
	g.imports["context"] = true

	// Generate implementation stub
	fmt.Fprintf(&buf, "// %sImpl implements the %sTool interface\n", toolName, toolName)
	fmt.Fprintf(&buf, "type %sImpl struct{}\n\n", toolName)

	fmt.Fprintf(&buf, "// Execute implements %sTool\n", toolName)
	fmt.Fprintf(&buf, "func (t *%sImpl) Execute(ctx context.Context, input *%s) (*%s, error) {\n", toolName, inputTypeName, outputTypeName)
	fmt.Fprintf(&buf, "\t// TODO: Implement tool logic\n")
	fmt.Fprintf(&buf, "\treturn nil, fmt.Errorf(\"not implemented\")\n")
	fmt.Fprintf(&buf, "}\n\n")

	// Add fmt import for error
	g.imports["fmt"] = true

	return buf.String(), nil
}

func (g *Generator) generateResourceCode(resource *mcpspec.ResourceDefinition) (string, error) {
	var buf bytes.Buffer
	name := toGoName(resource.Name)

	fmt.Fprintf(&buf, "// %sResource represents the %s resource\n", name, resource.Name)
	fmt.Fprintf(&buf, "type %sResource interface {\n", name)
	fmt.Fprintf(&buf, "\t// Read reads the resource\n")
	fmt.Fprintf(&buf, "\tRead(ctx context.Context, uri string) ([]byte, error)\n")
	fmt.Fprintf(&buf, "}\n\n")

	g.imports["context"] = true

	// Implementation stub
	fmt.Fprintf(&buf, "type %sImpl struct{}\n\n", name)
	fmt.Fprintf(&buf, "func (r *%sImpl) Read(ctx context.Context, uri string) ([]byte, error) {\n", name)
	fmt.Fprintf(&buf, "\treturn nil, fmt.Errorf(\"not implemented\")\n")
	fmt.Fprintf(&buf, "}\n\n")

	g.imports["fmt"] = true
	return buf.String(), nil
}

func (g *Generator) generatePromptCode(prompt *mcpspec.PromptDefinition) (string, error) {
	var buf bytes.Buffer
	name := toGoName(prompt.Name)

	// Generate input struct from arguments
	inputName := name + "PromptInput"
	fmt.Fprintf(&buf, "type %s struct {\n", inputName)
	for _, arg := range prompt.Arguments {
		fieldName := toGoName(arg.Name)
		tag := arg.Name
		if !arg.Required {
			tag += ",omitempty"
		}
		// Assuming string type for prompt args for now as they are template vars mostly
		fieldType := "string"
		if !arg.Required {
			fieldType = "*string"
		}
		fmt.Fprintf(&buf, "\t%s %s `json:\"%s\"` // %s\n", fieldName, fieldType, tag, arg.Description)
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "// %sPrompt represents the %s prompt\n", name, prompt.Name)
	fmt.Fprintf(&buf, "type %sPrompt interface {\n", name)
	fmt.Fprintf(&buf, "\t// Get returns the prompt\n")
	fmt.Fprintf(&buf, "\tGet(ctx context.Context, input *%s) (interface{}, error)\n", inputName)
	fmt.Fprintf(&buf, "}\n\n")

	g.imports["context"] = true

	// Implementation stub
	fmt.Fprintf(&buf, "type %sImpl struct{}\n\n", name)
	fmt.Fprintf(&buf, "func (p *%sImpl) Get(ctx context.Context, input *%s) (interface{}, error) {\n", name, inputName)
	fmt.Fprintf(&buf, "\treturn nil, fmt.Errorf(\"not implemented\")\n")
	fmt.Fprintf(&buf, "}\n\n")

	g.imports["fmt"] = true
	return buf.String(), nil
}

// GenerateFromJSONSchema generates Go types from a JSON schema
func (g *Generator) GenerateFromJSONSchema(name string, schema *mcpspec.JSONSchema) (string, error) {
	var buf bytes.Buffer

	// Add package declaration
	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)

	// Generate the type
	if err := g.generateType(&buf, name, schema); err != nil {
		return "", err
	}

	return g.finalizeSource(&buf)
}

// generateType generates a Go type from a JSON schema
func (g *Generator) generateType(buf *bytes.Buffer, name string, schema *mcpspec.JSONSchema) error {
	// Handle references
	if schema.Ref != "" {
		// TODO: Implement reference resolution
		return fmt.Errorf("references not yet implemented: %s", schema.Ref)
	}

	// Handle type based on schema type
	switch schema.Type {
	case "object":
		return g.generateStruct(buf, name, schema)
	case "array":
		return g.generateArray(buf, name, schema)
	case "string", "number", "integer", "boolean":
		return g.generateSimpleType(buf, name, schema)
	default:
		if len(schema.Properties) > 0 {
			// Object type implied
			return g.generateStruct(buf, name, schema)
		}
		return fmt.Errorf("unsupported schema type: %s", schema.Type)
	}
}

// generateStruct generates a Go struct from an object schema
func (g *Generator) generateStruct(buf *bytes.Buffer, name string, schema *mcpspec.JSONSchema) error {
	// Write struct comment if description exists
	if schema.Description != "" {
		fmt.Fprintf(buf, "// %s %s\n", name, schema.Description)
	} else {
		fmt.Fprintf(buf, "// %s represents %s\n", name, toHumanReadable(name))
	}

	fmt.Fprintf(buf, "type %s struct {\n", name)

	// Sort properties for consistent output
	propNames := make([]string, 0, len(schema.Properties))
	for propName := range schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)

	// Generate struct fields
	for _, propName := range propNames {
		prop := schema.Properties[propName]

		fieldName := toGoName(propName)
		fieldType := g.jsonSchemaToGoType(prop)

		// Check if field is required
		isRequired := containsString(schema.Required, propName)

		// Add field comment
		if prop.Description != "" {
			fmt.Fprintf(buf, "\t// %s %s\n", fieldName, prop.Description)
		}

		// Handle pointer types for optional fields
		if !isRequired && needsPointer(fieldType) {
			fieldType = "*" + fieldType
		}

		// Add JSON tag
		jsonTag := propName
		if !isRequired {
			jsonTag += ",omitempty"
		}

		fmt.Fprintf(buf, "\t%s %s `json:\"%s\"`\n", fieldName, fieldType, jsonTag)
	}

	fmt.Fprintf(buf, "}\n\n")
	return nil
}

// generateArray generates a Go array/slice type
func (g *Generator) generateArray(buf *bytes.Buffer, name string, schema *mcpspec.JSONSchema) error {
	if schema.Items == nil {
		return fmt.Errorf("array schema missing items definition")
	}

	itemType := g.jsonSchemaToGoType(schema.Items)

	// Generate type alias
	if schema.Description != "" {
		fmt.Fprintf(buf, "// %s %s\n", name, schema.Description)
	}
	fmt.Fprintf(buf, "type %s []%s\n\n", name, itemType)

	return nil
}

// generateSimpleType generates a simple type alias
func (g *Generator) generateSimpleType(buf *bytes.Buffer, name string, schema *mcpspec.JSONSchema) error {
	goType := g.jsonSchemaToGoType(schema)

	if schema.Description != "" {
		fmt.Fprintf(buf, "// %s %s\n", name, schema.Description)
	}
	fmt.Fprintf(buf, "type %s %s\n\n", name, goType)

	// Generate enum constants if present
	if len(schema.Enum) > 0 {
		fmt.Fprintf(buf, "// %s values\n", name)
		fmt.Fprintf(buf, "const (\n")
		for i, val := range schema.Enum {
			constName := fmt.Sprintf("%s%s", name, toGoName(fmt.Sprintf("%v", val)))
			if i == 0 {
				fmt.Fprintf(buf, "\t%s %s = \"%v\"\n", constName, name, val)
			} else {
				fmt.Fprintf(buf, "\t%s = \"%v\"\n", constName, val)
			}
		}
		fmt.Fprintf(buf, ")\n\n")
	}

	return nil
}

// jsonSchemaToGoType converts a JSON schema type to a Go type
func (g *Generator) jsonSchemaToGoType(schema *mcpspec.JSONSchema) string {
	switch schema.Type {
	case "string":
		if schema.Format == "date-time" {
			g.imports["time"] = true
			return "time.Time"
		}
		return "string"
	case "number":
		return "float64"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		if schema.Items != nil {
			return "[]" + g.jsonSchemaToGoType(schema.Items)
		}
		return "[]interface{}"
	case "object":
		if len(schema.Properties) == 0 {
			if schema.AdditionalProperties != nil {
				return "map[string]interface{}"
			}
			return "struct{}"
		}
		// For nested objects, we generate an inline struct
		var fields []string
		for propName, prop := range schema.Properties {
			fieldName := toGoName(propName)
			fieldType := g.jsonSchemaToGoType(prop)
			isRequired := containsString(schema.Required, propName)

			// Handle pointer types for optional fields
			if !isRequired && needsPointer(fieldType) {
				fieldType = "*" + fieldType
			}

			// Build field definition
			fieldDef := fmt.Sprintf("\t%s %s", fieldName, fieldType)

			// Add JSON tag
			jsonTag := propName
			if !isRequired {
				jsonTag += ",omitempty"
			}
			fieldDef += fmt.Sprintf(" `json:\"%s\"`", jsonTag)

			fields = append(fields, fieldDef)
		}
		return "struct{\n" + strings.Join(fields, "\n") + "\n\t}"
	default:
		return "interface{}"
	}
}

// finalizeSource adds imports and formats the generated source
func (g *Generator) finalizeSource(buf *bytes.Buffer) (string, error) {
	var finalBuf bytes.Buffer

	// Write package declaration
	lines := strings.Split(buf.String(), "\n")
	if len(lines) > 0 {
		finalBuf.WriteString(lines[0] + "\n\n")
	}

	// Write imports if any
	if len(g.imports) > 0 {
		finalBuf.WriteString("import (\n")

		// Sort imports
		importList := make([]string, 0, len(g.imports))
		for imp := range g.imports {
			importList = append(importList, imp)
		}
		sort.Strings(importList)

		for _, imp := range importList {
			fmt.Fprintf(&finalBuf, "\t\"%s\"\n", imp)
		}
		finalBuf.WriteString(")\n\n")
	}

	// Write the rest of the code
	if len(lines) > 1 {
		finalBuf.WriteString(strings.Join(lines[1:], "\n"))
	}

	// Format the source
	formatted, err := format.Source(finalBuf.Bytes())
	if err != nil {
		// Return unformatted if formatting fails
		return finalBuf.String(), nil
	}

	return string(formatted), nil
}

// Utility functions

func toGoName(s string) string {
	// Convert snake_case or kebab-case to PascalCase
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	for i, part := range parts {
		parts[i] = strings.Title(part)
	}

	return strings.Join(parts, "")
}

func toHumanReadable(s string) string {
	// Convert PascalCase to human readable
	var result []string
	var current []rune

	for _, r := range s {
		if len(current) > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, string(current))
			current = []rune{r}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		result = append(result, string(current))
	}

	return strings.ToLower(strings.Join(result, " "))
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func needsPointer(goType string) bool {
	// Basic types that need pointers for optional fields
	switch goType {
	case "string", "int", "int64", "float64", "bool", "time.Time":
		return true
	}
	// Arrays and maps don't need pointers
	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return false
	}
	// Custom types need pointers
	return true
}
