// Package scaffold provides project scaffolding functionality
package scaffold

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Config represents scaffolding configuration
type Config struct {
	Language    string
	Template    string
	Output      string
	Author      string
	License     string
	CI          string
	Verbose     bool
	DryRun      bool
	ProjectName string
}

// Scaffolder provides project scaffolding functionality
type Scaffolder struct {
	config    *Config
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// ProjectData contains data for template rendering
type ProjectData struct {
	Name        string
	Language    string
	Template    string
	Author      string
	License     string
	CI          string
	Year        int
	Date        string
	ModulePath  string
	PackageName string
	Description string
	Version     string
	
	// Language-specific data
	GoModule      string
	TSPackage     string
	PythonPackage string
	RustCrate     string
	JavaPackage   string
	
	// CI/CD data
	GitHubActions bool
	GitLabCI      bool
	Jenkins       bool
	
	// Features
	HasTests        bool
	HasDocs         bool
	HasExamples     bool
	HasMiddleware   bool
	HasDocker       bool
	HasMakefile     bool
	HasGitIgnore    bool
	HasReadme       bool
	HasChangelog    bool
	HasContributing bool
	HasLicense      bool
}

//go:embed templates/*
var embeddedTemplates embed.FS

// New creates a new scaffolder
func New(config *Config) (*Scaffolder, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	scaffolder := &Scaffolder{
		config:    config,
		templates: make(map[string]*template.Template),
		funcs:     createTemplateFuncs(),
	}

	if err := scaffolder.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return scaffolder, nil
}

// Init initializes a new MCP project
func (s *Scaffolder) Init(ctx context.Context) error {
	if s.config.ProjectName == "" {
		return fmt.Errorf("project name required for init command")
	}

	if s.config.Verbose {
		fmt.Printf("Initializing %s project: %s\n", s.config.Language, s.config.ProjectName)
	}

	data := s.createProjectData()
	
	// Create project directory
	projectDir := filepath.Join(s.config.Output, s.config.ProjectName)
	if err := s.createDirectory(projectDir); err != nil {
		return err
	}

	// Generate project files
	templateKey := fmt.Sprintf("%s/project/%s", s.config.Language, s.config.Template)
	return s.generateFromTemplate(templateKey, data, projectDir)
}

// CreateServer creates a new MCP server project
func (s *Scaffolder) CreateServer(ctx context.Context) error {
	if s.config.ProjectName == "" {
		return fmt.Errorf("project name required for server command")
	}

	if s.config.Verbose {
		fmt.Printf("Creating %s server project: %s\n", s.config.Language, s.config.ProjectName)
	}

	data := s.createProjectData()
	data.Description = "MCP server implementation"
	
	// Create project directory
	projectDir := filepath.Join(s.config.Output, s.config.ProjectName)
	if err := s.createDirectory(projectDir); err != nil {
		return err
	}

	// Generate server files
	templateKey := fmt.Sprintf("%s/server/%s", s.config.Language, s.config.Template)
	return s.generateFromTemplate(templateKey, data, projectDir)
}

// CreateClient creates a new MCP client project
func (s *Scaffolder) CreateClient(ctx context.Context) error {
	if s.config.ProjectName == "" {
		return fmt.Errorf("project name required for client command")
	}

	if s.config.Verbose {
		fmt.Printf("Creating %s client project: %s\n", s.config.Language, s.config.ProjectName)
	}

	data := s.createProjectData()
	data.Description = "MCP client implementation"
	
	// Create project directory
	projectDir := filepath.Join(s.config.Output, s.config.ProjectName)
	if err := s.createDirectory(projectDir); err != nil {
		return err
	}

	// Generate client files
	templateKey := fmt.Sprintf("%s/client/%s", s.config.Language, s.config.Template)
	return s.generateFromTemplate(templateKey, data, projectDir)
}

// AddTool adds a new tool to an existing project
func (s *Scaffolder) AddTool(ctx context.Context) error {
	if s.config.ProjectName == "" {
		return fmt.Errorf("tool name required for tool command")
	}

	if s.config.Verbose {
		fmt.Printf("Adding tool: %s\n", s.config.ProjectName)
	}

	data := s.createProjectData()
	data.Name = s.config.ProjectName // Tool name in this case
	
	// Use current directory as project root
	projectDir := s.config.Output
	
	// Generate tool files
	templateKey := fmt.Sprintf("%s/tool", s.config.Language)
	return s.generateFromTemplate(templateKey, data, projectDir)
}

// CreatePlugin creates a new MCP plugin project
func (s *Scaffolder) CreatePlugin(ctx context.Context) error {
	if s.config.ProjectName == "" {
		return fmt.Errorf("plugin name required for plugin command")
	}

	if s.config.Verbose {
		fmt.Printf("Creating %s plugin project: %s\n", s.config.Language, s.config.ProjectName)
	}

	data := s.createProjectData()
	data.Description = "MCP plugin implementation"
	
	// Create project directory
	projectDir := filepath.Join(s.config.Output, s.config.ProjectName)
	if err := s.createDirectory(projectDir); err != nil {
		return err
	}

	// Generate plugin files
	templateKey := fmt.Sprintf("%s/plugin/%s", s.config.Language, s.config.Template)
	return s.generateFromTemplate(templateKey, data, projectDir)
}

// createProjectData creates project data for template rendering
func (s *Scaffolder) createProjectData() *ProjectData {
	now := time.Now()
	
	data := &ProjectData{
		Name:        s.config.ProjectName,
		Language:    s.config.Language,
		Template:    s.config.Template,
		Author:      s.config.Author,
		License:     s.config.License,
		CI:          s.config.CI,
		Year:        now.Year(),
		Date:        now.Format("2006-01-02"),
		Version:     "0.1.0",
		Description: fmt.Sprintf("MCP %s project", s.config.ProjectName),
		
		// Features based on template
		HasTests:        s.config.Template != "basic",
		HasDocs:         s.config.Template == "advanced" || s.config.Template == "enterprise",
		HasExamples:     s.config.Template == "advanced" || s.config.Template == "enterprise",
		HasMiddleware:   s.config.Template == "enterprise",
		HasDocker:       s.config.Template == "enterprise",
		HasMakefile:     s.config.Template != "basic",
		HasGitIgnore:    true,
		HasReadme:       true,
		HasChangelog:    s.config.Template == "advanced" || s.config.Template == "enterprise",
		HasContributing: s.config.Template == "enterprise",
		HasLicense:      s.config.License != "",
		
		// CI/CD
		GitHubActions: s.config.CI == "github",
		GitLabCI:      s.config.CI == "gitlab",
		Jenkins:       s.config.CI == "jenkins",
	}

	// Set language-specific data
	switch s.config.Language {
	case "go":
		data.GoModule = fmt.Sprintf("github.com/%s/%s", s.config.Author, s.config.ProjectName)
		data.ModulePath = data.GoModule
		data.PackageName = s.config.ProjectName
	case "typescript":
		data.TSPackage = s.config.ProjectName
		data.PackageName = s.config.ProjectName
	case "python":
		data.PythonPackage = strings.ReplaceAll(s.config.ProjectName, "-", "_")
		data.PackageName = data.PythonPackage
	case "rust":
		data.RustCrate = s.config.ProjectName
		data.PackageName = s.config.ProjectName
	case "java":
		data.JavaPackage = fmt.Sprintf("com.%s.%s", s.config.Author, strings.ReplaceAll(s.config.ProjectName, "-", ""))
		data.PackageName = data.JavaPackage
	}

	return data
}

// loadTemplates loads embedded templates
func (s *Scaffolder) loadTemplates() error {
	return fs.WalkDir(embeddedTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		content, err := fs.ReadFile(embeddedTemplates, path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		name := strings.TrimSuffix(path, ".tmpl")
		name = strings.TrimPrefix(name, "templates/")

		tmpl, err := template.New(name).Funcs(s.funcs).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		s.templates[name] = tmpl
		return nil
	})
}

// generateFromTemplate generates files from a template
func (s *Scaffolder) generateFromTemplate(templateKey string, data *ProjectData, outputDir string) error {
	// This is a simplified implementation - in practice, you'd have a directory
	// structure of templates and generate multiple files
	
	// For now, generate a few key files
	files := []string{
		"main",
		"readme",
		"gitignore",
		"dockerfile",
		"makefile",
		"go.mod",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
		"pom.xml",
	}

	for _, file := range files {
		templateName := fmt.Sprintf("%s/%s", templateKey, file)
		if tmpl, exists := s.templates[templateName]; exists {
			fileName := s.getFileName(file, data.Language)
			filePath := filepath.Join(outputDir, fileName)
			
			if err := s.generateFile(tmpl, data, filePath); err != nil {
				return fmt.Errorf("failed to generate %s: %w", fileName, err)
			}
		}
	}

	return nil
}

// generateFile generates a single file from template
func (s *Scaffolder) generateFile(tmpl *template.Template, data *ProjectData, filePath string) error {
	if s.config.DryRun {
		fmt.Printf("Would generate: %s\n", filePath)
		return nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if s.config.Verbose {
		fmt.Printf("Generated: %s\n", filePath)
	}

	return nil
}

// getFileName returns the appropriate filename for a template
func (s *Scaffolder) getFileName(templateName, language string) string {
	switch templateName {
	case "main":
		switch language {
		case "go":
			return "main.go"
		case "typescript":
			return "index.ts"
		case "python":
			return "main.py"
		case "rust":
			return "src/main.rs"
		case "java":
			return "src/main/java/Main.java"
		}
	case "readme":
		return "README.md"
	case "gitignore":
		return ".gitignore"
	case "dockerfile":
		return "Dockerfile"
	case "makefile":
		return "Makefile"
	case "go.mod":
		if language == "go" {
			return "go.mod"
		}
	case "package.json":
		if language == "typescript" {
			return "package.json"
		}
	case "pyproject.toml":
		if language == "python" {
			return "pyproject.toml"
		}
	case "Cargo.toml":
		if language == "rust" {
			return "Cargo.toml"
		}
	case "pom.xml":
		if language == "java" {
			return "pom.xml"
		}
	}
	return templateName
}

// createDirectory creates a directory if it doesn't exist
func (s *Scaffolder) createDirectory(path string) error {
	if s.config.DryRun {
		fmt.Printf("Would create directory: %s\n", path)
		return nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if s.config.Verbose {
		fmt.Printf("Created directory: %s\n", path)
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	supportedLanguages := []string{"go", "typescript", "python", "rust", "java"}
	
	found := false
	for _, lang := range supportedLanguages {
		if config.Language == lang {
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("unsupported language: %s (supported: %v)", config.Language, supportedLanguages)
	}
	
	supportedTemplates := []string{"basic", "advanced", "enterprise"}
	
	found = false
	for _, tmpl := range supportedTemplates {
		if config.Template == tmpl {
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("unsupported template: %s (supported: %v)", config.Template, supportedTemplates)
	}
	
	return nil
}

// createTemplateFuncs creates template functions
func createTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toLower":     strings.ToLower,
		"toUpper":     strings.ToUpper,
		"toTitle":     strings.Title,
		"replace":     strings.ReplaceAll,
		"trim":        strings.TrimSpace,
		"contains":    strings.Contains,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"split":       strings.Split,
		"join":        strings.Join,
		"default":     defaultValue,
		"formatDate":  formatDate,
		"formatYear":  formatYear,
		"kebabCase":   toKebabCase,
		"snakeCase":   toSnakeCase,
		"pascalCase":  toPascalCase,
		"camelCase":   toCamelCase,
	}
}

// Template function implementations

func defaultValue(def interface{}, val interface{}) interface{} {
	if val == nil || val == "" {
		return def
	}
	return val
}

func formatDate(date time.Time) string {
	return date.Format("2006-01-02")
}

func formatYear(date time.Time) string {
	return date.Format("2006")
}

func toKebabCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", "-"))
}

func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", "_"))
}

func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})
	
	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.Title(strings.ToLower(word)))
	}
	return result.String()
}

func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
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