// Package codegen provides Go code generation from MCP trace analysis
package codegen

import (
	"encoding/json"
	"fmt"
	"strings"
	
	"github.com/tmc/mcp/exp/trace"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// Generator generates Go code from analyzed trace state
type Generator struct {
	packageName string
}

// NewGenerator creates a new code generator
func NewGenerator(packageName string) *Generator {
	return &Generator{
		packageName: packageName,
	}
}

// GeneratePackage generates the package declaration
func (g *Generator) GeneratePackage() string {
	return fmt.Sprintf("package %s", g.packageName)
}

// GenerateImports generates import statements based on state
func (g *Generator) GenerateImports(state trace.State) string {
	imports := []string{
		`"context"`,
		`"encoding/json"`,
		`"fmt"`,
	}
	
	if state.HasServer || state.HasClient {
		imports = append(imports, `"github.com/tmc/mcp"`)
		imports = append(imports, `"github.com/tmc/mcp/modelcontextprotocol"`)
	}
	
	if len(state.Tools) > 0 && hasOutputSchema(state.Tools) {
		imports = append(imports, `"github.com/tmc/mcp/modelcontextprotocol/draft"`)
	}
	
	return fmt.Sprintf("import (\n\t%s\n)", strings.Join(imports, "\n\t"))
}

// GenerateServerType generates the server type and constructor
func (g *Generator) GenerateServerType(state trace.State) string {
	var code strings.Builder
	
	code.WriteString("// MCPServer implements the MCP protocol\n")
	code.WriteString("type MCPServer struct {\n")
	code.WriteString("\t*mcp.Server\n")
	
	// Add fields for discovered features
	if len(state.Tools) > 0 {
		code.WriteString("\ttools map[string]*Tool\n")
	}
	if len(state.Resources) > 0 {
		code.WriteString("\tresources map[string]*Resource\n")
	}
	if len(state.Subscriptions) > 0 {
		code.WriteString("\tsubscriptions map[string][]chan ResourceUpdate\n")
	}
	
	code.WriteString("}\n\n")
	
	// Constructor
	code.WriteString("// NewMCPServer creates a new MCP server\n")
	code.WriteString("func NewMCPServer() *MCPServer {\n")
	code.WriteString("\tserver := &MCPServer{\n")
	code.WriteString("\t\tServer: mcp.NewServer(),\n")
	
	if len(state.Tools) > 0 {
		code.WriteString("\t\ttools: make(map[string]*Tool),\n")
	}
	if len(state.Resources) > 0 {
		code.WriteString("\t\tresources: make(map[string]*Resource),\n")
	}
	if len(state.Subscriptions) > 0 {
		code.WriteString("\t\tsubscriptions: make(map[string][]chan ResourceUpdate),\n")
	}
	
	code.WriteString("\t}\n\n")
	
	// Initialize server info
	if state.ServerInfo != nil {
		code.WriteString(fmt.Sprintf("\tserver.SetInfo(\"%s\", \"%s\")\n",
			state.ServerInfo.Name, state.ServerInfo.Version))
	}
	
	// Initialize capabilities
	if state.Capabilities != nil {
		code.WriteString("\n\t// Configure capabilities\n")
		if state.Capabilities.Tools != nil {
			code.WriteString("\tserver.EnableTools()\n")
		}
		if state.Capabilities.Resources != nil {
			code.WriteString("\tserver.EnableResources()\n")
		}
	}
	
	// Register tools
	if len(state.Tools) > 0 {
		code.WriteString("\n\t// Register tools\n")
		for _, tool := range state.Tools {
			code.WriteString(fmt.Sprintf("\tserver.RegisterTool(\"%s\")\n", tool.Tool.Name))
		}
	}
	
	code.WriteString("\n\treturn server\n")
	code.WriteString("}\n")
	
	return code.String()
}

// GenerateClientType generates the client type
func (g *Generator) GenerateClientType(state trace.State) string {
	var code strings.Builder
	
	code.WriteString("// MCPClient implements the MCP client\n")
	code.WriteString("type MCPClient struct {\n")
	code.WriteString("\t*mcp.Client\n")
	code.WriteString("}\n\n")
	
	code.WriteString("// NewMCPClient creates a new MCP client\n")
	code.WriteString("func NewMCPClient() *MCPClient {\n")
	code.WriteString("\treturn &MCPClient{\n")
	code.WriteString("\t\tClient: mcp.NewClient(),\n")
	code.WriteString("\t}\n")
	code.WriteString("}\n")
	
	return code.String()
}

// GenerateTool generates a tool implementation
func (g *Generator) GenerateTool(tool *trace.ToolInfo) string {
	var code strings.Builder
	
	// Tool documentation
	if tool.Tool.Description != nil {
		code.WriteString(fmt.Sprintf("// %s implements the %s tool\n", 
			toCamelCase(tool.Tool.Name), tool.Tool.Name))
		code.WriteString(fmt.Sprintf("// %s\n", *tool.Tool.Description))
	}
	
	// Handler function
	handlerName := fmt.Sprintf("handle%s", toCamelCase(tool.Tool.Name))
	code.WriteString(fmt.Sprintf("func (s *MCPServer) %s(ctx context.Context, params json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {\n", handlerName))
	
	// Parse parameters
	if tool.Tool.InputSchema.Properties != nil && len(tool.Tool.InputSchema.Properties) > 0 {
		code.WriteString("\t// Parse input parameters\n")
		code.WriteString("\tvar input struct {\n")
		for name, schema := range tool.Tool.InputSchema.Properties {
			goType := schemaToGoType(schema)
			jsonTag := toSnakeCase(name)
			code.WriteString(fmt.Sprintf("\t\t%s %s `json:\"%s\"`\n", 
				toCamelCase(name), goType, jsonTag))
		}
		code.WriteString("\t}\n")
		code.WriteString("\tif err := json.Unmarshal(params, &input); err != nil {\n")
		code.WriteString("\t\treturn nil, fmt.Errorf(\"invalid parameters: %w\", err)\n")
		code.WriteString("\t}\n\n")
	}
	
	// Validation
	if len(tool.Tool.InputSchema.Required) > 0 {
		code.WriteString("\t// Validate required fields\n")
		for _, required := range tool.Tool.InputSchema.Required {
			fieldName := toCamelCase(required)
			code.WriteString(fmt.Sprintf("\tif input.%s == \"\" {\n", fieldName))
			code.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s is required\")\n", required))
			code.WriteString("\t}\n")
		}
		code.WriteString("\n")
	}
	
	// Implementation placeholder
	code.WriteString("\t// TODO: Implement tool logic\n")
	code.WriteString("\t// Examples from trace:\n")
	for i, example := range tool.Examples {
		if i >= 3 { // Limit examples
			break
		}
		exampleJSON, _ := json.MarshalIndent(example.Arguments, "\t// ", "\t")
		code.WriteString(fmt.Sprintf("\t// Example %d: %s\n", i+1, exampleJSON))
	}
	code.WriteString("\n")
	
	// Return result
	code.WriteString("\treturn &modelcontextprotocol.CallToolResult{\n")
	code.WriteString("\t\tContent: []modelcontextprotocol.Content{\n")
	code.WriteString("\t\t\t{\n")
	code.WriteString("\t\t\t\tType: \"text\",\n")
	code.WriteString("\t\t\t\tText: \"Tool executed successfully\",\n")
	code.WriteString("\t\t\t},\n")
	code.WriteString("\t\t},\n")
	code.WriteString("\t}, nil\n")
	code.WriteString("}\n")
	
	return code.String()
}

