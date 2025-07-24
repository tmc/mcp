// Package codegen provides the core code generation functionality for mcp-gen
package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/config"
	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/templates"
	"github.com/tmc/mcp"
)

// Generator provides code generation capabilities
type Generator struct {
	config     *config.Config
	templates  *templates.TemplateEngine
	analyzer   *SchemaAnalyzer
	validators map[string]Validator
}

// SchemaAnalyzer analyzes MCP schemas and extracts code generation information
type SchemaAnalyzer struct {
	config *config.Config
}

// Validator validates generated code
type Validator interface {
	Validate(code string) error
}

// GenerationResult contains the result of code generation
type GenerationResult struct {
	Files       []GeneratedFile
	Errors      []error
	Warnings    []string
	Duration    time.Duration
	Language    string
	PackageName string
}

// GeneratedFile represents a generated file
type GeneratedFile struct {
	Path     string
	Content  string
	Language string
	Type     string // "client", "server", "types", "tests", "docs"
}

// New creates a new code generator
func New(cfg *config.Config) (*Generator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	templateEngine := templates.NewTemplateEngine(cfg)
	if err := templateEngine.LoadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	analyzer := &SchemaAnalyzer{config: cfg}
	validators := make(map[string]Validator)
	
	// Register language validators
	validators["go"] = &GoValidator{}
	validators["typescript"] = &TypeScriptValidator{}
	validators["python"] = &PythonValidator{}
	validators["rust"] = &RustValidator{}
	validators["java"] = &JavaValidator{}

	return &Generator{
		config:     cfg,
		templates:  templateEngine,
		analyzer:   analyzer,
		validators: validators,
	}, nil
}

// GenerateClientFromServer generates a client SDK from a running MCP server
func (g *Generator) GenerateClientFromServer(ctx context.Context, serverPath string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s client from server: %s\n", g.config.Language, serverPath)
	}

	// Extract tools from server
	tools, err := g.extractToolsFromServer(ctx, serverPath)
	if err != nil {
		return fmt.Errorf("failed to extract tools from server: %w", err)
	}

	// Generate client code
	return g.generateClient(tools)
}

// GenerateClientFromSchema generates a client SDK from a schema file
func (g *Generator) GenerateClientFromSchema(ctx context.Context, schemaPath string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s client from schema: %s\n", g.config.Language, schemaPath)
	}

	// Parse schema file
	tools, err := g.parseSchemaFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse schema file: %w", err)
	}

	// Generate client code
	return g.generateClient(tools)
}

// GenerateServerStub generates a server stub from tools definition
func (g *Generator) GenerateServerStub(ctx context.Context, toolsPath string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s server stub from: %s\n", g.config.Language, toolsPath)
	}

	// Parse tools definition
	tools, err := g.parseToolsFile(toolsPath)
	if err != nil {
		return fmt.Errorf("failed to parse tools file: %w", err)
	}

	// Generate server code
	return g.generateServer(tools)
}

// GenerateTypes generates types from JSON schema
func (g *Generator) GenerateTypes(ctx context.Context, schemaPath string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s types from schema: %s\n", g.config.Language, schemaPath)
	}

	// Parse schema
	schema, err := g.parseJSONSchema(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Generate types
	return g.generateTypesFromSchema(schema)
}

// GenerateDocs generates documentation
func (g *Generator) GenerateDocs(ctx context.Context, input string) error {
	if g.config.Verbose {
		fmt.Printf("Generating documentation from: %s\n", input)
	}

	// Determine input type and extract information
	var tools []templates.Tool
	var err error

	if strings.HasSuffix(input, ".json") {
		tools, err = g.parseSchemaFile(input)
	} else {
		tools, err = g.extractToolsFromServer(ctx, input)
	}

	if err != nil {
		return fmt.Errorf("failed to extract tools for documentation: %w", err)
	}

	// Generate documentation
	return g.generateDocumentation(tools)
}

