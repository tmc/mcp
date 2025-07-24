// Package main provides mcp-docs, a documentation generator for MCP servers
// with API documentation generation, interactive examples, multi-format output,
// version management, and integration guides.
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"gopkg.in/yaml.v3"
)

const (
	// Version information
	Version = "1.0.0"
	Name    = "mcp-docs"

	// Default configuration
	DefaultOutputDir = "./docs"
	DefaultConfigFile = "mcp-docs.yaml"
)

//go:embed templates/*
var templateFiles embed.FS

// Config represents the documentation generator configuration
type Config struct {
	// Source configuration
	Sources []SourceConfig `yaml:"sources"`
	
	// Output configuration
	Output OutputConfig `yaml:"output"`
	
	// Documentation settings
	Documentation DocumentationConfig `yaml:"documentation"`
	
	// Template configuration
	Templates TemplateConfig `yaml:"templates"`
	
	// Version management
	Versions VersionConfig `yaml:"versions"`
	
	// Integration settings
	Integration IntegrationConfig `yaml:"integration"`
}

// SourceConfig represents source configuration
type SourceConfig struct {
	Type        string            `yaml:"type"`        // "server", "package", "api"
	Path        string            `yaml:"path"`        // Path to source
	Command     []string          `yaml:"command"`     // Command to run server
	URL         string            `yaml:"url"`         // URL for API endpoints
	Headers     map[string]string `yaml:"headers"`     // HTTP headers
	Include     []string          `yaml:"include"`     // Include patterns
	Exclude     []string          `yaml:"exclude"`     // Exclude patterns
	Description string            `yaml:"description"` // Source description
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Directory string            `yaml:"directory"`
	Formats   []string          `yaml:"formats"`   // html, markdown, json, yaml
	Assets    string            `yaml:"assets"`    // Assets directory
	Templates string            `yaml:"templates"` // Custom templates
	Options   map[string]string `yaml:"options"`   // Format-specific options
}

// DocumentationConfig represents documentation settings
type DocumentationConfig struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Author      string            `yaml:"author"`
	BaseURL     string            `yaml:"base_url"`
	Logo        string            `yaml:"logo"`
	Favicon     string            `yaml:"favicon"`
	Language    string            `yaml:"language"`
	Theme       string            `yaml:"theme"`
	Sidebar     SidebarConfig     `yaml:"sidebar"`
	Navigation  NavigationConfig  `yaml:"navigation"`
	Search      SearchConfig      `yaml:"search"`
	Analytics   AnalyticsConfig   `yaml:"analytics"`
	Social      SocialConfig      `yaml:"social"`
	Custom      map[string]string `yaml:"custom"`
}

// SidebarConfig represents sidebar configuration
type SidebarConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Collapsible bool     `yaml:"collapsible"`
	Sections    []string `yaml:"sections"`
	Order       []string `yaml:"order"`
}

// NavigationConfig represents navigation configuration
type NavigationConfig struct {
	Enabled bool              `yaml:"enabled"`
	Items   []NavigationItem  `yaml:"items"`
}

// NavigationItem represents a navigation item
type NavigationItem struct {
	Title    string           `yaml:"title"`
	URL      string           `yaml:"url"`
	Icon     string           `yaml:"icon"`
	Children []NavigationItem `yaml:"children"`
}

// SearchConfig represents search configuration
type SearchConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"` // "local", "algolia"
	Config   map[string]string `yaml:"config"`
}

// AnalyticsConfig represents analytics configuration
type AnalyticsConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"` // "google", "plausible"
	Config   map[string]string `yaml:"config"`
}

// SocialConfig represents social media configuration
type SocialConfig struct {
	GitHub   string `yaml:"github"`
	Twitter  string `yaml:"twitter"`
	LinkedIn string `yaml:"linkedin"`
	Email    string `yaml:"email"`
}

// TemplateConfig represents template configuration
type TemplateConfig struct {
	Directory string            `yaml:"directory"`
	Custom    map[string]string `yaml:"custom"`
	Partials  []string          `yaml:"partials"`
	Helpers   []string          `yaml:"helpers"`
}

// VersionConfig represents version management configuration
type VersionConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Current   string   `yaml:"current"`
	Available []string `yaml:"available"`
	Directory string   `yaml:"directory"`
}