// GenerateHandler generates a generic handler
func (g *Generator) GenerateHandler(handler *trace.HandlerInfo) string {
	var code strings.Builder
	
	methodName := strings.ReplaceAll(handler.Method, "/", "_")
	handlerName := fmt.Sprintf("handle%s", toCamelCase(methodName))
	
	code.WriteString(fmt.Sprintf("// %s handles %s requests\n", handlerName, handler.Method))
	code.WriteString(fmt.Sprintf("func (s *MCPServer) %s(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {\n", handlerName))
	
	// Parse parameters if known
	if handler.ParamType != "" && handler.ParamType != "Object" {
		code.WriteString(fmt.Sprintf("\tvar input %s\n", handler.ParamType))
		code.WriteString("\tif err := json.Unmarshal(params, &input); err != nil {\n")
		code.WriteString("\t\treturn nil, fmt.Errorf(\"invalid parameters: %w\", err)\n")
		code.WriteString("\t}\n\n")
	}
	
	// Implementation placeholder
	code.WriteString("\t// TODO: Implement handler logic\n\n")
	
	// Return result
	if handler.ResultType != "" && handler.ResultType != "Object" {
		code.WriteString(fmt.Sprintf("\tresult := %s{\n", handler.ResultType))
		code.WriteString("\t\t// TODO: Populate result\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn json.Marshal(result)\n")
	} else {
		code.WriteString("\treturn json.Marshal(map[string]any{\n")
		code.WriteString("\t\t\"status\": \"ok\",\n")
		code.WriteString("\t})\n")
	}
	
	code.WriteString("}\n")
	
	return code.String()
}

// GenerateMain generates the main function
func (g *Generator) GenerateMain(state trace.State) string {
	var code strings.Builder
	
	code.WriteString("func main() {\n")
	
	if state.HasServer {
		code.WriteString("\t// Create and run server\n")
		code.WriteString("\tserver := NewMCPServer()\n")
		code.WriteString("\n")
		code.WriteString("\t// Use stdio transport by default\n")
		code.WriteString("\ttransport := mcp.NewStdioTransport()\n")
		code.WriteString("\n")
		code.WriteString("\tif err := server.Serve(transport); err != nil {\n")
		code.WriteString("\t\tfmt.Fprintf(os.Stderr, \"Server error: %v\\n\", err)\n")
		code.WriteString("\t\tos.Exit(1)\n")
		code.WriteString("\t}\n")
	} else if state.HasClient {
		code.WriteString("\t// Create and run client\n")
		code.WriteString("\tclient := NewMCPClient()\n")
		code.WriteString("\n")
		code.WriteString("\t// TODO: Implement client logic\n")
	}
	
	code.WriteString("}\n")
	
	return code.String()
}

// Helper functions

func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == '/'
	})
	
	for i := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(parts[i])
		} else {
			parts[i] = strings.Title(parts[i])
		}
	}
	
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

func schemaToGoType(schema json.RawMessage) string {
	var schemaObj struct {
		Type string `json:"type"`
	}
	
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return "interface{}"
	}
	
	switch schemaObj.Type {
	case "string":
		return "string"
	case "number", "integer":
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

func hasOutputSchema(tools []*trace.ToolInfo) bool {
	// In real implementation, check if any tool has output schema
	// This would require analyzing the draft format
	return false
}