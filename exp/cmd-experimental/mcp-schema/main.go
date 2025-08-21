package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	programName = "mcp-schema"
	version     = "0.1.0"
)

// Command represents a CLI command
type Command interface {
	Name() string
	Usage() string
	Execute(ctx context.Context, args []string) error
}

// GenerateCommand handles schema generation
type GenerateCommand struct {
	packagePath string
	outputDir   string
	format      string
	protocol    string
	verbose     bool
}

func (c *GenerateCommand) Name() string {
	return "generate"
}

func (c *GenerateCommand) Usage() string {
	return "Generate JSON schemas from Go types"
}

func (c *GenerateCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)

	fs.StringVar(&c.packagePath, "package", "", "Go package path to analyze")
	fs.StringVar(&c.outputDir, "output", "./schemas", "Output directory for schemas")
	fs.StringVar(&c.format, "format", "json", "Output format (json, yaml)")
	fs.StringVar(&c.protocol, "protocol", "stable", "Protocol version (stable, draft)")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if c.packagePath == "" {
		return fmt.Errorf("package path is required")
	}

	generator := NewSchemaGenerator(c)
	return generator.Generate(ctx)
}

// DiffCommand handles schema comparison
type DiffCommand struct {
	oldSchema  string
	newSchema  string
	outputFile string
	format     string
	verbose    bool
}

func (c *DiffCommand) Name() string {
	return "diff"
}

func (c *DiffCommand) Usage() string {
	return "Compare schemas for breaking changes"
}

func (c *DiffCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)

	fs.StringVar(&c.oldSchema, "old", "", "Old schema file")
	fs.StringVar(&c.newSchema, "new", "", "New schema file")
	fs.StringVar(&c.outputFile, "output", "", "Output file (default: stdout)")
	fs.StringVar(&c.format, "format", "json", "Output format (json, yaml, markdown)")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if c.oldSchema == "" || c.newSchema == "" {
		return fmt.Errorf("both old and new schema files are required")
	}

	differ := NewSchemaDiffer(c)
	return differ.Compare(ctx)
}

// DocsCommand handles documentation generation
type DocsCommand struct {
	inputDir  string
	outputDir string
	format    string
	template  string
	verbose   bool
}

func (c *DocsCommand) Name() string {
	return "docs"
}

func (c *DocsCommand) Usage() string {
	return "Generate documentation from schemas"
}

func (c *DocsCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)

	fs.StringVar(&c.inputDir, "input", "./schemas", "Input directory containing schemas")
	fs.StringVar(&c.outputDir, "output", "./docs", "Output directory for documentation")
	fs.StringVar(&c.format, "format", "markdown", "Output format (markdown, html)")
	fs.StringVar(&c.template, "template", "", "Custom template file")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	docGen := NewDocumentationGenerator(c)
	return docGen.Generate(ctx)
}