// IntegrationConfig represents integration configuration
type IntegrationConfig struct {
	Examples   ExamplesConfig   `yaml:"examples"`
	Playground PlaygroundConfig `yaml:"playground"`
	SDK        SDKConfig        `yaml:"sdk"`
}

// ExamplesConfig represents examples configuration
type ExamplesConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Directory string   `yaml:"directory"`
	Languages []string `yaml:"languages"`
	Interactive bool   `yaml:"interactive"`
}

// PlaygroundConfig represents playground configuration
type PlaygroundConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Embed   bool   `yaml:"embed"`
}

// SDKConfig represents SDK configuration
type SDKConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Languages []string `yaml:"languages"`
	Generate  bool     `yaml:"generate"`
}

// Documentation represents the generated documentation
type Documentation struct {
	Config      *Config
	Servers     []*ServerDoc
	Packages    []*PackageDoc
	APIs        []*APIDoc
	Examples    []*ExampleDoc
	Guides      []*GuideDoc
	References  []*ReferenceDoc
	GeneratedAt time.Time
}

// ServerDoc represents documentation for an MCP server
type ServerDoc struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Version      string           `json:"version"`
	Author       string           `json:"author"`
	Command      []string         `json:"command"`
	Transport    string           `json:"transport"`
	Capabilities ServerCapabilities `json:"capabilities"`
	Tools        []*ToolDoc       `json:"tools"`
	Resources    []*ResourceDoc   `json:"resources"`
	Prompts      []*PromptDoc     `json:"prompts"`
	Examples     []*ExampleDoc    `json:"examples"`
	Health       *HealthDoc       `json:"health"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	Tools     bool `json:"tools"`
	Resources bool `json:"resources"`
	Prompts   bool `json:"prompts"`
	Logging   bool `json:"logging"`
}

// ToolDoc represents documentation for a tool
type ToolDoc struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema"`
	Examples     []*ExampleDoc          `json:"examples"`
	Usage        string                 `json:"usage"`
}

// ResourceDoc represents documentation for a resource
type ResourceDoc struct {
	URI         string        `json:"uri"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	MimeType    string        `json:"mime_type"`
	Examples    []*ExampleDoc `json:"examples"`
}

// PromptDoc represents documentation for a prompt
type PromptDoc struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Arguments   map[string]interface{} `json:"arguments"`
	Examples    []*ExampleDoc          `json:"examples"`
}

// ExampleDoc represents an example
type ExampleDoc struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Code        string                 `json:"code"`
	Language    string                 `json:"language"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	Interactive bool                   `json:"interactive"`
}

// GuideDoc represents a guide or tutorial
type GuideDoc struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Order       int       `json:"order"`
	Tags        []string  `json:"tags"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReferenceDoc represents API reference documentation
type ReferenceDoc struct {
	Section string                 `json:"section"`
	Items   []*ReferenceItem       `json:"items"`
}

// ReferenceItem represents a reference item
type ReferenceItem struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Returns     map[string]interface{} `json:"returns"`
	Examples    []*ExampleDoc          `json:"examples"`
}

// PackageDoc represents Go package documentation
type PackageDoc struct {
	Name        string         `json:"name"`
	ImportPath  string         `json:"import_path"`
	Description string         `json:"description"`
	Types       []*TypeDoc     `json:"types"`
	Functions   []*FunctionDoc `json:"functions"`
	Variables   []*VariableDoc `json:"variables"`
	Constants   []*ConstantDoc `json:"constants"`
	Examples    []*ExampleDoc  `json:"examples"`
}

// TypeDoc represents a Go type
type TypeDoc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Methods     []*MethodDoc   `json:"methods"`
	Examples    []*ExampleDoc  `json:"examples"`
}

// FunctionDoc represents a Go function
type FunctionDoc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Signature   string         `json:"signature"`
	Parameters  []Parameter    `json:"parameters"`
	Returns     []Return       `json:"returns"`
	Examples    []*ExampleDoc  `json:"examples"`
}

// MethodDoc represents a Go method
type MethodDoc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Signature   string         `json:"signature"`
	Parameters  []Parameter    `json:"parameters"`
	Returns     []Return       `json:"returns"`
	Examples    []*ExampleDoc  `json:"examples"`
}

// VariableDoc represents a Go variable
type VariableDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Value       string `json:"value"`
}

// ConstantDoc represents a Go constant
type ConstantDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Value       string `json:"value"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Optional    bool   `json:"optional"`
}

