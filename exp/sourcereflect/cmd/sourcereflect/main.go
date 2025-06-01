package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"

	"github.com/tmc/mcp/exp/sourcereflect"
)

func handleFunctionAnalysis(filename string, node *ast.File, fset *token.FileSet, funcName string, pretty bool, analyzeHints bool) {
	// Find the function declaration
	var funcDecl *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == funcName {
				funcDecl = fn
				return false
			}
		}
		return true
	})

	if funcDecl == nil {
		fmt.Fprintf(os.Stderr, "Function %q not found in %s\n", funcName, filename)
		os.Exit(1)
	}

	// Create MCP tool description
	tool := &sourcereflect.MCPToolDescription{
		Name:        funcName,
		Description: fmt.Sprintf("Function %s", funcName),
		InputSchema: funcToInputSchema(funcDecl),
	}

	// Analyze hints if requested
	if analyzeHints {
		hints, err := sourcereflect.AnalyzeSourceHints(filename, funcName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing hints: %v\n", err)
			os.Exit(1)
		}
		tool.Hints = hints
	}

	// Output result
	var output []byte
	var err error
	if pretty {
		output, err = json.MarshalIndent(tool, "", "  ")
	} else {
		output, err = json.Marshal(tool)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func funcToInputSchema(funcDecl *ast.FuncDecl) *sourcereflect.Schema {
	schema := &sourcereflect.Schema{
		Type:       "object",
		Properties: make(map[string]*sourcereflect.Schema),
		Required:   []string{},
	}

	// Process function parameters
	if funcDecl.Type.Params != nil {
		for i, field := range funcDecl.Type.Params.List {
			// Get parameter name
			var paramName string
			if len(field.Names) > 0 {
				paramName = field.Names[0].Name
			} else {
				paramName = fmt.Sprintf("arg%d", i)
			}

			// Convert type to schema
			paramSchema, err := astToSchema(field.Type, "")
			if err != nil {
				// Fallback to generic object type
				paramSchema = &sourcereflect.Schema{Type: "object"}
			}

			schema.Properties[paramName] = paramSchema
			schema.Required = append(schema.Required, paramName)
		}
	}

	return schema
}

func main() {
	var (
		pretty      = flag.Bool("pretty", false, "Output pretty-printed JSON")
		typeName    = flag.String("type", "", "Type name to generate schema for")
		funcName    = flag.String("func", "", "Function name to generate MCP tool description for")
		showCaller  = flag.Bool("caller", false, "Include caller information in schema")
		analyzeHints = flag.Bool("analyze-hints", false, "Analyze source code to determine MCP tool hints")
	)
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: sourcereflect [flags] <file.go>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate flags
	if *typeName == "" && *funcName == "" {
		fmt.Fprintf(os.Stderr, "Either -type or -func must be specified\n")
		flag.Usage()
		os.Exit(1)
	}

	if *typeName != "" && *funcName != "" {
		fmt.Fprintf(os.Stderr, "Cannot specify both -type and -func\n")
		flag.Usage()
		os.Exit(1)
	}

	filename := flag.Arg(0)
	
	// Read and parse the Go file
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Handle function analysis if requested
	if *funcName != "" {
		handleFunctionAnalysis(filename, node, fset, *funcName, *pretty, *analyzeHints)
		return
	}

	// Find the specified type
	var targetType ast.Expr
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name == *typeName {
				targetType = x.Type
				return false
			}
		}
		return true
	})

	if targetType == nil {
		fmt.Fprintf(os.Stderr, "Type %q not found in %s\n", *typeName, filename)
		os.Exit(1)
	}

	// Convert AST to schema
	schema, err := astToSchema(targetType, *typeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to schema: %v\n", err)
		os.Exit(1)
	}

	// Add caller info if requested
	if *showCaller {
		if schema.Additional == nil {
			schema.Additional = make(map[string]interface{})
		}
		schema.Additional["$sourceLocation"] = map[string]interface{}{
			"file": filename,
			"type": *typeName,
		}
	}

	// Output JSON
	var output []byte
	if *pretty {
		output, err = json.MarshalIndent(schema, "", "  ")
	} else {
		output, err = json.Marshal(schema)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

// astToSchema converts an AST type expression to a JSON schema
func astToSchema(expr ast.Expr, typeName string) (*sourcereflect.Schema, error) {
	switch x := expr.(type) {
	case *ast.StructType:
		return structToSchema(x, typeName)
	case *ast.Ident:
		return identToSchema(x)
	case *ast.ArrayType:
		elemSchema, err := astToSchema(x.Elt, "")
		if err != nil {
			return nil, err
		}
		return &sourcereflect.Schema{
			Type:  "array",
			Items: elemSchema,
		}, nil
	case *ast.MapType:
		valueSchema, err := astToSchema(x.Value, "")
		if err != nil {
			return nil, err
		}
		return &sourcereflect.Schema{
			Type: "object",
			Additional: map[string]interface{}{
				"additionalProperties": valueSchema,
			},
		}, nil
	case *ast.StarExpr:
		// Handle pointer types
		return astToSchema(x.X, typeName)
	case *ast.SelectorExpr:
		// Handle qualified types like time.Time
		pkg := ""
		if ident, ok := x.X.(*ast.Ident); ok {
			pkg = ident.Name
		}
		typeName := x.Sel.Name

		// Special cases for common types
		if pkg == "time" && typeName == "Time" {
			return &sourcereflect.Schema{
				Type:   "string",
				Format: "date-time",
			}, nil
		}

		// For other types, just use object with title
		return &sourcereflect.Schema{
			Type:  "object",
			Title: typeName,
		}, nil
	case *ast.InterfaceType:
		// Handle interface{} types
		return &sourcereflect.Schema{Type: "object"}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", expr)
	}
}

func structToSchema(s *ast.StructType, typeName string) (*sourcereflect.Schema, error) {
	schema := &sourcereflect.Schema{
		Type:       "object",
		Title:      typeName,
		Properties: make(map[string]*sourcereflect.Schema),
		Required:   []string{},
	}

	for _, field := range s.Fields.List {
		if len(field.Names) == 0 {
			continue // Skip embedded fields for now
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue // Skip unexported fields
		}

		// Parse JSON tag
		jsonName := fieldName
		isRequired := true
		
		if field.Tag != nil {
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			jsonTag := tag.Get("json")
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					jsonName = parts[0]
				}
				for _, part := range parts[1:] {
					if part == "omitempty" {
						isRequired = false
					}
				}
			}
		}

		// Convert field type to schema
		fieldSchema, err := astToSchema(field.Type, "")
		if err != nil {
			return nil, fmt.Errorf("error processing field %s: %w", fieldName, err)
		}

		schema.Properties[jsonName] = fieldSchema
		if isRequired {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema, nil
}

func identToSchema(ident *ast.Ident) (*sourcereflect.Schema, error) {
	switch ident.Name {
	case "string":
		return &sourcereflect.Schema{Type: "string"}, nil
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return &sourcereflect.Schema{Type: "integer"}, nil
	case "float32", "float64":
		return &sourcereflect.Schema{Type: "number"}, nil
	case "bool":
		return &sourcereflect.Schema{Type: "boolean"}, nil
	case "interface{}":
		return &sourcereflect.Schema{Type: "object"}, nil
	default:
		// For custom types, just use the type name
		return &sourcereflect.Schema{Type: "object", Title: ident.Name}, nil
	}
}