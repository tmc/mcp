package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func main() {
	var (
		format = flag.String("format", "text", "Output format: text or markdown")
		output = flag.String("output", "", "Output file (default: stdout)")
	)
	flag.Parse()

	schema := mcpscripttest.GetSchema()

	var out *os.File
	if *output == "" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()
	}

	switch *format {
	case "text":
		mcpscripttest.DumpSchema(out, schema)
	case "markdown", "md":
		markdown := mcpscripttest.GenerateMarkdown(schema)
		fmt.Fprint(out, markdown)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (use 'text' or 'markdown')\n", *format)
		os.Exit(1)
	}
}
