// Command mcpapi captures and compares declared MCP tool catalogs.
//
// Usage:
//
//	mcpapi snapshot -scope project -principal local -client default -- server args
//	mcpapi diff old.json new.json
//
// Snapshot performs initialization and tools/list only. Visibility labels must
// describe the actual caller configuration, never credentials. Diff prints JSON
// findings and exits 0 for supported compatible differences, 1 for incompatible
// differences, 2 for unknown compatibility, or 3 for usage and operational errors.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpcatalog"
)

func main() { os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr)) }
func run(ctx context.Context, args []string, out, errout io.Writer) int {
	fail := func(err error) int { fmt.Fprintln(errout, "mcpapi:", err); return 3 }
	if len(args) == 0 {
		return fail(fmt.Errorf("expected snapshot or diff"))
	}
	switch args[0] {
	case "snapshot":
		f := flag.NewFlagSet("snapshot", flag.ContinueOnError)
		f.SetOutput(errout)
		scope := f.String("scope", "", "visibility scope label")
		principal := f.String("principal", "", "principal label (not credentials)")
		clientLabel := f.String("client", "", "client configuration label")
		timeout := f.Duration("timeout", 30*time.Second, "capture timeout")
		if err := f.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return 3
		}
		command := f.Args()
		if len(command) == 0 || *scope == "" || *principal == "" || *clientLabel == "" || *timeout <= 0 {
			return fail(fmt.Errorf("snapshot requires visibility labels and a server command"))
		}
		ctx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stderr = errout
		client := mcp.NewClient(&mcp.Implementation{Name: "mcpapi", Version: "0.1.0"}, nil)
		session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		if err != nil {
			return fail(err)
		}
		s, err := mcpcatalog.Capture(ctx, session, mcpcatalog.Context{Scope: *scope, Principal: *principal, Client: *clientLabel})
		closeErr := session.Close()
		if err != nil {
			return fail(err)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
		if err := mcpcatalog.Write(out, s); err != nil {
			return fail(err)
		}
		return 0
	case "diff":
		if len(args) != 3 {
			return fail(fmt.Errorf("diff requires two snapshot files"))
		}
		load := func(path string) (*mcpcatalog.Snapshot, error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			return mcpcatalog.Read(f)
		}
		old, err := load(args[1])
		if err != nil {
			return fail(err)
		}
		new, err := load(args[2])
		if err != nil {
			return fail(err)
		}
		changes := mcpcatalog.Compare(old, new)
		if changes == nil {
			changes = []mcpcatalog.Change{}
		}
		if err := json.NewEncoder(out).Encode(changes); err != nil {
			return fail(err)
		}
		code := 0
		for _, c := range changes {
			if c.Kind == mcpcatalog.Unknown {
				code = 2
			} else if c.Kind == mcpcatalog.Incompatible && code == 0 {
				code = 1
			}
		}
		return code
	default:
		return fail(fmt.Errorf("unknown subcommand %q", args[0]))
	}
}
