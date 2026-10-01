// Workflow demonstrates recording and fixing a tool response regression.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpfixture"
	"github.com/tmc/mcp/mcptrace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "workflow:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workflow server|record|expect")
	}
	switch args[0] {
	case "server":
		flags := flag.NewFlagSet("server", flag.ContinueOnError)
		fixed := flags.Bool("fixed", false, "multiply rather than accidentally add")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return newServer(*fixed).Run(context.Background(), &sdk.StdioTransport{})
	case "record":
		flags := flag.NewFlagSet("record", flag.ContinueOnError)
		fixed := flags.Bool("fixed", false, "record corrected behavior")
		protocol := flags.String("protocol", "2026-07-28", "MCP protocol version to record")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 1 {
			return fmt.Errorf("record requires a trace file")
		}
		return record(flags.Arg(0), *fixed, *protocol)
	case "expect":
		flags := flag.NewFlagSet("expect", flag.ContinueOnError)
		text := flags.String("text", "", "reviewed correct response text")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 2 || *text == "" {
			return fmt.Errorf("expect requires -text correct-text observed.txt regression.json")
		}
		return expectation(flags.Arg(0), flags.Arg(1), *text)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func newServer(fixed bool) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "square", Version: "1"}, nil)
	server.AddTool(&sdk.Tool{Name: "square", Description: "Return n squared", InputSchema: map[string]any{
		"type": "object", "properties": map[string]any{"n": map[string]any{"type": "integer"}}, "required": []string{"n"},
	}}, func(_ context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		var args struct {
			N int `json:"n"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, err
		}
		result := args.N + args.N // The deliberately faulty implementation.
		if fixed {
			result = args.N * args.N
		}
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprint(result)}}}, nil
	})
	return server
}

func record(path string, fixed bool, protocol string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server, err := newServer(fixed).Connect(ctx, &sdk.IOTransport{Reader: right, Writer: right}, nil)
	if err != nil {
		return err
	}
	defer server.Close()
	trace := mcptrace.NewWriter(file)
	client := sdk.NewClient(&sdk.Implementation{Name: "workflow", Version: "1"}, &sdk.ClientOptions{Capabilities: &sdk.ClientCapabilities{}})
	session, err := client.Connect(ctx, &sdk.IOTransport{
		Reader: mcptrace.ReadCloser(left, trace, "recv"),
		Writer: mcptrace.WriteCloser(left, trace, "send"),
	}, &sdk.ClientSessionOptions{ProtocolVersion: protocol})
	if err != nil {
		return err
	}
	defer session.Close()
	if _, err := session.ListTools(ctx, &sdk.ListToolsParams{}); err != nil {
		return err
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "square", Arguments: map[string]any{"n": 3}})
	if err != nil {
		return err
	}
	if len(result.Content) != 1 {
		return fmt.Errorf("expected one text result")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		return fmt.Errorf("expected text result")
	}
	fmt.Println(text.Text)
	return session.Close()
}

func expectation(observed, path, text string) error {
	data, err := os.ReadFile(observed)
	if err != nil {
		return err
	}
	_, fixtureJSON, ok := strings.Cut(string(data), "-- fixture.json --\n")
	if !ok {
		return fmt.Errorf("observed file has no fixture.json archive member")
	}
	fixture, err := mcpfixture.Read(strings.NewReader(fixtureJSON))
	if err != nil {
		return err
	}
	// This is an explicit developer assertion, not the recorded observation.
	var expected map[string]json.RawMessage
	if err := json.Unmarshal(fixture.Expected, &expected); err != nil {
		return err
	}
	expected["content"], err = json.Marshal([]sdk.Content{&sdk.TextContent{Text: text}})
	if err != nil {
		return err
	}
	fixture.Expected, err = json.Marshal(expected)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(fixture)
}