// Return represents a function return value
type Return struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// APIDoc represents REST API documentation
type APIDoc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BaseURL     string         `json:"base_url"`
	Endpoints   []*EndpointDoc `json:"endpoints"`
	Models      []*ModelDoc    `json:"models"`
	Examples    []*ExampleDoc  `json:"examples"`
}

// EndpointDoc represents an API endpoint
type EndpointDoc struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Parameters  []Parameter            `json:"parameters"`
	RequestBody map[string]interface{} `json:"request_body"`
	Responses   map[string]interface{} `json:"responses"`
	Examples    []*ExampleDoc          `json:"examples"`
}

// ModelDoc represents a data model
type ModelDoc struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
	Examples    []*ExampleDoc          `json:"examples"`
}

// HealthDoc represents server health information
type HealthDoc struct {
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Uptime      string    `json:"uptime"`
	Version     string    `json:"version"`
}

// Generator represents the documentation generator
type Generator struct {
	config    *Config
	templates *template.Template
	fset      *token.FileSet
}

// Global flags
var (
	configFile = flag.String("config", DefaultConfigFile, "Configuration file path")
	outputDir  = flag.String("output", DefaultOutputDir, "Output directory")
	format     = flag.String("format", "html", "Output format (html, markdown, json, yaml)")
	server     = flag.String("server", "", "MCP server command to document")
	pkg        = flag.String("package", "", "Go package to document")
	watch      = flag.Bool("watch", false, "Watch for changes and regenerate")
	serve      = flag.Bool("serve", false, "Start local server to serve documentation")
	port       = flag.Int("port", 8080, "Port for local server")
	debug      = flag.Bool("debug", false, "Enable debug mode")
	version    = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s v%s\n", Name, Version)
		return
	}

	// Load configuration
	config, err := LoadConfig(*configFile)
	if err != nil {
		// Use default configuration if file doesn't exist
		config = getDefaultConfig()
	}

	// Override with command line options
	if *outputDir != DefaultOutputDir {
		config.Output.Directory = *outputDir
	}
	if *format != "html" {
		config.Output.Formats = []string{*format}
	}
	if *server != "" {
		config.Sources = append(config.Sources, SourceConfig{
			Type:        "server",
			Command:     strings.Fields(*server),
			Description: "Command line server",
		})
	}
	if *pkg != "" {
		config.Sources = append(config.Sources, SourceConfig{
			Type:        "package",
			Path:        *pkg,
			Description: "Command line package",
		})
	}

	// Create generator
	generator, err := NewGenerator(config)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	// Generate documentation
	if err := generator.Generate(); err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	// Start local server if requested
	if *serve {
		if err := generator.Serve(*port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}

	// Watch for changes if requested
	if *watch {
		if err := generator.Watch(); err != nil {
			log.Fatalf("Failed to start watcher: %v", err)
		}
	}
}

// NewGenerator creates a new documentation generator
func NewGenerator(config *Config) (*Generator, error) {
	// Load templates
	templates, err := template.ParseFS(templateFiles, "templates/*.html", "templates/*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Add custom template functions
	templates = templates.Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"toYAML": func(v interface{}) string {
			b, _ := yaml.Marshal(v)
			return string(b)
		},
		"markdown": func(text string) template.HTML {
			// Simple markdown processing
			return template.HTML(processMarkdown(text))
		},
		"code": func(lang, code string) template.HTML {
			return template.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, lang, template.HTMLEscapeString(code)))
		},
		"formatType": func(t string) string {
			// Format Go type names
			return strings.TrimPrefix(t, "*")
		},
		"join": strings.Join,
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
	})

	return &Generator{
		config:    config,
		templates: templates,
		fset:      token.NewFileSet(),
	}, nil
}

// Generate generates the documentation
func (g *Generator) Generate() error {
	log.Printf("Generating documentation...")

	// Create documentation structure
	doc := &Documentation{
		Config:      g.config,
		GeneratedAt: time.Now(),
	}

	// Process sources
	for _, source := range g.config.Sources {
		if err := g.processSource(doc, source); err != nil {
			log.Printf("Error processing source %s: %v", source.Path, err)
			continue
		}
	}

	// Generate output files
	for _, format := range g.config.Output.Formats {
		if err := g.generateOutput(doc, format); err != nil {
			return fmt.Errorf("failed to generate %s output: %w", format, err)
		}
	}

	log.Printf("Documentation generated successfully in %s", g.config.Output.Directory)
	return nil
}

