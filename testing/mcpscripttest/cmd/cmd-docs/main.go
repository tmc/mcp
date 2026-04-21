// cmd-docs analyzes mcpscripttest custom commands and generates documentation suggestions.
//
// cmd-docs discovers all custom commands registered in a codebase, analyzes their
// documentation completeness, and generates improvement suggestions. It supports
// multiple output formats (text, JSON, structured edits) for different use cases.
//
// Usage:
//
//	cmd-docs [options]
//
// Examples:
//
//	# Analyze commands in current directory and print text documentation
//	cmd-docs
//
//	# Generate JSON output for programmatic processing
//	cmd-docs -format json -output commands.json
//
//	# Focus on specific command
//	cmd-docs -cmd "mcp-server-start"
//
//	# Generate structured edit suggestions for documentation improvements
//	cmd-docs -format edits -output improvements.json
//
// The tool identifies command registrations in Go source files using AST parsing:
//
//	e.Cmds["command-name"] = handlerFunc
//	e.Commands["command-name"] = handlerFunc
//
// Documentation is inferred from function names and patterns, with improvement
// suggestions generated for gaps in coverage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	sourceDir  string        // Directory to analyze for Go source files
	outputFile string        // Output file path (empty = stdout)
	format     string        // Output format: text, json, or edits
	verbose    bool          // Enable verbose error reporting during analysis
	structEdit bool          // Enable structured edit suggestion output
	filterCmd  string        // Filter analysis to specific command by name
)

func init() {
	flag.StringVar(&sourceDir, "source", ".", "Source directory to analyze")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "text", "Output format: text, json, edits")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&structEdit, "structured", false, "Output structured edit suggestions")
	flag.StringVar(&filterCmd, "cmd", "", "Filter to specific command")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "cmd-docs - Generate documentation for mcpscripttest commands\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  cmd-docs [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Analyze commands in current directory\n")
		fmt.Fprintf(os.Stderr, "  cmd-docs\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Generate structured edits\n")
		fmt.Fprintf(os.Stderr, "  cmd-docs -structured -format edits\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Document specific command\n")
		fmt.Fprintf(os.Stderr, "  cmd-docs -cmd mcp-server-start\n")
	}
}

// Command represents a discovered custom command in the codebase.
//
// Each command corresponds to a registration in the source code and includes
// documentation, arguments, examples, and suggestions for improvement.
type Command struct {
	Name         string       `json:"name"`         // Command name (e.g., "mcp-server-start")
	File         string       `json:"file"`         // Source file containing the registration
	Line         int          `json:"line"`         // Line number in source file
	Function     string       `json:"function"`     // Handler function name
	Description  string       `json:"description"` // Documentation description
	Usage        string       `json:"usage"`       // Usage string
	Arguments    []Argument   `json:"arguments"`   // Command arguments
	Examples     []Example    `json:"examples"`    // Usage examples
	Registration Registration `json:"registration"` // Registration details
	Suggestions  []Suggestion `json:"suggestions"` // Improvement suggestions
}

// Argument represents a single command argument with documentation.
type Argument struct {
	Name        string `json:"name"`        // Argument name
	Type        string `json:"type"`        // Argument type (string, int, bool, etc.)
	Required    bool   `json:"required"`    // Whether argument is required
	Description string `json:"description"` // Argument documentation
}

// Example represents a command usage example with explanation.
type Example struct {
	Command     string `json:"command"`     // Example command line
	Description string `json:"description"` // Explanation of the example
}

// Registration contains metadata about how a command was registered.
type Registration struct {
	Type     string `json:"type"`     // Registration type ("Cmds", "Commands", etc.)
	Location string `json:"location"` // File and line location of registration
}

