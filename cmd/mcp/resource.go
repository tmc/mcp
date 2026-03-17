package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
)

func newResourceCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "Inspect and read MCP resources",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "ls",
			Short: "List resources",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
				defer cancel()
				sess, err := a.session(ctx)
				if err != nil {
					return err
				}
				resources, err := sess.ListResourcesAll(ctx)
				if err != nil {
					return err
				}
				if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
					data, err := json.MarshalIndent(resources, "", "  ")
					if err != nil {
						return err
					}
					return mcpcli.WriteOutput("", data)
				}
				for _, resource := range resources {
					if resource.Description != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", resource.URI, resource.Description)
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), resource.URI)
					}
				}
				return nil
			},
		},
		newResourceCatCommand(a),
		newResourceWatchCommand(a),
	)
	return cmd
}

func newResourceCatCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "cat <uri>",
		Short: "Read a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			result, err := sess.Client().ReadResource(ctx, mcp.ReadResourceRequest{URI: args[0]})
			if err != nil {
				return err
			}
			data, err := mcpcli.RenderResourceResult(result, a.output)
			if err != nil {
				return err
			}
			return mcpcli.WriteOutput("", data)
		},
	}
}

func newResourceWatchCommand(a *app) *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "watch <uri>",
		Short: "Watch a resource for changes by polling",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			var last []byte
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				result, err := sess.Client().ReadResource(ctx, mcp.ReadResourceRequest{URI: args[0]})
				if err != nil {
					return err
				}
				data, err := mcpcli.RenderResourceResult(result, a.output)
				if err != nil {
					return err
				}
				if !bytes.Equal(last, data) {
					if a.output == mcpcli.OutputNDJSON {
						event, err := json.Marshal(map[string]any{
							"time": time.Now().Format(time.RFC3339),
							"uri":  args[0],
							"data": json.RawMessage(data),
						})
						if err != nil {
							return err
						}
						if err := mcpcli.WriteOutput("", event); err != nil {
							return err
						}
					} else if err := mcpcli.WriteOutput("", data); err != nil {
						return err
					}
					last = append(last[:0], data...)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "polling interval")
	return cmd
}
