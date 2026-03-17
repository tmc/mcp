package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

const (
	toolName         = "mcpsh"
	toolVersion      = "0.1.0"
	defaultTimeout   = 15 * time.Second
	defaultProtoVers = mcp.LATEST_PROTOCOL_VERSION
	groupMeta        = "meta"
	groupTools       = "tools"
)

type bootstrapOptions struct {
	Cmd             string
	HTTPURL         string
	SSEURL          string
	Timeout         time.Duration
	ProtocolVersion string
	Raw             bool
	ServerStderr    bool
}

type backend interface {
	Initialize(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Notify(context.Context, string, any) error
	Close() error
}

type liveBackend struct {
	client *mcp.Client
}

func (b *liveBackend) Initialize(ctx context.Context, req mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return b.client.Initialize(ctx, req)
}

func (b *liveBackend) ListTools(ctx context.Context, req mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return b.client.ListTools(ctx, req)
}

func (b *liveBackend) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return b.client.CallTool(ctx, req)
}

func (b *liveBackend) Notify(ctx context.Context, method string, params any) error {
	return b.client.Notify(ctx, method, params)
}

func (b *liveBackend) Close() error {
	return b.client.Close()
}

type app struct {
	backend backend
	opts    bootstrapOptions
	server  *mcp.InitializeResult
	tools   []mcp.Tool
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	opts, err := parseBootstrapArgs(args)
	if err != nil {
		return err
	}

	root, app, err := buildRootCommand(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		if app != nil && app.backend != nil {
			_ = app.backend.Close()
		}
	}()

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	return root.Execute()
}

func buildRootCommand(ctx context.Context, opts bootstrapOptions) (*cobra.Command, *app, error) {
	root := &cobra.Command{
		Use:                toolName + " [flags] <tool>",
		Short:              "Dynamic shell for MCP tools",
		Long:               baseLongHelp(),
		Example:            rootExamples(),
		Version:            toolVersion,
		DisableSuggestions: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddGroup(&cobra.Group{ID: groupMeta, Title: "Support Commands"})
	root.AddCommand(newCompletionCommand())
	addPersistentFlags(root, &opts)

	if opts.transportCount() == 0 {
		return root, nil, nil
	}

	backend, err := newLiveBackend(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	app := &app{backend: backend, opts: opts}
	if err := app.load(ctx); err != nil {
		_ = backend.Close()
		return nil, nil, err
	}

	root.Short = shortHelp(app.server)
	root.Long = longHelp(app.server, app.tools)
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	root.AddCommand(newToolsCommand(app))
	addToolCommands(root, app)
	return root, app, nil
}

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts",
		Long:                  completionLongHelp(),
		GroupID:               groupMeta,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}

func newToolsCommand(app *app) *cobra.Command {
	return &cobra.Command{
		Use:     "tools",
		Short:   "List discovered tools",
		Long:    "List the tools exposed by the configured MCP server.",
		GroupID: groupMeta,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
			for _, tool := range app.tools {
				if _, err := fmt.Fprintf(w, "%s\t%s\n", cobraName(tool.Name), shortToolHelp(tool)); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}
}

func addPersistentFlags(cmd *cobra.Command, opts *bootstrapOptions) {
	if opts.Timeout == 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.ProtocolVersion == "" {
		opts.ProtocolVersion = defaultProtoVers
	}
	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.Cmd, "cmd", opts.Cmd, "shell command to start an MCP stdio server")
	flags.StringVar(&opts.HTTPURL, "http", opts.HTTPURL, "streamable HTTP MCP endpoint")
	flags.StringVar(&opts.SSEURL, "sse", opts.SSEURL, "SSE MCP endpoint")
	flags.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "request timeout")
	flags.StringVar(&opts.ProtocolVersion, "protocol-version", opts.ProtocolVersion, "MCP protocol version")
	flags.BoolVar(&opts.Raw, "raw", opts.Raw, "print raw JSON tool results")
	flags.BoolVar(&opts.ServerStderr, "server-stderr", opts.ServerStderr, "forward wrapped server stderr to stderr")
}

func (o bootstrapOptions) transportCount() int {
	n := 0
	if o.Cmd != "" {
		n++
	}
	if o.HTTPURL != "" {
		n++
	}
	if o.SSEURL != "" {
		n++
	}
	return n
}

func parseBootstrapArgs(args []string) (bootstrapOptions, error) {
	opts := bootstrapOptions{
		Timeout:         defaultTimeout,
		ProtocolVersion: defaultProtoVers,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := splitArg(arg)
		switch name {
		case "--cmd":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --cmd")
				}
				value = args[i]
			}
			opts.Cmd = value
		case "--http":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --http")
				}
				value = args[i]
			}
			opts.HTTPURL = value
		case "--sse":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --sse")
				}
				value = args[i]
			}
			opts.SSEURL = value
		case "--timeout":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --timeout")
				}
				value = args[i]
			}
			d, err := time.ParseDuration(value)
			if err != nil {
				return opts, fmt.Errorf("parse --timeout: %w", err)
			}
			opts.Timeout = d
		case "--protocol-version":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --protocol-version")
				}
				value = args[i]
			}
			opts.ProtocolVersion = value
		case "--raw":
			if !hasValue {
				opts.Raw = true
				continue
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("parse --raw: %w", err)
			}
			opts.Raw = v
		case "--server-stderr":
			if !hasValue {
				opts.ServerStderr = true
				continue
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("parse --server-stderr: %w", err)
			}
			opts.ServerStderr = v
		}
	}
	if opts.transportCount() > 1 {
		return opts, errors.New("choose exactly one of --cmd, --http, or --sse")
	}
	return opts, nil
}

