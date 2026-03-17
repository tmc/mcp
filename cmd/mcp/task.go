package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
)

func newTaskCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage durable MCP tasks",
	}
	cmd.AddCommand(
		newTaskSubmitCommand(a),
		newTaskPSCommand(a),
		newTaskGetCommand(a),
		newTaskResultCommand(a),
		newTaskCancelCommand(a),
		newTaskWatchCommand(a),
	)
	return cmd
}

func newTaskSubmitCommand(a *app) *cobra.Command {
	var rawArgs string
	var ttl int64
	cmd := &cobra.Command{
		Use:   "submit <tool> [key=value...]",
		Short: "Submit a tool call as a durable task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolArgs, err := parsePromptArgs(args[1:], rawArgs, false)
			if err != nil {
				return err
			}
			params := map[string]any{
				"name":      args[0],
				"arguments": toolArgs,
			}
			if ttl > 0 {
				params["task"] = map[string]any{"ttl": ttl}
			}
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			raw, err := sess.Client().CallRaw(ctx, string(mcp.MethodToolsCall), params)
			if err != nil {
				return err
			}
			return mcpcli.WriteOutput("", raw)
		},
	}
	cmd.Flags().StringVar(&rawArgs, "json", "", "JSON object of tool arguments")
	cmd.Flags().Int64Var(&ttl, "ttl", 0, "task time-to-live in seconds")
	return cmd
}

func newTaskPSCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "ps",
		Aliases: []string{"list"},
		Short:   "List tasks",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			var result mcp.ListTasksResult
			if err := sess.CallRaw(ctx, string(mcp.MethodTasksList), mcp.ListTasksRequest{}, &result); err != nil {
				return err
			}
			if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				return mcpcli.WriteOutput("", data)
			}
			for _, task := range result.Tasks {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", task.TaskID, task.Status, task.StatusMessage)
			}
			return nil
		},
	}
}

func newTaskGetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "get <task-id>",
		Short: "Get task status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeTaskStatus(cmd, a, args[0])
		},
	}
}

func newTaskResultCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "result <task-id>",
		Short: "Fetch task result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			raw, err := sess.Client().CallRaw(ctx, string(mcp.MethodTasksResult), mcp.GetTaskRequest{TaskID: args[0]})
			if err != nil {
				return err
			}
			return mcpcli.WriteOutput("", raw)
		},
	}
}

func newTaskCancelCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			var result map[string]any
			if err := sess.CallRaw(ctx, string(mcp.MethodTasksCancel), mcp.CancelTaskRequest{TaskID: args[0]}, &result); err != nil {
				return err
			}
			if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				return mcpcli.WriteOutput("", data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "cancelled %s\n", args[0])
			return nil
		},
	}
}

func newTaskWatchCommand(a *app) *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "watch <task-id>",
		Short: "Watch task status until completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			var last string
			for {
				status, err := taskStatus(ctx, a, args[0])
				if err != nil {
					return err
				}
				if data, _ := json.Marshal(status); string(data) != last {
					if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
						if err := mcpcli.WriteOutput("", data); err != nil {
							return err
						}
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", status.TaskID, status.Status, status.StatusMessage)
					}
					last = string(data)
				}
				if isTerminalTaskStatus(status.Status) {
					return nil
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

func writeTaskStatus(cmd *cobra.Command, a *app, id string) error {
	status, err := taskStatus(cmd.Context(), a, id)
	if err != nil {
		return err
	}
	if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		return mcpcli.WriteOutput("", data)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", status.TaskID, status.Status, status.StatusMessage)
	return nil
}

func taskStatus(ctx context.Context, a *app, id string) (*mcp.TaskInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()
	sess, err := a.session(ctx)
	if err != nil {
		return nil, err
	}
	var status mcp.TaskInfo
	if err := sess.CallRaw(ctx, string(mcp.MethodTasksGet), mcp.GetTaskRequest{TaskID: id}, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "input_required":
		return true
	default:
		return false
	}
}
