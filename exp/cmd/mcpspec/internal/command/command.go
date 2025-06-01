// Package command defines the core command interface for the MCP tools.
package command

import (
	"context"
	"io"
)

// Command defines the interface that all MCP commands must implement.
type Command interface {
	// Name returns the name of the command.
	Name() string

	// Execute runs the command with the given context and arguments.
	Execute(ctx context.Context, args []string) error

	// Usage returns a string describing how to use the command.
	Usage() string
}

// BaseCommand provides common functionality for MCP commands.
type BaseCommand struct {
	// Name of the command
	CommandName string

	// Standard input for the command
	Input io.Reader

	// Standard output for the command
	Output io.Writer

	// Standard error for the command
	Error io.Writer

	// Usage documentation
	UsageText string

	// Verbose mode flag
	Verbose bool
}

// Name returns the name of the command.
func (c *BaseCommand) Name() string {
	return c.CommandName
}

// Usage returns the usage documentation for the command.
func (c *BaseCommand) Usage() string {
	return c.UsageText
}