// JSONSchema represents a JSON schema
type JSONSchema struct {
	Schema      string                 `json:"$schema,omitempty"`
	ID          string                 `json:"$id,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	OneOf       []*JSONSchema          `json:"oneOf,omitempty"`
	AnyOf       []*JSONSchema          `json:"anyOf,omitempty"`
	AllOf       []*JSONSchema          `json:"allOf,omitempty"`
	Not         *JSONSchema            `json:"not,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	MinLength   *int                   `json:"minLength,omitempty"`
	MaxLength   *int                   `json:"maxLength,omitempty"`
	MinItems    *int                   `json:"minItems,omitempty"`
	MaxItems    *int                   `json:"maxItems,omitempty"`
	Examples    []interface{}          `json:"examples,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
}

// SchemaGenerator generates JSON schemas from Go types
type SchemaGenerator struct {
	config   *GenerateCommand
	schemas  map[string]*JSONSchema
	fileSet  *token.FileSet
	packages map[string]*ast.Package
}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator(config *GenerateCommand) *SchemaGenerator {
	return &SchemaGenerator{
		config:   config,
		schemas:  make(map[string]*JSONSchema),
		fileSet:  token.NewFileSet(),
		packages: make(map[string]*ast.Package),
	}
}

// Generate generates schemas from Go packages
func (g *SchemaGenerator) Generate(ctx context.Context) error {
	if g.config.verbose {
		log.Printf("Generating schemas from package: %s", g.config.packagePath)
	}

	// Parse the package
	pkgs, err := parser.ParseDir(g.fileSet, g.config.packagePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	g.packages = pkgs

	// Generate schemas for each package
	for pkgName, pkg := range pkgs {
		if g.config.verbose {
			log.Printf("Processing package: %s", pkgName)
		}

		if err := g.processPackage(pkg); err != nil {
			return fmt.Errorf("failed to process package %s: %w", pkgName, err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(g.config.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write schemas to files
	for name, schema := range g.schemas {
		filename := filepath.Join(g.config.outputDir, name+".json")
		if err := g.writeSchema(filename, schema); err != nil {
			return fmt.Errorf("failed to write schema %s: %w", name, err)
		}

		if g.config.verbose {
			log.Printf("Generated schema: %s", filename)
		}
	}

	return nil
}

// processPackage processes a Go package to extract type information
func (g *SchemaGenerator) processPackage(pkg *ast.Package) error {
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.TypeSpec:
				if node.Name.IsExported() {
					g.processType(node.Name.Name, node.Type)
				}
			}
			return true
		})
	}
	return nil
}

// processType processes a Go type and generates a JSON schema
func (g *SchemaGenerator) processType(name string, t ast.Expr) {
	schema := &JSONSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Title:  name,
		ID:     fmt.Sprintf("#/%s", name),
	}

	switch typ := t.(type) {
	case *ast.StructType:
		g.processStruct(schema, typ)
	case *ast.InterfaceType:
		g.processInterface(schema, typ)
	case *ast.ArrayType:
		g.processArray(schema, typ)
	case *ast.MapType:
		g.processMap(schema, typ)
	case *ast.Ident:
		g.processIdent(schema, typ)
	}

	g.schemas[name] = schema
}

// processStruct processes a struct type
func (g *SchemaGenerator) processStruct(schema *JSONSchema, structType *ast.StructType) {
	schema.Type = "object"
	schema.Properties = make(map[string]*JSONSchema)

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.IsExported() {
				fieldSchema := &JSONSchema{}
				g.processFieldType(fieldSchema, field.Type)

				// Extract JSON tag information
				jsonName := name.Name
				required := true

				if field.Tag != nil {
					tag := strings.Trim(field.Tag.Value, "`")
					if jsonTag := parseJSONTag(tag); jsonTag != "" {
						if jsonTag == "-" {
							continue // Skip this field
						}
						parts := strings.Split(jsonTag, ",")
						if len(parts) > 0 && parts[0] != "" {
							jsonName = parts[0]
						}
						// Check for omitempty
						for _, part := range parts[1:] {
							if part == "omitempty" {
								required = false
								break
							}
						}
					}
				}

				// Extract field documentation
				if field.Doc != nil {
					fieldSchema.Description = extractDocumentation(field.Doc.Text())
				}

				schema.Properties[jsonName] = fieldSchema

				if required {
					schema.Required = append(schema.Required, jsonName)
				}
			}
		}
	}

	// Sort required fields for consistency
	sort.Strings(schema.Required)
}

// processInterface processes an interface type
func (g *SchemaGenerator) processInterface(schema *JSONSchema, interfaceType *ast.InterfaceType) {
	// For interfaces, we'll create a more flexible schema
	schema.Type = "object"
	schema.Description = "Interface type - flexible object structure"

	// Add basic properties that most interfaces might have
	schema.Properties = map[string]*JSONSchema{
		"type": {
			Type:        "string",
			Description: "Type identifier",
		},
	}
}

// processArray processes an array or slice type
func (g *SchemaGenerator) processArray(schema *JSONSchema, arrayType *ast.ArrayType) {
	schema.Type = "array"
	schema.Items = &JSONSchema{}
	g.processFieldType(schema.Items, arrayType.Elt)
}

// processMap processes a map type
func (g *SchemaGenerator) processMap(schema *JSONSchema, mapType *ast.MapType) {
	schema.Type = "object"

	// For maps, we'll use additionalProperties
	additionalProps := &JSONSchema{}
	g.processFieldType(additionalProps, mapType.Value)

	// Note: JSON Schema doesn't have a direct equivalent to Go's additionalProperties
	// This is a simplified representation
	schema.Properties = map[string]*JSONSchema{
		"additionalProperties": additionalProps,
	}
}

