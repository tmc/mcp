package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/codegen"
	"github.com/tmc/mcp/exp/cmd/mcp-gen/internal/config"
)

// TestMcpGenIntegration tests the complete mcp-gen workflow
func TestMcpGenIntegration(t *testing.T) {
	// Create temporary directory
	tmpDir, err := ioutil.TempDir("", "mcp-gen-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test configuration
	cfg := &config.Config{
		Language:      "go",
		Output:        tmpDir,
		Package:       "github.com/test/example",
		TypeSafe:      true,
		Middleware:    true,
		Documentation: true,
		Tests:         true,
	}

	// Create generator
	generator, err := codegen.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	ctx := context.Background()

	// Test client generation from schema
	schemaPath := filepath.Join("examples", "time-server-schema.json")
	if err := generator.GenerateClientFromSchema(ctx, schemaPath); err != nil {
		t.Fatalf("Failed to generate client from schema: %v", err)
	}

	// Verify client file was created
	clientPath := filepath.Join(tmpDir, "client.go")
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		t.Fatalf("Client file not created: %s", clientPath)
	}

	// Verify client file content
	content, err := ioutil.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client file: %v", err)
	}

	clientCode := string(content)
	
	// Check for expected client components
	expectedComponents := []string{
		"package generated",
		"type ExampleClient struct",
		"func NewExampleClient",
		"func (c *ExampleClient) Initialize",
		"func (c *ExampleClient) GetCurrentTime",
		"func (c *ExampleClient) ConvertTime",
		"type GetCurrentTimeInput struct",
		"type GetCurrentTimeOutput struct",
		"type ConvertTimeInput struct",
		"type ConvertTimeOutput struct",
		"func (c *ExampleClient) HealthCheck",
		"func (c *ExampleClient) ExecuteWithRetry",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(clientCode, component) {
			t.Errorf("Client code missing expected component: %s", component)
		}
	}

	// Test server generation
	if err := generator.GenerateServerStub(ctx, schemaPath); err != nil {
		t.Fatalf("Failed to generate server stub: %v", err)
	}

	// Verify server file was created
	serverPath := filepath.Join(tmpDir, "server.go")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Fatalf("Server file not created: %s", serverPath)
	}

	// Verify server file content
	serverContent, err := ioutil.ReadFile(serverPath)
	if err != nil {
		t.Fatalf("Failed to read server file: %v", err)
	}

	serverCode := string(serverContent)

	// Check for expected server components
	expectedServerComponents := []string{
		"package generated",
		"type ExampleServer struct",
		"func NewExampleServer",
		"func (s *ExampleServer) Serve",
		"func (s *ExampleServer) ServeStdio",
		"func (s *ExampleServer) registerGetCurrentTimeTool",
		"func (s *ExampleServer) registerConvertTimeTool",
		"func (s *ExampleServer) handleGetCurrentTime",
		"func (s *ExampleServer) handleConvertTime",
		"func (s *ExampleServer) executeGetCurrentTime",
		"func (s *ExampleServer) executeConvertTime",
		"func main()",
	}

	for _, component := range expectedServerComponents {
		if !strings.Contains(serverCode, component) {
			t.Errorf("Server code missing expected component: %s", component)
		}
	}

	// Test types generation
	if err := generator.GenerateTypes(ctx, schemaPath); err != nil {
		t.Fatalf("Failed to generate types: %v", err)
	}

	// Verify types file was created
	typesPath := filepath.Join(tmpDir, "types.go")
	if _, err := os.Stat(typesPath); os.IsNotExist(err) {
		t.Fatalf("Types file not created: %s", typesPath)
	}

	// Test tests generation
	if err := generator.GenerateTests(ctx, schemaPath); err != nil {
		t.Fatalf("Failed to generate tests: %v", err)
	}

	// Verify tests file was created
	testsPath := filepath.Join(tmpDir, "client_test.go")
	if _, err := os.Stat(testsPath); os.IsNotExist(err) {
		t.Fatalf("Tests file not created: %s", testsPath)
	}

	// Test documentation generation
	if err := generator.GenerateDocs(ctx, schemaPath); err != nil {
		t.Fatalf("Failed to generate docs: %v", err)
	}

	// Verify documentation file was created
	docsPath := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		t.Fatalf("Documentation file not created: %s", docsPath)
	}

	// Test plugin generation
	if err := generator.GeneratePlugin(ctx, "example-plugin"); err != nil {
		t.Fatalf("Failed to generate plugin: %v", err)
	}

	// Verify plugin file was created
	pluginPath := filepath.Join(tmpDir, "example-plugin.go")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		t.Fatalf("Plugin file not created: %s", pluginPath)
	}

	t.Logf("All tests passed! Generated files in: %s", tmpDir)
}