// GenerateTests generates test suites
func (g *Generator) GenerateTests(ctx context.Context, input string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s tests from: %s\n", g.config.Language, input)
	}

	// Determine input type and extract information
	var tools []templates.Tool
	var err error

	if strings.HasSuffix(input, ".json") {
		tools, err = g.parseSchemaFile(input)
	} else if isDirectory(input) {
		// Parse generated code directory
		tools, err = g.extractToolsFromGeneratedCode(input)
	} else {
		tools, err = g.extractToolsFromServer(ctx, input)
	}

	if err != nil {
		return fmt.Errorf("failed to extract tools for tests: %w", err)
	}

	// Generate tests
	return g.generateTestSuite(tools)
}

// GeneratePlugin generates plugin boilerplate
func (g *Generator) GeneratePlugin(ctx context.Context, pluginName string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s plugin: %s\n", g.config.Language, pluginName)
	}

	// Generate plugin code
	return g.generatePluginBoilerplate(pluginName)
}

// extractToolsFromServer extracts tools from a running MCP server
func (g *Generator) extractToolsFromServer(ctx context.Context, serverPath string) ([]templates.Tool, error) {
	// Create a temporary client to connect to server
	client, err := mcp.NewClient(mcp.StdioTransport())
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Start server process
	cmd := exec.CommandContext(ctx, serverPath)
	cmd.Stderr = os.Stderr
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	
	defer func() {
		stdin.Close()
		stdout.Close()
		cmd.Process.Kill()
	}()

	// Initialize client
	initReq := mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		Capabilities: mcp.ClientCapabilities{
			Experimental: map[string]interface{}{},
			Sampling:     map[string]interface{}{},
		},
		ClientInfo: mcp.Implementation{
			Name:    "mcp-gen",
			Version: "0.1.0",
		},
	}

	_, err = client.Initialize(ctx, initReq)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	// List tools
	toolsResp, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert to template tools
	var tools []templates.Tool
	for _, tool := range toolsResp.Tools {
		templateTool := templates.Tool{
			Name:        tool.Name,
			Description: tool.Description,
		}

		// Parse input schema
		if tool.InputSchema != nil {
			var schema map[string]interface{}
			if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
				templateTool.InputSchema = schema
				templateTool.InputType = g.analyzer.GenerateTypeName(tool.Name, "Input")
			}
		}

		tools = append(tools, templateTool)
	}

	return tools, nil
}

// parseSchemaFile parses a schema file and extracts tools
func (g *Generator) parseSchemaFile(path string) ([]templates.Tool, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Check if it's a tools response or individual tool
	if result, ok := schema["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if toolsArray, ok := resultMap["tools"]; ok {
				return g.parseToolsArray(toolsArray)
			}
		}
	}

	// Single tool
	if _, ok := schema["name"]; ok {
		tool, err := g.parseTool(schema)
		if err != nil {
			return nil, err
		}
		return []templates.Tool{tool}, nil
	}

	return nil, fmt.Errorf("unsupported schema format")
}

// parseToolsFile parses a tools definition file
func (g *Generator) parseToolsFile(path string) ([]templates.Tool, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}

	var toolsData interface{}
	if err := json.Unmarshal(data, &toolsData); err != nil {
		return nil, fmt.Errorf("failed to parse tools file: %w", err)
	}

	return g.parseToolsArray(toolsData)
}

