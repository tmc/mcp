package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func main() {
	var (
		seed   = flag.Int64("seed", 0, "Random seed (0 for current time)")
		output = flag.String("output", "", "Output file (default: stdout)")
		count  = flag.Int("count", 1, "Number of scripts to generate")
	)
	flag.Parse()

	// Use current time as seed if not specified
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	for i := 0; i < *count; i++ {
		generator := mcpscripttest.NewFuzzGenerator(*seed + int64(i))
		script := generator.Generate()

		if *output == "" {
			fmt.Println(script)
			if i < *count-1 {
				fmt.Println("---") // Separator between multiple scripts
			}
		} else {
			filename := *output
			if *count > 1 {
				filename = fmt.Sprintf("%s.%d", *output, i)
			}

			if err := os.WriteFile(filename, []byte(script), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file %s: %v\n", filename, err)
				os.Exit(1)
			}
			fmt.Printf("Generated script written to %s\n", filename)
		}
	}
}
