// mcp2go generates Go source code from MCP tool descriptions and JSON schemas
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/mcpspec"
	"github.com/tmc/mcp/exp/sourcegen"
)

func main() {
	var (
		outputDir   = flag.String("output", ".", "output directory for generated files")
		packageName = flag.String("package", "generated", "package name for generated code")
		inputType   = flag.String("type", "auto", "input type: mcp, jsonschema, or auto")
		fileName    = flag.String("name", "", "name for generated file (defaults to input name)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <input-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGenerate Go source code from MCP tool descriptions or JSON schemas\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s tool.json                    # auto-detect format\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -type mcp mcp-tool.json      # MCP tool description\n  %s -type jsonschema schema.json # JSON schema\n  %s -type mcpspec server.mcpspec # MCP server spec\n", os.Args[0], os.Args[0], os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Read input file
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create generator
	gen := sourcegen.NewGenerator(*packageName)

	// Determine input type
	actualType := *inputType
	if actualType == "auto" {
		actualType = detectInputType(data)
		if actualType == "" {
			fmt.Fprintf(os.Stderr, "Could not auto-detect input type. Please specify with -type flag.\n")
			os.Exit(1)
		}
		if *inputType == "auto" {
			fmt.Printf("Auto-detected input type: %s\n", actualType)
		}
	}

	// Generate code based on type
	var output string
	var outputFileName string

	switch actualType {
	case "mcp":
		var tool mcpspec.ToolDefinition
		if err := json.Unmarshal(data, &tool); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing MCP tool description: %v\n", err)
			os.Exit(1)
		}

		output, err = gen.GenerateFromTool(&tool)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
			os.Exit(1)
		}

		if *fileName == "" {
			outputFileName = strings.ToLower(tool.Name) + "_tool.go"
		}

	case "mcp-response":
		// Parse MCP tools response format
		var response struct {
			Result struct {
				Tools []mcpspec.ToolDefinition `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing MCP tools response: %v\n", err)
			os.Exit(1)
		}

		// Generate code for all tools
		var outputs []string
		for _, tool := range response.Result.Tools {
			toolOutput, err := gen.GenerateFromTool(&tool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating code for tool %s: %v\n", tool.Name, err)
				os.Exit(1)
			}
			// Extract just the type definitions (skip package and imports)
			lines := strings.Split(toolOutput, "\n")
			var typeLines []string
			inType := false
			for _, line := range lines {
				if strings.HasPrefix(line, "// ") || strings.HasPrefix(line, "type ") {
					inType = true
				}
				if inType && line != "" {
					typeLines = append(typeLines, line)
				}
				if line == "" && inType {
					typeLines = append(typeLines, line)
				}
			}
			outputs = append(outputs, strings.Join(typeLines, "\n"))
		}

		// Combine outputs with single package and imports
		var finalBuf bytes.Buffer
		fmt.Fprintf(&finalBuf, "package %s\n\n", gen.GetPackageName())

		// Add imports
		if len(gen.GetImports()) > 0 {
			fmt.Fprintf(&finalBuf, "import (\n")
			for imp := range gen.GetImports() {
				fmt.Fprintf(&finalBuf, "\t\"%s\"\n", imp)
			}
			fmt.Fprintf(&finalBuf, ")\n\n")
		}

		// Add all type definitions
		for _, output := range outputs {
			finalBuf.WriteString(output)
			finalBuf.WriteString("\n")
		}

		output = finalBuf.String()

		if *fileName == "" {
			outputFileName = "tools.go"
		}

	case "jsonschema":
		var schema mcpspec.JSONSchema
		if err := json.Unmarshal(data, &schema); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON schema: %v\n", err)
			os.Exit(1)
		}

		// Use filename as type name
		typeName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		typeName = toGoName(typeName)

		output, err = gen.GenerateFromJSONSchema(typeName, &schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
			os.Exit(1)
		}

		if *fileName == "" {
			outputFileName = strings.ToLower(typeName) + ".go"
		}

	case "mcpspec":
		spec, err := mcpspec.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing MCPSpec: %v\n", err)
			os.Exit(1)
		}

		output, err = gen.GenerateFromSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
			os.Exit(1)
		}

		if *fileName == "" {
			outputFileName = strings.ToLower(spec.Server.Name) + "_server.go"
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown input type: %s\n", actualType)
		os.Exit(1)
	}

	// Use specified filename if provided
	if *fileName != "" {
		outputFileName = *fileName
		if !strings.HasSuffix(outputFileName, ".go") {
			outputFileName += ".go"
		}
	}

	// Write output
	outputPath := filepath.Join(*outputDir, outputFileName)

	// Create output directory if needed
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write the file
	if err := ioutil.WriteFile(outputPath, []byte(output), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", outputPath)
}

// detectInputType attempts to determine the input type from the JSON structure
func detectInputType(data []byte) string {
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return ""
	}

	// Check for MCP tools response format (from ListToolsResult)
	if result, hasResult := generic["result"]; hasResult {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if tools, hasTools := resultMap["tools"]; hasTools {
				if toolsArray, ok := tools.([]interface{}); ok && len(toolsArray) > 0 {
					// Check if first tool has expected fields
					if tool, ok := toolsArray[0].(map[string]interface{}); ok {
						if _, hasName := tool["name"]; hasName {
							if _, hasInputSchema := tool["inputSchema"]; hasInputSchema {
								return "mcp-response"
							}
						}
					}
				}
			}
		}
	}

	// Check for single MCP tool description fields
	if _, hasName := generic["name"]; hasName {
		if _, hasInputSchema := generic["inputSchema"]; hasInputSchema {
			return "mcp"
		}
	}

	// Check for JSON schema fields
	if _, hasType := generic["type"]; hasType {
		return "jsonschema"
	}
	if _, hasProperties := generic["properties"]; hasProperties {
		return "jsonschema"
	}
	if _, hasSchema := generic["$schema"]; hasSchema {
		return "jsonschema"
	}

	// Check for MCPSpec
	if _, hasSpecVersion := generic["specVersion"]; hasSpecVersion {
		return "mcpspec"
	}

	return ""
}

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
