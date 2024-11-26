package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp/internal/mcptest"
)

func main() {
	var (
		verbose = flag.Bool("v", false, "verbose output")
		fail    = flag.Bool("f", false, "fail fast (stop on first failure)")
		timeout = flag.Duration("timeout", 10*time.Second, "timeout for each test")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mcptest [flags] <test.txtar...>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	var failed bool
	for _, pattern := range flag.Args() {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatal(err)
		}
		for _, match := range matches {
			start := time.Now()
			if *verbose {
				fmt.Printf("=== RUN   %s\n", match)
			}
			var output strings.Builder
			err := runTest(ctx, match, &output, *timeout)
			duration := time.Since(start).Round(time.Millisecond)

			if err != nil {
				failed = true
				fmt.Printf("--- FAIL: %s (%s)\n%s\n", match, duration, output.String())
				if *fail {
					os.Exit(1)
				}
			} else if *verbose {
				fmt.Printf("--- PASS: %s (%s)\n%s", match, duration, output.String())
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}

func runTest(ctx context.Context, scriptPath string, output *strings.Builder, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return mcptest.RunTxTarFile(context.Background(), scriptPath, output)
}

func indent(s string) string {
	return "\t" + strings.ReplaceAll(s, "\n", "\n\t")
}
