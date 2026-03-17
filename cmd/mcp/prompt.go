package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
)

func newPromptCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Inspect and render MCP prompts",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "ls",
			Short: "List prompts",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
				defer cancel()
				sess, err := a.session(ctx)
				if err != nil {
					return err
				}
				prompts, err := sess.ListPromptsAll(ctx)
				if err != nil {
					return err
				}
				if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
					data, err := json.MarshalIndent(prompts, "", "  ")
					if err != nil {
						return err
					}
					return mcpcli.WriteOutput("", data)
				}
				for _, prompt := range prompts {
					if prompt.Description != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", prompt.Name, prompt.Description)
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), prompt.Name)
					}
				}
				return nil
			},
		},
		newPromptGetCommand(a, false),
		newPromptGetCommand(a, true),
	)
	return cmd
}

func newPromptGetCommand(a *app, render bool) *cobra.Command {
	var argJSON string
	var useEditor bool
	cmd := &cobra.Command{
		Use:   "get <name> [key=value...]",
		Short: "Retrieve a prompt",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if render {
				cmd.Use = "render <name> [key=value...]"
			}
			promptArgs, err := parsePromptArgs(args[1:], argJSON, useEditor)
			if err != nil {
				return err
			}
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			result, err := sess.Client().GetPrompt(ctx, mcp.GetPromptRequest{
				Name:      args[0],
				Arguments: promptArgs,
			})
			if err != nil {
				return err
			}
			mode := a.output
			if render && mode == mcpcli.OutputText {
				mode = mcpcli.OutputText
			}
			data, err := mcpcli.RenderPromptResult(result, mode)
			if err != nil {
				return err
			}
			return mcpcli.WriteOutput("", data)
		},
	}
	if render {
		cmd.Use = "render <name> [key=value...]"
		cmd.Short = "Render a prompt as terminal-friendly text"
	} else {
		cmd.Use = "get <name> [key=value...]"
		cmd.Short = "Get a prompt result"
	}
	cmd.Flags().StringVar(&argJSON, "arg-json", "", "JSON object of prompt arguments")
	cmd.Flags().BoolVar(&useEditor, "editor", false, "edit prompt arguments as JSON in $EDITOR")
	return cmd
}

func parsePromptArgs(parts []string, argJSON string, useEditor bool) (map[string]interface{}, error) {
	args := make(map[string]interface{})
	if argJSON != "" {
		if err := json.Unmarshal([]byte(argJSON), &args); err != nil {
			return nil, err
		}
	}
	if useEditor {
		raw, err := mcpcli.EditTempFile([]byte("{\n  \n}\n"), ".json")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid argument %q", part)
		}
		args[key] = parseScalar(value)
	}
	return args, nil
}

func parseScalar(value string) any {
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return value
}
