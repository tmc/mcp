package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"golang.org/x/term"
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
	ConfigFile      string
	ServerName      string
	Timeout         time.Duration
	ProtocolVersion string
	Raw             bool
	ServerStderr    bool
	SpyRecord       string
	SpyUI           bool
	SpyOpen         bool
	SpyPretty       bool
	SpySpecFile     string

	// configServers is populated by resolveConfig for multi-server mode.
	configServers []configServerEntry
}

// configServerEntry holds resolved config for one server from .mcp.json.
type configServerEntry struct {
	name    string
	cfg     mcpServerConfig
	command string // pre-built shell command (empty for url-based)
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

// serverConn represents a single connected MCP server.
type serverConn struct {
	name    string
	backend backend
	info    *mcp.InitializeResult
	tools   []mcp.Tool
}

type app struct {
	servers []*serverConn
	opts    bootstrapOptions
}

// allTools returns all tools across all servers, sorted by display name.
func (a *app) allTools() []namespacedTool {
	multi := len(a.servers) > 1
	var all []namespacedTool
	for _, srv := range a.servers {
		for _, tool := range srv.tools {
			nt := namespacedTool{
				server: srv,
				tool:   tool,
			}
			if multi {
				nt.prefix = srv.name
			}
			all = append(all, nt)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].displayName() < all[j].displayName()
	})
	return all
}

// namespacedTool pairs a tool with its owning server.
type namespacedTool struct {
	server *serverConn
	tool   mcp.Tool
	prefix string
}

func (nt namespacedTool) displayName() string {
	if nt.prefix == "" {
		return cobraName(nt.tool.Name)
	}
	return cobraName(nt.prefix) + "/" + cobraName(nt.tool.Name)
}

func (a *app) reloadTools(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var errs []error
	for _, srv := range a.servers {
		ctx, cancel := context.WithTimeout(ctx, a.opts.Timeout)
		tools, err := listAllTools(ctx, srv.backend)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: list tools: %w", srv.name, err))
			continue
		}
		srv.tools = tools
	}
	return errors.Join(errs...)
}

func (a *app) close() {
	for _, srv := range a.servers {
		_ = srv.backend.Close()
	}
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

	root, a, err := buildRootCommand(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		if a != nil {
			a.close()
		}
	}()

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	return root.Execute()
}

func buildRootCommand(ctx context.Context, opts bootstrapOptions) (*cobra.Command, *app, error) {
	loaded, err := connectServers(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	return newRootCommand(opts, loaded), loaded, nil
}

// connectServers builds an app with all configured server connections.
func connectServers(ctx context.Context, opts bootstrapOptions) (*app, error) {
	// Multi-server from config.
	if opts.configServers != nil {
		a := &app{opts: opts}
		var errs []error
		for _, entry := range opts.configServers {
			srvOpts := opts
			srvOpts.Cmd = ""
			srvOpts.HTTPURL = ""
			srvOpts.SSEURL = ""
			if entry.cfg.URL != "" {
				srvOpts.HTTPURL = entry.cfg.URL
			} else {
				srvOpts.Cmd = entry.command
			}
			b, err := newLiveBackend(ctx, srvOpts)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", entry.name, err))
				continue
			}
			conn := &serverConn{name: entry.name, backend: b}
			if err := conn.load(ctx, opts); err != nil {
				_ = b.Close()
				errs = append(errs, fmt.Errorf("%s: %w", entry.name, err))
				continue
			}
			a.servers = append(a.servers, conn)
		}
		if len(a.servers) == 0 {
			return nil, fmt.Errorf("no servers connected: %w", errors.Join(errs...))
		}
		if len(errs) > 0 {
			// Partial success — log but continue.
			fmt.Fprintf(os.Stderr, "warning: some servers failed to connect: %v\n", errors.Join(errs...))
		}
		return a, nil
	}
	// Single server from --cmd/--http/--sse.
	if opts.transportCount() == 0 {
		return nil, nil
	}
	b, err := newLiveBackend(ctx, opts)
	if err != nil {
		return nil, err
	}
	conn := &serverConn{name: "default", backend: b}
	if err := conn.load(ctx, opts); err != nil {
		_ = b.Close()
		return nil, err
	}
	return &app{servers: []*serverConn{conn}, opts: opts}, nil
}

func (sc *serverConn) load(ctx context.Context, opts bootstrapOptions) error {
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	result, err := sc.backend.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: opts.ProtocolVersion,
		ClientInfo: mcp.Implementation{
			Name:    toolName,
			Version: toolVersion,
		},
		Capabilities: mcp.ClientCapabilities{},
	})
	if err != nil {
		return fmt.Errorf("initialize server: %w", err)
	}
	sc.info = result
	_ = sc.backend.Notify(ctx, "notifications/initialized", map[string]any{})

	tools, err := listAllTools(ctx, sc.backend)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}
	sc.tools = tools
	return nil
}