// processIdent processes an identifier type
func (g *SchemaGenerator) processIdent(schema *JSONSchema, ident *ast.Ident) {
	switch ident.Name {
	case "string":
		schema.Type = "string"
	case "int", "int8", "int16", "int32", "int64":
		schema.Type = "integer"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		schema.Type = "integer"
		schema.Minimum = toFloat64Ptr(0)
	case "float32", "float64":
		schema.Type = "number"
	case "bool":
		schema.Type = "boolean"
	case "interface{}":
		// For interface{}, we allow any type
		schema.OneOf = []*JSONSchema{
			{Type: "string"},
			{Type: "number"},
			{Type: "integer"},
			{Type: "boolean"},
			{Type: "object"},
			{Type: "array"},
			{Type: "null"},
		}
	default:
		// Reference to another type
		schema.Type = "object"
		schema.Description = fmt.Sprintf("Reference to %s", ident.Name)
	}
}

// processFieldType processes a field type
func (g *SchemaGenerator) processFieldType(schema *JSONSchema, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		g.processIdent(schema, t)
	case *ast.StarExpr:
		// Pointer type - process the underlying type
		g.processFieldType(schema, t.X)
	case *ast.ArrayType:
		g.processArray(schema, t)
	case *ast.MapType:
		g.processMap(schema, t)
	case *ast.SelectorExpr:
		// Handle qualified identifiers like time.Time
		if x, ok := t.X.(*ast.Ident); ok {
			typeName := x.Name + "." + t.Sel.Name
			g.processKnownType(schema, typeName)
		}
	case *ast.InterfaceType:
		g.processInterface(schema, t)
	}
}

// processKnownType handles known types like time.Time
func (g *SchemaGenerator) processKnownType(schema *JSONSchema, typeName string) {
	switch typeName {
	case "time.Time":
		schema.Type = "string"
		schema.Format = "date-time"
	case "json.RawMessage":
		schema.Type = "object"
		schema.Description = "Raw JSON message"
	default:
		schema.Type = "object"
		schema.Description = fmt.Sprintf("Reference to %s", typeName)
	}
}

// writeSchema writes a schema to a file
func (g *SchemaGenerator) writeSchema(filename string, schema *JSONSchema) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schema)
}

// SchemaDiffer compares schemas for breaking changes
type SchemaDiffer struct {
	config *DiffCommand
}

// NewSchemaDiffer creates a new schema differ
func NewSchemaDiffer(config *DiffCommand) *SchemaDiffer {
	return &SchemaDiffer{config: config}
}

// SchemaDiff represents the differences between two schemas
type SchemaDiff struct {
	Summary    DiffSummary    `json:"summary"`
	Changes    []SchemaChange `json:"changes"`
	Breaking   []SchemaChange `json:"breaking"`
	Deprecated []SchemaChange `json:"deprecated"`
	Added      []SchemaChange `json:"added"`
	Removed    []SchemaChange `json:"removed"`
	Modified   []SchemaChange `json:"modified"`
	Timestamp  time.Time      `json:"timestamp"`
	OldVersion string         `json:"oldVersion"`
	NewVersion string         `json:"newVersion"`
}

// DiffSummary provides a summary of changes
type DiffSummary struct {
	TotalChanges    int  `json:"totalChanges"`
	BreakingChanges int  `json:"breakingChanges"`
	AddedItems      int  `json:"addedItems"`
	RemovedItems    int  `json:"removedItems"`
	ModifiedItems   int  `json:"modifiedItems"`
	IsCompatible    bool `json:"isCompatible"`
}

// SchemaChange represents a single change
type SchemaChange struct {
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue,omitempty"`
	Breaking    bool        `json:"breaking"`
	Severity    string      `json:"severity"`
}

