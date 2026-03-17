package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp/internal/mcpcli"
)

const (
	toolName    = "mcp"
	toolVersion = "0.1.0"
)

type bootstrapOptions struct {
	mcpcli.Config
	Output string
}

type app struct {
	cfg    mcpcli.Config
	output mcpcli.OutputMode

	mu   sync.Mutex
	sess *mcpcli.Session
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	opts, err := parseBootstrapArgs(args)
	if err != nil {
		return err
	}
	output, err := mcpcli.ParseOutputMode(opts.Output)
	if err != nil {
		return err
	}
	a := &app{cfg: opts.Config, output: output}
	root, err := buildRoot(ctx, a, opts)
	if err != nil {
		return err
	}
	defer a.close()
	root.SetArgs(args)
	return root.Execute()
}

func buildRoot(ctx context.Context, a *app, opts bootstrapOptions) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:                "mcp",
		Short:              "Unix-native CLI for Model Context Protocol servers",
		Long:               "mcp connects to MCP servers and exposes tools, resources, prompts, tasks, roots, logs, and a TUI-friendly monitor.",
		Version:            toolVersion,
		DisableSuggestions: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	addPersistentFlags(root, &opts)
	root.AddCommand(newCompletionCommand())
	root.AddCommand(newInspectCommand(a))
	root.AddCommand(newResourceCommand(a))
	root.AddCommand(newPromptCommand(a))
	root.AddCommand(newRootCommand(a))
	root.AddCommand(newLogCommand(a))
	root.AddCommand(newTaskCommand(a))
	root.AddCommand(newUICommand(a))

	toolCmd, err := newToolCommand(ctx, a, opts)
	if err != nil {
		return nil, err
	}
	root.AddCommand(toolCmd)
	return root, nil
}

func addPersistentFlags(cmd *cobra.Command, opts *bootstrapOptions) {
	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.Cmd, "cmd", opts.Cmd, "shell command to start an MCP stdio server")
	flags.StringVar(&opts.HTTPURL, "http", opts.HTTPURL, "streamable HTTP MCP endpoint")
	flags.StringVar(&opts.SSEURL, "sse", opts.SSEURL, "SSE MCP endpoint")
	flags.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "request timeout")
	flags.StringVar(&opts.ProtocolVersion, "protocol-version", opts.ProtocolVersion, "MCP protocol version")
	flags.BoolVar(&opts.ServerStderr, "server-stderr", opts.ServerStderr, "forward wrapped server stderr to stderr")
	flags.StringVar(&opts.StateDir, "state-dir", opts.StateDir, "directory for local CLI state")
	flags.StringVar(&opts.Output, "output", opts.Output, "output mode: text, json, ndjson")
}

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{
			"bash", "zsh", "fish", "powershell",
		},
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

func parseBootstrapArgs(args []string) (bootstrapOptions, error) {
	opts := bootstrapOptions{
		Config: mcpcli.DefaultConfig(),
		Output: string(mcpcli.OutputText),
	}
	for i := 0; i < len(args); i++ {
		name, value, hasValue := splitArg(args[i])
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
		case "--state-dir":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --state-dir")
				}
				value = args[i]
			}
			opts.StateDir = value
		case "--output":
			if !hasValue {
				i++
				if i >= len(args) {
					return opts, errors.New("missing value for --output")
				}
				value = args[i]
			}
			opts.Output = value
		}
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

func (a *app) session(ctx context.Context) (*mcpcli.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess != nil {
		return a.sess, nil
	}
	sess, err := mcpcli.Connect(ctx, a.cfg)
	if err != nil {
		return nil, err
	}
	a.sess = sess
	return sess, nil
}

func (a *app) stateStore() (*mcpcli.StateStore, error) {
	return mcpcli.OpenStateStore(a.cfg.StateDir)
}

func (a *app) close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess != nil {
		_ = a.sess.Close()
		a.sess = nil
	}
}

func cmdContext(cmd *cobra.Command, d time.Duration) (context.Context, context.CancelFunc) {
	if d == 0 {
		return context.WithCancel(cmd.Context())
	}
	return context.WithTimeout(cmd.Context(), d)
}

func newInspectCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect server metadata and negotiated capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			init := sess.InitializeResult()
			if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
				data, err := json.MarshalIndent(init, "", "  ")
				if err != nil {
					return err
				}
				return mcpcli.WriteOutput("", data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Server: %s %s\n", init.ServerInfo.Name, init.ServerInfo.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Protocol: %s\n", init.ProtocolVersion)
			if init.Instructions != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Instructions: %s\n", init.Instructions)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Capabilities:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  tools: %v\n", init.Capabilities.Tools != nil)
			fmt.Fprintf(cmd.OutOrStdout(), "  resources: %v\n", init.Capabilities.Resources != nil)
			fmt.Fprintf(cmd.OutOrStdout(), "  prompts: %v\n", init.Capabilities.Prompts != nil)
			fmt.Fprintf(cmd.OutOrStdout(), "  logging: %v\n", init.Capabilities.Logging != nil)
			fmt.Fprintf(cmd.OutOrStdout(), "  completions: %v\n", init.Capabilities.Completions != nil)
			fmt.Fprintf(cmd.OutOrStdout(), "  tasks: %v\n", init.Capabilities.Tasks != nil)
			return nil
		},
	}
}
