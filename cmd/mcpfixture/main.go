// Command mcpfixture emits tool regression fixtures and explicitly checks them.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpfixture"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mcpfixture:", err)
		os.Exit(1)
	}
}
func run(args []string, in io.Reader, out, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "check" {
		return check(args[1:], stderr)
	}
	flags := flag.NewFlagSet("mcpfixture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	record := flags.Int("record", 0, "one-based tools/call request record to extract")
	session := flags.String("session", "", "session= metadata scope (default: anonymous single session)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: mcpfixture -record N [-session name] [trace]\n       mcpfixture check [-timeout duration] [-transport stdio|http] fixture.json -- server [args...]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err == flag.ErrHelp {
		return nil
	} else if err != nil {
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("at most one trace file is allowed")
	}
	if flags.NArg() == 1 {
		file, err := os.Open(flags.Arg(0))
		if err != nil {
			return fmt.Errorf("open trace: %w", err)
		}
		defer file.Close()
		in = file
	}
	return mcpfixture.Convert(in, out, mcpfixture.Options{Record: *record, Session: *session})
}
func check(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("mcpfixture check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	transport := flags.String("transport", "stdio", "stdio or http")
	timeout := flags.Duration("timeout", 10*time.Second, "total execution timeout")
	if err := flags.Parse(args); err == flag.ErrHelp {
		return nil
	} else if err != nil {
		return err
	}
	rest := flags.Args()
	if len(rest) < 3 || rest[1] != "--" || (*transport != "stdio" && *transport != "http") || (*transport == "http" && len(rest) != 3) {
		return fmt.Errorf("check requires fixture.json -- server [args...]")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	file, err := os.Open(rest[0])
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}
	defer file.Close()
	fixture, err := mcpfixture.Read(file)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	command := exec.CommandContext(ctx, rest[2], rest[3:]...)
	command.Stderr = stderr
	opts, err := mcpfixture.ClientOptions(fixture)
	if err != nil {
		return err
	}
	client := sdk.NewClient(fixture.ClientInfo, opts)
	var connection sdk.Transport = &sdk.CommandTransport{Command: command, TerminateDuration: 100 * time.Millisecond}
	if *transport == "http" {
		connection = &sdk.StreamableClientTransport{Endpoint: rest[2]}
	}
	requested := fixture.RequestedProtocolVersion
	if requested == "" {
		requested = fixture.ProtocolVersion
	}
	session, err := client.Connect(ctx, connection, &sdk.ClientSessionOptions{ProtocolVersion: requested})
	if err != nil {
		return fmt.Errorf("connect server: %w", err)
	}
	defer session.Close()
	return mcpfixture.Check(ctx, session, fixture)
}