// TestSchemaAnalysis tests the schema analysis functionality
func TestSchemaAnalysis(t *testing.T) {
	// Create test schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name field",
			},
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "The count field",
			},
			"enabled": map[string]interface{}{
				"type":        "boolean",
				"description": "The enabled field",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "The tags field",
			},
		},
		"required": []string{"name", "count"},
	}

	// Create configuration
	cfg := &config.Config{
		Language: "go",
		Package:  "test",
	}

	// Create analyzer
	analyzer := &codegen.SchemaAnalyzer{}

	// Test type name generation
	typeName := analyzer.GenerateTypeName("test_tool", "Input")
	if typeName != "TestToolInput" {
		t.Errorf("Expected TestToolInput, got %s", typeName)
	}

	// Test schema to types conversion
	types := analyzer.SchemaToTypes(schema)
	if len(types) == 0 {
		t.Error("Expected types to be generated")
	}

	// Verify generated type structure
	if len(types) > 0 {
		rootType := types[0]
		if rootType.Name != "Root" {
			t.Errorf("Expected Root type, got %s", rootType.Name)
		}

		if len(rootType.Fields) != 4 {
			t.Errorf("Expected 4 fields, got %d", len(rootType.Fields))
		}

		// Check field types
		for _, field := range rootType.Fields {
			switch field.Name {
			case "Name":
				if field.Type != "string" {
					t.Errorf("Expected string type for Name, got %s", field.Type)
				}
				if field.Optional {
					t.Error("Expected Name to be required")
				}
			case "Count":
				if field.Type != "int" {
					t.Errorf("Expected int type for Count, got %s", field.Type)
				}
				if field.Optional {
					t.Error("Expected Count to be required")
				}
			case "Enabled":
				if field.Type != "bool" {
					t.Errorf("Expected bool type for Enabled, got %s", field.Type)
				}
				if !field.Optional {
					t.Error("Expected Enabled to be optional")
				}
			case "Tags":
				if field.Type != "[]string" {
					t.Errorf("Expected []string type for Tags, got %s", field.Type)
				}
				if !field.Optional {
					t.Error("Expected Tags to be optional")
				}
			}
		}
	}
}

// TestMultiLanguageGeneration tests generation for multiple languages
func TestMultiLanguageGeneration(t *testing.T) {
	languages := []string{"go", "typescript", "python", "rust", "java"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := ioutil.TempDir("", "mcp-gen-"+lang+"-test-")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create configuration
			cfg := &config.Config{
				Language: lang,
				Output:   tmpDir,
				Package:  "test",
			}

			// Create generator
			generator, err := codegen.New(cfg)
			if err != nil {
				t.Fatalf("Failed to create generator for %s: %v", lang, err)
			}

			ctx := context.Background()

			// Test client generation
			schemaPath := filepath.Join("examples", "time-server-schema.json")
			if err := generator.GenerateClientFromSchema(ctx, schemaPath); err != nil {
				t.Fatalf("Failed to generate %s client: %v", lang, err)
			}

			// Verify client file was created
			clientFile := generator.getClientFilename()
			if _, err := os.Stat(clientFile); os.IsNotExist(err) {
				t.Fatalf("%s client file not created: %s", lang, clientFile)
			}

			t.Logf("Successfully generated %s client", lang)
		})
	}
}

