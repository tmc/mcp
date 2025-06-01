package internal

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Command represents a scripttest command with its documentation
type Command struct {
	Name        string
	Usage       string
	Description string
	Examples    []string
}

// Schema represents the complete schema of available scripttest commands
type Schema struct {
	CoreCommands []Command
	MCPCommands  []Command
	Directives   []Command
}

// GetSchema returns the complete schema of available scripttest commands
func GetSchema() *Schema {
	return &Schema{
		CoreCommands: []Command{
			{
				Name:  "exec",
				Usage: "exec [command] [args...]",
				Description: "Execute a command and capture its output",
				Examples: []string{
					"exec echo hello",
					"exec go test -v",
					"exec mcp-serve -- go run server.go",
				},
			},
			{
				Name:  "stdin",
				Usage: "stdin [content]",
				Description: "Provide input to the next exec command",
				Examples: []string{
					"stdin {\"jsonrpc\":\"2.0\",\"method\":\"initialize\",\"id\":1}",
					"stdin hello world",
				},
			},
			{
				Name:  "stdout",
				Usage: "stdout [expected content]",
				Description: "Assert that the previous command's stdout contains the expected content",
				Examples: []string{
					"stdout \"result\"",
					"stdout jsonrpc",
				},
			},
			{
				Name:  "stderr",
				Usage: "stderr [expected content]",
				Description: "Assert that the previous command's stderr contains the expected content",
				Examples: []string{
					"stderr error:",
					"stderr 'failed to connect'",
				},
			},
			{
				Name:  "cat",
				Usage: "cat [filename]",
				Description: "Display the contents of a file",
				Examples: []string{
					"cat output.json",
					"cat server.log",
				},
			},
			{
				Name:  "cp",
				Usage: "cp [source] [destination]",
				Description: "Copy a file from source to destination",
				Examples: []string{
					"cp template.json test.json",
					"cp config.yaml backup.yaml",
				},
			},
			{
				Name:  "mv",
				Usage: "mv [source] [destination]",
				Description: "Move/rename a file from source to destination",
				Examples: []string{
					"mv old.txt new.txt",
					"mv temp.log final.log",
				},
			},
			{
				Name:  "rm",
				Usage: "rm [filename]",
				Description: "Remove a file",
				Examples: []string{
					"rm temp.txt",
					"rm *.log",
				},
			},
			{
				Name:  "mkdir",
				Usage: "mkdir [directory]",
				Description: "Create a directory",
				Examples: []string{
					"mkdir output",
					"mkdir -p logs/test",
				},
			},
			{
				Name:  "cd",
				Usage: "cd [directory]",
				Description: "Change the current directory",
				Examples: []string{
					"cd testdir",
					"cd ..",
				},
			},
			{
				Name:  "env",
				Usage: "env [NAME=value]",
				Description: "Set an environment variable",
				Examples: []string{
					"env GOCOVERDIR=/tmp/coverage",
					"env DEBUG=1",
				},
			},
			{
				Name:  "cmp",
				Usage: "cmp [file1] [file2]",
				Description: "Compare two files for equality",
				Examples: []string{
					"cmp expected.json actual.json",
					"cmp output.txt golden.txt",
				},
			},
			{
				Name:  "grep",
				Usage: "grep [pattern] [file]",
				Description: "Search for a pattern in a file",
				Examples: []string{
					"grep error server.log",
					"grep -v debug output.txt",
				},
			},
			{
				Name:  "sleep",
				Usage: "sleep [duration]",
				Description: "Sleep for the specified duration",
				Examples: []string{
					"sleep 1s",
					"sleep 500ms",
				},
			},
			{
				Name:  "wait",
				Usage: "wait [process]",
				Description: "Wait for a background process to complete",
				Examples: []string{
					"wait $server",
					"wait",
				},
			},
		},
		MCPCommands: []Command{
			{
				Name:  "mcp-send",
				Usage: "mcp-send [json-rpc message]",
				Description: "Send a JSON-RPC message to an MCP server",
				Examples: []string{
					`mcp-send {"jsonrpc":"2.0","method":"initialize","id":1}`,
					`mcp-send {"jsonrpc":"2.0","method":"tools/list","id":2}`,
				},
			},
			{
				Name:  "mcp-recv",
				Usage: "mcp-recv [expected pattern]",
				Description: "Receive and assert on a JSON-RPC response",
				Examples: []string{
					`mcp-recv "result"`,
					`mcp-recv {"id":1}`,
				},
			},
			{
				Name:  "mcp-serve",
				Usage: "mcp-serve -- [server command]",
				Description: "Start an MCP server and capture its communication",
				Examples: []string{
					"mcp-serve -- go run server.go",
					"mcp-serve -- node server.js",
				},
			},
			{
				Name:  "mcp-trace",
				Usage: "mcp-trace [output.mcp]",
				Description: "Enable MCP trace recording to the specified file",
				Examples: []string{
					"mcp-trace server.mcp",
					"mcp-trace test-run.mcp",
				},
			},
		},
		Directives: []Command{
			{
				Name:  "!",
				Usage: "! [command]",
				Description: "Negate the expected result (expect command to fail)",
				Examples: []string{
					"! exec false",
					"! exec test -f missing.txt",
				},
			},
			{
				Name:  "?",
				Usage: "? [command]",
				Description: "Ignore the exit status of the command",
				Examples: []string{
					"? exec grep pattern file.txt",
					"? exec rm temp.txt",
				},
			},
			{
				Name:  "#",
				Usage: "# [comment]",
				Description: "Add a comment (ignored by the test runner)",
				Examples: []string{
					"# This is a test comment",
					"# Setup phase",
				},
			},
			{
				Name:  "skip",
				Usage: "skip [condition]",
				Description: "Skip the test based on a condition",
				Examples: []string{
					"skip windows",
					"skip !linux",
				},
			},
			{
				Name:  "[condition]",
				Usage: "[condition] [command]",
				Description: "Conditionally execute a command",
				Examples: []string{
					"[linux] exec chmod +x script.sh",
					"[!windows] exec ./script.sh",
				},
			},
		},
	}
}