func newRootCommand(opts bootstrapOptions, app *app) *cobra.Command {
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
			if len(args) == 0 && app != nil && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
				return runInteractiveShell(cmd, opts, app)
			}
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddGroup(&cobra.Group{ID: groupMeta, Title: "Support Commands"})
	root.AddCommand(newCompletionCommand())
	addPersistentFlags(root, &opts)

	if app == nil || len(app.servers) == 0 {
		return root
	}

	allTools := app.allTools()
	root.Short = shortHelpMulti(app)
	root.Long = longHelpMulti(app, allTools)
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	root.AddCommand(newToolsCommand(app))
	root.AddCommand(newServersCommand(app))
	root.AddCommand(newShellCommand(opts, app))
	addToolCommandsMulti(root, app, allTools)
	return root
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
		Long:    "List the tools exposed by the configured MCP server(s).",
		GroupID: groupMeta,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
			for _, nt := range app.allTools() {
				if _, err := fmt.Fprintf(w, "%s\t%s\n", nt.displayName(), shortToolHelp(nt.tool)); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}
}

func newServersCommand(app *app) *cobra.Command {
	return &cobra.Command{
		Use:     "servers",
		Short:   "List connected servers",
		Long:    "List the MCP servers that are currently connected.",
		GroupID: groupMeta,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
			for _, srv := range app.servers {
				name := srv.name
				info := ""
				if srv.info != nil && srv.info.ServerInfo.Name != "" {
					info = srv.info.ServerInfo.Name
					if srv.info.ServerInfo.Version != "" {
						info += " " + srv.info.ServerInfo.Version
					}
				}
				tools := len(srv.tools)
				if _, err := fmt.Fprintf(w, "%s\t%s\t%d tools\n", name, info, tools); err != nil {
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
	flags.StringVar(&opts.ConfigFile, "config", opts.ConfigFile, "path to .mcp.json config file (auto-discovered if omitted)")
	flags.StringVar(&opts.ServerName, "server", opts.ServerName, "server name from .mcp.json config")
	flags.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "request timeout")
	flags.StringVar(&opts.ProtocolVersion, "protocol-version", opts.ProtocolVersion, "MCP protocol version")
	flags.BoolVar(&opts.Raw, "raw", opts.Raw, "print raw JSON tool results")
	flags.BoolVar(&opts.ServerStderr, "server-stderr", opts.ServerStderr, "forward wrapped server stderr to stderr")
	flags.StringVar(&opts.SpyRecord, "spy-record", opts.SpyRecord, "record wrapped stdio server traffic with mcpspy")
	flags.BoolVar(&opts.SpyUI, "spy-ui", opts.SpyUI, "serve wrapped stdio server traffic in the mcpspy web UI")
	flags.BoolVar(&opts.SpyOpen, "spy-open", opts.SpyOpen, "open the mcpspy UI in a browser")
	flags.BoolVar(&opts.SpyPretty, "spy-pretty", opts.SpyPretty, "pretty-print mcpspy JSON output")
	flags.StringVar(&opts.SpySpecFile, "spy-spec-file", opts.SpySpecFile, "write observed mcpspy spec output to this .mcpspec path")
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
		case "--spy-record":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --spy-record")
				}
				value = args[i]
			}
			opts.SpyRecord = value
		case "--spy-ui":
			if !hasValue {
				opts.SpyUI = true
				continue
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("parse --spy-ui: %w", err)
			}
			opts.SpyUI = v
		case "--spy-open":
			if !hasValue {
				opts.SpyOpen = true
				continue
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("parse --spy-open: %w", err)
			}
			opts.SpyOpen = v
		case "--spy-pretty":
			if !hasValue {
				opts.SpyPretty = true
				continue
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("parse --spy-pretty: %w", err)
			}
			opts.SpyPretty = v
		case "--spy-spec-file":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --spy-spec-file")
				}
				value = args[i]
			}
			opts.SpySpecFile = value
		case "--config":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --config")
				}
				value = args[i]
			}
			opts.ConfigFile = value
		case "--server":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --server")
				}
				value = args[i]
			}
			opts.ServerName = value
		}
	}
	if opts.SpyOpen && !opts.SpyUI {
		return opts, errors.New("--spy-open requires --spy-ui")
	}
	if opts.Cmd != "" && opts.spyEnabled() && opts.SpyPretty {
		return opts, errors.New("--spy-pretty is not supported when wrapping a live stdio server")
	}
	// Resolve --config/--server before transport validation.
	if err := resolveConfig(&opts); err != nil {
		return opts, err
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
	clientOpts := []mcp.ClientOption{}
	if opts.Cmd != "" {
		clientOpts = append(clientOpts, mcp.WithFramer(mcp.LineFramer()))
	}
	client, err := mcp.NewClient(transport, clientOpts...)
	if err != nil {
		return nil, err
	}
	return &liveBackend{client: client}, nil
}