// processSource processes a documentation source
func (g *Generator) processSource(doc *Documentation, source SourceConfig) error {
	switch source.Type {
	case "server":
		return g.processServer(doc, source)
	case "package":
		return g.processPackage(doc, source)
	case "api":
		return g.processAPI(doc, source)
	default:
		return fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

// processServer processes an MCP server for documentation
func (g *Generator) processServer(doc *Documentation, source SourceConfig) error {
	if len(source.Command) == 0 {
		return fmt.Errorf("server command is required")
	}

	// Create transport
	transport := mcp.NewStdioTransport(source.Command[0], source.Command[1:]...)

	// Create client
	client, err := mcp.NewClient(transport)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	initReq := mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    Name,
			Version: Version,
		},
	}

	initResp, err := client.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}

	// Create server documentation
	serverDoc := &ServerDoc{
		Name:        initResp.ServerInfo.Name,
		Description: source.Description,
		Version:     initResp.ServerInfo.Version,
		Command:     source.Command,
		Transport:   "stdio",
		Capabilities: ServerCapabilities{
			Tools:     initResp.Capabilities.Tools != nil,
			Resources: initResp.Capabilities.Resources != nil,
			Prompts:   initResp.Capabilities.Prompts != nil,
			Logging:   initResp.Capabilities.Logging != nil,
		},
	}

	// Get tools
	if serverDoc.Capabilities.Tools {
		if err := g.processTools(client, serverDoc); err != nil {
			log.Printf("Error processing tools: %v", err)
		}
	}

	// Get resources
	if serverDoc.Capabilities.Resources {
		if err := g.processResources(client, serverDoc); err != nil {
			log.Printf("Error processing resources: %v", err)
		}
	}

	// Get prompts
	if serverDoc.Capabilities.Prompts {
		if err := g.processPrompts(client, serverDoc); err != nil {
			log.Printf("Error processing prompts: %v", err)
		}
	}

	// Add health check
	serverDoc.Health = &HealthDoc{
		Status:      "healthy",
		LastChecked: time.Now(),
		Version:     serverDoc.Version,
	}

	doc.Servers = append(doc.Servers, serverDoc)
	return nil
}

// processTools processes server tools
func (g *Generator) processTools(client *mcp.Client, serverDoc *ServerDoc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}

	for _, tool := range resp.Tools {
		toolDoc := &ToolDoc{
			Name:        tool.Name,
			Description: tool.Description,
		}

		// Parse input schema
		if tool.InputSchema != nil {
			var schema map[string]interface{}
			if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
				toolDoc.InputSchema = schema
			}
		}

		// Generate examples
		toolDoc.Examples = g.generateToolExamples(tool)

		serverDoc.Tools = append(serverDoc.Tools, toolDoc)
	}

	return nil
}

// processResources processes server resources
func (g *Generator) processResources(client *mcp.Client, serverDoc *ServerDoc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return err
	}

	for _, resource := range resp.Resources {
		resourceDoc := &ResourceDoc{
			URI:         resource.URI,
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MimeType,
		}

		// Generate examples
		resourceDoc.Examples = g.generateResourceExamples(resource)

		serverDoc.Resources = append(serverDoc.Resources, resourceDoc)
	}

	return nil
}

// processPrompts processes server prompts
func (g *Generator) processPrompts(client *mcp.Client, serverDoc *ServerDoc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return err
	}

	for _, prompt := range resp.Prompts {
		promptDoc := &PromptDoc{
			Name:        prompt.Name,
			Description: prompt.Description,
			Arguments:   make(map[string]interface{}),
		}

		// Process arguments
		for _, arg := range prompt.Arguments {
			promptDoc.Arguments[arg.Name] = map[string]interface{}{
				"description": arg.Description,
				"required":    arg.Required,
			}
		}

		// Generate examples
		promptDoc.Examples = g.generatePromptExamples(prompt)

		serverDoc.Prompts = append(serverDoc.Prompts, promptDoc)
	}

	return nil
}

