package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"strings"

	"github.com/tmc/mcp/exp/json2go"
)

func main() {
	var (
		input    = flag.String("input", "-", "Input JSON file (- for stdin)")
		output   = flag.String("output", "-", "Output Go file (- for stdout)")
		pkgName  = flag.String("package", "main", "Package name for generated code")
		typeName = flag.String("type", "Generated", "Name for the generated type")
		tags     = flag.String("tags", "json", "Struct tags to include (comma-separated)")
		prefix   = flag.String("prefix", "", "Prefix for all generated types")
		verbose  = flag.Bool("verbose", false, "Verbose output")
		noFormat = flag.Bool("no-format", false, "Skip formatting the output")
		jsonRPC  = flag.Bool("jsonrpc", false, "Handle JSON-RPC format")
		schema   = flag.Bool("schema", false, "Input is JSON Schema")
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

	// Create converter
	converter := json2go.NewConverter(json2go.Options{
		PackageName: *pkgName,
		TypeName:    *typeName,
		Tags:        strings.Split(*tags, ","),
		Prefix:      *prefix,
		Verbose:     *verbose,
	})

	// Convert based on input type
	var code string
	if *jsonRPC {
		code, err = converter.ConvertJSONRPC(inputData)
	} else if *schema {
		code, err = converter.ConvertJSONSchema(inputData)
	} else {
		code, err = converter.ConvertJSON(inputData)
	}

	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	// Format code unless disabled
	if !*noFormat {
		formatted, err := format.Source([]byte(code))
		if err != nil {
			if *verbose {
				log.Printf("Warning: Failed to format code: %v", err)
			}
		} else {
			code = string(formatted)
		}
	}

	// Write output
	if *output == "-" {
		fmt.Print(code)
	} else {
		if err := os.WriteFile(*output, []byte(code), 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
		if *verbose {
			fmt.Printf("Generated Go code written to %s\n", *output)
		}
	}
}

func showUsage() {
	fmt.Fprintf(os.Stderr, `json2go - Convert JSON to Go structs

Usage:
  json2go [options] < input.json > output.go
  json2go -input data.json -output types.go

Examples:
  # Convert JSON from API response
  curl https://api.example.com/data | json2go -type APIResponse

  # Convert JSON Schema to Go types
  json2go -schema -input schema.json -package models

  # Convert JSON-RPC request/response
  json2go -jsonrpc -input request.json -type RPCRequest

Options:
`)
	flag.PrintDefaults()
}