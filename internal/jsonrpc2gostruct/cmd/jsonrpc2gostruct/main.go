package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/internal/jsonrpc2gostruct"
)

var (
	packageName = flag.String("package", "main", "Package name for the generated Go code")
	structName  = flag.String("struct", "JSONRPCRequest", "Name of the struct to generate")
	outputFile  = flag.String("out", "", "Output file (default: stdout)")
	batchMode   = flag.Bool("batch", false, "Process multiple schema files in batch mode")
	inputDir    = flag.String("dir", "", "Directory containing schema files for batch mode")
	filePattern = flag.String("pattern", "*.json", "File pattern for batch mode")
)

func main() {
	flag.Parse()

	if *batchMode {
		if *inputDir == "" {
			fmt.Fprintf(os.Stderr, "Error: -dir is required in batch mode\n")
			os.Exit(1)
		}
		processBatch()
		return
	}

	// Single file mode
	var input []byte
	var err error

	if flag.NArg() > 0 {
		// Read from specified file
		input, err = ioutil.ReadFile(flag.Arg(0))
	} else {
		// Read from stdin
		input, err = ioutil.ReadAll(os.Stdin)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	processInput(input)
}

func processInput(input []byte) {
	// Try as a JSON-RPC request first
	var jsonrpcRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
	}

	// Check if it's a JSON-RPC request
	isJSONRPC := false
	if err := json.Unmarshal(input, &jsonrpcRequest); err == nil {
		if jsonrpcRequest.JSONRPC != "" && jsonrpcRequest.Method != "" {
			isJSONRPC = true
		}
	}

	var output string
	var err error

	if isJSONRPC {
		// Process as a JSON-RPC request
		methodName := jsonrpcRequest.Method
		// Convert method name to a struct name (e.g. "tools/call" -> "ToolsCall")
		structPrefix := convertMethodToStructName(methodName)
		if *structName != "JSONRPCRequest" {
			structPrefix = *structName
		}

		output, err = jsonrpc2gostruct.ParseJSONRPCRequestToStruct(input, *packageName, structPrefix)
	} else {
		// Process as a regular JSON schema
		output, err = jsonrpc2gostruct.GenerateGoStruct(input, *packageName, *structName)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Go struct: %v\n", err)
		os.Exit(1)
	}

	writeOutput(output)
}

func processBatch() {
	pattern := filepath.Join(*inputDir, *filePattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No files found matching pattern: %s\n", pattern)
		os.Exit(1)
	}

	schemas := make(map[string][]byte)
	for _, file := range files {
		// Use the filename without extension as the struct name
		base := filepath.Base(file)
		structName := strings.TrimSuffix(base, filepath.Ext(base))
		structName = convertToStructName(structName)

		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
			continue
		}

		schemas[structName] = data
	}

	output, err := jsonrpc2gostruct.GenerateMultipleStructs(schemas, *packageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Go structs: %v\n", err)
		os.Exit(1)
	}

	writeOutput(output)
}

func writeOutput(output string) {
	if *outputFile == "" {
		// Write to stdout
		fmt.Print(output)
	} else {
		// Write to file
		err := ioutil.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", *outputFile)
	}
}

func convertMethodToStructName(method string) string {
	parts := strings.Split(method, "/")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

func convertToStructName(name string) string {
	// Handle dashes, underscores, etc.
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	for i, part := range parts {
		parts[i] = strings.Title(part)
	}

	return strings.Join(parts, "")
}