// Suggestion represents a documentation improvement suggestion.
//
// Suggestions are generated for commands with incomplete documentation,
// missing examples, or undocumented arguments.
type Suggestion struct {
	Type      string `json:"type"`      // Suggestion type (e.g., "documentation")
	Field     string `json:"field"`     // Which field needs improvement (description, arguments, etc.)
	Current   string `json:"current"`   // Current value or "none" if missing
	Suggested string `json:"suggested"` // Suggested improvement
	Reason    string `json:"reason"`    // Why this suggestion is needed
}

type Edit struct {
	File        string `json:"file"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	OldText     string `json:"old_text"`
	NewText     string `json:"new_text"`
	Description string `json:"description"`
}

func main() {
	flag.Parse()

	// Find commands
	commands, err := findCommands(sourceDir)
	if err != nil {
		log.Fatalf("Error finding commands: %v", err)
	}

	// Filter if requested
	if filterCmd != "" {
		filtered := []Command{}
		for _, cmd := range commands {
			if cmd.Name == filterCmd {
				filtered = append(filtered, cmd)
			}
		}
		commands = filtered
	}

	// Analyze and suggest improvements
	for i := range commands {
		analyzeCommand(&commands[i])
		generateSuggestions(&commands[i])
	}

	// Sort by name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	// Output
	var out io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	switch format {
	case "json":
		outputJSON(out, commands)
	case "edits":
		outputEdits(out, commands)
	default:
		outputText(out, commands)
	}
}

// findCommands recursively searches the given directory for all custom command
// registrations in Go source files.
//
// It walks the directory tree, parses each .go file (excluding _test.go files),
// and extracts all command registrations. If verbose is enabled, parse errors
// are logged but don't stop the search.
func findCommands(dir string) ([]Command, error) {
	var commands []Command

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "_test.go") {
			cmds, err := parseFile(path)
			if err != nil {
				if verbose {
					log.Printf("Error parsing %s: %v", path, err)
				}
				return nil
			}
			commands = append(commands, cmds...)
		}

		return nil
	})

	return commands, err
}

// parseFile parses a single Go source file and extracts all command registrations.
//
// It identifies command registrations in two patterns:
//   - e.Cmds["command-name"] = handler
//   - e.Commands["command-name"] = handler
//
// The parser extracts the command name, handler function, file location, and
// registration type. It uses Go's AST to ensure accurate parsing.
func parseFile(filename string) ([]Command, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var commands []Command

	// Look for command registrations in assignment statements
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			// Look for e.Cmds["name"] = ... or e.Commands["name"] = ...
			if len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if indexExpr, ok := x.Lhs[0].(*ast.IndexExpr); ok {
					if selExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
						if selExpr.Sel.Name == "Cmds" || selExpr.Sel.Name == "Commands" {
							if lit, ok := indexExpr.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING {
								cmdName := strings.Trim(lit.Value, `"`)
								cmd := Command{
									Name: cmdName,
									File: filename,
									Line: fset.Position(x.Pos()).Line,
									Registration: Registration{
										Type:     selExpr.Sel.Name,
										Location: fset.Position(x.Pos()).String(),
									},
								}

								// Extract function info - can be direct identifier or function call
								if ident, ok := x.Rhs[0].(*ast.Ident); ok {
									cmd.Function = ident.Name
								} else if callExpr, ok := x.Rhs[0].(*ast.CallExpr); ok {
									if ident, ok := callExpr.Fun.(*ast.Ident); ok {
										cmd.Function = ident.Name
									}
								}

								commands = append(commands, cmd)
							}
						}
					}
				}
			}
		}
		return true
	})

	return commands, nil
}

// analyzeCommand performs automatic documentation inference for a command.
//
// It infers description, usage, arguments, and examples based on the command
// name and common patterns. This provides a baseline for documentation that
// can be improved by explicit documentation in source code.
func analyzeCommand(cmd *Command) {
	cmd.Description = inferDescription(cmd)
	cmd.Usage = inferUsage(cmd)
	cmd.Arguments = inferArguments(cmd)
	cmd.Examples = findExamples(cmd)
}

