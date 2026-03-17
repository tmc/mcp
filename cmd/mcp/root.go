package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

func newRootCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Manage client roots exposed to MCP servers",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "ls",
			Short: "List configured roots",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				store, err := a.stateStore()
				if err != nil {
					return err
				}
				roots, err := store.List()
				if err != nil {
					return err
				}
				for _, root := range roots {
					if root.Name != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", root.URI, root.Name)
						continue
					}
					fmt.Fprintln(cmd.OutOrStdout(), root.URI)
				}
				return nil
			},
		},
		newRootAddCommand(a),
		&cobra.Command{
			Use:   "rm <root>",
			Short: "Remove a configured root",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				store, err := a.stateStore()
				if err != nil {
					return err
				}
				return store.RemoveRoot(rootURI(args[0]))
			},
		},
	)
	return cmd
}

func newRootAddCommand(a *app) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add <root>",
		Short: "Add a configured root",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.stateStore()
			if err != nil {
				return err
			}
			return store.AddRoot(mcp.Root{
				URI:  rootURI(args[0]),
				Name: name,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "optional root display name")
	return cmd
}

func rootURI(s string) string {
	if strings.Contains(s, "://") {
		return s
	}
	abs, err := filepath.Abs(s)
	if err != nil {
		return s
	}
	if _, err := os.Stat(abs); err != nil {
		return abs
	}
	return (&url.URL{Scheme: "file", Path: abs}).String()
}
