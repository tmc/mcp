// Command mcpcorpus extracts selected tool arguments without executing tools.
//
// Usage:
//
//	mcpcorpus -tool search -out testdata/fuzz/FuzzSearch trace.mcp
//	mcpcorpus -tool search -format json -out arguments < trace.mcp
//
// The output directory must not exist. All input is validated before any output
// is created. Files have deterministic SHA-256 names. Formats are fuzz (one
// []byte argument in Go's native corpus format) and json (raw arguments).
// Missing or null arguments are errors; no input is silently replaced with {}.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tmc/mcp/mcpcorpus"
)

func main() { os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
func run(ctx context.Context, args []string, input io.Reader, out, errout io.Writer) int {
	fail := func(err error) int { fmt.Fprintln(errout, "mcpcorpus:", err); return 1 }
	f := flag.NewFlagSet("mcpcorpus", flag.ContinueOnError)
	f.SetOutput(errout)
	tool := f.String("tool", "", "tool name to select")
	dir := f.String("out", "", "new output directory")
	format := f.String("format", "fuzz", "fuzz or json")
	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if *tool == "" || *dir == "" || f.NArg() > 1 || (*format != "fuzz" && *format != "json") {
		return fail(fmt.Errorf("required: -tool name -out new-directory [-format fuzz|json] [trace-file]"))
	}
	if f.NArg() == 1 {
		file, err := os.Open(f.Arg(0))
		if err != nil {
			return fail(err)
		}
		defer file.Close()
		input = file
	}
	seeds, err := mcpcorpus.Extract(input, *tool)
	if err != nil {
		return fail(err)
	}
	if len(seeds) == 0 {
		return fail(fmt.Errorf("no calls for selected tool %q", *tool))
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	// Reserve a new directory atomically. Never merge into an existing corpus.
	if err := os.Mkdir(*dir, 0755); err != nil {
		return fail(err)
	}
	created := []string{}
	complete := false
	defer func() {
		if !complete {
			for _, path := range created {
				os.Remove(path)
			}
			os.Remove(*dir)
		}
	}()
	for _, seed := range seeds {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		name := seed.Name
		if *format == "json" {
			name += ".json"
		}
		path := filepath.Join(*dir, name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return fail(err)
		}
		created = append(created, path)
		if *format == "json" {
			err = mcpcorpus.WriteJSON(file, seed)
		} else {
			err = mcpcorpus.WriteFuzz(file, seed)
		}
		closeErr := file.Close()
		if err != nil {
			return fail(err)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
	}
	complete = true
	if _, err := fmt.Fprintf(out, "wrote %d seeds\n", len(seeds)); err != nil {
		return fail(err)
	}
	return 0
}