func newTransport(opts bootstrapOptions) (mcp.Transport, error) {
	switch {
	case opts.Cmd != "":
		return commandTransport(wrappedCommand(opts), serverStderr(opts)), nil
	case opts.SSEURL != "":
		return mcp.NewSSEClientTransport(opts.SSEURL, nil)
	case opts.HTTPURL != "":
		return mcp.NewStreamableClientTransport(opts.HTTPURL, nil), nil
	default:
		return nil, errors.New("no server transport configured; pass --cmd, --http, or --sse")
	}
}

func serverStderr(opts bootstrapOptions) io.Writer {
	if opts.ServerStderr || opts.SpyUI {
		return os.Stderr
	}
	return io.Discard
}

func (o bootstrapOptions) spyEnabled() bool {
	return o.SpyRecord != "" || o.SpyUI || o.SpyOpen || o.SpyPretty || o.SpySpecFile != ""
}

func wrappedCommand(opts bootstrapOptions) string {
	if !opts.spyEnabled() {
		return opts.Cmd
	}
	args := spyCommandParts()
	if opts.SpyRecord != "" {
		args = append(args, "-f", opts.SpyRecord)
	}
	if opts.SpyUI {
		args = append(args, "-l")
	}
	if opts.SpyOpen {
		args = append(args, "-open")
	}
	if opts.SpySpecFile != "" {
		args = append(args, "--spec-file", opts.SpySpecFile)
	}
	args = append(args, "--pass-through")
	if !opts.ServerStderr {
		args = append(args, "-no-stderr")
	}
	args = append(args, "--")
	if runtime.GOOS == "windows" {
		args = append(args, "cmd", "/C", opts.Cmd)
	} else {
		args = append(args, "sh", "-lc", opts.Cmd)
	}
	return joinShellCommand(args)
}

func joinShellCommand(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`!&|;<>()[]{}*?~#") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func spyCommandParts() []string {
	if _, err := os.Stat(filepath.Join("exp", "cmd", "mcpspy", "main.go")); err == nil {
		return []string{"go", "run", "./exp/cmd/mcpspy"}
	}
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "mcpspy")
		if info, err := os.Stat(sibling); err == nil && info.Mode().IsRegular() {
			return []string{sibling}
		}
	}
	if path, err := exec.LookPath("mcpspy"); err == nil {
		return []string{path}
	}
	return []string{"mcpspy"}
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

func shortHelpMulti(app *app) string {
	if len(app.servers) == 1 {
		srv := app.servers[0]
		if srv.info != nil && srv.info.ServerInfo.Name != "" {
			return fmt.Sprintf("Dynamic shell for %s", srv.info.ServerInfo.Name)
		}
	}
	if len(app.servers) > 1 {
		return fmt.Sprintf("Dynamic shell for %d MCP servers", len(app.servers))
	}
	return "Dynamic shell for MCP tools"
}

func longHelpMulti(app *app, tools []namespacedTool) string {
	var b strings.Builder
	b.WriteString(baseLongHelp())
	if len(app.servers) == 1 {
		srv := app.servers[0]
		if srv.info != nil && srv.info.ServerInfo.Name != "" {
			fmt.Fprintf(&b, "\n\nServer:\n  %s %s", srv.info.ServerInfo.Name, srv.info.ServerInfo.Version)
		}
		if srv.info != nil && srv.info.Instructions != "" {
			fmt.Fprintf(&b, "\n\nInstructions:\n  %s", strings.ReplaceAll(srv.info.Instructions, "\n", "\n  "))
		}
	} else {
		fmt.Fprintf(&b, "\n\nServers:")
		for _, srv := range app.servers {
			name := srv.name
			if srv.info != nil && srv.info.ServerInfo.Name != "" {
				name = srv.info.ServerInfo.Name
			}
			fmt.Fprintf(&b, "\n  %s (%d tools)", name, len(srv.tools))
		}
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
  mcpsh --cmd 'server --stdio'
  mcpsh --cmd 'server --stdio' shell
  mcpsh --cmd 'server --stdio' --spy-record session.mcp --spy-ui
  mcpsh --http http://127.0.0.1:8080/mcp tools
  mcpsh --sse http://127.0.0.1:8080/sse completion zsh
  mcpsh --config .mcp.json --server cdp tools
  mcpsh --config .mcp.json                       # auto-select if single server
  mcpsh --server cdp                              # auto-discover .mcp.json`
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