func splitArg(arg string) (name, value string, hasValue bool) {
	if !strings.HasPrefix(arg, "--") {
		return arg, "", false
	}
	name = arg
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		name = arg[:idx]
		value = arg[idx+1:]
		hasValue = true
	}
	return name, value, hasValue
}

func newLiveBackend(ctx context.Context, opts bootstrapOptions) (backend, error) {
	transport, err := newTransport(opts)
	if err != nil {
		return nil, err
	}
	client, err := mcp.NewClient(transport)
	if err != nil {
		return nil, err
	}
	return &liveBackend{client: client}, nil
}

func newTransport(opts bootstrapOptions) (mcp.Transport, error) {
	switch {
	case opts.Cmd != "":
		return commandTransport(opts.Cmd, serverStderr(opts)), nil
	case opts.SSEURL != "":
		return mcp.NewSSEClientTransport(opts.SSEURL, nil)
	case opts.HTTPURL != "":
		return mcp.NewStreamableClientTransport(opts.HTTPURL, nil), nil
	default:
		return nil, errors.New("no server transport configured; pass --cmd, --http, or --sse")
	}
}

func serverStderr(opts bootstrapOptions) io.Writer {
	if opts.ServerStderr {
		return os.Stderr
	}
	return io.Discard
}

func (a *app) load(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.opts.Timeout)
	defer cancel()

	result, err := a.backend.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: a.opts.ProtocolVersion,
		ClientInfo: mcp.Implementation{
			Name:    toolName,
			Version: toolVersion,
		},
		Capabilities: mcp.ClientCapabilities{},
	})
	if err != nil {
		return fmt.Errorf("initialize server: %w", err)
	}
	a.server = result
	_ = a.backend.Notify(ctx, "notifications/initialized", map[string]any{})

	tools, err := listAllTools(ctx, a.backend)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}
	a.tools = tools
	return nil
}

func listAllTools(ctx context.Context, backend backend) ([]mcp.Tool, error) {
	cursor := ""
	var all []mcp.Tool
	for {
		result, err := backend.ListTools(ctx, mcp.ListToolsRequest{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Tools...)
		if result.NextCursor == "" || result.NextCursor == cursor {
			break
		}
		cursor = result.NextCursor
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all, nil
}

func shortHelp(init *mcp.InitializeResult) string {
	if init == nil || init.ServerInfo.Name == "" {
		return "Dynamic shell for MCP tools"
	}
	return fmt.Sprintf("Dynamic shell for %s", init.ServerInfo.Name)
}

func longHelp(init *mcp.InitializeResult, tools []mcp.Tool) string {
	var b strings.Builder
	b.WriteString(baseLongHelp())
	if init != nil && init.ServerInfo.Name != "" {
		fmt.Fprintf(&b, "\n\nServer:\n  %s %s", init.ServerInfo.Name, init.ServerInfo.Version)
	}
	if init != nil && init.Instructions != "" {
		fmt.Fprintf(&b, "\n\nInstructions:\n  %s", strings.ReplaceAll(init.Instructions, "\n", "\n  "))
	}
	if len(tools) > 0 {
		fmt.Fprintf(&b, "\n\nDiscovered %d tools. Run %q to list them or %q for generated shell completion.", len(tools), toolName+" tools", toolName+" completion")
	}
	return b.String()
}

func baseLongHelp() string {
	return "mcpsh connects to an MCP server, discovers its tools, and exposes them as shell-friendly subcommands."
}

func completionLongHelp() string {
	return `Generate shell completion scripts for mcpsh.

To load completions:

Bash:
  source <(mcpsh completion bash)

Zsh:
  source <(mcpsh completion zsh)

Fish:
  mcpsh completion fish | source

PowerShell:
  mcpsh completion powershell | Out-String | Invoke-Expression`
}

func rootExamples() string {
	return `  mcpsh --cmd 'server --stdio' echo --message hello
  mcpsh --http http://127.0.0.1:8080/mcp tools
  mcpsh --sse http://127.0.0.1:8080/sse completion zsh`
}

func renderResult(result *mcp.CallToolResult, raw bool) ([]byte, error) {
	if raw {
		return json.MarshalIndent(result, "", "  ")
	}
	text := textOnlyResult(result)
	if text != "" {
		return []byte(text), nil
	}
	return json.MarshalIndent(result, "", "  ")
}

func textOnlyResult(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	lines := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		m, ok := item.(map[string]any)
		if !ok {
			return ""
		}
		if kind, _ := m["type"].(string); kind != "text" {
			return ""
		}
		text, ok := m["text"].(string)
		if !ok {
			return ""
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

func marshalArguments(args map[string]any) (json.RawMessage, error) {
	if len(args) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func decodeJSONObject(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(map[string]any)
	}
	return out, nil
}

func writeResult(cmd *cobra.Command, result *mcp.CallToolResult, raw bool) error {
	data, err := renderResult(result, raw)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if bytes.HasSuffix(data, []byte("\n")) {
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}
