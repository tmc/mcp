// Command mcpinspect summarizes JSON-RPC exchanges in an MCP trace.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tmc/mcp/mcptrace"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mcpinspect:", err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, stderr io.Writer) error {
	flags := flag.NewFlagSet("mcpinspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "write one JSON event per line (duration is nanoseconds)")
	session := flags.String("session", "", "default single-session scope; session= metadata overrides it")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: mcpinspect [-json] [-session name] [trace]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("at most one input file is allowed")
	}
	if flags.NArg() == 1 {
		f, err := os.Open(flags.Arg(0))
		if err != nil {
			return fmt.Errorf("open trace: %w", err)
		}
		defer f.Close()
		in = f
	}
	conversation := mcptrace.NewConversation(in, *session)
	encoder := json.NewEncoder(out)
	for {
		e, err := conversation.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("input truncated; conclusions are incomplete: %w", err)
		}
		if *jsonOutput {
			err = encoder.Encode(e)
		} else {
			_, err = fmt.Fprintln(out, e.String())
		}
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
}