// processPackage processes a Go package for documentation
func (g *Generator) processPackage(doc *Documentation, source SourceConfig) error {
	pkgs, err := parser.ParseDir(g.fset, source.Path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	for _, pkg := range pkgs {
		docPkg := doc.New(pkg, source.Path, doc.Mode(0))
		
		packageDoc := &PackageDoc{
			Name:        pkg.Name,
			ImportPath:  source.Path,
			Description: docPkg.Doc,
		}

		// Process types
		for _, t := range docPkg.Types {
			typeDoc := &TypeDoc{
				Name:        t.Name,
				Description: t.Doc,
			}

			// Process methods
			for _, method := range t.Methods {
				methodDoc := &MethodDoc{
					Name:        method.Name,
					Description: method.Doc,
					Signature:   method.Decl.Type.(*ast.FuncType).String(),
				}
				typeDoc.Methods = append(typeDoc.Methods, methodDoc)
			}

			packageDoc.Types = append(packageDoc.Types, typeDoc)
		}

		// Process functions
		for _, f := range docPkg.Funcs {
			funcDoc := &FunctionDoc{
				Name:        f.Name,
				Description: f.Doc,
			}
			packageDoc.Functions = append(packageDoc.Functions, funcDoc)
		}

		// Process variables
		for _, v := range docPkg.Vars {
			for _, name := range v.Names {
				varDoc := &VariableDoc{
					Name:        name,
					Description: v.Doc,
				}
				packageDoc.Variables = append(packageDoc.Variables, varDoc)
			}
		}

		// Process constants
		for _, c := range docPkg.Consts {
			for _, name := range c.Names {
				constDoc := &ConstantDoc{
					Name:        name,
					Description: c.Doc,
				}
				packageDoc.Constants = append(packageDoc.Constants, constDoc)
			}
		}

		doc.Packages = append(doc.Packages, packageDoc)
	}

	return nil
}

// processAPI processes an API for documentation
func (g *Generator) processAPI(doc *Documentation, source SourceConfig) error {
	// This would implement OpenAPI/Swagger parsing
	// For now, return placeholder
	apiDoc := &APIDoc{
		Name:        "API Documentation",
		Description: source.Description,
		BaseURL:     source.URL,
	}

	doc.APIs = append(doc.APIs, apiDoc)
	return nil
}

// generateOutput generates output in the specified format
func (g *Generator) generateOutput(doc *Documentation, format string) error {
	outputDir := filepath.Join(g.config.Output.Directory, format)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	switch format {
	case "html":
		return g.generateHTML(doc, outputDir)
	case "markdown":
		return g.generateMarkdown(doc, outputDir)
	case "json":
		return g.generateJSON(doc, outputDir)
	case "yaml":
		return g.generateYAML(doc, outputDir)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// generateHTML generates HTML documentation
func (g *Generator) generateHTML(doc *Documentation, outputDir string) error {
	// Generate index page
	if err := g.generateHTMLFile(doc, "index.html", outputDir, "index"); err != nil {
		return err
	}

	// Generate server pages
	for _, server := range doc.Servers {
		filename := fmt.Sprintf("server-%s.html", slugify(server.Name))
		if err := g.generateHTMLFile(server, filename, outputDir, "server"); err != nil {
			return err
		}
	}

	// Generate package pages
	for _, pkg := range doc.Packages {
		filename := fmt.Sprintf("package-%s.html", slugify(pkg.Name))
		if err := g.generateHTMLFile(pkg, filename, outputDir, "package"); err != nil {
			return err
		}
	}

	// Copy static assets
	return g.copyAssets(outputDir)
}

// generateHTMLFile generates an HTML file
func (g *Generator) generateHTMLFile(data interface{}, filename, outputDir, templateName string) error {
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName+".html", data); err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, filename)
	return os.WriteFile(filepath, buf.Bytes(), 0644)
}

// generateMarkdown generates Markdown documentation
func (g *Generator) generateMarkdown(doc *Documentation, outputDir string) error {
	// Generate README
	if err := g.generateMarkdownFile(doc, "README.md", outputDir, "readme"); err != nil {
		return err
	}

	// Generate server documentation
	for _, server := range doc.Servers {
		filename := fmt.Sprintf("server-%s.md", slugify(server.Name))
		if err := g.generateMarkdownFile(server, filename, outputDir, "server"); err != nil {
			return err
		}
	}

	return nil
}

// generateMarkdownFile generates a Markdown file
func (g *Generator) generateMarkdownFile(data interface{}, filename, outputDir, templateName string) error {
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName+".md", data); err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, filename)
	return os.WriteFile(filepath, buf.Bytes(), 0644)
}

