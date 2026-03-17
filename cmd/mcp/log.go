package main

import (
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
)

func newLogCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Consume protocol logging and progress notifications",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "level <debug|info|warning|error>",
			Short: "Set protocol logging level",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
				defer cancel()
				sess, err := a.session(ctx)
				if err != nil {
					return err
				}
				return sess.Client().SetLoggingLevel(ctx, mcp.LoggingLevel(args[0]))
			},
		},
		newLogTailCommand(a),
	)
	return cmd
}

func newLogTailCommand(a *app) *cobra.Command {
	var level string
	var all bool
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Tail protocol notifications",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			if level != "" {
				if err := sess.Client().SetLoggingLevel(ctx, mcp.LoggingLevel(level)); err != nil {
					return err
				}
			}
			ch, unsubscribe := sess.Subscribe(64)
			defer unsubscribe()
			for {
				select {
				case <-ctx.Done():
					return nil
				case event := <-ch:
					if !all && event.Method != string(mcp.MethodLogging) && event.Method != string(mcp.MethodProgress) {
						continue
					}
					if a.output == mcpcli.OutputNDJSON || a.output == mcpcli.OutputJSON {
						data, err := json.Marshal(event)
						if err != nil {
							return err
						}
						if err := mcpcli.WriteOutput("", data); err != nil {
							return err
						}
						continue
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", event.Time.Format("15:04:05"), event.Method)
					if len(event.Params) > 0 {
						fmt.Fprintln(cmd.OutOrStdout(), string(event.Params))
					}
				}
			}
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "set logging level before tailing")
	cmd.Flags().BoolVar(&all, "all", false, "show all notifications, not just logs and progress")
	return cmd
}