func inferDescription(cmd *Command) string {
	// Try to infer from function name
	name := cmd.Function
	if strings.HasSuffix(name, "Cmd") {
		name = strings.TrimSuffix(name, "Cmd")
	}

	// Convert camelCase to sentence
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	desc := re.ReplaceAllString(name, `$1 $2`)
	desc = strings.ToLower(desc)

	// Common patterns
	switch {
	case strings.Contains(desc, "server start"):
		return "Start a server process"
	case strings.Contains(desc, "send"):
		return "Send data to a process"
	case strings.Contains(desc, "verify"):
		return "Verify output or behavior"
	case strings.Contains(desc, "test"):
		return "Run a test"
	default:
		return fmt.Sprintf("Execute %s operation", desc)
	}
}

func inferUsage(cmd *Command) string {
	// Infer usage from command name
	switch {
	case strings.HasPrefix(cmd.Name, "mcp-server-"):
		return fmt.Sprintf("%s <server-name> [options]", cmd.Name)
	case strings.HasPrefix(cmd.Name, "mcp-"):
		return fmt.Sprintf("%s [options] <args>", cmd.Name)
	default:
		return fmt.Sprintf("%s <args>", cmd.Name)
	}
}

func inferArguments(cmd *Command) []Argument {
	// Common patterns
	var args []Argument

	switch {
	case strings.Contains(cmd.Name, "server"):
		args = append(args, Argument{
			Name:        "server-name",
			Type:        "string",
			Required:    true,
			Description: "Name of the server",
		})
	case strings.Contains(cmd.Name, "send"):
		args = append(args, Argument{
			Name:        "data",
			Type:        "string",
			Required:    true,
			Description: "Data to send",
		})
	}

	return args
}

func findExamples(cmd *Command) []Example {
	// Generate basic examples
	var examples []Example

	switch {
	case cmd.Name == "mcp-server-start":
		examples = append(examples, Example{
			Command:     "mcp-server-start myserver -- go run server.go",
			Description: "Start a Go server",
		})
	case strings.HasPrefix(cmd.Name, "mcp-"):
		examples = append(examples, Example{
			Command:     fmt.Sprintf("%s --help", cmd.Name),
			Description: "Show help information",
		})
	}

	return examples
}

// generateSuggestions analyzes a command's documentation and generates
// improvement suggestions for missing or incomplete documentation.
//
// Suggestions are created for:
//   - Missing or generic descriptions
//   - Undocumented command arguments
//   - Missing usage examples
//
// These suggestions help prioritize documentation efforts and ensure
// comprehensive command coverage.
func generateSuggestions(cmd *Command) {
	// Generate improvement suggestions for missing or generic descriptions
	if cmd.Description == "" || strings.Contains(cmd.Description, "Execute") {
		cmd.Suggestions = append(cmd.Suggestions, Suggestion{
			Type:      "documentation",
			Field:     "description",
			Current:   cmd.Description,
			Suggested: fmt.Sprintf("TODO: Add detailed description for %s", cmd.Name),
			Reason:    "Missing or generic description",
		})
	}

	// Suggest documenting command arguments if none were found
	if len(cmd.Arguments) == 0 && !strings.HasSuffix(cmd.Name, "-help") {
		cmd.Suggestions = append(cmd.Suggestions, Suggestion{
			Type:      "documentation",
			Field:     "arguments",
			Current:   "none",
			Suggested: "TODO: Document command arguments",
			Reason:    "Commands typically have arguments that should be documented",
		})
	}

	// Suggest adding examples to commands without them
	if len(cmd.Examples) == 0 {
		cmd.Suggestions = append(cmd.Suggestions, Suggestion{
			Type:      "documentation",
			Field:     "examples",
			Current:   "none",
			Suggested: fmt.Sprintf("%s <example-usage>", cmd.Name),
			Reason:    "Examples help users understand usage",
		})
	}
}

