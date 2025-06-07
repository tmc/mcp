// cmd-docs analyzes mcpscripttest custom commands and generates documentation suggestions
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
	sourceDir  string
	outputFile string
	format     string
	verbose    bool
	structEdit bool
	filterCmd  string
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

type Command struct {
	Name         string       `json:"name"`
	File         string       `json:"file"`
	Line         int          `json:"line"`
	Function     string       `json:"function"`
	Description  string       `json:"description"`
	Usage        string       `json:"usage"`
	Arguments    []Argument   `json:"arguments"`
	Examples     []Example    `json:"examples"`
	Registration Registration `json:"registration"`
	Suggestions  []Suggestion `json:"suggestions"`
}

type Argument struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type Example struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type Registration struct {
	Type     string `json:"type"` // "Cmds", "Commands", etc.
	Location string `json:"location"`
}

type Suggestion struct {
	Type      string `json:"type"`
	Field     string `json:"field"`
	Current   string `json:"current"`
	Suggested string `json:"suggested"`
	Reason    string `json:"reason"`
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

func parseFile(filename string) ([]Command, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var commands []Command

	// Look for command registrations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			// Look for e.Cmds["name"] = ...
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

								// Extract function info
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

func analyzeCommand(cmd *Command) {
	// Try to find documentation from comments
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

func generateSuggestions(cmd *Command) {
	// Generate improvement suggestions
	if cmd.Description == "" || strings.Contains(cmd.Description, "Execute") {
		cmd.Suggestions = append(cmd.Suggestions, Suggestion{
			Type:      "documentation",
			Field:     "description",
			Current:   cmd.Description,
			Suggested: fmt.Sprintf("TODO: Add detailed description for %s", cmd.Name),
			Reason:    "Missing or generic description",
		})
	}

	if len(cmd.Arguments) == 0 && !strings.HasSuffix(cmd.Name, "-help") {
		cmd.Suggestions = append(cmd.Suggestions, Suggestion{
			Type:      "documentation",
			Field:     "arguments",
			Current:   "none",
			Suggested: "TODO: Document command arguments",
			Reason:    "Commands typically have arguments that should be documented",
		})
	}

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

func outputJSON(w io.Writer, commands []Command) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(commands)
}

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