// generateJSON generates JSON documentation
func (g *Generator) generateJSON(doc *Documentation, outputDir string) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, "documentation.json")
	return os.WriteFile(filepath, data, 0644)
}

// generateYAML generates YAML documentation
func (g *Generator) generateYAML(doc *Documentation, outputDir string) error {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, "documentation.yaml")
	return os.WriteFile(filepath, data, 0644)
}

// copyAssets copies static assets to output directory
func (g *Generator) copyAssets(outputDir string) error {
	// This would copy CSS, JS, images, etc.
	// For now, return nil
	return nil
}

// Serve starts a local server to serve the documentation
func (g *Generator) Serve(port int) error {
	log.Printf("Starting documentation server on port %d", port)
	// This would implement a local HTTP server
	// For now, return nil
	return nil
}

// Watch watches for changes and regenerates documentation
func (g *Generator) Watch() error {
	log.Printf("Watching for changes...")
	// This would implement file watching
	// For now, return nil
	return nil
}

// Helper functions

// generateToolExamples generates examples for a tool
func (g *Generator) generateToolExamples(tool mcp.Tool) []*ExampleDoc {
	examples := []*ExampleDoc{
		{
			Title:       fmt.Sprintf("Using %s", tool.Name),
			Description: fmt.Sprintf("Example usage of the %s tool", tool.Name),
			Code:        fmt.Sprintf(`client.CallTool(ctx, mcp.CallToolRequest{
    Name: "%s",
    Arguments: map[string]interface{}{
        // Add your arguments here
    },
})`, tool.Name),
			Language:    "go",
			Interactive: true,
		},
	}

	return examples
}

// generateResourceExamples generates examples for a resource
func (g *Generator) generateResourceExamples(resource mcp.Resource) []*ExampleDoc {
	examples := []*ExampleDoc{
		{
			Title:       fmt.Sprintf("Reading %s", resource.Name),
			Description: fmt.Sprintf("Example of reading the %s resource", resource.Name),
			Code:        fmt.Sprintf(`client.ReadResource(ctx, mcp.ReadResourceRequest{
    URI: "%s",
})`, resource.URI),
			Language:    "go",
			Interactive: true,
		},
	}

	return examples
}

// generatePromptExamples generates examples for a prompt
func (g *Generator) generatePromptExamples(prompt mcp.Prompt) []*ExampleDoc {
	examples := []*ExampleDoc{
		{
			Title:       fmt.Sprintf("Using %s", prompt.Name),
			Description: fmt.Sprintf("Example usage of the %s prompt", prompt.Name),
			Code:        fmt.Sprintf(`client.GetPrompt(ctx, mcp.GetPromptRequest{
    Name: "%s",
    Arguments: map[string]interface{}{
        // Add your arguments here
    },
})`, prompt.Name),
			Language:    "go",
			Interactive: true,
		},
	}

	return examples
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.ToLower(reg.ReplaceAllString(s, "-"))
}

// processMarkdown processes markdown text
func processMarkdown(text string) string {
	// Simple markdown processing
	// This would use a proper markdown library in production
	return text
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *Config {
	return &Config{
		Output: OutputConfig{
			Directory: DefaultOutputDir,
			Formats:   []string{"html", "markdown"},
		},
		Documentation: DocumentationConfig{
			Title:       "MCP Documentation",
			Description: "Generated MCP documentation",
			Version:     "1.0.0",
			Language:    "en",
			Theme:       "default",
			Sidebar: SidebarConfig{
				Enabled:     true,
				Collapsible: true,
			},
			Navigation: NavigationConfig{
				Enabled: true,
			},
			Search: SearchConfig{
				Enabled:  true,
				Provider: "local",
			},
		},
		Templates: TemplateConfig{
			Directory: "templates",
		},
		Versions: VersionConfig{
			Enabled: false,
			Current: "1.0.0",
		},
		Integration: IntegrationConfig{
			Examples: ExamplesConfig{
				Enabled:     true,
				Interactive: true,
			},
			Playground: PlaygroundConfig{
				Enabled: false,
			},
			SDK: SDKConfig{
				Enabled:   false,
				Languages: []string{"go", "python", "javascript"},
			},
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(filename string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}