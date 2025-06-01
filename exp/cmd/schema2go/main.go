package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/schema2go"
)

func main() {
	var (
		input       = flag.String("input", "-", "Input schema file (- for stdin)")
		output      = flag.String("output", "-", "Output Go file (- for stdout)")
		schemaType  = flag.String("type", "auto", "Schema type: auto, json, openapi, proto, mcp")
		packageName = flag.String("package", "main", "Go package name")
		prefix      = flag.String("prefix", "", "Prefix for generated types")
		tags        = flag.String("tags", "json", "Struct tags (comma-separated)")
		imports     = flag.String("imports", "", "Additional imports (comma-separated)")
		noValidate  = flag.Bool("no-validate", false, "Skip schema validation")
		noComments  = flag.Bool("no-comments", false, "Skip generating comments")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Read input
	var inputData []byte
	var err error
	
	if *input == "-" {
		inputData, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read from stdin: %v", err)
		}
	} else {
		inputData, err = os.ReadFile(*input)
		if err != nil {
			log.Fatalf("Failed to read input file: %v", err)
		}
	}

	// Auto-detect schema type if needed
	if *schemaType == "auto" {
		*schemaType = detectSchemaType(inputData, *input)
		if *verbose {
			fmt.Fprintf(os.Stderr, "Detected schema type: %s\n", *schemaType)
		}
	}

	// Create generator
	generator := schema2go.NewGenerator(schema2go.Options{
		PackageName: *packageName,
		Prefix:      *prefix,
		Tags:        strings.Split(*tags, ","),
		Imports:     parseImports(*imports),
		NoValidate:  *noValidate,
		NoComments:  *noComments,
		Verbose:     *verbose,
	})

	// Generate Go code
	var code string
	switch *schemaType {
	case "json":
		code, err = generator.FromJSONSchema(inputData)
	case "openapi":
		code, err = generator.FromOpenAPI(inputData)
	case "proto":
		code, err = generator.FromProtobuf(inputData)
	case "mcp":
		code, err = generator.FromMCPSchema(inputData)
	default:
		log.Fatalf("Unknown schema type: %s", *schemaType)
	}

	if err != nil {
		log.Fatalf("Code generation failed: %v", err)
	}

	// Format the generated code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to format code: %v\n", err)
			fmt.Fprintf(os.Stderr, "Generated code:\n%s\n", code)
		}
		formatted = []byte(code)
	}

	// Write output
	if *output == "-" {
		fmt.Print(string(formatted))
	} else {
		if err := os.WriteFile(*output, formatted, 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Generated Go code written to %s\n", *output)
		}
	}
}

func detectSchemaType(data []byte, filename string) string {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		// Could be JSON Schema or OpenAPI
		if isOpenAPI(data) {
			return "openapi"
		}
		return "json"
	case ".yaml", ".yml":
		// Likely OpenAPI
		return "openapi"
	case ".proto":
		return "proto"
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		// Check for OpenAPI indicators
		if _, ok := jsonData["openapi"]; ok {
			return "openapi"
		}
		if _, ok := jsonData["swagger"]; ok {
			return "openapi"
		}
		
		// Check for JSON Schema indicators
		if _, ok := jsonData["$schema"]; ok {
			return "json"
		}
		if _, ok := jsonData["type"]; ok {
			return "json"
		}
		
		// Check for MCP schema indicators
		if _, ok := jsonData["tools"]; ok {
			return "mcp"
		}
		if _, ok := jsonData["resources"]; ok {
			return "mcp"
		}
	}

	// Default to JSON Schema
	return "json"
}

func isOpenAPI(data []byte) bool {
	// Simple check for OpenAPI markers
	dataStr := string(data)
	return strings.Contains(dataStr, `"openapi"`) || 
	       strings.Contains(dataStr, `"swagger"`) ||
	       strings.Contains(dataStr, `"paths"`)
}

func parseImports(imports string) []string {
	if imports == "" {
		return nil
	}
	
	// Split and clean imports
	parts := strings.Split(imports, ",")
	result := make([]string, 0, len(parts))
	
	for _, imp := range parts {
		imp = strings.TrimSpace(imp)
		if imp != "" {
			// Add quotes if not present
			if !strings.HasPrefix(imp, `"`) {
				imp = `"` + imp + `"`
			}
			result = append(result, imp)
		}
	}
	
	return result
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `schema2go - Generate Go code from schemas

Usage:
  schema2go [options] < schema.json > types.go
  schema2go -input api.yaml -output models.go

Examples:
  schema2go -type json < schema.json           # JSON Schema to Go
  schema2go -type openapi api.yaml             # OpenAPI to Go
  schema2go -type proto service.proto          # Protocol Buffers to Go
  schema2go -type mcp tools.json               # MCP schema to Go

Schema Types:
  auto    - Auto-detect schema type (default)
  json    - JSON Schema
  openapi - OpenAPI/Swagger specification
  proto   - Protocol Buffers
  mcp     - Model Context Protocol schema

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
The tool generates idiomatic Go code with:
  - Proper type definitions
  - JSON/YAML/XML struct tags
  - Validation methods (optional)
  - Documentation from schema descriptions
  - Custom prefixes for type names
  - Additional imports as needed
`)
	}
}