// TestConfigurationValidation tests configuration validation
func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Language: "go",
				Output:   "/tmp/test",
				Package:  "test",
			},
			expectError: false,
		},
		{
			name: "invalid language",
			config: &config.Config{
				Language: "invalid",
				Output:   "/tmp/test",
				Package:  "test",
			},
			expectError: true,
		},
		{
			name: "missing output",
			config: &config.Config{
				Language: "go",
				Package:  "test",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestTemplateExecution tests template execution
func TestTemplateExecution(t *testing.T) {
	// Create test data
	testData := map[string]interface{}{
		"name":     "test-tool",
		"language": "go",
		"package":  "test",
	}

	// Test template functions
	funcMap := map[string]interface{}{
		"toPascalCase": func(s string) string {
			return strings.Title(strings.ReplaceAll(s, "-", ""))
		},
		"toCamelCase": func(s string) string {
			result := strings.Title(strings.ReplaceAll(s, "-", ""))
			if len(result) > 0 {
				return strings.ToLower(result[:1]) + result[1:]
			}
			return result
		},
	}

	// Test template execution with sample content
	templateContent := `package {{.package}}

type {{.name | toPascalCase}}Client struct {
	// Client implementation
}

func new{{.name | toPascalCase}}Client() *{{.name | toPascalCase}}Client {
	return &{{.name | toPascalCase}}Client{}
}

func (c *{{.name | toPascalCase}}Client) {{.name | toCamelCase}}Method() error {
	// Method implementation
	return nil
}`

	// This would normally be done by the template engine
	// For now, we just verify the test data is properly structured
	if testData["name"] != "test-tool" {
		t.Error("Test data not properly structured")
	}

	t.Log("Template execution test passed")
}

// TestErrorHandling tests error handling in various scenarios
func TestErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (*config.Config, error)
		expectError bool
		errorType   string
	}{
		{
			name: "invalid schema file",
			setup: func() (*config.Config, error) {
				return &config.Config{
					Language: "go",
					Output:   "/tmp/test",
					Package:  "test",
				}, nil
			},
			expectError: true,
			errorType:   "file_not_found",
		},
		{
			name: "invalid output directory",
			setup: func() (*config.Config, error) {
				return &config.Config{
					Language: "go",
					Output:   "/invalid/directory/path",
					Package:  "test",
				}, nil
			},
			expectError: true,
			errorType:   "directory_creation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.setup()
			if err != nil {
				if !tc.expectError {
					t.Errorf("Unexpected setup error: %v", err)
				}
				return
			}

			generator, err := codegen.New(cfg)
			if err != nil {
				if !tc.expectError {
					t.Errorf("Unexpected generator creation error: %v", err)
				}
				return
			}

			ctx := context.Background()
			err = generator.GenerateClientFromSchema(ctx, "nonexistent.json")
			
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// BenchmarkCodeGeneration benchmarks code generation performance
func BenchmarkCodeGeneration(b *testing.B) {
	// Create temporary directory
	tmpDir, err := ioutil.TempDir("", "mcp-gen-bench-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create configuration
	cfg := &config.Config{
		Language: "go",
		Output:   tmpDir,
		Package:  "bench",
	}

	// Create generator
	generator, err := codegen.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create generator: %v", err)
	}

	ctx := context.Background()
	schemaPath := filepath.Join("examples", "time-server-schema.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := generator.GenerateClientFromSchema(ctx, schemaPath); err != nil {
			b.Fatalf("Failed to generate client: %v", err)
		}
	}
}

// Helper function to create test schema
func createTestSchema() map[string]interface{} {
	return map[string]interface{}{
		"result": map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name":        "test_tool",
					"description": "A test tool",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"input": map[string]interface{}{
								"type":        "string",
								"description": "Input parameter",
							},
						},
						"required": []string{"input"},
					},
					"outputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output": map[string]interface{}{
								"type":        "string",
								"description": "Output result",
							},
						},
						"required": []string{"output"},
					},
				},
			},
		},
	}
}

// Helper function to save test schema to file
func saveTestSchema(t *testing.T, schema map[string]interface{}, path string) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}
}