// parseToolsArray parses an array of tools
func (g *Generator) parseToolsArray(data interface{}) ([]templates.Tool, error) {
	toolsArray, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("tools must be an array")
	}

	var tools []templates.Tool
	for _, toolData := range toolsArray {
		toolMap, ok := toolData.(map[string]interface{})
		if !ok {
			continue
		}

		tool, err := g.parseTool(toolMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tool: %w", err)
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// parseTool parses a single tool definition
func (g *Generator) parseTool(data map[string]interface{}) (templates.Tool, error) {
	tool := templates.Tool{}

	if name, ok := data["name"].(string); ok {
		tool.Name = name
	}

	if desc, ok := data["description"].(string); ok {
		tool.Description = desc
	}

	if inputSchema, ok := data["inputSchema"]; ok {
		if schemaMap, ok := inputSchema.(map[string]interface{}); ok {
			tool.InputSchema = schemaMap
			tool.InputType = g.analyzer.GenerateTypeName(tool.Name, "Input")
		}
	}

	if outputSchema, ok := data["outputSchema"]; ok {
		if schemaMap, ok := outputSchema.(map[string]interface{}); ok {
			tool.OutputSchema = schemaMap
			tool.OutputType = g.analyzer.GenerateTypeName(tool.Name, "Output")
		}
	}

	return tool, nil
}

// parseJSONSchema parses a JSON schema file
func (g *Generator) parseJSONSchema(path string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	return schema, nil
}

// extractToolsFromGeneratedCode extracts tools from generated code directory
func (g *Generator) extractToolsFromGeneratedCode(dir string) ([]templates.Tool, error) {
	// TODO: Implement code analysis to extract tools
	return []templates.Tool{}, nil
}

// generateClient generates client code
func (g *Generator) generateClient(tools []templates.Tool) error {
	templateName := fmt.Sprintf("%s/client", g.config.Language)
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Tools:    tools,
		Client: &templates.Client{
			Name:        g.config.GetPackageName() + "Client",
			Description: "Generated MCP client",
			Tools:       tools,
		},
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}

	// Write client file
	filename := g.getClientFilename()
	return g.writeFile(filename, content)
}

// generateServer generates server code
func (g *Generator) generateServer(tools []templates.Tool) error {
	templateName := fmt.Sprintf("%s/server", g.config.Language)
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Tools:    tools,
		Server: &templates.Server{
			Name:        g.config.GetPackageName() + "Server",
			Description: "Generated MCP server",
			Tools:       tools,
		},
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate server: %w", err)
	}

	// Write server file
	filename := g.getServerFilename()
	return g.writeFile(filename, content)
}

// generateTypesFromSchema generates types from schema
func (g *Generator) generateTypesFromSchema(schema map[string]interface{}) error {
	templateName := fmt.Sprintf("%s/types", g.config.Language)
	
	types := g.analyzer.SchemaToTypes(schema)
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Types:    types,
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}

	// Write types file
	filename := g.getTypesFilename()
	return g.writeFile(filename, content)
}

// generateDocumentation generates documentation
func (g *Generator) generateDocumentation(tools []templates.Tool) error {
	templateName := "docs/markdown"
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Tools:    tools,
		Docs: &templates.Documentation{
			Title:       g.config.GetPackageName(),
			Description: "Generated MCP documentation",
			Tools:       tools,
		},
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	// Write documentation file
	filename := filepath.Join(g.config.Output, "README.md")
	return g.writeFile(filename, content)
}

// generateTestSuite generates test suite
func (g *Generator) generateTestSuite(tools []templates.Tool) error {
	templateName := fmt.Sprintf("%s/tests", g.config.Language)
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Tools:    tools,
		Tests: &templates.TestSuite{
			Name:  g.config.GetPackageName() + "Tests",
			Tools: tools,
		},
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate tests: %w", err)
	}

	// Write tests file
	filename := g.getTestsFilename()
	return g.writeFile(filename, content)
}

// generatePluginBoilerplate generates plugin boilerplate
func (g *Generator) generatePluginBoilerplate(pluginName string) error {
	templateName := fmt.Sprintf("%s/plugin", g.config.Language)
	
	data := &templates.TemplateData{
		Config:   g.config,
		Language: g.config.Language,
		Package:  g.config.GetPackageName(),
		Plugin: &templates.Plugin{
			Name:        pluginName,
			Description: "Generated MCP plugin",
		},
	}

	content, err := g.templates.ExecuteTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to generate plugin: %w", err)
	}

	// Write plugin file
	filename := g.getPluginFilename(pluginName)
	return g.writeFile(filename, content)
}

