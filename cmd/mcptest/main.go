package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/tmc/mcp/internal/mcptest"
)

func main() {
    var (
        verbose = flag.Bool("v", false, "verbose output")
        fail    = flag.Bool("f", false, "fail fast (stop on first failure)")
    )
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: mcptest [flags] <test.txtar...>\n")
        fmt.Fprintf(os.Stderr, "       mcptest [flags] <dir>\n")
        flag.PrintDefaults()
    }
    flag.Parse()

    if flag.NArg() < 1 {
        flag.Usage()
        os.Exit(1)
    }

    var failed bool
    for _, pattern := range flag.Args() {
        matches, err := filepath.Glob(pattern)
        if err != nil {
            log.Fatal(err)
        }
        if len(matches) == 0 {
            // If not a glob, try as directory
            if info, err := os.Stat(pattern); err == nil && info.IsDir() {
                matches = []string{filepath.Join(pattern, "*.txtar")}
            }
        }
        for _, match := range matches {
            if *verbose {
                fmt.Printf("=== RUN   %s\n", match)
            }
            if err := runTest(match); err != nil {
                failed = true
                fmt.Printf("--- FAIL: %s\n%s\n", match, indent(err.Error()))
                if *fail {
                    os.Exit(1)
                }
            } else if *verbose {
                fmt.Printf("--- PASS: %s\n", match)
            }
        }
    }

    if failed {
        os.Exit(1)
    }
}

func runTest(scriptPath string) error {
    dir := filepath.Dir(scriptPath)
    return mcptest.RunTXTARFile(context.Background(), scriptPath, dir)
}

func indent(s string) string {
    return "\t" + strings.ReplaceAll(s, "\n", "\n\t")
}