// outputText formats commands as human-readable text documentation.
//
// This is the default output format, suitable for reading in a terminal or
// including in documentation. It displays all command information including
// descriptions, usage, arguments, examples, and improvement suggestions.
func outputText(w io.Writer, commands []Command) {
	fmt.Fprintf(w, "=== MCPScriptTest Commands Documentation ===\n\n")

	for _, cmd := range commands {
		fmt.Fprintf(w, "## %s\n", cmd.Name)
		fmt.Fprintf(w, "File: %s:%d\n", cmd.File, cmd.Line)
		fmt.Fprintf(w, "Function: %s\n", cmd.Function)
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "Description: %s\n", cmd.Description)
		fmt.Fprintf(w, "Usage: %s\n", cmd.Usage)

		if len(cmd.Arguments) > 0 {
			fmt.Fprintf(w, "\nArguments:\n")
			for _, arg := range cmd.Arguments {
				req := ""
				if arg.Required {
					req = " (required)"
				}
				fmt.Fprintf(w, "  - %s: %s%s - %s\n", arg.Name, arg.Type, req, arg.Description)
			}
		}

		if len(cmd.Examples) > 0 {
			fmt.Fprintf(w, "\nExamples:\n")
			for _, ex := range cmd.Examples {
				fmt.Fprintf(w, "  %s\n", ex.Command)
				fmt.Fprintf(w, "    # %s\n", ex.Description)
			}
		}

		if len(cmd.Suggestions) > 0 {
			fmt.Fprintf(w, "\nSuggestions:\n")
			for _, sug := range cmd.Suggestions {
				fmt.Fprintf(w, "  - %s: %s\n", sug.Field, sug.Reason)
				fmt.Fprintf(w, "    Current: %s\n", sug.Current)
				fmt.Fprintf(w, "    Suggested: %s\n", sug.Suggested)
			}
		}

		fmt.Fprintf(w, "\n---\n\n")
	}

	fmt.Fprintf(w, "Total commands found: %d\n", len(commands))
}

// outputJSON formats commands as JSON for programmatic processing.
//
// This format is suitable for tools, APIs, and integration with other systems.
// Each command is serialized with full details including suggestions.
func outputJSON(w io.Writer, commands []Command) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(commands)
}

// outputEdits generates structured edit suggestions for documentation improvements.
//
// This format is designed for automated application of documentation updates
// to source files. Each edit includes the file, line numbers, and suggested
// documentation block to add.
func outputEdits(w io.Writer, commands []Command) {
	var edits []Edit

	// Generate edits for each command
	for _, cmd := range commands {
		if len(cmd.Suggestions) > 0 {
			// Generate a documentation block edit
			docBlock := generateDocBlock(cmd)

			edit := Edit{
				File:        cmd.File,
				StartLine:   cmd.Line - 1, // Insert before command registration
				EndLine:     cmd.Line - 1,
				OldText:     "",
				NewText:     docBlock,
				Description: fmt.Sprintf("Add documentation for %s command", cmd.Name),
			}

			edits = append(edits, edit)
		}
	}

	// Sort edits by file and line
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].File != edits[j].File {
			return edits[i].File < edits[j].File
		}
		return edits[i].StartLine < edits[j].StartLine
	})

	// Output as JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(edits)
}

func generateDocBlock(cmd Command) string {
	tmpl := `// {{.Name}} {{.Description}}
// Usage: {{.Usage}}
{{if .Arguments}}// Arguments:
{{range .Arguments}}//   {{.Name}} ({{.Type}}){{if .Required}} - required{{end}}: {{.Description}}
{{end}}{{end}}{{if .Examples}}// Examples:
{{range .Examples}}//   {{.Command}}
{{end}}{{end}}`

	t := template.Must(template.New("doc").Parse(tmpl))
	var buf strings.Builder
	t.Execute(&buf, cmd)
	return buf.String()
}