// Compare compares two schemas
func (d *SchemaDiffer) Compare(ctx context.Context) error {
	if d.config.verbose {
		log.Printf("Comparing schemas: %s -> %s", d.config.oldSchema, d.config.newSchema)
	}

	// Load schemas
	oldSchema, err := d.loadSchema(d.config.oldSchema)
	if err != nil {
		return fmt.Errorf("failed to load old schema: %w", err)
	}

	newSchema, err := d.loadSchema(d.config.newSchema)
	if err != nil {
		return fmt.Errorf("failed to load new schema: %w", err)
	}

	// Compare schemas
	diff := d.compareSchemas(oldSchema, newSchema)

	// Output results
	var output io.Writer = os.Stdout
	if d.config.outputFile != "" {
		file, err := os.Create(d.config.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	switch d.config.format {
	case "json":
		return d.outputJSON(output, diff)
	case "markdown":
		return d.outputMarkdown(output, diff)
	default:
		return fmt.Errorf("unsupported format: %s", d.config.format)
	}
}

// loadSchema loads a schema from a file
func (d *SchemaDiffer) loadSchema(filename string) (*JSONSchema, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var schema JSONSchema
	if err := json.NewDecoder(file).Decode(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// compareSchemas compares two schemas and returns the differences
func (d *SchemaDiffer) compareSchemas(oldSchema, newSchema *JSONSchema) *SchemaDiff {
	diff := &SchemaDiff{
		Timestamp:  time.Now(),
		OldVersion: oldSchema.ID,
		NewVersion: newSchema.ID,
	}

	// Compare properties
	d.compareProperties(diff, "", oldSchema.Properties, newSchema.Properties)

	// Compare required fields
	d.compareRequired(diff, "", oldSchema.Required, newSchema.Required)

	// Compare types
	if oldSchema.Type != newSchema.Type {
		change := SchemaChange{
			Type:        "type_change",
			Path:        "type",
			Description: fmt.Sprintf("Type changed from %s to %s", oldSchema.Type, newSchema.Type),
			OldValue:    oldSchema.Type,
			NewValue:    newSchema.Type,
			Breaking:    true,
			Severity:    "error",
		}
		diff.Changes = append(diff.Changes, change)
		diff.Breaking = append(diff.Breaking, change)
	}

	// Calculate summary
	diff.Summary = d.calculateSummary(diff)

	return diff
}

// compareProperties compares properties between schemas
func (d *SchemaDiffer) compareProperties(diff *SchemaDiff, basePath string, oldProps, newProps map[string]*JSONSchema) {
	// Find added properties
	for name, newProp := range newProps {
		if _, exists := oldProps[name]; !exists {
			path := d.joinPath(basePath, name)
			change := SchemaChange{
				Type:        "property_added",
				Path:        path,
				Description: fmt.Sprintf("Property '%s' was added", name),
				NewValue:    newProp.Type,
				Breaking:    false,
				Severity:    "info",
			}
			diff.Changes = append(diff.Changes, change)
			diff.Added = append(diff.Added, change)
		}
	}

	// Find removed properties
	for name, oldProp := range oldProps {
		if _, exists := newProps[name]; !exists {
			path := d.joinPath(basePath, name)
			change := SchemaChange{
				Type:        "property_removed",
				Path:        path,
				Description: fmt.Sprintf("Property '%s' was removed", name),
				OldValue:    oldProp.Type,
				Breaking:    true,
				Severity:    "error",
			}
			diff.Changes = append(diff.Changes, change)
			diff.Removed = append(diff.Removed, change)
			diff.Breaking = append(diff.Breaking, change)
		}
	}

	// Find modified properties
	for name, oldProp := range oldProps {
		if newProp, exists := newProps[name]; exists {
			path := d.joinPath(basePath, name)
			d.compareSchemaProperties(diff, path, oldProp, newProp)
		}
	}
}

// compareRequired compares required fields
func (d *SchemaDiffer) compareRequired(diff *SchemaDiff, basePath string, oldRequired, newRequired []string) {
	oldSet := make(map[string]bool)
	for _, field := range oldRequired {
		oldSet[field] = true
	}

	newSet := make(map[string]bool)
	for _, field := range newRequired {
		newSet[field] = true
	}

	// Find newly required fields
	for field := range newSet {
		if !oldSet[field] {
			path := d.joinPath(basePath, field)
			change := SchemaChange{
				Type:        "required_added",
				Path:        path,
				Description: fmt.Sprintf("Field '%s' is now required", field),
				Breaking:    true,
				Severity:    "error",
			}
			diff.Changes = append(diff.Changes, change)
			diff.Breaking = append(diff.Breaking, change)
		}
	}

	// Find fields no longer required
	for field := range oldSet {
		if !newSet[field] {
			path := d.joinPath(basePath, field)
			change := SchemaChange{
				Type:        "required_removed",
				Path:        path,
				Description: fmt.Sprintf("Field '%s' is no longer required", field),
				Breaking:    false,
				Severity:    "info",
			}
			diff.Changes = append(diff.Changes, change)
		}
	}
}

// compareSchemaProperties compares individual schema properties
func (d *SchemaDiffer) compareSchemaProperties(diff *SchemaDiff, path string, oldProp, newProp *JSONSchema) {
	// Compare types
	if oldProp.Type != newProp.Type {
		change := SchemaChange{
			Type:        "type_change",
			Path:        path + ".type",
			Description: fmt.Sprintf("Type changed from %s to %s", oldProp.Type, newProp.Type),
			OldValue:    oldProp.Type,
			NewValue:    newProp.Type,
			Breaking:    true,
			Severity:    "error",
		}
		diff.Changes = append(diff.Changes, change)
		diff.Breaking = append(diff.Breaking, change)
	}

	// Compare formats
	if oldProp.Format != newProp.Format {
		change := SchemaChange{
			Type:        "format_change",
			Path:        path + ".format",
			Description: fmt.Sprintf("Format changed from %s to %s", oldProp.Format, newProp.Format),
			OldValue:    oldProp.Format,
			NewValue:    newProp.Format,
			Breaking:    isBreakingFormatChange(oldProp.Format, newProp.Format),
			Severity:    "warning",
		}
		diff.Changes = append(diff.Changes, change)
		if change.Breaking {
			diff.Breaking = append(diff.Breaking, change)
		}
	}

	// Recursively compare nested properties
	if oldProp.Properties != nil || newProp.Properties != nil {
		d.compareProperties(diff, path, oldProp.Properties, newProp.Properties)
	}
}

// calculateSummary calculates the diff summary
func (d *SchemaDiffer) calculateSummary(diff *SchemaDiff) DiffSummary {
	summary := DiffSummary{
		TotalChanges:    len(diff.Changes),
		BreakingChanges: len(diff.Breaking),
		AddedItems:      len(diff.Added),
		RemovedItems:    len(diff.Removed),
		ModifiedItems:   len(diff.Modified),
		IsCompatible:    len(diff.Breaking) == 0,
	}

	return summary
}

// joinPath joins path components
func (d *SchemaDiffer) joinPath(basePath, component string) string {
	if basePath == "" {
		return component
	}
	return basePath + "." + component
}

// outputJSON outputs diff in JSON format
func (d *SchemaDiffer) outputJSON(w io.Writer, diff *SchemaDiff) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(diff)
}

// outputMarkdown outputs diff in Markdown format
func (d *SchemaDiffer) outputMarkdown(w io.Writer, diff *SchemaDiff) error {
	fmt.Fprintf(w, "# Schema Diff Report\n\n")
	fmt.Fprintf(w, "**Generated:** %s\n", diff.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "**Old Version:** %s\n", diff.OldVersion)
	fmt.Fprintf(w, "**New Version:** %s\n\n", diff.NewVersion)

	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "- **Total Changes:** %d\n", diff.Summary.TotalChanges)
	fmt.Fprintf(w, "- **Breaking Changes:** %d\n", diff.Summary.BreakingChanges)
	fmt.Fprintf(w, "- **Added Items:** %d\n", diff.Summary.AddedItems)
	fmt.Fprintf(w, "- **Removed Items:** %d\n", diff.Summary.RemovedItems)
	fmt.Fprintf(w, "- **Modified Items:** %d\n", diff.Summary.ModifiedItems)

	if diff.Summary.IsCompatible {
		fmt.Fprintf(w, "- **Compatibility:** ✅ Compatible\n\n")
	} else {
		fmt.Fprintf(w, "- **Compatibility:** ❌ Breaking changes detected\n\n")
	}

	if len(diff.Breaking) > 0 {
		fmt.Fprintf(w, "## Breaking Changes\n\n")
		for _, change := range diff.Breaking {
			fmt.Fprintf(w, "- **%s** `%s`: %s\n", change.Type, change.Path, change.Description)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(diff.Added) > 0 {
		fmt.Fprintf(w, "## Added Items\n\n")
		for _, change := range diff.Added {
			fmt.Fprintf(w, "- **%s** `%s`: %s\n", change.Type, change.Path, change.Description)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(diff.Removed) > 0 {
		fmt.Fprintf(w, "## Removed Items\n\n")
		for _, change := range diff.Removed {
			fmt.Fprintf(w, "- **%s** `%s`: %s\n", change.Type, change.Path, change.Description)
		}
		fmt.Fprintf(w, "\n")
	}

	return nil
}

// DocumentationGenerator generates documentation from schemas
type DocumentationGenerator struct {
	config *DocsCommand
}

// NewDocumentationGenerator creates a new documentation generator
func NewDocumentationGenerator(config *DocsCommand) *DocumentationGenerator {
	return &DocumentationGenerator{config: config}
}

// Generate generates documentation from schemas
func (g *DocumentationGenerator) Generate(ctx context.Context) error {
	if g.config.verbose {
		log.Printf("Generating documentation from: %s", g.config.inputDir)
	}

	// Find all schema files
	schemaFiles, err := filepath.Glob(filepath.Join(g.config.inputDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find schema files: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(g.config.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process each schema file
	for _, schemaFile := range schemaFiles {
		if err := g.processSchemaFile(schemaFile); err != nil {
			return fmt.Errorf("failed to process schema file %s: %w", schemaFile, err)
		}
	}

	return nil
}

// processSchemaFile processes a single schema file
func (g *DocumentationGenerator) processSchemaFile(filename string) error {
	// Load schema
	schema, err := g.loadSchema(filename)
	if err != nil {
		return err
	}

	// Generate documentation
	baseName := strings.TrimSuffix(filepath.Base(filename), ".json")
	outputFile := filepath.Join(g.config.outputDir, baseName+".md")

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return g.generateSchemaDoc(file, schema)
}

// loadSchema loads a schema from file
func (g *DocumentationGenerator) loadSchema(filename string) (*JSONSchema, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var schema JSONSchema
	if err := json.NewDecoder(file).Decode(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// generateSchemaDoc generates documentation for a schema
func (g *DocumentationGenerator) generateSchemaDoc(w io.Writer, schema *JSONSchema) error {
	fmt.Fprintf(w, "# %s\n\n", schema.Title)

	if schema.Description != "" {
		fmt.Fprintf(w, "%s\n\n", schema.Description)
	}

	fmt.Fprintf(w, "## Properties\n\n")

	if schema.Properties != nil {
		g.generatePropertiesDoc(w, schema.Properties, schema.Required, 0)
	}

	return nil
}

// generatePropertiesDoc generates documentation for properties
func (g *DocumentationGenerator) generatePropertiesDoc(w io.Writer, properties map[string]*JSONSchema, required []string, depth int) {
	requiredSet := make(map[string]bool)
	for _, field := range required {
		requiredSet[field] = true
	}

	// Sort properties for consistent output
	var names []string
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prop := properties[name]

		// Generate property header
		indent := strings.Repeat("  ", depth)
		requiredText := ""
		if requiredSet[name] {
			requiredText = " (required)"
		}

		fmt.Fprintf(w, "%s- **%s**%s: %s", indent, name, requiredText, prop.Type)

		if prop.Format != "" {
			fmt.Fprintf(w, " (%s)", prop.Format)
		}

		fmt.Fprintf(w, "\n")

		if prop.Description != "" {
			fmt.Fprintf(w, "%s  %s\n", indent, prop.Description)
		}

		// Handle nested properties
		if prop.Properties != nil {
			g.generatePropertiesDoc(w, prop.Properties, prop.Required, depth+1)
		}

		fmt.Fprintf(w, "\n")
	}
}

// Helper functions

func parseJSONTag(tag string) string {
	re := regexp.MustCompile(`json:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractDocumentation(docText string) string {
	lines := strings.Split(docText, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "//") {
			result = append(result, line)
		}
	}

	return strings.Join(result, " ")
}

func toFloat64Ptr(v float64) *float64 {
	return &v
}

func isBreakingFormatChange(oldFormat, newFormat string) bool {
	// Define breaking format changes
	breakingChanges := map[string][]string{
		"date-time": {"date", "time"},
		"email":     {""},
		"uri":       {""},
		"uuid":      {""},
	}

	if allowed, exists := breakingChanges[oldFormat]; exists {
		for _, allowedFormat := range allowed {
			if newFormat == allowedFormat {
				return false
			}
		}
		return true
	}

	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", programName)
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  generate    Generate JSON schemas from Go types\n")
		fmt.Fprintf(os.Stderr, "  diff        Compare schemas for breaking changes\n")
		fmt.Fprintf(os.Stderr, "  docs        Generate documentation from schemas\n")
		fmt.Fprintf(os.Stderr, "  version     Show version information\n")
		os.Exit(1)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "generate":
		cmd := &GenerateCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "diff":
		cmd := &DiffCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "docs":
		cmd := &DocsCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("%s version %s\n", programName, version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