// DumpSchema outputs the schema in a readable format
func DumpSchema(w io.Writer, s *Schema) {
	fmt.Fprintln(w, "MCPScriptTest Schema")
	fmt.Fprintln(w, "===================")
	fmt.Fprintln(w)

	// Core Commands
	fmt.Fprintln(w, "Core Commands")
	fmt.Fprintln(w, "-------------")
	dumpCommands(w, s.CoreCommands)
	fmt.Fprintln(w)

	// MCP Commands
	fmt.Fprintln(w, "MCP Commands")
	fmt.Fprintln(w, "------------")
	dumpCommands(w, s.MCPCommands)
	fmt.Fprintln(w)

	// Directives
	fmt.Fprintln(w, "Directives")
	fmt.Fprintln(w, "----------")
	dumpCommands(w, s.Directives)
}

func dumpCommands(w io.Writer, commands []Command) {
	// Sort commands by name for consistent output
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	for _, cmd := range commands {
		fmt.Fprintf(w, "### %s\n", cmd.Name)
		fmt.Fprintf(w, "Usage: %s\n", cmd.Usage)
		fmt.Fprintf(w, "Description: %s\n", cmd.Description)
		if len(cmd.Examples) > 0 {
			fmt.Fprintln(w, "Examples:")
			for _, ex := range cmd.Examples {
				fmt.Fprintf(w, "  %s\n", ex)
			}
		}
		fmt.Fprintln(w)
	}
}

// GenerateMarkdown generates a Markdown documentation of the schema
func GenerateMarkdown(s *Schema) string {
	var b strings.Builder
	
	b.WriteString("# MCPScriptTest Command Reference\n\n")
	b.WriteString("This document describes all available commands and directives in MCPScriptTest.\n\n")
	
	b.WriteString("## Core Commands\n\n")
	b.WriteString("These are the standard scripttest commands available in all tests.\n\n")
	for _, cmd := range s.CoreCommands {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", cmd.Name))
		b.WriteString(fmt.Sprintf("**Usage:** `%s`\n\n", cmd.Usage))
		b.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))
		if len(cmd.Examples) > 0 {
			b.WriteString("**Examples:**\n```\n")
			for _, ex := range cmd.Examples {
				b.WriteString(fmt.Sprintf("%s\n", ex))
			}
			b.WriteString("```\n\n")
		}
	}
	
	b.WriteString("## MCP-Specific Commands\n\n")
	b.WriteString("These commands are specific to testing MCP servers and clients.\n\n")
	for _, cmd := range s.MCPCommands {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", cmd.Name))
		b.WriteString(fmt.Sprintf("**Usage:** `%s`\n\n", cmd.Usage))
		b.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))
		if len(cmd.Examples) > 0 {
			b.WriteString("**Examples:**\n```\n")
			for _, ex := range cmd.Examples {
				b.WriteString(fmt.Sprintf("%s\n", ex))
			}
			b.WriteString("```\n\n")
		}
	}
	
	b.WriteString("## Directives and Modifiers\n\n")
	b.WriteString("These modify how commands are executed or control test flow.\n\n")
	for _, cmd := range s.Directives {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", cmd.Name))
		b.WriteString(fmt.Sprintf("**Usage:** `%s`\n\n", cmd.Usage))
		b.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))
		if len(cmd.Examples) > 0 {
			b.WriteString("**Examples:**\n```\n")
			for _, ex := range cmd.Examples {
				b.WriteString(fmt.Sprintf("%s\n", ex))
			}
			b.WriteString("```\n\n")
		}
	}
	
	return b.String()
}