// Helper methods

func (g *Generator) getClientFilename() string {
	switch g.config.Language {
	case "go":
		return filepath.Join(g.config.Output, "client.go")
	case "typescript":
		return filepath.Join(g.config.Output, "client.ts")
	case "python":
		return filepath.Join(g.config.Output, "client.py")
	case "rust":
		return filepath.Join(g.config.Output, "client.rs")
	case "java":
		return filepath.Join(g.config.Output, "Client.java")
	}
	return filepath.Join(g.config.Output, "client")
}

func (g *Generator) getServerFilename() string {
	switch g.config.Language {
	case "go":
		return filepath.Join(g.config.Output, "server.go")
	case "typescript":
		return filepath.Join(g.config.Output, "server.ts")
	case "python":
		return filepath.Join(g.config.Output, "server.py")
	case "rust":
		return filepath.Join(g.config.Output, "server.rs")
	case "java":
		return filepath.Join(g.config.Output, "Server.java")
	}
	return filepath.Join(g.config.Output, "server")
}

func (g *Generator) getTypesFilename() string {
	switch g.config.Language {
	case "go":
		return filepath.Join(g.config.Output, "types.go")
	case "typescript":
		return filepath.Join(g.config.Output, "types.ts")
	case "python":
		return filepath.Join(g.config.Output, "types.py")
	case "rust":
		return filepath.Join(g.config.Output, "types.rs")
	case "java":
		return filepath.Join(g.config.Output, "Types.java")
	}
	return filepath.Join(g.config.Output, "types")
}

func (g *Generator) getTestsFilename() string {
	switch g.config.Language {
	case "go":
		return filepath.Join(g.config.Output, "client_test.go")
	case "typescript":
		return filepath.Join(g.config.Output, "client.test.ts")
	case "python":
		return filepath.Join(g.config.Output, "test_client.py")
	case "rust":
		return filepath.Join(g.config.Output, "tests.rs")
	case "java":
		return filepath.Join(g.config.Output, "ClientTest.java")
	}
	return filepath.Join(g.config.Output, "tests")
}

func (g *Generator) getPluginFilename(pluginName string) string {
	switch g.config.Language {
	case "go":
		return filepath.Join(g.config.Output, pluginName+".go")
	case "typescript":
		return filepath.Join(g.config.Output, pluginName+".ts")
	case "python":
		return filepath.Join(g.config.Output, pluginName+".py")
	case "rust":
		return filepath.Join(g.config.Output, pluginName+".rs")
	case "java":
		return filepath.Join(g.config.Output, strings.Title(pluginName)+".java")
	}
	return filepath.Join(g.config.Output, pluginName)
}

func (g *Generator) writeFile(filename, content string) error {
	if g.config.DryRun {
		fmt.Printf("--- %s ---\n", filename)
		fmt.Println(content)
		return nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Validate code if validator exists
	if validator, ok := g.validators[g.config.Language]; ok {
		if err := validator.Validate(content); err != nil {
			return fmt.Errorf("validation failed for %s: %w", filename, err)
		}
	}

	// Write file
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if g.config.Verbose {
		fmt.Printf("Generated: %s\n", filename)
	}

	return nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Language-specific validators

type GoValidator struct{}

func (v *GoValidator) Validate(code string) error {
	// TODO: Implement Go validation
	return nil
}

type TypeScriptValidator struct{}

func (v *TypeScriptValidator) Validate(code string) error {
	// TODO: Implement TypeScript validation
	return nil
}

type PythonValidator struct{}

func (v *PythonValidator) Validate(code string) error {
	// TODO: Implement Python validation
	return nil
}

type RustValidator struct{}

func (v *RustValidator) Validate(code string) error {
	// TODO: Implement Rust validation
	return nil
}

type JavaValidator struct{}

func (v *JavaValidator) Validate(code string) error {
	// TODO: Implement Java validation
	return nil
}